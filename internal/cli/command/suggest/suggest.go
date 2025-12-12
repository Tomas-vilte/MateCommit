package suggest

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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
			return fmt.Errorf("%s", msg)
		}

		cfg.Language = command.String("lang")

		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error al guardar la configuración: %w", err)
		}

		fmt.Println(t.GetMessage("analyzing_changes", 0, nil))

		issueNumber := command.Int("issue")
		var suggestions []models.CommitSuggestion
		var err error

		if issueNumber > 0 {
			msg := t.GetMessage("issue_including_context", 0, map[string]interface{}{
				"Number": issueNumber,
			})
			fmt.Println(msg)
			suggestions, err = f.commitService.GenerateSuggestionsWithIssue(ctx, int(count), issueNumber)
		} else {
			suggestions, err = f.commitService.GenerateSuggestions(ctx, int(count))
		}

		if err != nil {
			msg := t.GetMessage("suggestion_generation_error", 0, map[string]interface{}{"Error": err})
			return fmt.Errorf("%s", msg)
		}

		return f.commitHandler.HandleSuggestions(ctx, suggestions)
	}
}
