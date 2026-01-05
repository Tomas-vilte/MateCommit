package issues

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// FromPlanOptions configures issue generation from a plan file.
type FromPlanOptions struct {
	File     string
	DryRun   bool
	AssignMe bool
	Labels   []string
}

func (f *IssuesCommandFactory) newFromPlanCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "from-plan",
		Usage: t.GetMessage("issue_from_plan.usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    t.GetMessage("issue_from_plan.file_flag", 0, nil),
				Required: true,
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   t.GetMessage("issue_from_plan.dry_run_flag", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "assign-me",
				Aliases: []string{"a"},
				Usage:   t.GetMessage("issue_from_plan.assign_me_flag", 0, nil),
			},
			&cli.StringSliceFlag{
				Name:    "labels",
				Aliases: []string{"l"},
				Usage:   t.GetMessage("issue_from_plan.labels_flag", 0, nil),
			},
		},
		Action: f.createFromPlanAction(t, cfg),
	}
}

func (f *IssuesCommandFactory) createFromPlanAction(t *i18n.Translations, cfg *config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		opts := FromPlanOptions{
			File:     cmd.String("file"),
			DryRun:   cmd.Bool("dry-run"),
			AssignMe: cmd.Bool("assign-me"),
			Labels:   cmd.StringSlice("labels"),
		}

		content, err := os.ReadFile(opts.File)
		if err != nil {
			errMsg := t.GetMessage("issue_from_plan.error_reading_file", 0,
				struct{ Error string }{err.Error()})
			return fmt.Errorf("%s", errMsg)
		}

		if len(content) == 0 {
			errMsg := t.GetMessage("issue_from_plan.error_empty_file", 0, nil)
			return fmt.Errorf("%s", errMsg)
		}

		issueService, err := f.issueServiceProvider(ctx)
		if err != nil {
			return fmt.Errorf("failed to create issue service: %w", err)
		}

		ui.PrintInfo(t.GetMessage("issue_from_plan.parsing_plan", 0, nil))

		result, err := issueService.GenerateFromDescription(ctx, string(content), false, false, nil)
		if err != nil {
			errMsg := t.GetMessage("issue_from_plan.error_parsing_plan", 0,
				struct{ Error string }{err.Error()})
			return fmt.Errorf("%s", errMsg)
		}

		if len(opts.Labels) > 0 {
			result.Labels = mergeUniqueLabels(result.Labels, opts.Labels)
		}

		if opts.DryRun {
			return showPreview(result, t, cfg)
		}

		var assignees []string
		if opts.AssignMe {
			user, err := issueService.GetAuthenticatedUser(ctx)
			if err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to get authenticated user: %v", err))
			} else {
				assignees = []string{user}
			}
		}

		ui.PrintInfo(t.GetMessage("issue_from_plan.creating_issues", 0, nil))

		issue, err := issueService.CreateIssue(ctx, result, assignees)
		if err != nil {
			errMsg := t.GetMessage("issue_from_plan.error_creating_issue", 0,
				struct {
					Number int
					Error  string
				}{1, err.Error()})
			ui.PrintError(os.Stdout, errMsg)
			return err
		}

		emoji := ""
		if cfg.UseEmoji {
			emoji = "✅ "
		}
		fmt.Printf("  %s#%d: %s\n", emoji, issue.Number, result.Title)

		ui.PrintSuccess(os.Stdout, t.GetMessage("issue_from_plan.created_summary", 0,
			struct{ Count int }{1}))

		return nil
	}
}

// showPreview displays the generated issue without creating it.
func showPreview(result *models.IssueGenerationResult, t *i18n.Translations, cfg *config.Config) error {
	separator := strings.Repeat("\u2500", 60)
	fmt.Println()
	fmt.Println(separator)

	emoji := ""
	if cfg.UseEmoji {
		emoji = "\U0001F4CB "
	}

	ui.PrintInfo(fmt.Sprintf("%s%s", emoji, t.GetMessage("issue_from_plan.preview_header", 0, nil)))
	fmt.Println()

	ui.PrintKeyValue("Title", result.Title)
	fmt.Println()

	if result.Description != "" {
		ui.PrintInfo("Description:")
		fmt.Println(result.Description)
		fmt.Println()
	}

	if len(result.Labels) > 0 {
		ui.PrintInfo(fmt.Sprintf("Labels: %s", strings.Join(result.Labels, ", ")))
	}

	fmt.Println(separator)
	fmt.Println()

	return nil
}

// mergeUniqueLabels combines two label slices, removing duplicates.
func mergeUniqueLabels(existing, additional []string) []string {
	labelSet := make(map[string]bool)

	for _, label := range existing {
		if label != "" {
			labelSet[label] = true
		}
	}

	for _, label := range additional {
		if label != "" {
			labelSet[label] = true
		}
	}

	result := make([]string, 0, len(labelSet))
	for label := range labelSet {
		result = append(result, label)
	}

	return result
}
