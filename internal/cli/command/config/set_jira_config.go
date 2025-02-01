package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetJiraConfigCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "jira",
		Usage: t.GetMessage("jira_config_command_usage.jira_config_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "base-url",
				Aliases: []string{"u"},
				Usage:   t.GetMessage("jira_config_command_usage.jira_config_base_url_usage", 0, nil),
			},
			&cli.StringFlag{
				Name:  "api-key",
				Usage: t.GetMessage("jira_config_command_usage.jira_config_api_key_usage", 0, nil),
			},
			&cli.StringFlag{
				Name:  "email",
				Usage: t.GetMessage("jira_config_command_usage.jira_config_email_usage", 0, nil),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			baseURL := command.String("base-url")
			apiKey := command.String("api-key")
			email := command.String("email")

			if baseURL == "" || apiKey == "" || email == "" {
				msg := t.GetMessage("jira_config_command_usage.jira_config_missing_fields", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			cfg.JiraConfig.BaseURL = baseURL
			cfg.JiraConfig.APIKey = apiKey
			cfg.JiraConfig.Email = email

			if err := config.SaveConfig(cfg); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}

			fmt.Println(t.GetMessage("jira_config_command_usage.jira_config_success", 0, nil))
			return nil
		},
	}

}
