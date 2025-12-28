package stats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/services/cost"
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
			&cli.BoolFlag{
				Name:    "breakdown",
				Aliases: []string{"b"},
				Usage:   t.GetMessage("stats.breakdown_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "forecast",
				Aliases: []string{"f"},
				Usage:   t.GetMessage("stats.forecast_flag", 0, nil),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			manager, err := cost.NewManager(0)
			if err != nil {
				return fmt.Errorf(t.GetMessage("stats.error_init", 0, nil)+": %w", err)
			}

			showMonthly := cmd.Bool("monthly")
			showBreakdown := cmd.Bool("breakdown")
			showForecast := cmd.Bool("forecast")

			if showBreakdown {
				return c.showBreakdown(manager, t)
			}

			if showMonthly || showForecast {
				return c.showMonthlyStats(manager, t, showForecast)
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
	dim := color.New(color.FgHiBlack)

	_, _ = cyan.Printf("\n📊 %s - %s\n", t.GetMessage("stats.daily_title", 0, nil), today)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if len(todayRecords) == 0 {
		fmt.Printf("%s\n\n", t.GetMessage("stats.no_activity", 0, nil))
		_, _ = dim.Println(t.GetMessage("stats.tip_run_suggest", 0, nil))
		return nil
	}

	_, _ = dim.Println(t.GetMessage("stats.activity_log", 0, nil))
	for _, record := range todayRecords {
		cacheIndicator := ""
		if record.CacheHit {
			cacheIndicator = green.Sprint(" [CACHE]")
		}

		tokensInfo := dim.Sprintf("(%d→%d tok)", record.TokensInput, record.TokensOutput)

		fmt.Printf("%s - %s: %s %s%s\n",
			record.Timestamp.Format("15:04"),
			record.Command,
			yellow.Sprintf("$%.4f", record.CostUSD),
			tokensInfo,
			cacheIndicator,
		)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = cyan.Printf("%s: ", t.GetMessage("stats.total_today", 0, nil))
	_, _ = yellow.Printf("$%.4f USD\n", total)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	_, _ = dim.Println(t.GetMessage("stats.tip_monthly", 0, nil))
	_, _ = dim.Println(t.GetMessage("stats.tip_breakdown", 0, nil))
	fmt.Println()

	return nil
}

func (c *StatsCommand) showMonthlyStats(manager *cost.Manager, t *i18n.Translations, showForecast bool) error {
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
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	_, _ = cyan.Printf("\n📅 %s\n", t.GetMessage("stats.monthly_title", 0, struct{ Month string }{time.Now().Format("January 2006")}))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if len(monthRecords) == 0 {
		fmt.Printf("%s\n\n", t.GetMessage("stats.no_activity", 0, nil))
		return nil
	}

	dailyTotals := make(map[string]float64)
	maxDaily := 0.0
	for _, record := range monthRecords {
		day := record.Timestamp.Format("2006-01-02")
		dailyTotals[day] += record.CostUSD
		if dailyTotals[day] > maxDaily {
			maxDaily = dailyTotals[day]
		}
	}

	_, _ = dim.Println(t.GetMessage("stats.daily_totals_label", 0, nil))
	days := make([]string, 0, len(dailyTotals))
	for day := range dailyTotals {
		days = append(days, day)
	}
	for i := 0; i < len(days); i++ {
		for j := i + 1; j < len(days); j++ {
			if days[i] > days[j] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}

	for _, day := range days {
		dayTotal := dailyTotals[day]
		barLength := int((dayTotal / maxDaily) * 10)
		bar := strings.Repeat("█", barLength) + strings.Repeat("░", 10-barLength)

		fmt.Printf("%s: %s  %s\n",
			day,
			yellow.Sprintf("$%.4f", dayTotal),
			cyan.Sprint(bar),
		)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	_, _ = cyan.Printf("%s: ", t.GetMessage("stats.total_month", 0, nil))
	_, _ = yellow.Printf("$%.4f USD\n", total)

	daysWithActivity := len(dailyTotals)
	if daysWithActivity > 0 {
		avgPerDay := total / float64(daysWithActivity)
		_, _ = dim.Printf("%s: $%.4f USD\n", t.GetMessage("stats.average_per_day", 0, nil), avgPerDay)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	if showForecast {
		forecast, err := manager.GetForecast()
		if err == nil {
			_, _ = cyan.Println(t.GetMessage("stats.forecast_title", 0, nil))
			_, _ = dim.Printf("   %s\n", t.GetMessage("stats.forecast_days_label", 0, struct {
				Current int
				Total   int
			}{forecast.DaysElapsed, forecast.DaysInMonth}))
			_, _ = dim.Printf("   %s\n", t.GetMessage("stats.forecast_daily_avg_label", 0, struct{ Avg float64 }{forecast.DailyAverage}))
			_, _ = yellow.Printf("   %s\n", t.GetMessage("stats.forecast_projected_label", 0, struct{ Amount float64 }{forecast.ProjectedMonthEnd}))
			fmt.Println()
		}
	}

	hitRate, saved, err := manager.GetCacheStats()
	if err == nil && hitRate > 0 {
		_, _ = green.Println(t.GetMessage("stats.cache_hit_rate_label", 0, struct {
			Rate  float64
			Saved float64
		}{hitRate, saved}))
		fmt.Println()
	}

	if !showForecast {
		_, _ = dim.Println(t.GetMessage("stats.tip_forecast", 0, nil))
	}
	_, _ = dim.Println(t.GetMessage("stats.tip_breakdown", 0, nil))
	fmt.Println()

	return nil
}

func (c *StatsCommand) showBreakdown(manager *cost.Manager, t *i18n.Translations) error {
	breakdown, err := manager.GetBreakdownByCommand()
	if err != nil {
		return err
	}

	if breakdown.TotalCalls == 0 {
		fmt.Printf("\n%s\n\n", t.GetMessage("stats.no_activity", 0, nil))
		return nil
	}

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	dim := color.New(color.FgHiBlack)
	green := color.New(color.FgGreen)

	_, _ = cyan.Printf("\n📊 Usage Breakdown - %s\n", time.Now().Format("January 2006"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	_, _ = dim.Println(t.GetMessage("stats.by_command_label", 0, nil))

	maxCmdLen := 10
	for _, stat := range breakdown.ByCommand {
		if len(stat.Command) > maxCmdLen {
			maxCmdLen = len(stat.Command)
		}
	}

	fmt.Printf("%-*s │ %8s │ %10s │ %8s\n",
		maxCmdLen,
		t.GetMessage("stats.column_command", 0, nil),
		t.GetMessage("stats.column_calls", 0, nil),
		t.GetMessage("stats.column_cost", 0, nil),
		t.GetMessage("stats.column_percent", 0, nil))
	fmt.Printf("%s─┼─%s─┼─%s─┼─%s\n",
		strings.Repeat("─", maxCmdLen),
		strings.Repeat("─", 8),
		strings.Repeat("─", 10),
		strings.Repeat("─", 8))

	for _, stat := range breakdown.ByCommand {
		percentage := (stat.TotalCost / breakdown.TotalCost) * 100

		cmdColor := color.New(color.FgWhite)
		switch stat.Command {
		case "suggest":
			cmdColor = color.New(color.FgCyan)
		case "summarize-pr":
			cmdColor = color.New(color.FgMagenta)
		case "issue":
			cmdColor = color.New(color.FgBlue)
		}

		fmt.Printf("%-*s │ %s │ %s │ %7s%%\n",
			maxCmdLen, cmdColor.Sprint(stat.Command),
			yellow.Sprintf("%8d", stat.CallCount),
			yellow.Sprintf("$%8.4f", stat.TotalCost),
			fmt.Sprintf("%7.1f", percentage),
		)

		if stat.CacheHitRate > 0 {
			_, _ = dim.Printf("%s   └─ %s %.0f%%\n",
				strings.Repeat(" ", maxCmdLen),
				t.GetMessage("stats.cache_hits_label", 0, nil),
				stat.CacheHitRate)
		}
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%s %s │ %s\n",
		t.GetMessage("stats.total_label", 0, nil),
		yellow.Sprintf("%d %s", breakdown.TotalCalls, t.GetMessage("stats.calls_text", 0, nil)),
		yellow.Sprintf("$%.4f USD", breakdown.TotalCost))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for _, stat := range breakdown.ByCommand {
		if stat.Command == "suggest" && stat.CallCount > 0 {
			_, _ = green.Println(t.GetMessage("stats.avg_cost_per_commit_label", 0, struct{ Cost float64 }{stat.AvgCost}))
			fmt.Println()
			break
		}
	}

	return nil
}
