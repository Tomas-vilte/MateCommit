package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func newSetAPIKeyCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-api-key",
		Usage: t.GetMessage("commands.set_api_key_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "key",
				Aliases:  []string{"k"},
				Usage:    t.GetMessage("flags.gemini_api_key", 0, nil),
				Required: true,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			apiKey := command.String("key")
			if len(apiKey) < 10 {
				msg := t.GetMessage("api.invalid_key", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			cfg.GeminiAPIKey = apiKey
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}

			fmt.Printf("%s\n", t.GetMessage("api.key_configured", 0, nil))
			fmt.Printf("%s\n", t.GetMessage("api.key_configuration_help", 0, nil))
			return nil
		},
	}
}
