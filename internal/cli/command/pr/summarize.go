package pr

import (
	"context"
	"fmt"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
	"strings"
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
	var defaultRepo string
	if cfg.ActiveVCSProvider != "" {
		if vcsConfig, exists := cfg.VCSConfigs[cfg.ActiveVCSProvider]; exists {
			defaultRepo = fmt.Sprintf("%s/%s", vcsConfig.Owner, vcsConfig.Repo)
		}
	}

	return &cli.Command{
		Name:    "summarize-pr",
		Aliases: []string{"spr"},
		Usage:   t.GetMessage("vcs_summary.pr_summary_usage", 0, nil),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				Aliases: []string{"r"},
				Usage:   t.GetMessage("vcs_summary.repo_flag_usage", 0, nil),
				Value:   defaultRepo,
			},
			&cli.IntFlag{
				Name:     "pr-number",
				Aliases:  []string{"n"},
				Usage:    t.GetMessage("vcs_summary.pr_number_usage", 0, nil),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "context",
				Aliases: []string{"c"},
				Usage:   t.GetMessage("vcs_summary.additional_context_usage", 0, nil),
				Value:   "",
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			prNumber := command.Int("pr-number")
			repo := command.String("repo")
			additionalContext := command.String("context")

			if repo == "" || prNumber == 0 {
				return fmt.Errorf("%s", t.GetMessage("error.no_repo_configured", 0, nil))
			}

			if _, _, err := parseRepo(repo); err != nil {
				return fmt.Errorf(t.GetMessage("error.invalid_repo_format", 0, nil)+": %w", err)
			}

			summary, err := c.prService.SummarizePR(ctx, int(prNumber), additionalContext)
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

func parseRepo(repo string) (owner string, repoName string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("formato invalido, debe ser owner/repo")
	}
	return parts[0], parts[1], nil
}
