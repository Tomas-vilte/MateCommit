package github

import (
	"context"
	"os"

	"github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/builder"
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

func (m *MockPRService) List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	return args.Get(0).([]*github.PullRequest), args.Get(1).(*github.Response), args.Error(2)
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

func (m *MockIssuesService) ListByRepo(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	return args.Get(0).([]*github.Issue), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockIssuesService) Get(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number)
	return args.Get(0).(*github.Issue), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockIssuesService) Edit(ctx context.Context, owner, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	args := m.Called(ctx, owner, repo, number, issue)
	return args.Get(0).(*github.Issue), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockIssuesService) Create(ctx context.Context, owner, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	args := m.Called(ctx, owner, repo, issue)
	return args.Get(0).(*github.Issue), args.Get(1).(*github.Response), args.Error(2)
}

type MockRepoService struct {
	mock.Mock
}

func (m *MockRepoService) GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
	args := m.Called(ctx, owner, repo, sha, opts)
	return args.Get(0).(*github.RepositoryCommit), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockRepoService) CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error) {
	args := m.Called(ctx, owner, repo, base, head, opts)
	return args.Get(0).(*github.CommitsComparison), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockRepoService) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	args := m.Called(ctx, owner, repo, path, opts)
	return args.Get(0).(*github.RepositoryContent), args.Get(1).([]*github.RepositoryContent), args.Get(2).(*github.Response), args.Error(3)
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

func (m *MockReleaseService) UploadReleaseAsset(ctx context.Context, owner, repo string, id int64, opt *github.UploadOptions, file *os.File) (*github.ReleaseAsset, *github.Response, error) {
	args := m.Called(ctx, owner, repo, id, opt, file)
	return args.Get(0).(*github.ReleaseAsset), args.Get(1).(*github.Response), args.Error(2)
}

type MockBinaryPackager struct {
	mock.Mock
}

func (m *MockBinaryPackager) BuildAndPackageAll(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type MockBinaryBuilderFactory struct {
	mock.Mock
}

func (m *MockBinaryBuilderFactory) NewBuilder(mainPath, binaryName string, opts ...builder.Option) binaryBuilder {
	// Variadic arguments in mock.Called need careful handling.
	// We'll pass them as a single argument for simplicity in this mock.
	args := m.Called(mainPath, binaryName, opts)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(binaryBuilder)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) Get(ctx context.Context, user string) (*github.User, *github.Response, error) {
	args := m.Called(ctx, user)
	return args.Get(0).(*github.User), args.Get(1).(*github.Response), args.Error(2)
}
