package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newSetCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     t.GetMessage("config_set_usage", 0, nil),
		ArgsUsage: t.GetMessage("config_set_args_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() < 2 {
				ui.PrintError(os.Stdout, t.GetMessage("config_set_error_args", 0, nil))
				return fmt.Errorf("missing arguments")
			}

			key := strings.ToLower(command.Args().Get(0))
			value := command.Args().Get(1)

			switch key {
			case "lang", "language":
				if isValidLanguage(value) {
					cfg.Language = value
				} else {
					return fmt.Errorf("invalid language: %s", value)
				}
			case "emoji", "use_emoji":
				boolVal, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				cfg.UseEmoji = boolVal
			case "count", "suggestions_count":
				intVal, err := strconv.Atoi(value)
				if err != nil || intVal < 1 || intVal > 10 {
					return fmt.Errorf("invalid count (must be 1-10): %s", value)
				}
				cfg.SuggestionsCount = intVal
			case "active-ai", "active_ai":
				cfg.AIConfig.ActiveAI = config.AI(value)
			case "model":
				if cfg.AIConfig.ActiveAI != "" {
					if cfg.AIConfig.Models == nil {
						cfg.AIConfig.Models = make(map[config.AI]config.Model)
					}
					cfg.AIConfig.Models[cfg.AIConfig.ActiveAI] = config.Model(value)
				} else {
					return fmt.Errorf("no active AI provider configured")
				}
			case "active-vcs", "active_vcs":
				cfg.ActiveVCSProvider = value
			case "git-name":
				cfg.GitFallback.UserName = value
			case "git-email":
				cfg.GitFallback.UserEmail = value
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			if err := config.SaveConfig(cfg); err != nil {
				ui.PrintError(os.Stdout, t.GetMessage("ui_error.error_saving_config", 0, nil))
				return err
			}

			ui.PrintSuccess(os.Stdout, t.GetMessage("config_set_success", 0, struct {
				Key   string
				Value string
			}{Key: key, Value: value}))

			return nil
		},
	}
}
