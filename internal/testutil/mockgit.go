// Package testutil provides shared testify mocks reused across the test
// suites of multiple packages. It is only ever imported from _test.go files,
// so it never becomes part of the production binary.
package testutil

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/models"
)

// MockGitService is a shared mock covering every method exposed by
// internal/git.GitService across the various narrower interfaces that
// consume it.
type MockGitService struct {
	mock.Mock
}

func (m *MockGitService) HasStagedChanges(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockGitService) AddFileToStaging(ctx context.Context, file string) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockGitService) GetChangedFiles(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitService) GetChangedFilesWithStatus(ctx context.Context) ([]models.GitChange, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *MockGitService) GetDiff(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitService) GetDiffForFiles(ctx context.Context, files []string) (string, error) {
	args := m.Called(ctx, files)
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

func (m *MockGitService) GetRecentCommitMessages(ctx context.Context, count int) ([]string, error) {
	args := m.Called(ctx, count)
	return args.Get(0).([]string), args.Error(1)
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

func (m *MockGitService) FetchTags(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGitService) ValidateGitConfig(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGitService) ValidateTagExists(ctx context.Context, tag string) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}

func (m *MockGitService) GetRepoRoot(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}
