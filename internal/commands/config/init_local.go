package config

import (
	"bufio"
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

func runQuickSetupLocal(reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	if err := configureLanguage(reader, cfg, t); err != nil {
		return err
	}
	if err := config.SaveLocalConfig(cfg); err != nil {
		fmt.Println(t.GetMessage("config_save.error_saving_config", 0, struct{ Error string }{err.Error()}))
		return fmt.Errorf("error saving local configuration: %w", err)
	}

	fmt.Println(t.GetMessage("config_local.saved_successfully", 0, nil))
	return nil
}

func runFullSetupLocal(ctx context.Context, command *cli.Command, reader *bufio.Reader, cfg *config.Config, t *i18n.Translations) error {
	if err := configureWelcome(ctx, reader, cfg, t); err != nil {
		return err
	}
	if err := configureLanguage(reader, cfg, t); err != nil {
		return err
	}
	if err := configureVCS(reader, cfg, t); err != nil {
		return err
	}
	if err := configureTickets(reader, cfg, t); err != nil {
		return err
	}
	if err := config.SaveLocalConfig(cfg); err != nil {
		fmt.Println(t.GetMessage("config_save.error_saving_config", 0, struct{ Error string }{err.Error()}))
		return fmt.Errorf("error saving local configuration: %w", err)
	}

	printConfigSummary(cfg, t)
	fmt.Println(t.GetMessage("config_local.saved_successfully", 0, nil))

	if ui.AskConfirmation(t.GetMessage("init.prompt_run_again", 0, nil)) {
		return runFullSetupLocal(ctx, command, reader, cfg, t)
	}

	return nil
}
