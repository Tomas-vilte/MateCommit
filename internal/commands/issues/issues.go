package issues

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// IssueGeneratorService is a minimal interface for testing purposes
type IssueGeneratorService interface {
	GenerateFromDiff(ctx context.Context, hint string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error)
	GenerateFromDescription(ctx context.Context, description string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error)
	GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool, autoTemplate bool) (*models.IssueGenerationResult, error)
	GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error)
	CreateIssue(ctx context.Context, result *models.IssueGenerationResult, assignees []string) (*models.Issue, error)
	GetAuthenticatedUser(ctx context.Context) (string, error)
	InferBranchName(issueNumber int, labels []string) string
	LinkIssueToPR(ctx context.Context, prNumber int, issueNumber int) error
}

// IssueTemplateService is a minimal interface for testing purposes
type IssueTemplateService interface {
	InitializeTemplates(ctx context.Context, force bool) error
	GetTemplatesDir(ctx context.Context) (string, error)
	ListTemplates(ctx context.Context) ([]models.TemplateMetadata, error)
}

type IssueServiceProvider func(ctx context.Context) (IssueGeneratorService, error)

// IssuesCommandFactory is the factory to create the issues command.
type IssuesCommandFactory struct {
	issueServiceProvider IssueServiceProvider
	templateService      IssueTemplateService
}

// NewIssuesCommandFactory creates a new instance of the factory.
func NewIssuesCommandFactory(issueServiceProvider IssueServiceProvider, templateService IssueTemplateService) *IssuesCommandFactory {
	return &IssuesCommandFactory{
		issueServiceProvider: issueServiceProvider,
		templateService:      templateService,
	}
}

// CreateCommand creates the main issues command with its subcommands.
func (f *IssuesCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "issue",
		Aliases: []string{"i"},
		Usage:   t.GetMessage("issue.command_usage", 0, nil),
		Commands: []*cli.Command{
			f.newGenerateCommand(t, cfg),
			f.newLinkCommand(t, cfg),
			f.newTemplateCommand(t, cfg),
			f.newFromPlanCommand(t, cfg),
		},
	}
}

// newGenerateCommand creates the 'generate' subcommand.
func (f *IssuesCommandFactory) newGenerateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:          "generate",
		Aliases:       []string{"g"},
		Usage:         t.GetMessage("issue.generate_usage", 0, nil),
		Flags:         f.createGenerateFlags(t),
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action:        f.createGenerateAction(t, cfg),
	}
}

// createGenerateFlags defines the flags for the generate command.
func (f *IssuesCommandFactory) createGenerateFlags(t *i18n.Translations) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "from-diff",
			Aliases: []string{"d"},
			Usage:   t.GetMessage("issue.flag_from_diff", 0, nil),
		},
		&cli.IntFlag{
			Name:    "from-pr",
			Aliases: []string{"p", "pr"},
			Usage:   t.GetMessage("issue.flag_from_pr", 0, nil),
			Value:   0,
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"m"},
			Usage:   t.GetMessage("issue.flag_description", 0, nil),
		},
		&cli.StringFlag{
			Name:    "hint",
			Aliases: []string{"h"},
			Usage:   t.GetMessage("issue.flag_hint", 0, nil),
		},
		&cli.StringFlag{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   t.GetMessage("issue.flag_template", 0, nil),
		},
		&cli.BoolFlag{
			Name:  "no-labels",
			Usage: t.GetMessage("issue.flag_no_labels", 0, nil),
		},
		&cli.BoolFlag{
			Name:  "dry-run",
			Usage: t.GetMessage("issue.flag_dry_run", 0, nil),
		},
		&cli.BoolFlag{
			Name:    "assign-me",
			Aliases: []string{"a"},
			Usage:   t.GetMessage("issue.flag_assign_me", 0, nil),
		},
		&cli.BoolFlag{
			Name:    "checkout",
			Aliases: []string{"c"},
			Usage:   t.GetMessage("issue.flag_checkout", 0, nil),
		},
		&cli.BoolFlag{
			Name:  "auto-template",
			Usage: t.GetMessage("issue.auto_template_flag", 0, nil),
			Value: true,
		},
	}
}

