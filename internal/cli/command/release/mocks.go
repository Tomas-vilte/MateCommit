package release

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/mock"
)

type MockReleaseService struct {
	mock.Mock
}

func (m *MockReleaseService) AnalyzeNextRelease(ctx context.Context) (*models.Release, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Release), args.Error(1)
}

func (m *MockReleaseService) GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	args := m.Called(ctx, release)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReleaseNotes), args.Error(1)
}

func (m *MockReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error {
	args := m.Called(ctx, release, notes, draft)
	return args.Error(0)
}

func (m *MockReleaseService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *MockReleaseService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.VCSRelease), args.Error(1)
}

func (m *MockReleaseService) UpdateRelease(ctx context.Context, version, body string) error {
	args := m.Called(ctx, version, body)
	return args.Error(0)
}

func (m *MockReleaseService) EnrichReleaseContext(ctx context.Context, release *models.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockReleaseService) UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error {
	args := m.Called(release, notes)
	return args.Error(0)
}

func (m *MockReleaseService) CommitChangelog(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockReleaseService) UpdateAppVersion(version string) error {
	args := m.Called(version)
	return args.Error(0)
}

type MockGitService struct {
	mock.Mock
}

func (m *MockGitService) GetChangedFiles(ctx context.Context) ([]models.GitChange, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *MockGitService) GetDiff(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) HasStagedChanges(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockGitService) CreateCommit(ctx context.Context, message string) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockGitService) AddFileToStaging(ctx context.Context, file string) error {
	args := m.Called(ctx, file)
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

func (m *MockGitService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *MockGitService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *MockGitService) GetRecentCommitMessages(ctx context.Context, count int) (string, error) {
	args := m.Called(ctx, count)
	return args.String(0), args.Error(1)
}
