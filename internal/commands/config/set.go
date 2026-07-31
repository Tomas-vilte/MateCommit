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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "local",
				Aliases: []string{"l"},
				Usage:   t.GetMessage("config_local.set_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "global",
				Aliases: []string{"g"},
				Usage:   t.GetMessage("config_local.global_flag", 0, nil),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			if command.Args().Len() < 2 {
				return ui.PrintError(os.Stdout, t.GetMessage("config_set_error_args", 0, nil))
			}

			key := strings.ToLower(command.Args().Get(0))
			value := command.Args().Get(1)

			targetCfg, useLocal, err := resolveTargetConfig(command, cfg, t)
			if err != nil {
				return ui.PrintError(os.Stdout, err.Error())
			}

			switch key {
			case "lang", "language":
				if isValidLanguage(value) {
					targetCfg.Language = value
				} else {
					return ui.PrintError(os.Stdout, fmt.Sprintf("invalid language: %s", value))
				}
			case "emoji", "use_emoji":
				boolVal, parseErr := strconv.ParseBool(value)
				if parseErr != nil {
					return ui.PrintError(os.Stdout, fmt.Sprintf("invalid boolean value: %s", value))
				}
				targetCfg.UseEmoji = boolVal
			case "count", "suggestions_count":
				intVal, parseErr := strconv.Atoi(value)
				if parseErr != nil || intVal < 1 || intVal > 10 {
					return ui.PrintError(os.Stdout, fmt.Sprintf("invalid count (must be 1-10): %s", value))
				}
				targetCfg.SuggestionsCount = intVal
			case "active-ai", "active_ai":
				targetCfg.AIConfig.ActiveAI = config.AI(value)
			case "model":
				if targetCfg.AIConfig.ActiveAI != "" {
					if targetCfg.AIConfig.Models == nil {
						targetCfg.AIConfig.Models = make(map[config.AI]config.Model)
					}
					targetCfg.AIConfig.Models[targetCfg.AIConfig.ActiveAI] = config.Model(value)
				} else {
					return ui.PrintError(os.Stdout, "no active AI provider configured")
				}
			case "active-vcs", "active_vcs":
				targetCfg.ActiveVCSProvider = value
			case "git.name", "git-name":
				targetCfg.GitFallback.UserName = value
			case "git.email", "git-email":
				targetCfg.GitFallback.UserEmail = value
			default:
				return ui.PrintError(os.Stdout, fmt.Sprintf("unknown configuration key: %s", key))
			}

			if useLocal {
				err = config.SaveLocalConfig(targetCfg)
			} else {
				err = config.SaveConfig(targetCfg)
			}

			if err != nil {
				return ui.PrintError(os.Stdout, t.GetMessage("ui_error.error_saving_config", 0, nil))
			}

			scope := "global"
			if useLocal {
				scope = "local"
			}
			ui.PrintSuccess(os.Stdout, t.GetMessage("config_set_success", 0, struct {
				Key   string
				Value string
				Scope string
			}{Key: key, Value: value, Scope: scope}))

			return nil
		},
	}
}
