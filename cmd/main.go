package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/ai/gemini"
	"github.com/thomas-vilte/matecommit/internal/commands/cache"
	"github.com/thomas-vilte/matecommit/internal/commands/completion"
	"github.com/thomas-vilte/matecommit/internal/commands/config"
	"github.com/thomas-vilte/matecommit/internal/commands/handler"
	"github.com/thomas-vilte/matecommit/internal/commands/issues"
	"github.com/thomas-vilte/matecommit/internal/commands/pull_requests"
	"github.com/thomas-vilte/matecommit/internal/commands/release"
	"github.com/thomas-vilte/matecommit/internal/commands/stats"
	"github.com/thomas-vilte/matecommit/internal/commands/suggests_commits"
	"github.com/thomas-vilte/matecommit/internal/commands/update"
	cfg "github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/git"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/services"
	"github.com/thomas-vilte/matecommit/internal/tickets"
	"github.com/thomas-vilte/matecommit/internal/tickets/jira"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/thomas-vilte/matecommit/internal/vcs"
	"github.com/thomas-vilte/matecommit/internal/vcs/github"
	"github.com/thomas-vilte/matecommit/internal/version"
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

	cfgApp, err := cfg.LoadConfigWithHierarchy(homeDir)
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
	gitService.SetFallback(cfgApp.GitFallback.UserName, cfgApp.GitFallback.UserEmail)
	isCompletion := checkCompletion()

	commitAI, prAI, issueAI := initAIProviders(ctx, cfgApp, translations, isCompletion)
	vcsClient := initVCSClient(ctx, gitService, cfgApp, isCompletion)
	ticketMgr := initTicketManager(ctx, cfgApp, isCompletion)

	commitService, prService, issueService, templateService := initServices(cfgApp, gitService, commitAI, prAI, issueAI, vcsClient, ticketMgr)

	commitHandler := handler.NewSuggestionHandler(gitService, vcsClient, translations)
	commands := setupCommands(translations, cfgApp, gitService, commitService, prService, issueService, templateService, commitHandler)

	startBackgroundVersionCheck()

	return &cli.Command{
		Name:                  "mate-commit",
		Usage:                 translations.GetMessage("app_usage", 0, nil),
		Version:               version.Version,
		Description:           translations.GetMessage("app_description", 0, nil),
		Commands:              commands,
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Usage: translations.GetMessage("flags_global.debug_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   translations.GetMessage("flags_global.verbose_flag", 0, nil),
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			logger.Initialize(c.Bool("debug"), c.Bool("verbose"))
			handleVersionNotification(translations)
			return ctx, nil
		},
	}, nil
}

func checkCompletion() bool {
	for _, arg := range os.Args {
		if arg == "completion" || arg == "--generate-shell-completion" {
			return true
		}
	}
	return false
}

func initAIProviders(ctx context.Context, cfgApp *cfg.Config, t *i18n.Translations, isCompletion bool) (ai.CommitSummarizer, ai.PRSummarizer, ai.IssueContentGenerator) {
	if cfgApp.AIConfig.ActiveAI == "" {
		return nil, nil, nil
	}

	onConfirmation := createConfirmationCallback(t)

	switch cfgApp.AIConfig.ActiveAI {
	case "gemini":
		commitAI, err := gemini.NewGeminiCommitSummarizer(ctx, cfgApp, onConfirmation)
		if err != nil && !isCompletion {
			logger.Warn(ctx, "could not create CommitSummarizer", "error", err)
			logger.Info(ctx, "AI is not configured. You can configure it with 'mate-commit config init'")
		}

		prAI, err := gemini.NewGeminiPRSummarizer(ctx, cfgApp, onConfirmation)
		if err != nil && !isCompletion {
			logger.Debug(ctx, "could not create PRSummarizer", "error", err)
		}

		issueAI, err := gemini.NewGeminiIssueContentGenerator(ctx, cfgApp, onConfirmation)
		if err != nil && !isCompletion {
			logger.Debug(ctx, "could not create IssueContentGenerator", "error", err)
		}

		return commitAI, prAI, issueAI
	default:
		if !isCompletion {
			logger.Warn(ctx, "unsupported AI provider", "provider", cfgApp.AIConfig.ActiveAI)
		}
		return nil, nil, nil
	}
}

func initVCSClient(ctx context.Context, gitService *git.GitService, cfgApp *cfg.Config, isCompletion bool) vcs.VCSClient {
	owner, repo, provider, err := gitService.GetRepoInfo(ctx)
	if err != nil {
		logger.Debug(ctx, "VCS client not available", "reason", "not in a git repository or no remote configured")
		return nil
	}

	vcsConfig, exists := cfgApp.VCSConfigs[provider]
	if !exists {
		vcsConfig, exists = cfgApp.VCSConfigs[cfgApp.ActiveVCSProvider]
		if !exists {
			if !isCompletion {
				logger.Debug(ctx, "VCS provider not configured", "provider", provider)
			}
			return nil
		}
		provider = cfgApp.ActiveVCSProvider
	}

	switch provider {
	case "github":
		if vcsConfig.Token == "" {
			if !isCompletion {
				logger.Debug(ctx, "GitHub token is empty", "error", domainErrors.ErrTokenMissing)
			}
			return nil
		}
		return github.NewGitHubClient(owner, repo, vcsConfig.Token)
	default:
		if !isCompletion {
			logger.Debug(ctx, "unsupported VCS provider", "provider", provider)
		}
		return nil
	}
}