func (f *IssuesCommandFactory) createGenerateAction(t *i18n.Translations, cfg *config.Config) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		log := logger.FromContext(ctx)
		start := time.Now()

		fromDiff := command.Bool("from-diff")
		fromPR := command.Int("from-pr")
		description := command.String("description")
		hint := command.String("hint")
		noLabels := command.Bool("no-labels")
		dryRun := command.Bool("dry-run")
		assignMe := command.Bool("assign-me")
		checkoutBranch := command.Bool("checkout")
		templateName := command.String("template")
		autoTemplate := command.Bool("auto-template")

		log.Info("executing issue generate command",
			"from_diff", fromDiff,
			"from_pr", fromPR,
			"has_description", description != "",
			"has_hint", hint != "",
			"no_labels", noLabels,
			"dry_run", dryRun,
			"assign_me", assignMe,
			"checkout_branch", checkoutBranch,
			"template", templateName)

		sourcesCount := 0
		if fromDiff {
			sourcesCount++
		}
		if fromPR > 0 {
			sourcesCount++
		}
		if description != "" {
			sourcesCount++
		}

		if sourcesCount == 0 {
			log.Error("no input source provided",
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_no_input", 0, nil))
			return fmt.Errorf("%s", t.GetMessage("issue.error_no_input", 0, nil))
		}

		if sourcesCount > 1 {
			log.Error("multiple input sources provided",
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_multiple_sources", 0, nil))
			return fmt.Errorf("%s", t.GetMessage("issue.error_multiple_sources", 0, nil))
		}

		ui.PrintSectionBanner(t.GetMessage("issue.banner", 0, nil))

		issueService, err := f.issueServiceProvider(ctx)
		if err != nil {
			log.Error("failed to create issue service",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.error_generating", 0, nil), err))
			return err
		}

		var spinnerMsg string
		if fromPR > 0 {
			spinnerMsg = t.GetMessage("issue.analyzing_pr", 0, struct{ Number int }{fromPR})
		} else {
			spinnerMsg = t.GetMessage("issue.analyzing", 0, nil)
		}

		spinner := ui.NewSmartSpinner(spinnerMsg)
		spinner.Start()

		var result *models.IssueGenerationResult

		if templateName != "" {
			result, err = issueService.GenerateWithTemplate(ctx, templateName, hint, fromDiff, description, noLabels)
		} else if fromDiff {
			result, err = issueService.GenerateFromDiff(ctx, hint, noLabels, autoTemplate)
		} else if fromPR > 0 {
			result, err = issueService.GenerateFromPR(ctx, fromPR, hint, noLabels, autoTemplate)
		} else {
			result, err = issueService.GenerateFromDescription(ctx, description, noLabels, autoTemplate)
		}

		spinner.Stop()

		if err != nil {
			log.Error("failed to generate issue",
				"error", err,
				"from_diff", fromDiff,
				"from_pr", fromPR,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err, t)
			return err
		}

		log.Debug("issue generated",
			"title", result.Title,
			"labels_count", len(result.Labels))

		f.printPreview(result, t, cfg)
		ui.PrintTokenUsage(result.Usage, t)

		if dryRun {
			ui.PrintInfo(t.GetMessage("issue.dry_run_complete", 0, nil))
			return nil
		}

		if !f.promptConfirmation(t) {
			ui.PrintInfo(t.GetMessage("issue.cancelled", 0, nil))
			return nil
		}

		var assignees []string
		if assignMe {
			spinner = ui.NewSmartSpinner(t.GetMessage("issue.getting_user", 0, nil))
			spinner.Start()

			username, err := issueService.GetAuthenticatedUser(ctx)
			spinner.Stop()

			if err != nil {
				ui.PrintWarning(fmt.Sprintf("%s: %v", t.GetMessage("issue.warn_assignee_failed", 0, nil), err))
			} else {
				assignees = []string{username}
				ui.PrintInfo(t.GetMessage("issue.will_assign", 0, struct{ User string }{username}))
			}
		}

		spinner = ui.NewSmartSpinner(t.GetMessage("issue.creating", 0, nil))
		spinner.Start()

		issue, err := issueService.CreateIssue(ctx, result, assignees)
		spinner.Stop()

		if err != nil {
			log.Error("failed to create issue",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err, t)
			return err
		}

		log.Info("issue created successfully",
			"issue_number", issue.Number,
			"issue_url", issue.URL,
			"assignees_count", len(assignees),
			"duration_ms", time.Since(start).Milliseconds())

		ui.PrintSuccess(os.Stdout, t.GetMessage("issue.created_successfully", 0, struct {
			Number int
			URL    string
		}{issue.Number, issue.URL}))

		if fromPR > 0 {
			if err := issueService.LinkIssueToPR(ctx, fromPR, issue.Number); err != nil {
				ui.PrintWarning(t.GetMessage("issue.link_error", 0, struct {
					PR    int
					Error error
				}{fromPR, err}))
			} else {
				ui.PrintInfo(t.GetMessage("issue.link_success", 0, struct {
					PR    int
					Issue int
				}{fromPR, issue.Number}))
			}
		}

		if checkoutBranch {
			branchName := issueService.InferBranchName(issue.Number, result.Labels)

			ui.PrintInfo(t.GetMessage("issue.creating_branch", 0, struct{ Branch string }{branchName}))

			if err := f.checkoutBranch(branchName); err != nil {
				ui.PrintWarning(fmt.Sprintf("%s: %v", t.GetMessage("issue.warn_checkout_failed", 0, nil), err))
			} else {
				ui.PrintSuccess(os.Stdout, t.GetMessage("issue.branch_created", 0, struct{ Branch string }{branchName}))
			}
		}

		return nil
	}
}

// handleCreationError is now replaced by ui.HandleAppError

// printPreview shows a preview of the issue to be created.
func (f *IssuesCommandFactory) printPreview(result *models.IssueGenerationResult, t *i18n.Translations, cfg *config.Config) {
	separator := strings.Repeat("\u2500", 60)

	fmt.Println()
	fmt.Println(separator)

	emoji := ""
	if cfg.UseEmoji {
		emoji = "\U0001F4CB "
	}

	ui.PrintInfo(fmt.Sprintf("%s%s", emoji, t.GetMessage("issue.preview_title", 0, nil)))
	fmt.Println()

	ui.PrintKeyValue(t.GetMessage("issue.preview_title_label", 0, nil), result.Title)
	fmt.Println()

	ui.PrintInfo(fmt.Sprintf("%s:", t.GetMessage("issue.preview_description_label", 0, nil)))
	fmt.Println(result.Description)
	fmt.Println()

	if len(result.Labels) > 0 {
		ui.PrintInfo(fmt.Sprintf("%s: %s", t.GetMessage("issue.preview_labels_label", 0, nil), strings.Join(result.Labels, ", ")))
	}

	fmt.Println(separator)
	fmt.Println()
}

// promptConfirmation requests confirmation from the user to create the issue.
func (f *IssuesCommandFactory) promptConfirmation(t *i18n.Translations) bool {
	reader := bufio.NewReader(os.Stdin)

	prompt := t.GetMessage("issue.confirm_prompt", 0, nil)
	fmt.Printf("%s: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))

	return response == "" || response == "y" || response == "yes"
}

func (f *IssuesCommandFactory) checkoutBranch(branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %w (output: %s)", err, string(output))
	}
	return nil
}

func (f *IssuesCommandFactory) newLinkCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "link",
		Aliases: []string{"l"},
		Usage:   t.GetMessage("issue.link_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "pr",
				Aliases:  []string{"p"},
				Usage:    t.GetMessage("issue.flag_pr_number", 0, nil),
				Required: true,
			},
			&cli.IntFlag{
				Name:     "issue",
				Aliases:  []string{"i"},
				Usage:    t.GetMessage("issue.flag_issue_number", 0, nil),
				Required: true,
			},
		},
		Action: f.createLinkAction(t, cfg),
	}
}

