package pull_requests

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// PRService is a minimal interface for testing purposes
type PRService interface {
	SummarizePR(ctx context.Context, prNumber int, progress func(models.ProgressEvent)) (models.PRSummary, error)
}

// PRServiceProvider is a function that returns a PRService on demand
type PRServiceProvider func(ctx context.Context) (PRService, error)

type SummarizeCommand struct {
	prProvider PRServiceProvider
}

func NewSummarizeCommand(prProvider PRServiceProvider) *SummarizeCommand {
	return &SummarizeCommand{
		prProvider: prProvider,
	}
}

func (c *SummarizeCommand) CreateCommand(t *i18n.Translations, _ *cfg.Config) *cli.Command {
	return &cli.Command{
		Name:    "summarize-pr",
		Aliases: []string{"spr"},
		Usage:   t.GetMessage("vcs_summary.pr_summary_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "pr-number",
				Aliases:  []string{"n"},
				Usage:    t.GetMessage("vcs_summary.pr_number_usage", 0, nil),
				Required: true,
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			prService, err := c.prProvider(ctx)
			if err != nil {
				return fmt.Errorf(t.GetMessage("error.pr_service_creation_error", 0, nil)+": %w", err)
			}
			prNumber := cmd.Int("pr-number")
			if prNumber == 0 {
				return fmt.Errorf("%s", t.GetMessage("error.pr_number_required", 0, nil))
			}

			spinner := ui.NewSmartSpinner(t.GetMessage("ui.fetching_pr_info", 0, map[string]interface{}{
				"Number": prNumber,
			}))
			spinner.Start()

			summary, err := prService.SummarizePR(ctx, prNumber, func(event models.ProgressEvent) {
				msg := ""
				switch event.Type {
				case models.ProgressIssuesDetected:
					issues := event.Data["Issues"].([]string)
					msg = t.GetMessage("vcs_summary.issues_detected", 0, map[string]interface{}{
						"PRNumber": event.Data["PRNumber"],
						"Issues":   fmt.Sprintf("%v", issues),
					})
				case models.ProgressIssuesClosing:
					msg = t.GetMessage("vcs_summary.issues_closing", 0, map[string]interface{}{
						"Count": event.Data["Count"],
					})
				case models.ProgressBreakingChanges:
					msg = t.GetMessage("vcs_summary.breaking_changes_detected", 0, map[string]interface{}{
						"Count": event.Data["Count"],
					})
				case models.ProgressTestPlan:
					msg = t.GetMessage("vcs_summary.test_plan_generated", 0, nil)
				default:
					msg = event.Message
				}
				if msg != "" {
					spinner.Log(msg)
				}
			})
			if err != nil {
				spinner.Error(t.GetMessage("ui.error_generating_pr_summary", 0, nil))
				ui.HandleAppError(err, t)
				return fmt.Errorf(t.GetMessage("error.pr_summary_error", 0, nil)+": %w", err)
			}

			spinner.Success(t.GetMessage("ui.pr_updated_successfully", 0, map[string]interface{}{
				"Number": prNumber,
				"Title":  summary.Title,
			}))

			if summary.Usage != nil {
				fmt.Println()
				ui.PrintTokenUsage(summary.Usage, t)
			}

			return nil
		},
	}
}
