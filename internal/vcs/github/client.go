package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v80/github"
	"github.com/thomas-vilte/matecommit/internal/builder"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/vcs"
	"golang.org/x/oauth2"
)

var _ vcs.VCSClient = (*GitHubClient)(nil)

type PullRequestsService interface {
	Create(ctx context.Context, owner, repo string, pull *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	Edit(ctx context.Context, owner, repo string, number int, pr *github.PullRequest) (*github.PullRequest, *github.Response, error)
	List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	GetRaw(ctx context.Context, owner, repo string, number int, opts github.RawOptions) (string, *github.Response, error)
}

type IssuesService interface {
	ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	ListByRepo(ctx context.Context, owner, repo string, opts *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	Get(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error)
	Edit(ctx context.Context, owner, repo string, number int, issue *github.IssueRequest) (*github.Issue, *github.Response, error)
	Create(ctx context.Context, owner, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) // ← NEW
}

type RepositoriesService interface {
	GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)
	CompareCommits(ctx context.Context, owner, repo, base, head string, opts *github.ListOptions) (*github.CommitsComparison, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}

type ReleasesService interface {
	CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error)
	GetReleaseByTag(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, *github.Response, error)
	EditRelease(ctx context.Context, owner, repo string, id int64, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error)
	UploadReleaseAsset(ctx context.Context, owner, repo string, id int64, opt *github.UploadOptions, file *os.File) (*github.ReleaseAsset, *github.Response, error)
}

type UsersService interface {
	Get(ctx context.Context, user string) (*github.User, *github.Response, error)
}

// binaryBuilder is a minimal interface for testing purposes
type binaryBuilder interface {
	BuildAndPackageAll(ctx context.Context, progressCh chan<- models.BuildProgress) ([]string, error)
}

// binaryBuilderFactory is a minimal interface for testing purposes
type binaryBuilderFactory interface {
	NewBuilder(mainPath, binaryName string, opts ...builder.Option) binaryBuilder
}

// defaultBinaryBuilderFactoryAdapter adapts builder.DefaultBinaryBuilderFactory to binaryBuilderFactory
type defaultBinaryBuilderFactoryAdapter struct {
	*builder.DefaultBinaryBuilderFactory
}

func (a *defaultBinaryBuilderFactoryAdapter) NewBuilder(mainPath, binaryName string, opts ...builder.Option) binaryBuilder {
	return a.DefaultBinaryBuilderFactory.NewBuilder(mainPath, binaryName, opts...)
}

type GitHubClient struct {
	prService            PullRequestsService
	issuesService        IssuesService
	repoService          RepositoriesService
	releaseService       ReleasesService
	usersService         UsersService
	owner                string
	repo                 string
	token                string
	httpClient           *http.Client
	mainPath             string
	binaryBuilderFactory binaryBuilderFactory
}

func NewGitHubClient(owner, repo, token string) *GitHubClient {
	token = strings.TrimSpace(token)
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(httpClient)
	return &GitHubClient{
		prService:            client.PullRequests,
		issuesService:        client.Issues,
		repoService:          client.Repositories,
		releaseService:       client.Repositories,
		usersService:         client.Users,
		owner:                owner,
		repo:                 repo,
		token:                token,
		httpClient:           httpClient,
		mainPath:             "./cmd/main.go",
		binaryBuilderFactory: &defaultBinaryBuilderFactoryAdapter{&builder.DefaultBinaryBuilderFactory{}},
	}
}

func NewGitHubClientWithServices(
	prService PullRequestsService,
	issuesService IssuesService,
	repoService RepositoriesService,
	releaseService ReleasesService,
	usersService UsersService,
	owner string,
	repo string,
) *GitHubClient {
	return &GitHubClient{
		prService:            prService,
		issuesService:        issuesService,
		repoService:          repoService,
		usersService:         usersService,
		releaseService:       releaseService,
		owner:                owner,
		repo:                 repo,
		token:                "",
		httpClient:           &http.Client{},
		mainPath:             "./cmd/main.go",
		binaryBuilderFactory: &defaultBinaryBuilderFactoryAdapter{&builder.DefaultBinaryBuilderFactory{}},
	}
}

func (ghc *GitHubClient) SetMainPath(path string) {
	if path != "" {
		ghc.mainPath = path
	}
}

func (ghc *GitHubClient) GetAuthenticatedUser(ctx context.Context) (string, error) {
	user, resp, err := ghc.usersService.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return "", domainErrors.ErrGitHubTokenInvalid.
				WithContext("operation", "get authenticated user")
		}
		return "", fmt.Errorf("error obtaining authenticated user: %w", err)
	}

	if user.Login == nil {
		return "", fmt.Errorf("authenticated user has no login")
	}

	return *user.Login, nil
}

func (ghc *GitHubClient) GetFileAtTag(ctx context.Context, tag, filepath string) (string, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: tag,
	}

	fileContent, _, _, err := ghc.repoService.GetContents(ctx, ghc.owner, ghc.repo, filepath, opts)
	if err != nil {
		return "", err
	}

	if fileContent == nil {
		return "", fmt.Errorf("file not found: %s in %s", filepath, tag)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("error decoding file content: %w", err)
	}

	return content, nil
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
