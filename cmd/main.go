package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Tomas-vilte/MateCommit/internal/cli/command/cache"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/completion"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/config"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/handler"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/issues"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/pull_requests"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/release"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/stats"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/suggests_commits"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/update"
	"github.com/Tomas-vilte/MateCommit/internal/cli/registry"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/di"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/factory"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/jira"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/Tomas-vilte/MateCommit/internal/version"
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

	container := di.NewContainer(cfgApp, translations)

	if err := container.RegisterAIProvider("gemini", gemini.NewGeminiProviderFactory()); err != nil {
		log.Printf("Warning: no se pudo registrar el proveedor Gemini: %v", err)
	}

	if err := container.RegisterVCSProvider("github", github.NewGitHubProviderFactory()); err != nil {
		log.Printf("Warning: no se pudo registrar el proveedor de GitHub: %v", err)
	}

	if err := container.RegisterTicketProvider("jira", jira.NewJiraProviderFactory()); err != nil {
		log.Printf("Warning: no se pudo registrar el proveedor Jira: %v", err)
	}

	gitService := git.NewGitService(translations)
	container.SetGitService(gitService)

	ctx := context.Background()
	commitService, err := container.GetCommitService(ctx)
	if err != nil {
		log.Printf("Warning: la inicialización del servicio de confirmación falló: %v", err)
		log.Println("La IA no está configurada. Podés configurarla con 'matecommit config init'")
	}

	var vcsClient = container.GetVCSRegistry()
	vcsClientInstance, _ := vcsClient.CreateClientFromConfig(ctx, gitService, cfgApp, translations)
	commitHandler := handler.NewSuggestionHandler(gitService, vcsClientInstance, translations)
	aiSummarizer, err := container.GetPRSummarizer(ctx)
	if err != nil {
		log.Printf("Warning: no se pudo crear el servicio de IA para PRs: %v", err)
		aiSummarizer = nil
	}

	prServiceFactory := factory.NewPrServiceFactory(cfgApp, translations, aiSummarizer, gitService)
	prCommand := pull_requests.NewSummarizeCommand(prServiceFactory)

	registerCommand := registry.NewRegistry(cfgApp, translations)

	if err := registerCommand.Register("suggest", suggests_commits.NewSuggestCommandFactory(commitService, commitHandler)); err != nil {
		log.Fatalf("Error al registrar el comando 'suggest': %v", err)
	}

	issueServiceProvider := func(ctx context.Context) (ports.IssueGeneratorService, error) {
		return container.GetIssueGeneratorService(ctx)
	}

	if err := registerCommand.Register("issue", issues.NewIssuesCommandFactory(issueServiceProvider, container.GetIssueTemplateService())); err != nil {
		log.Fatalf("Error al registrar el comando 'issue': %v", err)
	}

	if err := registerCommand.Register("config", config.NewConfigCommandFactory()); err != nil {
		log.Fatalf("Error al registrar el comando 'config': %v", err)
	}

	if err := registerCommand.Register("doctor", config.NewDoctorCommand()); err != nil {
		log.Fatalf("Error al registrar el comando 'doctor': %v", err)
	}

	if err := registerCommand.Register("summarize-pr", prCommand); err != nil {
		log.Fatalf("Error al registrar el comando 'summarize-pr': %v", err)
	}

	if err := registerCommand.Register("release", release.NewReleaseCommandFactory(gitService, cfgApp)); err != nil {
		log.Fatalf("Error al registrar el comando 'release': %v", err)
	}

	if err := registerCommand.Register("update", update.NewUpdateCommandFactory("v1.4.0")); err != nil {
		log.Fatalf("Error al registrar el comando 'update': %v", err)
	}

	commands := registerCommand.CreateCommands()
	commands = append(commands, completion.NewCompletionCommand(translations))
	commands = append(commands, stats.NewStatsCommand().CreateCommand(translations, cfgApp))
	commands = append(commands, cache.NewCacheCommand().CreateCommand(translations, cfgApp))

	helpCommand := &cli.Command{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   translations.GetMessage("help_command_usage", 0, nil),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowAppHelp(cmd)
		},
	}
	commands = append(commands, helpCommand)

	go func() {
		checker := services.NewVersionUpdater(version.FullVersion(), translations)
		checker.CheckForUpdates(context.Background())
	}()

	return &cli.Command{
		Name:                  "mate-commit",
		Usage:                 translations.GetMessage("app_usage", 0, nil),
		Version:               version.Version,
		Description:           translations.GetMessage("app_description", 0, nil),
		Commands:              commands,
		EnableShellCompletion: true,
	}, nil
}
