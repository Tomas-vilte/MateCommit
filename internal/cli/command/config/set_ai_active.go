package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetAIActiveCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-ai-active",
		Usage: t.GetMessage("config_models.config_set_ai_active_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			ai := command.Args().First()
			if ai == "" {
				fmt.Println(t.GetMessage("config_models.config_available_ais", 0, nil))
				for _, validAI := range []config.AI{config.AIGemini, config.AIOpenAI} {
					fmt.Printf("- %s\n", validAI)
				}
				msg := t.GetMessage("config_models.error_missing_ai", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			switch config.AI(ai) {
			case config.AIGemini, config.AIOpenAI:
				cfg.AIConfig.ActiveAI = config.AI(ai)
			default:
				msg := t.GetMessage("config_models.error_invalid_ai", 0, map[string]interface{}{
					"AI": ai,
				})
				return fmt.Errorf("%s", msg)
			}

			if err := config.SaveConfig(cfg); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}
			fmt.Println(t.GetMessage("config_models.config_set_ai_active_success", 0, map[string]interface{}{
				"AI": ai,
			}))
			return nil
		},
	}
}
