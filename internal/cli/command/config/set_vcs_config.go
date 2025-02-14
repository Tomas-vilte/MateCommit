package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetVCSConfigCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-vcs",
		Usage: t.GetMessage("vcs_summary.config_set_vcs_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "provider",
				Aliases:  []string{"p"},
				Usage:    t.GetMessage("vcs_summary.config_set_vcs_provider_usage", 0, nil),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    t.GetMessage("vcs_summary.config_set_vcs_token_usage", 0, nil),
				Required: false,
			},
			&cli.StringFlag{
				Name:     "owner",
				Aliases:  []string{"o"},
				Usage:    t.GetMessage("vcs_summary.config_set_vcs_owner_usage", 0, nil),
				Required: false,
			},
			&cli.StringFlag{
				Name:     "repo",
				Aliases:  []string{"r"},
				Usage:    t.GetMessage("vcs_summary.config_set_vcs_repo_usage", 0, nil),
				Required: false,
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			provider := command.String("provider")

			if cfg.VCSConfigs == nil {
				cfg.VCSConfigs = make(map[string]config.VCSConfig)
			}

			vcsConfig, exists := cfg.VCSConfigs[provider]
			if !exists {
				vcsConfig = config.VCSConfig{Provider: provider}
			}

			if token := command.String("token"); token != "" {
				vcsConfig.Token = token
			}
			if owner := command.String("owner"); owner != "" {
				vcsConfig.Owner = owner
			}
			if repo := command.String("repo"); repo != "" {
				vcsConfig.Repo = repo
			}

			cfg.VCSConfigs[provider] = vcsConfig

			if err := config.SaveConfig(cfg); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}

			fmt.Println(t.GetMessage("vcs_summary.config_vcs_updated", 0, map[string]interface{}{
				"Provider": provider,
			}))
			return nil
		},
	}
}
