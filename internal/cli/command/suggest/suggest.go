package suggest

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/handler"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
)

func NewCommand(cfg *config.Config, t *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:        "suggest",
		Aliases:     []string{"s"},
		Usage:       t.GetMessage("suggest_command_usage", 0, nil),
		Description: t.GetMessage("suggest_command_description", 0, nil),
		Flags:       suggestFlags(cfg, t),
		Action:      suggestAction(cfg, t),
	}
}

func suggestFlags(cfg *config.Config, t *i18n.Translations) []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "count",
			Aliases: []string{"n"},
			Value:   3,
			Usage:   t.GetMessage("suggest_count_flag_usage", 0, nil),
		},
		&cli.StringFlag{
			Name:    "lang",
			Aliases: []string{"l"},
			Value:   cfg.DefaultLang,
			Usage:   t.GetMessage("suggest_lang_flag_usage", 0, nil),
		},
		&cli.BoolFlag{
			Name:    "no-emoji",
			Aliases: []string{"ne"},
			Usage:   t.GetMessage("suggest_no_emoji_flag_usage", 0, nil),
		},
		&cli.IntFlag{
			Name:    "max-length",
			Aliases: []string{"ml"},
			Value:   72,
			Usage:   t.GetMessage("suggest_max_length_flag_usage", 0, nil),
		},
	}
}

func suggestAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		gitService := git.NewGitService()
		count := command.Int("count")
		if count < 1 || count > 10 {
			msg := t.GetMessage("invalid_suggestions_count", 0, map[string]interface{}{
				"Min": 1,
				"Max": 10,
			})
			return fmt.Errorf("%s", msg)
		}

		commitConfig := &config.CommitConfig{
			Language:  config.GetLocaleConfig(command.String("lang")),
			MaxLength: int(command.Int("max-length")),
			UseEmoji:  !command.Bool("no-emoji"),
		}

		geminiService, err := gemini.NewGeminiService(ctx, cfg.GeminiAPIKey, commitConfig, t)
		if err != nil {
			msg := t.GetMessage("gemini_init_error", 0, map[string]interface{}{"Error": err})
			return fmt.Errorf("%s", msg)
		}

		fmt.Println(t.GetMessage("analyzing_changes", 0, nil))
		commitService := services.NewCommitService(gitService, geminiService)
		suggestions, err := commitService.GenerateSuggestions(ctx, int(count), cfg.Format)
		if err != nil {
			msg := t.GetMessage("suggestion_generation_error", 0, map[string]interface{}{"Error": err})
			return fmt.Errorf("%s", msg)
		}

		return handler.HandleSuggestions(suggestions, gitService, t)
	}
}
