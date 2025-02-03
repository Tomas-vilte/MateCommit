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

			supportedLanguages := []string{"en", "es"}
			var validLang bool
			for _, supportedLang := range supportedLanguages {
				if lang == supportedLang {
					validLang = true
					break
				}
			}

			if !validLang {
				msg := t.GetMessage("config_models.error_invalid_language", 0, map[string]interface{}{
					"Language": lang,
				})
				return fmt.Errorf("%s", msg)
			}

			cfgCopy := *cfg
			cfgCopy.Language = lang

			cfg.Language = lang
			if err := config.SaveConfig(&cfgCopy); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}

			cfg.Language = lang

			fmt.Printf("%s\n", t.GetMessage("language_configured", 0, map[string]interface{}{"Lang": lang}))
			return nil
		},
	}
}