// createLinkAction creates the action to link a PR to an issue.
func (f *IssuesCommandFactory) createLinkAction(t *i18n.Translations, _ *config.Config) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		log := logger.FromContext(ctx)
		start := time.Now()

		prNumber := command.Int("pr")
		issueNumber := command.Int("issue")

		log.Info("executing issue link command",
			"pr_number", prNumber,
			"issue_number", issueNumber)

		if prNumber <= 0 {
			log.Error("invalid PR number",
				"pr_number", prNumber,
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_invalid_pr", 0, nil))
			return fmt.Errorf("invalid PR number")
		}

		if issueNumber <= 0 {
			log.Error("invalid issue number",
				"issue_number", issueNumber,
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_invalid_issue", 0, nil))
			return fmt.Errorf("invalid issue number")
		}

		ui.PrintSectionBanner(t.GetMessage("issue.link_banner", 0, nil))

		issueService, err := f.issueServiceProvider(ctx)
		if err != nil {
			log.Error("failed to create issue service",
				"error", err,
				"duration_ms", time.Since(start).Milliseconds())
			ui.HandleAppError(err, t)
			return err
		}

		spinner := ui.NewSmartSpinner(t.GetMessage("issue.linking", 0, struct {
			PR    int
			Issue int
		}{prNumber, issueNumber}))
		spinner.Start()

		err = issueService.LinkIssueToPR(ctx, prNumber, issueNumber)
		spinner.Stop()

		if err != nil {
			log.Error("failed to link issue to PR",
				"error", err,
				"pr_number", prNumber,
				"issue_number", issueNumber,
				"duration_ms", time.Since(start).Milliseconds())
			ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.error_linking", 0, nil), err))
			return err
		}

		log.Info("issue linked to PR successfully",
			"pr_number", prNumber,
			"issue_number", issueNumber,
			"duration_ms", time.Since(start).Milliseconds())

		ui.PrintSuccess(os.Stdout, t.GetMessage("issue.link_success", 0, struct {
			PR    int
			Issue int
		}{prNumber, issueNumber}))

		return nil
	}
}
