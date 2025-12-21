package issues

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// IssueGeneratorService is a minimal interface for testing purposes
type IssueGeneratorService interface {
	GenerateFromDiff(ctx context.Context, hint string, skipLabels bool) (*models.IssueGenerationResult, error)
	GenerateFromDescription(ctx context.Context, description string, skipLabels bool) (*models.IssueGenerationResult, error)
	GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool) (*models.IssueGenerationResult, error)
	GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error)
	CreateIssue(ctx context.Context, result *models.IssueGenerationResult, assignees []string) (*models.Issue, error)
	GetAuthenticatedUser(ctx context.Context) (string, error)
	InferBranchName(issueNumber int, labels []string) string
	LinkIssueToPR(ctx context.Context, prNumber int, issueNumber int) error
}

// IssueTemplateService is a minimal interface for testing purposes
type IssueTemplateService interface {
	InitializeTemplates(force bool) error
	GetTemplatesDir() (string, error)
	ListTemplates() ([]models.TemplateMetadata, error)
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
	}
}

func (f *IssuesCommandFactory) createGenerateAction(t *i18n.Translations, cfg *config.Config) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		fromDiff := command.Bool("from-diff")
		fromPR := command.Int("from-pr")
		description := command.String("description")
		hint := command.String("hint")
		noLabels := command.Bool("no-labels")
		dryRun := command.Bool("dry-run")
		assignMe := command.Bool("assign-me")
		checkoutBranch := command.Bool("checkout")
		templateName := command.String("template")

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
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_no_input", 0, nil))
			return fmt.Errorf("%s", t.GetMessage("issue.error_no_input", 0, nil))
		}

		if sourcesCount > 1 {
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_multiple_sources", 0, nil))
			return fmt.Errorf("%s", t.GetMessage("issue.error_multiple_sources", 0, nil))
		}

		ui.PrintSectionBanner(t.GetMessage("issue.banner", 0, nil))

		issueService, err := f.issueServiceProvider(ctx)
		if err != nil {
			ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.error_generating", 0, nil), err))
			return err
		}

		var spinnerMsg string
		if fromPR > 0 {
			spinnerMsg = t.GetMessage("issue.analyzing_pr", 0, map[string]interface{}{
				"Number": fromPR,
			})
		} else {
			spinnerMsg = t.GetMessage("issue.analyzing", 0, nil)
		}

		spinner := ui.NewSmartSpinner(spinnerMsg)
		spinner.Start()

		var result *models.IssueGenerationResult

		if templateName != "" {
			result, err = issueService.GenerateWithTemplate(ctx, templateName, hint, fromDiff, description, noLabels)
		} else if fromDiff {
			result, err = issueService.GenerateFromDiff(ctx, hint, noLabels)
		} else if fromPR > 0 {
			result, err = issueService.GenerateFromPR(ctx, fromPR, hint, noLabels)
		} else {
			result, err = issueService.GenerateFromDescription(ctx, description, noLabels)
		}

		spinner.Stop()

		if err != nil {
			ui.HandleAppError(err, t)
			return err
		}

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
				ui.PrintInfo(t.GetMessage("issue.will_assign", 0, map[string]interface{}{
					"User": username,
				}))
			}
		}

		spinner = ui.NewSmartSpinner(t.GetMessage("issue.creating", 0, nil))
		spinner.Start()

		issue, err := issueService.CreateIssue(ctx, result, assignees)
		spinner.Stop()

		if err != nil {
			ui.HandleAppError(err, t)
			return err
		}

		ui.PrintSuccess(os.Stdout, t.GetMessage("issue.created_successfully", 0, map[string]interface{}{
			"Number": issue.Number,
			"URL":    issue.URL,
		}))

		if fromPR > 0 {
			if err := issueService.LinkIssueToPR(ctx, fromPR, issue.Number); err != nil {
				ui.PrintWarning(t.GetMessage("issue.link_error", 0, map[string]interface{}{
					"PR":    fromPR,
					"Error": err,
				}))
			} else {
				ui.PrintInfo(t.GetMessage("issue.link_success", 0, map[string]interface{}{
					"PR":    fromPR,
					"Issue": issue.Number,
				}))
			}
		}

		if checkoutBranch {
			branchName := issueService.InferBranchName(issue.Number, result.Labels)

			ui.PrintInfo(t.GetMessage("issue.creating_branch", 0, map[string]interface{}{
				"Branch": branchName,
			}))

			if err := f.checkoutBranch(branchName); err != nil {
				ui.PrintWarning(fmt.Sprintf("%s: %v", t.GetMessage("issue.warn_checkout_failed", 0, nil), err))
			} else {
				ui.PrintSuccess(os.Stdout, t.GetMessage("issue.branch_created", 0, map[string]interface{}{
					"Branch": branchName,
				}))
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
		prNumber := command.Int("pr")
		issueNumber := command.Int("issue")

		if prNumber <= 0 {
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_invalid_pr", 0, nil))
			return fmt.Errorf("invalid PR number")
		}

		if issueNumber <= 0 {
			ui.PrintError(os.Stdout, t.GetMessage("issue.error_invalid_issue", 0, nil))
			return fmt.Errorf("invalid issue number")
		}

		ui.PrintSectionBanner(t.GetMessage("issue.link_banner", 0, nil))

		issueService, err := f.issueServiceProvider(ctx)
		if err != nil {
			ui.HandleAppError(err, t)
			return err
		}

		spinner := ui.NewSmartSpinner(t.GetMessage("issue.linking", 0, map[string]interface{}{
			"PR":    prNumber,
			"Issue": issueNumber,
		}))
		spinner.Start()

		err = issueService.LinkIssueToPR(ctx, prNumber, issueNumber)
		spinner.Stop()

		if err != nil {
			ui.PrintError(os.Stdout, fmt.Sprintf("%s: %v", t.GetMessage("issue.error_linking", 0, nil), err))
			return err
		}

		ui.PrintSuccess(os.Stdout, t.GetMessage("issue.link_success", 0, map[string]interface{}{
			"PR":    prNumber,
			"Issue": issueNumber,
		}))

		return nil
	}
}
