package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

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
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/providers"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/Tomas-vilte/MateCommit/internal/version"
	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
)

func main() {
	app, err := initializeApp()
	if err != nil {
		log.Fatalf("Error starting the CLI: %v", err)
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func initializeApp() (*cli.Command, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get user home directory: %w", err)
	}

	cfgApp, err := cfg.LoadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	translations, err := i18n.NewTranslations(cfgApp.Language, "")
	if err != nil {
		log.Fatalf("Error loading translations: %v", err)
	}

	if err = cfg.SaveConfig(cfgApp); err != nil {
		return nil, err
	}

	ctx := context.Background()

	gitService := git.NewGitService()

	var commitAI ports.CommitSummarizer
	var prAI ports.PRSummarizer
	var issueAI ports.IssueContentGenerator

	if cfgApp.AIConfig.ActiveAI != "" {
		onConfirmation := createConfirmationCallback(translations)

		commitAI, err = providers.NewCommitSummarizer(ctx, cfgApp, onConfirmation)
		if err != nil {
			log.Printf("Warning: could not create CommitSummarizer: %v", err)
			log.Println("AI is not configured. You can configure it with 'mate-commit config init'")
		}

		prAI, err = providers.NewPRSummarizer(ctx, cfgApp, onConfirmation)
		if err != nil {
			log.Printf("Warning: could not create PRSummarizer: %v", err)
			prAI = nil
		}

		issueAI, err = providers.NewIssueContentGenerator(ctx, cfgApp, onConfirmation)
		if err != nil {
			log.Printf("Warning: could not create IssueContentGenerator: %v", err)
			issueAI = nil
		}
	}

	vcsClient, err := providers.NewVCSClient(ctx, gitService, cfgApp)
	if err != nil {
		log.Printf("Warning: could not create VCS client: %v", err)
		vcsClient = nil
	}

	ticketMgr, err := providers.NewTicketManager(ctx, cfgApp)
	if err != nil {
		log.Printf("Warning: could not create Ticket manager: %v", err)
		ticketMgr = nil
	}

	commitService := services.NewCommitService(
		gitService,
		commitAI,
		services.WithTicketManager(ticketMgr),
		services.WithVCSClient(vcsClient),
		services.WithConfig(cfgApp),
	)

	prService := services.NewPRService(
		services.WithPRVCSClient(vcsClient),
		services.WithPRAIProvider(prAI),
		services.WithPRConfig(cfgApp),
	)

	templateService := services.NewIssueTemplateService(
		services.WithTemplateConfig(cfgApp),
	)

	issueService := services.NewIssueGeneratorService(
		gitService,
		issueAI,
		services.WithIssueVCSClient(vcsClient),
		services.WithIssueTemplateService(templateService),
		services.WithIssueConfig(cfgApp),
	)

	commitHandler := handler.NewSuggestionHandler(gitService, vcsClient, translations)

	issueProvider := func(ctx context.Context) (issues.IssueGeneratorService, error) {
		return issueService, nil
	}
	commands := []*cli.Command{
		suggests_commits.NewSuggestCommandFactory(commitService, commitHandler).CreateCommand(translations, cfgApp),
		issues.NewIssuesCommandFactory(
			issueProvider,
			templateService,
		).CreateCommand(translations, cfgApp),
		pull_requests.NewSummarizeCommand(func(ctx context.Context) (pull_requests.PRService, error) {
			return prService, nil
		}).CreateCommand(translations, cfgApp),
		release.NewReleaseCommandFactory(gitService, cfgApp).CreateCommand(translations, cfgApp),
		config.NewConfigCommandFactory().CreateCommand(translations, cfgApp),
		config.NewDoctorCommand().CreateCommand(translations, cfgApp),
		update.NewUpdateCommandFactory("v1.4.0").CreateCommand(translations, cfgApp),
		completion.NewCompletionCommand(translations),
		stats.NewStatsCommand().CreateCommand(translations, cfgApp),
		cache.NewCacheCommand().CreateCommand(translations, cfgApp),
	}

	commands = append(commands, &cli.Command{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   translations.GetMessage("help_command_usage", 0, nil),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowAppHelp(cmd)
		},
	})

	go func() {
		checker := services.NewVersionUpdater(
			services.WithCurrentVersion(version.FullVersion()),
		)
		_, _ = checker.CheckForUpdates(context.Background())
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

func createConfirmationCallback(t *i18n.Translations) ai.ConfirmationCallback {
	return func(result ai.ConfirmationResult) (string, bool) {
		ui.SuspendActiveSpinner()
		defer ui.ResumeSuspendedSpinner()

		cyan := color.New(color.FgCyan, color.Bold)
		yellow := color.New(color.FgYellow)
		hasSuggestion := result.SuggestedModel != "" && result.SuggestedModel != result.CurrentModel

		fmt.Println()
		_, _ = cyan.Println(t.GetMessage("cost.confirmation_separator", 0, nil))
		_, _ = cyan.Println(t.GetMessage("cost.confirmation_header", 0, nil))
		_, _ = cyan.Println(t.GetMessage("cost.confirmation_separator", 0, nil))

		if result.RationaleKey != "" {
			rationale := t.GetMessage(result.RationaleKey, 0, nil)
			_, _ = yellow.Println(t.GetMessage("cost.routing_suggestion", 0, map[string]interface{}{
				"Rationale": rationale,
			}))
			_, _ = yellow.Println(t.GetMessage("cost.routing_suggested_model", 0, map[string]interface{}{
				"Suggested": result.SuggestedModel,
				"Current":   result.CurrentModel,
			}))
		}

		fmt.Println(t.GetMessage("cost.confirmation_input_tokens", 0, map[string]interface{}{
			"Tokens": yellow.Sprintf("%d", result.InputTokens),
		}))
		fmt.Println(t.GetMessage("cost.confirmation_output_tokens", 0, map[string]interface{}{
			"Tokens": yellow.Sprintf("%d", result.OutputTokens),
		}))

		costLabel := t.GetMessage("cost.confirmation_estimated_cost", 0, map[string]interface{}{
			"Cost": yellow.Sprintf("$%.4f", result.EstimatedCost),
		})
		if hasSuggestion {
			fmt.Printf("%s (%s)\n", costLabel, result.SuggestedModel)
		} else {
			fmt.Println(costLabel)
		}

		_, _ = cyan.Println(t.GetMessage("cost.confirmation_separator", 0, nil))

		if hasSuggestion {
			fmt.Printf("%s ", t.GetMessage("cost.confirmation_use_suggested", 0, nil))
			fmt.Printf("%s\n", color.HiBlackString(t.GetMessage("cost.confirmation_use_suggested_help", 0, nil)))
		} else {
			fmt.Printf("%s ", t.GetMessage("cost.confirmation_prompt", 0, nil))
		}

		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))

		if !hasSuggestion {
			proceed := response == "" || response == "y" || response == "yes"
			if proceed {
				return "original", true
			}
			return "", false
		}

		switch response {
		case "", "y", "yes":
			return "suggested", true
		case "m", "stay":
			return "original", true
		case "c", "cancel", "n", "no":
			return "", false
		default:
			return "", false
		}
	}
}
