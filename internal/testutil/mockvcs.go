package testutil

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
)

// MockVCSClient is a shared mock implementing internal/vcs.VCSClient, reused
// across packages that need to stub a version control provider in tests.
type MockVCSClient struct {
	mock.Mock
}

func (m *MockVCSClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	args := m.Called(ctx, prNumber, summary)
	return args.Error(0)
}

func (m *MockVCSClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).(models.PRData), args.Error(1)
}

func (m *MockVCSClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVCSClient) CreateLabel(ctx context.Context, name, color, description string) error {
	args := m.Called(ctx, name, color, description)
	return args.Error(0)
}

func (m *MockVCSClient) AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error {
	args := m.Called(ctx, prNumber, labels)
	return args.Error(0)
}

func (m *MockVCSClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error {
	args := m.Called(ctx, release, notes, draft, buildBinaries, progressCh)
	return args.Error(0)
}

func (m *MockVCSClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VCSRelease), args.Error(1)
}

func (m *MockVCSClient) UpdateRelease(ctx context.Context, version, body string) error {
	args := m.Called(ctx, version, body)
	return args.Error(0)
}

func (m *MockVCSClient) GetClosedIssuesBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.Issue, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]models.Issue), args.Error(1)
}

func (m *MockVCSClient) GetMergedPRsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.PullRequest, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]models.PullRequest), args.Error(1)
}

func (m *MockVCSClient) GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error) {
	args := m.Called(ctx, previousTag, currentTag)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVCSClient) GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error) {
	args := m.Called(ctx, previousTag, currentTag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FileStatistics), args.Error(1)
}

func (m *MockVCSClient) GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error) {
	args := m.Called(ctx, issueNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Issue), args.Error(1)
}

func (m *MockVCSClient) GetFileAtTag(ctx context.Context, tag, filepath string) (string, error) {
	args := m.Called(ctx, tag, filepath)
	return args.String(0), args.Error(1)
}

func (m *MockVCSClient) GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error) {
	args := m.Called(ctx, branchName, commits, prDescription)
	return args.Get(0).([]models.Issue), args.Error(1)
}

func (m *MockVCSClient) CreateIssue(ctx context.Context, title string, body string, labels []string, assignees []string) (*models.Issue, error) {
	args := m.Called(ctx, title, body, labels, assignees)
	return args.Get(0).(*models.Issue), args.Error(1)
}

func (m *MockVCSClient) GetAuthenticatedUser(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}
