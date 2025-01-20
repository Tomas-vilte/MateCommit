package config

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (f *ConfigCommandFactory) newShowCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: t.GetMessage("config_show_usage", 0, nil),
		Action: func(ctx context.Context, command *cli.Command) error {

			fmt.Println(t.GetMessage("current_config", 0, nil))
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("%s\n", t.GetMessage("language_label", 0, map[string]interface{}{"Lang": cfg.Language}))
			fmt.Printf("%s\n", t.GetMessage("emojis_label", 0, map[string]interface{}{"Emoji": cfg.UseEmoji}))
			fmt.Printf("%s\n", t.GetMessage("max_length_label", 0, map[string]interface{}{"MaxLength": cfg.MaxLength}))

			if cfg.GeminiAPIKey == "" {
				fmt.Println(t.GetMessage("api.key_not_set", 0, nil))
				fmt.Println(t.GetMessage("api.key_tip", 0, nil))
				fmt.Println(t.GetMessage("api.key_config_command", 0, nil))
			} else {
				fmt.Println(t.GetMessage("api.key_set", 0, nil))
			}
			return nil
		},
	}
}
