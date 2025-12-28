package suggests_commits

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// commitService is a minimal interface for testing purposes
type commitService interface {
	GenerateSuggestions(ctx context.Context, count int, issueNumber int, progress func(models.ProgressEvent)) ([]models.CommitSuggestion, error)
}

// commitHandler is a minimal interface for testing purposes
type commitHandler interface {
	HandleSuggestions(ctx context.Context, suggestions []models.CommitSuggestion) error
}

type gitService interface {
	ValidateGitConfig(ctx context.Context) error
	GetChangedFiles(ctx context.Context) ([]string, error)
	GetDiff(ctx context.Context) (string, error)
}

type SuggestCommandFactory struct {
	commitService commitService
	commitHandler commitHandler
	gitService    gitService
}

func NewSuggestCommandFactory(commitSvc commitService, commitHdlr commitHandler, gitSvc gitService) *SuggestCommandFactory {
	return &SuggestCommandFactory{
		commitService: commitSvc,
		commitHandler: commitHdlr,
		gitService:    gitSvc,
	}
}

func (f *SuggestCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:          "suggest",
		Aliases:       []string{"s"},
		Usage:         t.GetMessage("suggest_command_usage", 0, nil),
		Description:   t.GetMessage("suggest_command_description", 0, nil),
		Flags:         f.createFlags(cfg, t),
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action:        f.createAction(cfg, t),
	}
}

func (f *SuggestCommandFactory) createFlags(cfg *config.Config, t *i18n.Translations) []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "count",
			Aliases: []string{"n"},
			Value:   cfg.SuggestionsCount,
			Usage:   t.GetMessage("suggest_count_flag_usage", 0, nil),
		},
		&cli.StringFlag{
			Name:    "lang",
			Aliases: []string{"l"},
			Usage:   t.GetMessage("suggest_lang_flag_usage", 0, nil),
			Value:   cfg.Language,
		},
		&cli.BoolFlag{
			Name:    "no-emoji",
			Aliases: []string{"ne"},
			Value:   cfg.UseEmoji,
			Usage:   t.GetMessage("suggest_no_emoji_flag_usage", 0, nil),
		},
		&cli.IntFlag{
			Name:    "issue",
			Aliases: []string{"i"},
			Usage:   t.GetMessage("suggest_issue_flag_usage", 0, nil),
			Value:   0,
		},
		&cli.BoolFlag{
			Name:    "dry-run",
			Aliases: []string{"d"},
			Usage:   t.GetMessage("suggest_dry_run_flag_usage", 0, nil),
		},
	}
}

func (f *SuggestCommandFactory) createAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		log := logger.FromContext(ctx)

		count := command.Int("count")
		issueNumber := command.Int("issue")
		lang := command.String("lang")
		noEmoji := command.Bool("no-emoji")
		dryRun := command.Bool("dry-run")

		log.Info("executing suggest command",
			"count", count,
			"issue_number", issueNumber,
			"language", lang,
			"no_emoji", noEmoji,
			"dry_run", dryRun)

		if noEmoji {
			cfg.UseEmoji = false
		} else {
			cfg.UseEmoji = true
		}

		if count < 1 || count > 10 {
			msg := t.GetMessage("invalid_suggestions_count", 0, struct {
				Min int
				Max int
			}{1, 10})
			ui.PrintError(os.Stdout, msg)
			return fmt.Errorf("%s", msg)
		}

		cfg.Language = command.String("lang")

		if err := t.SetLanguage(cfg.Language); err != nil {
			_ = t.SetLanguage("en")
		}

		if err := config.SaveConfig(cfg); err != nil {
			ui.PrintError(os.Stdout, t.GetMessage("ui_error.error_saving_config", 0, nil))
			return fmt.Errorf("error saving configuration: %w", err)
		}

		if dryRun {
			return f.handleDryRun(ctx, t)
		}

		ui.PrintSectionBanner(t.GetMessage("ui.generating_suggestions_banner", 0, nil))

		if err := f.gitService.ValidateGitConfig(ctx); err != nil {
			ui.HandleAppError(err, t)
			return err
		}

		spinner := ui.NewSmartSpinner(t.GetMessage("analyzing_changes", 0, nil))
		spinner.Start()

		var suggestions []models.CommitSuggestion
		var err error

		start := time.Now()

		suggestions, err = f.commitService.GenerateSuggestions(ctx, count, issueNumber, func(event models.ProgressEvent) {
			msg := ""
			switch event.Type {
			case models.ProgressIssuesDetected:
				key := "issue_detected_auto"
				if !event.Data.IsAuto {
					key = "issue_using_manual"
				}
				msg = t.GetMessage(key, 0, event.Data)
			case models.ProgressGeneric:
				msg = event.Message
			default:
				msg = event.Message
			}

			if msg != "" {
				spinner.Log(msg)
			}
		})

		duration := time.Since(start)

		if err != nil {
			log.Error("failed to generate suggestions",
				"error", err,
				"duration_ms", duration.Milliseconds())
			spinner.Error(t.GetMessage("ui.error_generating_suggestions", 0, nil))
			ui.HandleAppError(err, t)
			return fmt.Errorf("%s", t.GetMessage("suggestion_generation_error", 0, struct{ Error error }{err}))
		}

		log.Info("suggestions generated successfully",
			"count", len(suggestions),
			"duration_ms", duration.Milliseconds())

		spinner.Stop()
		ui.PrintDuration(t.GetMessage("ui.suggestions_generated", 0, struct{ Count int }{len(suggestions)}), duration)
		return f.commitHandler.HandleSuggestions(ctx, suggestions)
	}
}

