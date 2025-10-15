package config

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newShowCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: t.GetMessage("config_show_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			fmt.Println(t.GetMessage("current_config", 0, nil))
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━\n")

			fmt.Printf("%s\n", t.GetMessage("language_label", 0, map[string]interface{}{"Lang": cfg.Language}))

			fmt.Printf("%s\n", t.GetMessage("emojis_label", 0, map[string]interface{}{"Emoji": cfg.UseEmoji}))

			if cfg.GeminiAPIKey == "" {
				fmt.Println(t.GetMessage("api.key_not_set", 0, nil))
				fmt.Println(t.GetMessage("api.key_tip", 0, nil))
				fmt.Println(t.GetMessage("api.key_config_command", 0, nil))
			} else {
				fmt.Println(t.GetMessage("api.key_set", 0, nil))
			}

			if cfg.UseTicket {
				fmt.Printf("%s\n", t.GetMessage("config_models.ticket_service_enabled", 0, map[string]interface{}{"Service": cfg.ActiveTicketService}))
				if cfg.ActiveTicketService == "jira" {
					fmt.Printf("%s\n", t.GetMessage("config_models.jira_config_label", 0, map[string]interface{}{
						"BaseURL": cfg.JiraConfig.BaseURL,
						"Email":   cfg.JiraConfig.Email,
					}))
				}
			} else {
				fmt.Println(t.GetMessage("config_models.ticket_service_disabled", 0, nil))
			}

			fmt.Printf("%s\n", t.GetMessage("config_models.active_ai_label", 0, map[string]interface{}{"IA": cfg.AIConfig.ActiveAI}))

			if len(cfg.AIConfig.Models) > 0 {
				fmt.Println(t.GetMessage("config_models.ai_models_label", 0, nil))
				for ai, model := range cfg.AIConfig.Models {
					fmt.Printf("- %s: %s\n", ai, model)
				}
			} else {
				fmt.Println(t.GetMessage("config_models.no_ai_models_configured", 0, nil))
			}

			return nil
		},
	}
}
