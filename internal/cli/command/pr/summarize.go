package pr

import (
	"context"
	"fmt"

	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
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

func (c *SummarizeCommand) CreateCommand(t *i18n.Translations, cfg *cfg.Config) *cli.Command {
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
			activeVCS := cfg.VCSConfigs[cfg.ActiveVCSProvider]

			if cfg.ActiveVCSProvider == "" || cfg.VCSConfigs == nil || activeVCS.Owner == "" {
				return fmt.Errorf("%s", t.GetMessage("error.no_repo_configured", 0, nil))
			}

			if activeVCS.Repo == "" {
				return fmt.Errorf("%s", t.GetMessage("error.invalid_repo_format", 0, nil))
			}

			prNumber := command.Int("pr-number")

			summary, err := c.prService.SummarizePR(ctx, int(prNumber))
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
