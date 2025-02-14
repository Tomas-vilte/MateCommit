package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetActiveVCSCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-active-vcs",
		Usage: t.GetMessage("vcs_summary.config_set_active_vcs_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "provider",
				Aliases:  []string{"p"},
				Usage:    t.GetMessage("vcs_summary.config_set_active_vcs_provider_usage", 0, nil),
				Required: true,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			provider := command.String("provider")

			if _, exists := cfg.VCSConfigs[provider]; !exists {
				msg := t.GetMessage("error.vcs_provider_not_configured", 0, map[string]interface{}{
					"Provider": provider,
				})
				return fmt.Errorf("%s", msg)
			}

			cfg.ActiveVCSProvider = provider

			if err := config.SaveConfig(cfg); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}

			fmt.Println(t.GetMessage("vcs_summary.config_active_vcs_updated", 0, map[string]interface{}{
				"Provider": provider,
			}))
			return nil
		},
	}
}
