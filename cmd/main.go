package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Tomas-vilte/MateCommit/internal/cli/command/config"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/handler"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/pull_requests"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/release"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/suggests_commits"
	"github.com/Tomas-vilte/MateCommit/internal/cli/registry"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/factory"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/jira"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
)

func main() {
	app, err := initializeApp()
	if err != nil {
		log.Fatalf("Error iniciando la cli: %v", err)
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

	translations, err := i18n.NewTranslations(cfgApp.Language, "")
	if err != nil {
		log.Fatalf("Error al cargar las traducciones: %v", err)
	}

	err = cfg.SaveConfig(cfgApp)
	if err != nil {
		return nil, err
	}

	gitService := git.NewGitService(translations)
	aiProvider, err := gemini.NewGeminiService(context.Background(), cfgApp, translations)
	if err != nil {
		log.Printf("Warning: %v", err)
		log.Println("La IA no está configurada. Podés configurarla con 'matecommit config init'")
		aiProvider = nil
	}

	aiSummarizer, err := gemini.NewGeminiPRSummarizer(context.Background(), cfgApp, translations)
	if err != nil {
		log.Printf("Warning: %v", err)
		log.Println("El resumidor de PRs está deshabilitado hasta configurar la IA (Gemini).")
		aiSummarizer = nil
	}

	ticketService := jira.NewJiraService(cfgApp, &http.Client{})

	// Inicializar VCS Client si es posible
	var vcsClient ports.VCSClient
	repoOwner, repoName, provider, err := gitService.GetRepoInfo(context.Background())
	if err == nil {
		if vcsConfig, ok := cfgApp.VCSConfigs[provider]; ok && provider == "github" {
			vcsClient = github.NewGitHubClient(repoOwner, repoName, vcsConfig.Token, translations)
		} else if cfgApp.ActiveVCSProvider != "" {
			if vcsConfig, ok := cfgApp.VCSConfigs[cfgApp.ActiveVCSProvider]; ok && cfgApp.ActiveVCSProvider == "github" {
				vcsClient = github.NewGitHubClient(repoOwner, repoName, vcsConfig.Token, translations)
			}
		}
	}

	commitService := services.NewCommitService(gitService, aiProvider, ticketService, nil, cfgApp, translations)

	commitHandler := handler.NewSuggestionHandler(gitService, vcsClient, translations)

	registerCommand := registry.NewRegistry(cfgApp, translations)

	prServiceFactory := factory.NewPrServiceFactory(cfgApp, translations, aiSummarizer, gitService)

	prCommand := pull_requests.NewSummarizeCommand(prServiceFactory)

	if err := registerCommand.Register("suggest", suggests_commits.NewSuggestCommandFactory(commitService, commitHandler)); err != nil {
		log.Fatalf("Error al registrar el comando 'suggest': %v", err)
	}

	if err := registerCommand.Register("config", config.NewConfigCommandFactory()); err != nil {
		log.Fatalf("Error al registrar el comando 'config': %v", err)
	}

	if err := registerCommand.Register("summarize-pr", prCommand); err != nil {
		log.Fatalf("Error al registrar el comando 'summarize-pr': %v", err)
	}

	if err := registerCommand.Register("release", release.NewReleaseCommandFactory(gitService, cfgApp)); err != nil {
		log.Fatalf("Error al registrar el comando 'release': %v", err)
	}

	commands := registerCommand.CreateCommands()
	helpCommand := &cli.Command{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   translations.GetMessage("help_command_usage", 0, nil),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowAppHelp(cmd)
		},
	}
	commands = append(commands, helpCommand)

	return &cli.Command{
		Name:        "mate-commit",
		Usage:       translations.GetMessage("app_usage", 0, nil),
		Version:     "1.4.0",
		Description: translations.GetMessage("app_description", 0, nil),
		Commands:    commands,
	}, nil
}
