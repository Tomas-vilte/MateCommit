package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetAIModelCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set-ai-model",
		Usage: t.GetMessage("config_models.config_set_ai_model_usage", 0, nil),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			args := cmd.Args()
			if args.Len() < 1 {
				// Mostrar un listado de IAs disponibles
				fmt.Println(t.GetMessage("config_models.config_available_ais", 0, nil))
				for _, validAI := range []config.AI{config.AIGemini, config.AIOpenAI} {
					fmt.Printf("- %s\n", validAI)
				}
				msg := t.GetMessage("config_models.error_missing_ai", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			ai := args.Get(0)
			model := args.Get(1)

			// Validar que la IA sea válida usando el enum
			var validModels []config.Model
			switch config.AI(ai) {
			case config.AIGemini:
				validModels = []config.Model{
					config.ModelGeminiV15Flash,
					config.ModelGeminiV15Pro,
					config.ModelGeminiV20Flash,
				}
			case config.AIOpenAI:
				validModels = []config.Model{
					config.ModelGPTV4o,
					config.ModelGPTV4oMini,
				}
			default:
				msg := t.GetMessage("config_models.error_invalid_ai", 0, map[string]interface{}{
					"AI": ai,
				})
				return fmt.Errorf("%s", msg)
			}

			if model == "" {
				currentModel := cfg.AIConfig.Models[config.AI(model)]
				if currentModel == "" {
					fmt.Println(t.GetMessage("config_models.config_no_model_selected_for_ai", 0, map[string]interface{}{
						"AI": ai,
					}))
				} else {
					fmt.Println(t.GetMessage("config_models.config_current_model_for_ai", 0, map[string]interface{}{
						"AI":    ai,
						"Model": currentModel,
					}))
				}
				// Mostrar un listado de modelos disponibles para la IA seleccionada
				fmt.Println(t.GetMessage("config_models.config_set_ai_model_usage", 0, map[string]interface{}{
					"AI": ai,
				}))
				for _, validModel := range validModels {
					fmt.Printf("- %s\n", validModel)
				}
				msg := t.GetMessage("config_models.error_missing_model", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			// Validar que el modelo sea válido
			valid := false
			for _, validModel := range validModels {
				if config.Model(model) == validModel {
					valid = true
					break
				}
			}
			if !valid {
				msg := t.GetMessage("config_models.error_invalid_model", 0, map[string]interface{}{
					"Model": model,
				})
				return fmt.Errorf("%s", msg)
			}

			if cfg.AIConfig.Models == nil {
				cfg.AIConfig.Models = make(map[config.AI]config.Model)
			}

			// Guardar el modelo seleccionado
			cfg.AIConfig.Models[config.AI(ai)] = config.Model(model)
			if err := config.SaveConfig(cfg); err != nil {
				msg := t.GetMessage("config_save.error_saving_config", 0, map[string]interface{}{
					"Error": err.Error(),
				})
				return fmt.Errorf("%s", msg)
			}
			fmt.Println(t.GetMessage("config_models.config_set_ai_model_success", 0, map[string]interface{}{
				"AI":    ai,
				"Model": model,
			}))
			return nil
		},
	}
}