func initTicketManager(ctx context.Context, cfgApp *cfg.Config, isCompletion bool) tickets.TicketManager {
	if cfgApp.ActiveTicketService == "" || !cfgApp.UseTicket {
		return nil
	}

	ticketCfg, exists := cfgApp.TicketProviders[cfgApp.ActiveTicketService]
	if !exists {
		if !isCompletion {
			logger.Debug(ctx, "ticket provider not configured", "provider", cfgApp.ActiveTicketService)
		}
		return nil
	}

	switch cfgApp.ActiveTicketService {
	case "jira":
		return jira.NewJiraService(ticketCfg.BaseURL, ticketCfg.APIKey, ticketCfg.Email, &http.Client{})
	default:
		if !isCompletion {
			logger.Debug(ctx, "unsupported ticket provider", "provider", cfgApp.ActiveTicketService)
		}
		return nil
	}
}

func initServices(cfgApp *cfg.Config, gitService *git.GitService, commitAI ai.CommitSummarizer, prAI ai.PRSummarizer, issueAI ai.IssueContentGenerator, vcsClient vcs.VCSClient, ticketMgr tickets.TicketManager) (*services.CommitService, *services.PRService, *services.IssueGeneratorService, *services.IssueTemplateService) {
	commitService := services.NewCommitService(
		gitService,
		commitAI,
		services.WithTicketManager(ticketMgr),
		services.WithVCSClient(vcsClient),
		services.WithConfig(cfgApp),
	)

	templateService := services.NewIssueTemplateService(
		services.WithTemplateConfig(cfgApp),
	)

	prService := services.NewPRService(
		services.WithPRVCSClient(vcsClient),
		services.WithPRAIProvider(prAI),
		services.WithPRConfig(cfgApp),
		services.WithPRTemplateService(templateService),
	)

	issueService := services.NewIssueGeneratorService(
		gitService,
		issueAI,
		services.WithIssueVCSClient(vcsClient),
		services.WithIssueTemplateService(templateService),
		services.WithIssueConfig(cfgApp),
	)

	return commitService, prService, issueService, templateService
}

func setupCommands(t *i18n.Translations, cfgApp *cfg.Config, gitService *git.GitService, commitService *services.CommitService, prService *services.PRService, issueService *services.IssueGeneratorService, templateService *services.IssueTemplateService, commitHandler *handler.SuggestionHandler) []*cli.Command {
	issueProvider := func(ctx context.Context) (issues.IssueGeneratorService, error) {
		return issueService, nil
	}

	commands := []*cli.Command{
		suggests_commits.NewSuggestCommandFactory(commitService, commitHandler, gitService).CreateCommand(t, cfgApp),
		issues.NewIssuesCommandFactory(issueProvider, templateService).CreateCommand(t, cfgApp),
		pull_requests.NewSummarizeCommand(func(ctx context.Context) (pull_requests.PRService, error) {
			return prService, nil
		}).CreateCommand(t, cfgApp),
		release.NewReleaseCommandFactory(gitService, cfgApp).CreateCommand(t, cfgApp),
		config.NewConfigCommandFactory().CreateCommand(t, cfgApp),
		config.NewDoctorCommand().CreateCommand(t, cfgApp),
		update.NewUpdateCommandFactory("v1.4.0").CreateCommand(t, cfgApp),
		completion.NewCompletionCommand(t),
		stats.NewStatsCommand().CreateCommand(t, cfgApp),
		cache.NewCacheCommand().CreateCommand(t, cfgApp),
	}

	commands = append(commands, &cli.Command{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   t.GetMessage("help_command_usage", 0, nil),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return cli.ShowAppHelp(cmd)
		},
	})

	return commands
}

func startBackgroundVersionCheck() {
	go func() {
		checker := services.NewVersionUpdater(
			services.WithCurrentVersion(version.FullVersion()),
		)
		_, _ = checker.CheckForUpdates(context.Background())
	}()
}

func handleVersionNotification(t *i18n.Translations) {
	args := os.Args
	if len(args) > 1 {
		cmdName := args[1]
		if cmdName != "update" && cmdName != "completion" {
			checker := services.NewVersionUpdater(
				services.WithCurrentVersion(version.FullVersion()),
			)

			updateInfo, err := checker.GetUpdateInfo()
			if err != nil || updateInfo == nil {
				return
			}

			yellow := color.New(color.FgYellow, color.Bold)
			cyan := color.New(color.FgCyan)

			fmt.Println()
			_, _ = cyan.Println(t.GetMessage("update.box_top", 0, nil))
			_, _ = yellow.Println(t.GetMessage("update.available", 0, map[string]interface{}{
				"Current": updateInfo.CurrentVersion,
				"Latest":  updateInfo.LatestVersion,
			}))
			fmt.Println(t.GetMessage("update.command", 0, map[string]interface{}{
				"Command": "matecommit update",
			}))
			_, _ = cyan.Println(t.GetMessage("update.box_bottom", 0, nil))
			fmt.Println()
		}
	}
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
