package suggests_commits

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
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

type SuggestCommandFactory struct {
	commitService commitService
	commitHandler commitHandler
}

func NewSuggestCommandFactory(commitSvc commitService, commitHdlr commitHandler) *SuggestCommandFactory {
	return &SuggestCommandFactory{
		commitService: commitSvc,
		commitHandler: commitHdlr,
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

		ui.PrintSectionBanner(t.GetMessage("ui.generating_suggestions_banner", 0, nil))

		spinner := ui.NewSmartSpinner(t.GetMessage("analyzing_changes", 0, nil))
		spinner.Start()

		issueNumber := command.Int("issue")
		var suggestions []models.CommitSuggestion
		var err error

		start := time.Now()

		suggestions, err = f.commitService.GenerateSuggestions(ctx, count, issueNumber, func(event models.ProgressEvent) {
			msg := ""
			switch event.Type {
			case models.ProgressIssuesDetected:
				title := event.Data["Title"].(string)
				id := event.Data["IssueID"].(int)
				isAuto := event.Data["IsAuto"].(bool)
				key := "issue_detected_auto"
				if !isAuto {
					key = "issue_using_manual"
				}
				msg = fmt.Sprintf("%s: #%d - %s", key, id, title) // TODO: use translations for key
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
			spinner.Error(t.GetMessage("ui.error_generating_suggestions", 0, nil))
			ui.HandleAppError(err, t)
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
