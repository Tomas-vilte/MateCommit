package config

import (
	"context"
	"errors"
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
				ui.PrintError(os.Stdout, t.GetMessage("config_set_error_args", 0, nil))
				return fmt.Errorf("missing arguments")
			}

			key := strings.ToLower(command.Args().Get(0))
			value := command.Args().Get(1)

			isLocalExplicit := command.Bool("local")
			isGlobalExplicit := command.Bool("global")

			useLocal := isLocalExplicit
			if !isGlobalExplicit && !isLocalExplicit {
				localPath := config.GetRepoConfigPath()
				useLocal = localPath != ""
			}

			targetCfg := cfg
			if useLocal {
				localPath := config.GetRepoConfigPath()
				if localPath == "" {
					return errors.New(t.GetMessage("config_local.not_in_repo", 0, nil))
				}

				localCfg, err := config.LoadConfig(localPath)
				if err != nil {
					localCfg, err = config.CreateDefaultConfig(localPath)
					if err != nil {
						return fmt.Errorf("error creating local config: %w", err)
					}
				}
				targetCfg = localCfg
			}

			switch key {
			case "lang", "language":
				if isValidLanguage(value) {
					targetCfg.Language = value
				} else {
					return fmt.Errorf("invalid language: %s", value)
				}
			case "emoji", "use_emoji":
				boolVal, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				targetCfg.UseEmoji = boolVal
			case "count", "suggestions_count":
				intVal, err := strconv.Atoi(value)
				if err != nil || intVal < 1 || intVal > 10 {
					return fmt.Errorf("invalid count (must be 1-10): %s", value)
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
					return fmt.Errorf("no active AI provider configured")
				}
			case "active-vcs", "active_vcs":
				targetCfg.ActiveVCSProvider = value
			case "git-name":
				targetCfg.GitFallback.UserName = value
			case "git-email":
				targetCfg.GitFallback.UserEmail = value
			default:
				return fmt.Errorf("unknown configuration key: %s", key)
			}

			var err error
			if useLocal {
				err = config.SaveLocalConfig(targetCfg)
			} else {
				err = config.SaveConfig(targetCfg)
			}

			if err != nil {
				ui.PrintError(os.Stdout, t.GetMessage("ui_error.error_saving_config", 0, nil))
				return err
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
