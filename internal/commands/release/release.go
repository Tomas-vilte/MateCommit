package release

import (
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/ai/gemini"
	cfg "github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/git"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/services"
	"github.com/thomas-vilte/matecommit/internal/vcs"
	"github.com/thomas-vilte/matecommit/internal/vcs/github"
	"github.com/urfave/cli/v3"
)

// releaseService is a minimal interface for testing purposes
type releaseService interface {
	AnalyzeNextRelease(ctx context.Context) (*models.Release, error)
	GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error)
	PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error
	CreateTag(ctx context.Context, version, message string) error
	PushTag(ctx context.Context, version string) error
	GetRelease(ctx context.Context, version string) (*models.VCSRelease, error)
	UpdateRelease(ctx context.Context, version, body string) error
	EnrichReleaseContext(ctx context.Context, release *models.Release) error
	UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error
	CommitChangelog(ctx context.Context, version string) error
	PushChanges(ctx context.Context) error
	UpdateAppVersion(ctx context.Context, version string) error
	ValidateMainBranch(ctx context.Context) error
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
	FetchTags(ctx context.Context) error
	ValidateGitConfig(ctx context.Context) error
	ValidateTagExists(ctx context.Context, tag string) error
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

func (r *ReleaseCommandFactory) createReleaseService(ctx context.Context, _ *i18n.Translations) (*services.ReleaseService, error) {
	var vcsClient vcs.VCSClient
	if r.config.ActiveVCSProvider != "" {
		if vcsConfig, ok := r.config.VCSConfigs[r.config.ActiveVCSProvider]; ok {
			owner, repo, _, err := r.gitService.GetRepoInfo(ctx)
			if err == nil && vcsConfig.Token != "" {
				vcsClient = github.NewGitHubClient(owner, repo, vcsConfig.Token)
			}
		}
	}

	var notesGen ai.ReleaseNotesGenerator
	if r.config.AIConfig.ActiveAI == "gemini" {
		owner, repo, _, err := r.gitService.GetRepoInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("error retrieving information from repository: %w", err)
		}

		gen, err := gemini.NewReleaseNotesGenerator(ctx, r.config, nil, owner, repo)
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
