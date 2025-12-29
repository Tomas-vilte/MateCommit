package vcs

import (
	"context"

	"github.com/thomas-vilte/matecommit/internal/models"
)

// VCSClient defines common methods to interact with the APIs of version control systems.
type VCSClient interface {
	// UpdatePR updates a Pull Request (title, body, and labels) in the provider.
	UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error
	// GetPR gets the PR data (for example, to extract commits, diff, etc.).
	GetPR(ctx context.Context, prNumber int) (models.PRData, error)
	// GetRepoLabels gets all available labels in the repository
	GetRepoLabels(ctx context.Context) ([]string, error)
	// CreateLabel creates a new label in the repository
	CreateLabel(ctx context.Context, name string, color string, description string) error
	// AddLabelsToPR adds specific labels to a PR
	AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error
	// CreateRelease creates a new release in the repository
	// buildBinaries indicates whether binaries should be compiled and uploaded (optional, only some providers support it)
	// progressCh is an optional channel to receive build/upload progress events
	CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error
	// GetRelease gets the release from the repository
	GetRelease(ctx context.Context, version string) (*models.VCSRelease, error)
	// UpdateRelease updates a release from the repository
	UpdateRelease(ctx context.Context, version, body string) error
	// GetClosedIssuesBetweenTags gets closed issues between two tags
	GetClosedIssuesBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.Issue, error)
	// GetMergedPRsBetweenTags gets merged PRs between two tags
	GetMergedPRsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.PullRequest, error)
	// GetContributorsBetweenTags gets contributors between two tags
	GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error)
	// GetFileStatsBetweenTags gets file statistics between two tags
	GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error)
	// GetIssue gets information for an issue/ticket by its number
	GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error)
	// GetFileAtTag gets the content of a file at a specific tag
	GetFileAtTag(ctx context.Context, tag, filepath string) (string, error)
	// GetPRIssues gets issues related to a PR based on branch name, commits, and description
	GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error)
	// UpdateIssueChecklist updates the checklist of an issue marking items as completed
	UpdateIssueChecklist(ctx context.Context, issueNumber int, indices []int) error
	// CreateIssue creates a new issue in the repository
	CreateIssue(ctx context.Context, title string, body string, labels []string, assignees []string) (*models.Issue, error)
	// GetAuthenticatedUser gets the current authenticated user
	GetAuthenticatedUser(ctx context.Context) (string, error)
}

// DependencyAnalyzer defines the interface to analyze dependencies for different languages
type DependencyAnalyzer interface {
	// CanHandle detects if this analyzer can handle the project
	CanHandle(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) bool

	// AnalyzeChanges analyzes dependency changes between two versions
	AnalyzeChanges(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error)

	// Name returns the name of the dependency manager
	Name() string
}
