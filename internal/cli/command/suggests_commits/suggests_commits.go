package suggests_commits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

type SuggestCommandFactory struct {
	commitService ports.CommitService
	commitHandler ports.CommitHandler
}

func NewSuggestCommandFactory(commitService ports.CommitService, commitHandler ports.CommitHandler) *SuggestCommandFactory {
	return &SuggestCommandFactory{
		commitService: commitService,
		commitHandler: commitHandler,
	}
}

func (f *SuggestCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:        "suggest",
		Aliases:     []string{"s"},
		Usage:       t.GetMessage("suggest_command_usage", 0, nil),
		Description: t.GetMessage("suggest_command_description", 0, nil),
		Flags:       f.createFlags(cfg, t),
		Action:      f.createAction(cfg, t),
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
	}
}

func (f *SuggestCommandFactory) createAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		emojiFlag := command.Bool("no-emoji")
		if emojiFlag {
			cfg.UseEmoji = false
		} else {
			cfg.UseEmoji = true
		}
		count := command.Int("count")
		if count < 1 || count > 10 {
			msg := t.GetMessage("invalid_suggestions_count", 0, map[string]interface{}{
				"Min": 1,
				"Max": 10,
			})
			ui.PrintError(msg)
			return fmt.Errorf("%s", msg)
		}

		cfg.Language = command.String("lang")

		if err := t.SetLanguage(cfg.Language); err != nil {
			_ = t.SetLanguage("en")
		}

		if err := config.SaveConfig(cfg); err != nil {
			ui.PrintError(t.GetMessage("ui_error.error_saving_config", 0, nil))
			return fmt.Errorf("error al guardar la configuración: %w", err)
		}

		ui.PrintSectionBanner(t.GetMessage("ui.generating_suggestions_banner", 0, nil))

		spinner := ui.NewSmartSpinner(t.GetMessage("analyzing_changes", 0, nil))
		spinner.Start()

		issueNumber := command.Int("issue")
		var suggestions []models.CommitSuggestion
		var err error

		start := time.Now()

		if issueNumber > 0 {
			spinner.UpdateMessage(t.GetMessage("ui.including_issue_context", 0, map[string]interface{}{
				"Number": issueNumber,
			}))
			time.Sleep(200 * time.Millisecond)

			spinner.Stop()
			ui.PrintInfo(t.GetMessage("ui.detected_issue", 0, map[string]interface{}{
				"Number": issueNumber,
			}))

			spinner = ui.NewSmartSpinner(t.GetMessage("ui.generating_with_issue", 0, nil))
			spinner.Start()

			suggestions, err = f.commitService.GenerateSuggestionsWithIssue(ctx, count, issueNumber, func(msg string) {
				spinner.Log(msg)
			})
		} else {
			spinner.UpdateMessage(t.GetMessage("ui.generating_with_ai", 0, nil))
			suggestions, err = f.commitService.GenerateSuggestions(ctx, count, func(msg string) {
				spinner.Log(msg)
			})
		}

		duration := time.Since(start)

		if err != nil {
			spinner.Error(t.GetMessage("ui.error_generating_suggestions", 0, nil))

			errStr := err.Error()
			if containsStr(errStr, "GEMINI_API_KEY") || containsStr(errStr, "API key") {
				ui.PrintErrorWithSuggestion(
					t.GetMessage("ui_error.gemini_api_key_missing", 0, nil),
					t.GetMessage("ui_error.run_config_init", 0, nil),
				)
			} else if containsStr(errStr, "GITHUB_TOKEN") {
				ui.PrintErrorWithSuggestion(
					t.GetMessage("ui_error.github_token_missing", 0, nil),
					t.GetMessage("ui_error.run_config_init", 0, nil),
				)
			} else if containsStr(errStr, "no differences detected") || containsStr(errStr, "no se detectaron cambios") {
				ui.PrintWarning(t.GetMessage("ui_error.no_changes_detected", 0, nil))
				ui.PrintInfo(t.GetMessage("ui_error.ensure_modified_files", 0, nil))
			} else {
				ui.PrintError(errStr)
			}

			return fmt.Errorf("%s", t.GetMessage("suggestion_generation_error", 0, map[string]interface{}{
				"Error": err,
			}))
		}

		spinner.Stop()
		ui.PrintDuration(t.GetMessage("ui.suggestions_generated", 0, map[string]interface{}{
			"Count": len(suggestions),
		}), duration)
		return f.commitHandler.HandleSuggestions(ctx, suggestions)
	}
}

func containsStr(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
