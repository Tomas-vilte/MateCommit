package github

import (
	"context"

	"github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/mock"
)

type MockPRService struct {
	mock.Mock
}

func (m *MockPRService) Edit(ctx context.Context, owner, repo string, number int, pr *github.PullRequest) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, pr)
	return args.Get(0).(*github.PullRequest), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockPRService) Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number)
	return args.Get(0).(*github.PullRequest), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockPRService) ListCommits(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, opts)
	return args.Get(0).([]*github.RepositoryCommit), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockPRService) GetRaw(ctx context.Context, owner, repo string, number int, opts github.RawOptions) (string, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, opts)
	if args.Get(1) == nil {
		return args.String(0), nil, args.Error(2)
	}
	return args.String(0), args.Get(1).(*github.Response), args.Error(2)
}

type MockIssuesService struct {
	mock.Mock
}

func (m *MockIssuesService) ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	return args.Get(0).([]*github.Label), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockIssuesService) CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error) {
	args := m.Called(ctx, owner, repo, label)
	return args.Get(0).(*github.Label), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockIssuesService) AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, labels)
	return args.Get(0).([]*github.Label), args.Get(1).(*github.Response), args.Error(2)
}

type MockRepoService struct {
	mock.Mock
}

func (m *MockRepoService) GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
	args := m.Called(ctx, owner, repo, sha, opts)
	return args.Get(0).(*github.RepositoryCommit), args.Get(1).(*github.Response), args.Error(2)
}

type MockReleaseService struct {
	mock.Mock
}

func (m *MockReleaseService) CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error) {
	args := m.Called(ctx, owner, repo, release)
	return args.Get(0).(*github.RepositoryRelease), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockReleaseService) GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error) {
	args := m.Called(ctx, owner, repo, tag)
	return args.Get(0).(*github.RepositoryRelease), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockReleaseService) EditRelease(ctx context.Context, owner, repo string, id int64, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error) {
	args := m.Called(ctx, owner, repo, id, release)
	return args.Get(0).(*github.RepositoryRelease), args.Get(1).(*github.Response), args.Error(2)
}
