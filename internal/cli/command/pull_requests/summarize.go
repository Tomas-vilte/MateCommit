package pull_requests

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/cli/completion_helper"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/ui"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type SummarizeCommand struct {
	prService ports.PRService
}

func NewSummarizeCommand(prService ports.PRService) *SummarizeCommand {
	return &SummarizeCommand{
		prService: prService,
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
			if c.prService == nil {
				return fmt.Errorf("%s", t.GetMessage("ai_missing.ai_missing_for_pr", 0, nil))
			}
			prNumber := cmd.Int("pr-number")
			if prNumber == 0 {
				return fmt.Errorf("%s", t.GetMessage("error.pr_number_required", 0, nil))
			}

			spinner := ui.NewSmartSpinner(t.GetMessage("ui.fetching_pr_info", 0, map[string]interface{}{
				"Number": prNumber,
			}))
			spinner.Start()

			summary, err := c.prService.SummarizePR(ctx, prNumber, func(msg string) {
				spinner.Log(msg)
			})
			if err != nil {
				spinner.Error(t.GetMessage("ui.error_generating_pr_summary", 0, nil))
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
