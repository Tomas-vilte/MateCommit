package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetTicketCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "ticket",
		Aliases: []string{"t"},
		Usage:   t.GetMessage("jira_config_command_usage.ticket_command_usage", 0, nil),
		Commands: []*cli.Command{
			{
				Name:    "disable",
				Aliases: []string{"d"},
				Usage:   t.GetMessage("jira_config_command_usage.disable_ticket_command_usage", 0, nil),
				Action: func(ctx context.Context, command *cli.Command) error {
					cfg.UseTicket = false
					cfg.ActiveTicketService = ""

					if err := config.SaveConfig(cfg); err != nil {
						return fmt.Errorf(t.GetMessage("error_saving_config", 0, nil), err)
					}

					fmt.Println(t.GetMessage("jira_config_command_usage.ticket_disabled_success", 0, nil))
					return nil
				},
			},
			{
				Name:    "enable",
				Aliases: []string{"e"},
				Usage:   t.GetMessage("jira_config_command_usage.enable_ticket_command_usage", 0, nil),
				Action: func(ctx context.Context, command *cli.Command) error {
					cfg.UseTicket = true

					if err := config.SaveConfig(cfg); err != nil {
						return fmt.Errorf(t.GetMessage("error_saving_config", 0, nil), err)
					}

					fmt.Println(t.GetMessage("jira_config_command_usage.ticket_enabled_success", 0, nil))
					return nil
				},
			},
		},
	}
}
