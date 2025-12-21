package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func (c *ConfigCommandFactory) newEditCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:   "edit",
		Usage:  t.GetMessage("config_edit_usage", 0, nil),
		Action: editConfigAction(cfg, t),
	}
}

func editConfigAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			if _, err := exec.LookPath("nano"); err == nil {
				editor = "nano"
			} else if _, err := exec.LookPath("vim"); err == nil {
				editor = "vim"
			} else {
				return fmt.Errorf("%s", t.GetMessage("config_save.error_no_editor", 0, nil))
			}
		}

		cmd := exec.Command(editor, cfg.PathFile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", t.GetMessage("config_save.error_opening_editor", 0, struct{ Error error }{err}), err)
		}

		return nil
	}
}
