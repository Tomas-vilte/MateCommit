package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/services/cost"
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
)

type StatsCommand struct{}

func NewStatsCommand() *StatsCommand {
	return &StatsCommand{}
}

func (c *StatsCommand) CreateCommand(t *i18n.Translations, _ *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "stats",
		Aliases: []string{"cost"},
		Usage:   t.GetMessage("stats.usage", 0, nil),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "monthly",
				Aliases: []string{"m"},
				Usage:   t.GetMessage("stats.monthly_flag", 0, nil),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			manager, err := cost.NewManager(0, t)
			if err != nil {
				return fmt.Errorf(t.GetMessage("stats.error_init", 0, nil)+": %w", err)
			}

			showMonthly := cmd.Bool("monthly")

			if showMonthly {
				return c.showMonthlyStats(manager, t)
			}
			return c.showDailyStats(manager, t)
		},
	}
}

func (c *StatsCommand) showDailyStats(manager *cost.Manager, t *i18n.Translations) error {
	total, err := manager.GetDailyTotal()
	if err != nil {
		return err
	}
	records, err := manager.GetHistory()
	if err != nil {
		return err
	}
	today := time.Now().Format("2006-01-02")
	var todayRecords []cost.ActivityRecord
	for _, r := range records {
		if r.Timestamp.Format("2006-01-02") == today {
			todayRecords = append(todayRecords, r)
		}
	}
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	_, _ = cyan.Printf("\n📊 %s\n", t.GetMessage("stats.daily_title", 0, nil))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(todayRecords) == 0 {
		fmt.Printf("\n%s\n\n", t.GetMessage("stats.no_activity", 0, nil))
		return nil
	}
	for _, record := range todayRecords {
		cacheIndicator := ""
		if record.CacheHit {
			cacheIndicator = green.Sprint(" [CACHE]")
		}
		fmt.Printf("%s - %s: %s%s\n",
			record.Timestamp.Format("15:04"),
			record.Command,
			yellow.Sprintf("$%.4f", record.CostUSD),
			cacheIndicator,
		)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = cyan.Printf("%s: ", t.GetMessage("stats.total_today", 0, nil))
	_, _ = yellow.Printf("$%.4f USD\n\n", total)
	return nil
}

func (c *StatsCommand) showMonthlyStats(manager *cost.Manager, t *i18n.Translations) error {
	total, err := manager.GetMonthlyTotal()
	if err != nil {
		return err
	}
	records, err := manager.GetHistory()
	if err != nil {
		return err
	}
	currentMonth := time.Now().Format("2006-01")
	var monthRecords []cost.ActivityRecord
	for _, r := range records {
		if r.Timestamp.Format("2006-01") == currentMonth {
			monthRecords = append(monthRecords, r)
		}
	}
	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	_, _ = cyan.Printf("\n📅 %s\n", t.GetMessage("stats.monthly_title", 0, map[string]interface{}{
		"Month": time.Now().Format("January 2006"),
	}))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(monthRecords) == 0 {
		fmt.Printf("\n%s\n\n", t.GetMessage("stats.no_activity", 0, nil))
		return nil
	}
	dailyTotals := make(map[string]float64)
	for _, record := range monthRecords {
		day := record.Timestamp.Format("2006-01-02")
		dailyTotals[day] += record.CostUSD
	}
	for day, dayTotal := range dailyTotals {
		fmt.Printf("%s: %s\n", day, yellow.Sprintf("$%.4f", dayTotal))
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = cyan.Printf("%s: ", t.GetMessage("stats.total_month", 0, nil))
	_, _ = yellow.Printf("$%.4f USD\n\n", total)
	return nil
}
