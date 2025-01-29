package main

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/config"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/handler"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/suggest"
	"github.com/Tomas-vilte/MateCommit/internal/cli/registry"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/jira"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
	"log"
	"net/http"
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

	cfgApp, err := cfg.LoadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	translations, err := i18n.NewTranslations(cfgApp.Language, "locales")
	if err != nil {
		log.Fatalf("Error loading translations: %v", err)
	}

	err = cfg.SaveConfig(cfgApp)
	if err != nil {
		return nil, err
	}

	gitService := git.NewGitService()
	aiProvider, err := gemini.NewGeminiService(context.Background(), cfgApp, translations)
	if err != nil {
		log.Fatalf("Error initializing AI service: %v", err)
	}

	ticketService := jira.NewJiraService(cfgApp, &http.Client{})

	commitService := services.NewCommitService(gitService, aiProvider, ticketService, cfgApp.UseTicket)

	commitHandler := handler.NewSuggestionHandler(gitService, translations)

	registerCommand := registry.NewRegistry(cfgApp, translations)

	if err := registerCommand.Register("suggest", suggest.NewSuggestCommandFactory(commitService, commitHandler)); err != nil {
		log.Fatalf("Error al registrar el comando 'suggest': %v", err)
	}

	if err := registerCommand.Register("config", config.NewConfigCommandFactory()); err != nil {
		log.Fatalf("Error al registrar el comando 'config': %v", err)
	}

	return &cli.Command{
		Name:        "mate-commit",
		Usage:       translations.GetMessage("app_usage", 0, nil),
		Version:     "1.0.0",
		Description: translations.GetMessage("app_description", 0, nil),
		Commands:    registerCommand.CreateCommands(),
	}, nil
}
