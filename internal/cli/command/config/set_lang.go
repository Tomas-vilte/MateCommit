package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetLangCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-lang",
		Usage: t.GetMessage("config_set_lang_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "lang",
				Aliases:  []string{"l"},
				Usage:    t.GetMessage("config_set_lang_flag_usage", 0, nil),
				Required: true,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			lang := command.String("lang")
			if lang != "en" && lang != "es" {
				msg := t.GetMessage("unsupported_language", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			cfg.Language = lang
			if err := config.SaveConfig(cfg); err != nil {
				return err
			}

			fmt.Printf("%s\n", t.GetMessage("language_configured", 0, map[string]interface{}{"Lang": lang}))
			return nil
		},
	}
}
