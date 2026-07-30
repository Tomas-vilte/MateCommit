package config

import (
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newShowCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: t.GetMessage("config_show_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			// LoadConfigWithHierarchy resolves to the repo-local config
			// entirely whenever one exists (not a field-by-field merge with
			// global) — so `cfg` here is local data whenever we're inside a
			// git repo, and it must be labeled as such instead of always
			// saying "Global".
			isLocal := config.GetRepoConfigPath() != ""
			if isLocal {
				fmt.Println(t.GetMessage("config_local.local_config_header", 0, nil))
			} else {
				fmt.Println(t.GetMessage("config_local.global_config_header", 0, nil))
			}
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━\n")

			fmt.Printf("%s\n", t.GetMessage("language_label", 0, struct{ Lang string }{cfg.Language}))

			fmt.Printf("%s\n", t.GetMessage("emojis_label", 0, struct{ Emoji bool }{cfg.UseEmoji}))

			hasGemini := false
			if providerCfg, exists := cfg.AIProviders["gemini"]; exists && providerCfg.APIKey != "" {
				hasGemini = true
			}

			if !hasGemini {
				fmt.Println(t.GetMessage("api.key_not_set", 0, nil))
				fmt.Println(t.GetMessage("api.key_tip", 0, nil))
				fmt.Println(t.GetMessage("api.key_config_command", 0, nil))
			} else {
				fmt.Println(t.GetMessage("api.key_set", 0, nil))
			}

			if cfg.UseTicket {
				fmt.Printf("%s\n", t.GetMessage("config_models.ticket_service_enabled", 0, struct{ Service string }{cfg.ActiveTicketService}))
				if cfg.ActiveTicketService == "jira" {
					jiraCfg := cfg.TicketProviders["jira"]
					fmt.Printf("%s\n", t.GetMessage("config_models.jira_config_label", 0, struct {
						BaseURL string
						Email   string
					}{
						BaseURL: jiraCfg.BaseURL,
						Email:   jiraCfg.Email,
					}))
				}
			} else {
				fmt.Println(t.GetMessage("config_models.ticket_service_disabled", 0, nil))
			}

			fmt.Printf("%s\n", t.GetMessage("config_models.active_ai_label", 0, struct{ IA config.AI }{cfg.AIConfig.ActiveAI}))

			if len(cfg.AIConfig.Models) > 0 {
				fmt.Println(t.GetMessage("config_models.ai_models_label", 0, nil))
				for ai, model := range cfg.AIConfig.Models {
					fmt.Printf("- %s: %s\n", ai, model)
				}
			} else {
				fmt.Println(t.GetMessage("config_models.no_ai_models_configured", 0, nil))
			}

			if cfg.GitFallback.UserName != "" || cfg.GitFallback.UserEmail != "" {
				fmt.Println()
				fmt.Println(t.GetMessage("config_git.fallback_header", 0, nil))
				if cfg.GitFallback.UserName != "" {
					fmt.Printf("%s\n", t.GetMessage("config_git.fallback_name", 0, struct{ Name string }{cfg.GitFallback.UserName}))
				}
				if cfg.GitFallback.UserEmail != "" {
					fmt.Printf("%s\n", t.GetMessage("config_git.fallback_email", 0, struct{ Email string }{cfg.GitFallback.UserEmail}))
				}
			}

			return nil
		},
	}
}
