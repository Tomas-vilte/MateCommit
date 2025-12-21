package release

import (
	"context"
	"fmt"

	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/Tomas-vilte/MateCommit/internal/providers"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
)

// releaseService is a minimal interface for testing purposes
type releaseService interface {
	AnalyzeNextRelease(ctx context.Context) (*models.Release, error)
	GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error)
	PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool) error
	CreateTag(ctx context.Context, version, message string) error
	PushTag(ctx context.Context, version string) error
	GetRelease(ctx context.Context, version string) (*models.VCSRelease, error)
	UpdateRelease(ctx context.Context, version, body string) error
	EnrichReleaseContext(ctx context.Context, release *models.Release) error
	UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error
	CommitChangelog(ctx context.Context, version string) error
	PushChanges(ctx context.Context) error
	UpdateAppVersion(version string) error
}

// gitService is a minimal interface for testing purposes
type gitService interface {
	GetChangedFiles(ctx context.Context) ([]string, error)
	GetDiff(ctx context.Context) (string, error)
	GetRecentCommitMessages(ctx context.Context, limit int) ([]string, error)
	GetRepoInfo(ctx context.Context) (string, string, string, error)
	GetCurrentBranch(ctx context.Context) (string, error)
	GetLastTag(ctx context.Context) (string, error)
	GetCommitCount(ctx context.Context) (int, error)
	GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error)
	CreateTag(ctx context.Context, version, message string) error
	PushTag(ctx context.Context, version string) error
	GetTagDate(ctx context.Context, version string) (string, error)
	AddFileToStaging(ctx context.Context, file string) error
	HasStagedChanges(ctx context.Context) bool
	CreateCommit(ctx context.Context, message string) error
	Push(ctx context.Context) error
}

type ReleaseCommandFactory struct {
	gitService gitService
	config     *cfg.Config
}

func NewReleaseCommandFactory(gitSvc *git.GitService, config *cfg.Config) *ReleaseCommandFactory {
	return &ReleaseCommandFactory{
		gitService: gitSvc,
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
			r.newEditCommand(t),
		},
	}
}

func (r *ReleaseCommandFactory) createReleaseService(ctx context.Context, t *i18n.Translations) (*services.ReleaseService, error) {
	var vcsClient ports.VCSClient
	if r.config.ActiveVCSProvider != "" {
		if vcsConfig, ok := r.config.VCSConfigs[r.config.ActiveVCSProvider]; ok {
			owner, repo, _, err := r.gitService.GetRepoInfo(ctx)
			if err == nil && vcsConfig.Token != "" {
				vcsClient = github.NewGitHubClient(owner, repo, vcsConfig.Token)
			}
		}
	}

	var notesGen ports.ReleaseNotesGenerator
	if r.config.AIConfig.ActiveAI != "" {
		owner, repo, _, err := r.gitService.GetRepoInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("error obteniendo informacion del repositorio: %w", err)
		}

		gen, err := providers.NewReleaseNotesGenerator(ctx, r.config, nil, owner, repo)
		if err != nil {
			return nil, err
		}
		notesGen = gen
	}

	return services.NewReleaseService(
		r.gitService,
		services.WithReleaseVCSClient(vcsClient),
		services.WithReleaseNotesGenerator(notesGen),
		services.WithReleaseConfig(r.config),
	), nil
}