// handleDryRun shows a preview of changes without making AI calls
func (f *SuggestCommandFactory) handleDryRun(ctx context.Context, t *i18n.Translations) error {
	log := logger.FromContext(ctx)
	log.Info("executing dry-run mode")

	cyan := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	dim := color.New(color.FgHiBlack)

	fmt.Println()
	_, _ = cyan.Println(t.GetMessage("stats.dry_run_banner", 0, nil))
	fmt.Println()

	if err := f.gitService.ValidateGitConfig(ctx); err != nil {
		ui.HandleAppError(err, t)
		return err
	}

	files, err := f.gitService.GetChangedFiles(ctx)
	if err != nil {
		ui.HandleAppError(err, t)
		return err
	}

	if len(files) == 0 {
		ui.PrintWarning(t.GetMessage("stats.dry_run_no_changes", 0, nil))
		_, _ = dim.Println("\n" + t.GetMessage("stats.dry_run_try_git_add", 0, nil))
		return nil
	}

	diff, err := f.gitService.GetDiff(ctx)
	if err != nil {
		ui.HandleAppError(err, t)
		return err
	}

	_, _ = cyan.Printf(t.GetMessage("stats.dry_run_changed_files", 0, nil)+"\n", len(files))
	for _, file := range files {
		fmt.Printf("   %s %s\n", color.CyanString("•"), file)
	}
	fmt.Println()

	additions, deletions := calculateDiffStats(diff)

	_, _ = cyan.Println(t.GetMessage("stats.dry_run_changes_summary", 0, nil) + ":")
	fmt.Printf("   %s %s: %s\n", green.Sprint("  +"), t.GetMessage("stats.dry_run_additions", 0, nil), green.Sprintf("%d lines", additions))
	fmt.Printf("   %s %s: %s\n", red.Sprint("  -"), t.GetMessage("stats.dry_run_deletions", 0, nil), red.Sprintf("%d lines", deletions))
	fmt.Printf("   %s %s: %s\n", yellow.Sprint("  ~"), t.GetMessage("stats.dry_run_files_modified", 0, nil), yellow.Sprintf("%d", len(files)))
	fmt.Println()

	estimatedTokens := estimateTokensFromDiff(diff)
	estimatedCost := float64(estimatedTokens) * 0.00001

	_, _ = cyan.Println(t.GetMessage("stats.dry_run_estimated_cost", 0, nil) + ":")
	fmt.Printf("   %s: ~%s\n", t.GetMessage("stats.dry_run_tokens", 0, nil), yellow.Sprintf("%d", estimatedTokens))
	fmt.Printf("   %s: ~%s\n", t.GetMessage("stats.dry_run_cost", 0, nil), yellow.Sprintf("$%.4f", estimatedCost))
	fmt.Println()

	_, _ = dim.Println(t.GetMessage("stats.dry_run_hint", 0, nil))
	fmt.Println()

	return nil
}

// calculateDiffStats parses a git diff and counts additions/deletions
func calculateDiffStats(diff string) (additions, deletions int) {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if line[0] == '+' && !strings.HasPrefix(line, "+++") {
			additions++
		} else if line[0] == '-' && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}
	return
}

// estimateTokensFromDiff provides a rough estimate of tokens needed
func estimateTokensFromDiff(diff string) int {
	baseTokens := len(diff) / 4
	overhead := 200
	return baseTokens + overhead
}
