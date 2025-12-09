package pr

import (
	"context"
	"fmt"

	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/factory"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type SummarizeCommand struct {
	prFactory factory.PRServiceFactoryInterface
}

func NewSummarizeCommand(prFactory factory.PRServiceFactoryInterface) *SummarizeCommand {
	return &SummarizeCommand{
		prFactory: prFactory,
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
		Action: func(ctx context.Context, command *cli.Command) error {
			prService, err := c.prFactory.CreatePRService(ctx)
			if err != nil {
				return fmt.Errorf(t.GetMessage("error.pr_service_creation_error", 0, nil)+": %w", err)
			}
			prNumber := command.Int("pr-number")

			summary, err := prService.SummarizePR(ctx, int(prNumber))
			if err != nil {
				return fmt.Errorf(t.GetMessage("error.pr_summary_error", 0, nil)+": %w", err)
			}

			fmt.Println(t.GetMessage("vcs_summary.pr_summary_success", 0, map[string]interface{}{
				"PRNumber": prNumber,
				"Title":    summary.Title,
			}))
			return nil
		},
	}
}
