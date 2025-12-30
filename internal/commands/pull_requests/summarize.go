package pull_requests

import (
	"context"
	"fmt"
	"time"

	"github.com/thomas-vilte/matecommit/internal/commands/completion_helper"
	cfg "github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/urfave/cli/v3"
)

// PRService is a minimal interface for testing purposes
type PRService interface {
	SummarizePR(ctx context.Context, prNumber int, hint string, progress func(models.ProgressEvent)) (models.PRSummary, error)
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
			&cli.StringFlag{
				Name:    "hint",
				Aliases: []string{"H"},
				Usage:   t.GetMessage("vcs_summary.hint_usage", 0, nil),
			},
		},
		ShellComplete: completion_helper.DefaultFlagComplete,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			log := logger.FromContext(ctx)
			start := time.Now()

			prNumber := cmd.Int("pr-number")
			hint := cmd.String("hint")

			log.Info("executing summarize-pr command",
				"pr_number", prNumber,
				"has_hint", hint != "")

			prService, err := c.prProvider(ctx)
			if err != nil {
				log.Error("failed to create PR service",
					"error", err,
					"duration_ms", time.Since(start).Milliseconds())
				return fmt.Errorf(t.GetMessage("error.pr_service_creation_error", 0, nil)+": %w", err)
			}

			if prNumber == 0 {
				log.Error("PR number is required",
					"duration_ms", time.Since(start).Milliseconds())
				return fmt.Errorf("%s", t.GetMessage("error.pr_number_required", 0, nil))
			}

			spinner := ui.NewSmartSpinner(t.GetMessage("ui.fetching_pr_info", 0, struct{ Number int }{prNumber}))
			spinner.Start()

			summary, err := prService.SummarizePR(ctx, prNumber, hint, func(event models.ProgressEvent) {
				msg := ""
				switch event.Type {
				case models.ProgressIssuesDetected:
					msg = t.GetMessage("vcs_summary.issues_detected", 0, event.Data)
				case models.ProgressIssuesClosing:
					msg = t.GetMessage("vcs_summary.issues_closing", 0, event.Data)
				case models.ProgressBreakingChanges:
					msg = t.GetMessage("vcs_summary.breaking_changes_detected", 0, event.Data)
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
				log.Error("failed to summarize PR",
					"error", err,
					"pr_number", prNumber,
					"duration_ms", time.Since(start).Milliseconds())
				spinner.Error(t.GetMessage("ui.error_generating_pr_summary", 0, nil))
				ui.HandleAppError(err)
				return fmt.Errorf(t.GetMessage("error.pr_summary_error", 0, nil)+": %w", err)
			}

			log.Info("PR summarized successfully",
				"pr_number", prNumber,
				"title", summary.Title,
				"labels_count", len(summary.Labels),
				"duration_ms", time.Since(start).Milliseconds())

			spinner.Success(t.GetMessage("ui.pr_updated_successfully", 0, struct {
				Number int
				Title  string
			}{prNumber, summary.Title}))

			if summary.Usage != nil {
				fmt.Println()
				ui.PrintTokenUsage(summary.Usage, t)
			}

			return nil
		},
	}
}
