package main

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/cli/registry"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	app, err := initializeApp()
	if err != nil {
		log.Fatalf("Error initializing application: %v", err)
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func initializeApp() (*cli.Command, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el directorio del usuario: %w", err)
	}

	cfg, err := config.LoadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	err = config.SaveConfig(cfg)
	if err != nil {
		return nil, err
	}

	t, err := i18n.NewTranslations(cfg.DefaultLang, "./locales")
	if err != nil {
		return nil, err
	}

	cmdRegistry := registry.NewCommandRegistry(cfg, t)

	return &cli.Command{
		Name:        "mate-commit",
		Usage:       t.GetMessage("app_usage", 0, nil),
		Version:     "1.0.0",
		Description: t.GetMessage("app_description", 0, nil),
		Commands:    cmdRegistry.RegisterCommands(),
	}, nil
}
