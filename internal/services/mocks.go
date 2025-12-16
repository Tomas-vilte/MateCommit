package services

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/mock"
)

type (
	MockGitService struct {
		mock.Mock
	}
	MockAIProvider struct {
		mock.Mock
	}

	MockJiraService struct {
		mock.Mock
	}

	MockVCSClient struct {
		mock.Mock
	}

	MockPRSummarizer struct {
		mock.Mock
	}

	MockReleaseNotesGenerator struct {
		mock.Mock
	}
)

func (m *MockJiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	args := m.Called(ticketID)
	return args.Get(0).(*models.TicketInfo), args.Error(1)
}

func (m *MockGitService) HasStagedChanges(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockGitService) AddFileToStaging(ctx context.Context, file string) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockGitService) GetChangedFiles(ctx context.Context) ([]models.GitChange, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *MockGitService) GetDiff(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) StageAllChanges(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGitService) CreateCommit(ctx context.Context, message string) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockGitService) GetCurrentBranch(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	args := m.Called(ctx)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockGitService) GetLastTag(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) GetCommitCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockGitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	args := m.Called(ctx, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Commit), args.Error(1)
}

func (m *MockGitService) GetCommitsBetweenTags(ctx context.Context, fromTag, toTag string) ([]models.Commit, error) {
	args := m.Called(ctx, fromTag, toTag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Commit), args.Error(1)
}

func (m *MockGitService) GetTagDate(ctx context.Context, tag string) (string, error) {
	args := m.Called(ctx, tag)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) GetRecentCommitMessages(ctx context.Context, count int) (string, error) {
	args := m.Called(ctx, count)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *MockGitService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockGitService) Push(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAIProvider) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	args := m.Called(ctx, info, count)
	return args.Get(0).([]models.CommitSuggestion), args.Error(1)
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

func (m *MockVCSClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error {
	args := m.Called(ctx, release, notes, draft)
	return args.Error(0)
}

func (m *MockVCSClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	args := m.Called(ctx, version)
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
	return args.Get(0).(*models.FileStatistics), args.Error(1)
}

func (m *MockVCSClient) GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error) {
	args := m.Called(ctx, issueNumber)
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

func (m *MockVCSClient) UpdateIssueChecklist(ctx context.Context, issueNumber int, indices []int) error {
	args := m.Called(ctx, issueNumber, indices)
	return args.Error(0)
}

func (m *MockPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error) {
	args := m.Called(ctx, prompt)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

func (m *MockReleaseNotesGenerator) GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	args := m.Called(ctx, release)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReleaseNotes), args.Error(1)
}
