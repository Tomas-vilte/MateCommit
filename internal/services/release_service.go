package services

import (
	"context"
	"fmt"

	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/dependency"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

type releaseGitService interface {
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
	GetRepoInfo(ctx context.Context) (string, string, string, error)
	GetCurrentBranch(ctx context.Context) (string, error)
	FetchTags(ctx context.Context) error
	ValidateTagExists(ctx context.Context, tag string) error
	GetRepoRoot(ctx context.Context) (string, error)
}
type ReleaseService struct {
	git         releaseGitService
	vcsClient   vcs.VCSClient
	notesGen    ai.ReleaseNotesGenerator
	depAnalyzer *dependency.AnalyzerRegistry
	config      *config.Config

	versionFileCache struct {
		file    string
		pattern string
		lang    string
	}
}
type ReleaseOption func(*ReleaseService)

func WithReleaseVCSClient(vcs vcs.VCSClient) ReleaseOption {
	return func(s *ReleaseService) {
		s.vcsClient = vcs
	}
}

func WithReleaseNotesGenerator(rng ai.ReleaseNotesGenerator) ReleaseOption {
	return func(s *ReleaseService) {
		s.notesGen = rng
	}
}

func WithReleaseConfig(cfg *config.Config) ReleaseOption {
	return func(s *ReleaseService) {
		s.config = cfg
	}
}
func NewReleaseService(
	gitSvc releaseGitService,
	opts ...ReleaseOption,
) *ReleaseService {
	s := &ReleaseService{
		git:         gitSvc,
		depAnalyzer: dependency.NewAnalyzerRegistry(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
func (s *ReleaseService) AnalyzeNextRelease(ctx context.Context) (*models.Release, error) {
	log := logger.FromContext(ctx)

	if s.config != nil && s.config.AutoFetchTags {
		if err := s.git.FetchTags(ctx); err != nil {
			log.Warn("failed to fetch tags, continuing with local tags", "error", err)
		}
	}

	lastTag, err := s.git.GetLastTag(ctx)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting last tag", err)
	}

	var queryTag string
	var previousVersion string

	if lastTag == "" {
		count, _ := s.git.GetCommitCount(ctx)
		if count == 0 {
			return nil, domainErrors.NewAppError(domainErrors.TypeGit, "no commits found in repository", nil)
		}

		queryTag = ""
		previousVersion = "v0.0.0"
		log.Info("no previous tag found, using v0.0.0 as baseline")

	} else {
		if !regex.SemVer.MatchString(lastTag) {
			log.Warn("last tag does not match semver format", "tag", lastTag)
			return nil, fmt.Errorf("%w: tag '%s'", domainErrors.ErrInvalidTagFormat, lastTag)
		}
		queryTag = lastTag
		previousVersion = lastTag
	}

	commits, err := s.git.GetCommitsSinceTag(ctx, queryTag)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting commits", err)
	}

	if len(commits) == 0 {
		return nil, domainErrors.ErrNoCommitsSinceLastRelease
	}

	validCommits := s.filterValidCommits(commits)
	if len(validCommits) == 0 && len(commits) > 0 {
		log.Warn("no conventional commits found, but commits exist",
			"total_commits", len(commits))
	}

	release := &models.Release{
		PreviousVersion: previousVersion,
		AllCommits:      commits,
	}

	s.categorizeCommits(release)

	newVersion, bump := s.calculateVersion(previousVersion, release)
	release.Version = newVersion
	release.VersionBump = bump

	if err := s.validateVersionIncrement(previousVersion, newVersion); err != nil {
		log.Warn("version increment validation failed", "error", err)
	}

	return release, nil
}
func (s *ReleaseService) GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	log := logger.FromContext(ctx)

	log.Info("generating release notes",
		"version", release.Version,
		"previous_version", release.PreviousVersion,
	)

	log.Debug("categorizing commits",
		"total_commits", len(release.AllCommits),
	)

	log.Debug("commits categorized",
		"featues", len(release.Features),
		"fixes", len(release.BugFixes),
		"breaking", len(release.Breaking),
		"other", len(release.Other),
	)
	if s.notesGen == nil {
		return s.generateBasicNotes(release), nil
	}

	return s.notesGen.GenerateNotes(ctx, release)
}
func (s *ReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error {
	log := logger.FromContext(ctx)

	log.Info("publishing release",
		"version", release.Version,
		"draft", draft,
		"build_binaries", buildBinaries)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for release publishing")
		return domainErrors.ErrConfigMissing
	}

	if err := s.vcsClient.CreateRelease(ctx, release, notes, draft, buildBinaries, progressCh); err != nil {
		log.Error("failed to publish release",
			"error", err,
			"version", release.Version)
		return err
	}

	log.Info("release published successfully",
		"version", release.Version,
		"draft", draft)

	return nil
}
func (s *ReleaseService) CreateTag(ctx context.Context, version, message string) error {
	log := logger.FromContext(ctx)

	log.Info("creating git tag",
		"version", version)

	if err := s.git.CreateTag(ctx, version, message); err != nil {
		log.Error("failed to create git tag",
			"error", err,
			"version", version)
		return err
	}

	log.Info("git tag created successfully",
		"version", version)

	return nil
}
func (s *ReleaseService) TagExists(ctx context.Context, version string) bool {
	if s.git == nil {
		return false
	}

	return s.git.ValidateTagExists(ctx, version) == nil
}
func (s *ReleaseService) PushTag(ctx context.Context, version string) error {
	log := logger.FromContext(ctx)

	log.Info("pushing git tag",
		"version", version)

	if err := s.git.PushTag(ctx, version); err != nil {
		log.Error("failed to push git tag",
			"error", err,
			"version", version)
		return err
	}

	log.Info("git tag pushed successfully",
		"version", version)

	return nil
}
func (s *ReleaseService) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	if s.vcsClient == nil {
		return nil, domainErrors.ErrVCSNotSupported
	}
	return s.vcsClient.GetRelease(ctx, version)
}
func (s *ReleaseService) UpdateRelease(ctx context.Context, version, body string) error {
	log := logger.FromContext(ctx)

	log.Info("updating release",
		"version", version)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for updating release")
		return domainErrors.ErrConfigMissing
	}

	if err := s.vcsClient.UpdateRelease(ctx, version, body); err != nil {
		log.Error("failed to update release",
			"error", err,
			"version", version)
		return err
	}

	log.Info("release updated successfully",
		"version", version)

	return nil
}
func (s *ReleaseService) EnrichReleaseContext(ctx context.Context, release *models.Release) error {
	log := logger.FromContext(ctx)

	log.Info("enriching release context",
		"version", release.Version,
		"previous_version", release.PreviousVersion)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for enriching release context")
		return domainErrors.ErrConfigMissing
	}

	// release.Version is the prospective next version — it isn't a real tag
	// yet during preview (and possibly not even during create, before the
	// tag is pushed), so comparisons need the actual current branch tip
	// instead. It points at the same commit a freshly created tag would.
	headRef := release.Version
	if s.git != nil {
		if branch, err := s.git.GetCurrentBranch(ctx); err == nil {
			headRef = branch
		} else {
			log.Warn("failed to resolve current branch for release comparisons, falling back to version string",
				"error", err)
		}
	}

	if issues, err := s.vcsClient.GetClosedIssuesBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.ClosedIssues = issues
		log.Debug("closed issues fetched",
			"count", len(issues))
	} else {
		log.Warn("failed to fetch closed issues", "error", err)
	}

	if prs, err := s.vcsClient.GetMergedPRsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.MergedPRs = prs
		log.Debug("merged PRs fetched",
			"count", len(prs))
	} else {
		log.Warn("failed to fetch merged PRs", "error", err)
	}

	if contributors, err := s.vcsClient.GetContributorsBetweenTags(ctx, release.PreviousVersion, headRef); err == nil {
		release.Contributors = contributors
		release.NewContributors = contributors
		log.Debug("contributors fetched",
			"count", len(contributors))
	} else {
		log.Warn("failed to fetch contributors", "error", err)
	}

	if stats, err := s.vcsClient.GetFileStatsBetweenTags(ctx, release.PreviousVersion, headRef); err == nil {
		release.FileStats = *stats
		log.Debug("file stats fetched",
			"files_changed", stats.FilesChanged,
			"insertions", stats.Insertions,
			"deletions", stats.Deletions)
	} else {
		log.Warn("failed to fetch file stats", "error", err)
	}

	if deps, err := s.analyzeDependencyChanges(ctx, release); err == nil {
		release.Dependencies = deps
		log.Debug("dependencies analyzed")
	} else {
		log.Warn("failed to analyze dependency changes", "error", err)
	}

	log.Info("release context enriched successfully")

	return nil
}
func (s *ReleaseService) ValidateMainBranch(ctx context.Context) error {
	log := logger.FromContext(ctx)

	branch, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeGit, "error getting current branch", err)
	}

	if branch != "main" && branch != "master" {
		return fmt.Errorf("%w: currently on '%s'", domainErrors.ErrInvalidBranch, branch)
	}

	log.Debug("branch validation passed",
		"branch", branch,
	)
	return nil
}
