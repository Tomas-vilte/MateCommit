package release

import (
	"context"

	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	geminiAI "github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
)

type ReleaseCommandFactory struct {
	gitService ports.GitService
	config     *cfg.Config
}

func NewReleaseCommandFactory(gitService ports.GitService, config *cfg.Config) *ReleaseCommandFactory {
	return &ReleaseCommandFactory{
		gitService: gitService,
		config:     config,
	}
}

func (r *ReleaseCommandFactory) CreateCommand(t *i18n.Translations, _ *cfg.Config) *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"r"},
		Usage:   t.GetMessage("release.command_usage", 0, nil),
		Commands: []*cli.Command{
			r.newPreviewCommand(t),
			r.newGenerateCommand(t),
			r.newCreateCommand(t),
			r.newPushCommand(t),
			r.newPublishCommand(t),
		},
	}
}

func (r *ReleaseCommandFactory) createReleaseService(ctx context.Context, t *i18n.Translations) (ports.ReleaseService, error) {
	var vcsClient ports.VCSClient
	if r.config.ActiveVCSProvider != "" {
		if vcsConfig, ok := r.config.VCSConfigs[r.config.ActiveVCSProvider]; ok {
			owner, repo, _, err := r.gitService.GetRepoInfo(ctx)
			if err == nil && vcsConfig.Token != "" {
				vcsClient = github.NewGitHubClient(owner, repo, vcsConfig.Token, t)
			}
		}
	}

	var notesGen ports.ReleaseNotesGenerator
	if r.config.GeminiAPIKey != "" {
		generator, err := geminiAI.NewReleaseNotesGenerator(ctx, r.config, t)
		if err == nil {
			notesGen = generator
		}
	}

	return services.NewReleaseService(r.gitService, vcsClient, notesGen, t), nil
}
