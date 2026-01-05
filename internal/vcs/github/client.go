package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v80/github"
	"github.com/thomas-vilte/matecommit/internal/builder"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"github.com/thomas-vilte/matecommit/internal/vcs"
	"golang.org/x/oauth2"
)

var _ vcs.VCSClient = (*GitHubClient)(nil)

type PullRequestsService interface {
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

var allowedLabels = map[string]struct {
	Color string
	Key   string
}{
	"feature":  {"00FF00", "label.feature"},
	"fix":      {"FF0000", "label.fix"},
	"refactor": {"FFA500", "label.refactor"},
	"docs":     {"0075CA", "label.docs"},
	"infra":    {"808080", "label.infra"},
	"test":     {"8A2BE2", "label.test"},
}

var labelDescriptions = map[string]string{
	"feature":  "New feature",
	"fix":      "Bug fix",
	"refactor": "Code refactor",
	"docs":     "Documentation",
	"infra":    "Infrastructure",
	"test":     "Test",
}

func NewGitHubClient(owner, repo, token string) *GitHubClient {
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

func (ghc *GitHubClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	pr := &github.PullRequest{
		Title: github.Ptr(summary.Title),
		Body:  github.Ptr(summary.Body),
	}

	_, resp, err := ghc.prService.Edit(ctx, ghc.owner, ghc.repo, prNumber, pr)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusTooManyRequests {
				return domainErrors.ErrGitHubRateLimit.
					WithContext("retry_after", resp.Header.Get("Retry-After")).
					WithContext("operation", "update PR")
			}
			if resp.StatusCode == http.StatusForbidden {
				return domainErrors.ErrGitHubInsufficientPerms.
					WithContext("operation", "update PR").
					WithContext("pr_number", prNumber).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			if resp.StatusCode == http.StatusForbidden {
				return domainErrors.ErrGitHubInsufficientPerms.
					WithContext("operation", "update PR").
					WithContext("pr_number", prNumber).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			if resp.StatusCode == http.StatusNotFound {
				return domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "update PR").
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
		}
		return fmt.Errorf("failed to update PR #%d: %w", prNumber, err)
	}

	if len(summary.Labels) > 0 {
		if err := ghc.AddLabelsToPR(ctx, prNumber, summary.Labels); err != nil {
			return fmt.Errorf("failed to add labels to PR #%d: %w", prNumber, err)
		}
	}

	return nil
}

func (ghc *GitHubClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	log := logger.FromContext(ctx)

	log.Debug("fetching github pull request",
		"owner", ghc.owner,
		"repo", ghc.repo,
		"pr_number", prNumber)

	pr, resp, err := ghc.prService.Get(ctx, ghc.owner, ghc.repo, prNumber)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return models.PRData{}, domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "get PR").
					WithContext("pr_number", prNumber)
			}
			if resp.StatusCode == http.StatusNotFound {
				return models.PRData{}, domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "get PR").
					WithContext("pr_number", prNumber).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
		}
		log.Error("failed to fetch github PR",
			"error", err,
			"owner", ghc.owner,
			"repo", ghc.repo,
			"pr_number", prNumber)
		return models.PRData{}, fmt.Errorf("failed to get PR #%d: %w", prNumber, err)
	}

	commits, _, err := ghc.prService.ListCommits(ctx, ghc.owner, ghc.repo, prNumber, &github.ListOptions{})
	if err != nil {
		return models.PRData{}, fmt.Errorf("failed to get commits for PR #%d: %w", prNumber, err)
	}

	prCommits := make([]models.Commit, len(commits))
	for i, commit := range commits {
		prCommits[i] = models.Commit{
			Message: commit.GetCommit().GetMessage(),
		}
	}

	prLabels := make([]string, len(pr.Labels))
	for i, label := range pr.Labels {
		prLabels[i] = label.GetName()
	}

	diff, resp, err := ghc.prService.GetRaw(ctx, ghc.owner, ghc.repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		// If 406 error (diff too large), use fallback commit by commit
		if resp != nil && resp.StatusCode == http.StatusNotAcceptable {
			log.Warn("PR diff too large, fetching diffs commit by commit",
				"pr_number", prNumber,
				"commits_count", len(commits))
			diff, err = ghc.getDiffFromCommits(ctx, commits)
			if err != nil {
				return models.PRData{}, fmt.Errorf("failed to get diff from commits for PR #%d: %w", prNumber, err)
			}
		} else {
			return models.PRData{}, fmt.Errorf("failed to get diff for PR #%d: %w", prNumber, err)
		}
	}

	prData := models.PRData{
		ID:          prNumber,
		Title:       pr.GetTitle(),
		Creator:     pr.GetUser().GetLogin(),
		Commits:     prCommits,
		Diff:        diff,
		BranchName:  pr.GetHead().GetRef(),
		Description: pr.GetBody(),
		Labels:      prLabels,
	}

	log.Debug("github PR fetched successfully",
		"pr_number", prNumber,
		"title", prData.Title,
		"commits_count", len(prCommits),
		"diff_size", len(diff))

	return prData, nil

}

func (ghc *GitHubClient) AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error {
	validLabels := ghc.validateAndFilterLabels(labels)
	if len(validLabels) == 0 {
		return nil
	}

	existingLabels, err := ghc.GetRepoLabels(ctx)
	if err != nil {
		return fmt.Errorf("failed to get repository labels: %w", err)
	}

	if err := ghc.ensureLabelsExist(ctx, existingLabels, validLabels); err != nil {
		return err
	}

	return ghc.addLabelsToIssue(ctx, prNumber, validLabels)
}

func (ghc *GitHubClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	labels, _, err := ghc.issuesService.ListLabels(ctx, ghc.owner, ghc.repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list repository labels: %w", err)
	}

	labelNames := make([]string, len(labels))
	for i, label := range labels {
		labelNames[i] = label.GetName()
	}
	return labelNames, nil
}

func (ghc *GitHubClient) CreateLabel(ctx context.Context, name, color, description string) error {
	_, _, err := ghc.issuesService.CreateLabel(ctx, ghc.owner, ghc.repo, &github.Label{
		Name:        github.Ptr(name),
		Color:       github.Ptr(color),
		Description: github.Ptr(description),
	})
	return err
}

func (ghc *GitHubClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool, progressCh chan<- models.BuildProgress) error {
	body := notes.Changelog
	if body == "" {
		body = fmt.Sprintf("%s\n\n", notes.Summary)
		if len(notes.Highlights) > 0 {
			body += "## Highlights\n\n"
			for _, h := range notes.Highlights {
				body += fmt.Sprintf("- %s\n", h)
			}
		}
	}

	releaseRequest := &github.RepositoryRelease{
		TagName:    github.Ptr(release.Version),
		Name:       github.Ptr(notes.Title),
		Body:       github.Ptr(body),
		Draft:      github.Ptr(draft),
		Prerelease: github.Ptr(false),
		MakeLatest: github.Ptr("true"),
	}

	createdRelease, resp, err := ghc.releaseService.CreateRelease(ctx, ghc.owner, ghc.repo, releaseRequest)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "create release").
					WithContext("version", release.Version)
			}
			if resp.StatusCode == http.StatusUnprocessableEntity {
				return domainErrors.ErrCreateRelease.
					WithContext("version", release.Version).
					WithContext("reason", "release already exists")
			}
			if resp.StatusCode == http.StatusNotFound {
				return domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "create release").
					WithContext("version", release.Version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			if resp.StatusCode == http.StatusForbidden {
				return domainErrors.ErrGitHubInsufficientPerms.
					WithContext("operation", "create release").
					WithContext("version", release.Version)
			}
		}
		return domainErrors.ErrCreateRelease.WithError(err).WithContext("version", release.Version)
	}

	if buildBinaries {
		if err := ghc.uploadBinaries(ctx, createdRelease.GetID(), release.Version, progressCh); err != nil {
			return fmt.Errorf("failed to upload binaries: %w", err)
		}
	}

	return nil
}

func (ghc *GitHubClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	release, resp, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, version)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "get release").
					WithContext("version", version)
			}
			if resp.StatusCode == http.StatusNotFound {
				return nil, domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "get release").
					WithContext("version", version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			return nil, domainErrors.ErrGetRelease.
				WithContext("version", version).
				WithContext("status_code", resp.StatusCode)
		}
		return nil, domainErrors.ErrGetRelease.WithError(err).WithContext("version", version)
	}

	return &models.VCSRelease{
		TagName: release.GetTagName(),
		Name:    release.GetName(),
		Body:    release.GetBody(),
		Draft:   release.GetDraft(),
		URL:     release.GetHTMLURL(),
	}, nil
}

func (ghc *GitHubClient) UpdateRelease(ctx context.Context, version, body string) error {
	release, resp, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, version)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "update release").
					WithContext("version", version)
			}
			if resp.StatusCode == http.StatusNotFound {
				return domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "update release").
					WithContext("version", version).
					WithContext("version", version).
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
			return domainErrors.ErrUpdateRelease.
				WithContext("version", version).
				WithContext("status_code", resp.StatusCode)
		}
		return domainErrors.ErrUpdateRelease.WithError(err).WithContext("version", version)
	}

	releaseUpdate := &github.RepositoryRelease{
		Body: github.Ptr(body),
	}

	_, _, err = ghc.releaseService.EditRelease(ctx, ghc.owner, ghc.repo, release.GetID(), releaseUpdate)
	if err != nil {
		return domainErrors.ErrUpdateRelease.WithError(err).WithContext("version", version)
	}
	return nil
}

func (ghc *GitHubClient) GetClosedIssuesBetweenTags(ctx context.Context, previousTag, _ string) ([]models.Issue, error) {
	prevRelease, _, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, previousTag)
	if err != nil {
		return nil, err
	}

	opts := &github.IssueListByRepoOptions{
		State:     "closed",
		Since:     prevRelease.GetCreatedAt().Time,
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []models.Issue
	for {
		issues, resp, err := ghc.issuesService.ListByRepo(ctx, ghc.owner, ghc.repo, opts)
		if err != nil {
			return nil, err
		}

		for _, issue := range issues {
			if issue.PullRequestLinks == nil {
				labels := make([]string, 0, len(issue.Labels))
				for _, label := range issue.Labels {
					labels = append(labels, label.GetName())
				}

				allIssues = append(allIssues, models.Issue{
					Number: issue.GetNumber(),
					Title:  issue.GetTitle(),
					Labels: labels,
					Author: issue.GetUser().GetLogin(),
					URL:    issue.GetHTMLURL(),
				})
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = resp.NextPage
	}

	return allIssues, nil
}

func (ghc *GitHubClient) GetMergedPRsBetweenTags(ctx context.Context, previousTag, _ string) ([]models.PullRequest, error) {
	prevRelease, _, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, previousTag)
	if err != nil {
		return nil, err
	}

	opts := &github.PullRequestListOptions{
		State:     "closed",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var allPRs []models.PullRequest
	for {
		prs, resp, err := ghc.prService.List(ctx, ghc.owner, ghc.repo, opts)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if pr.GetMerged() && pr.GetMergedAt().After(prevRelease.GetCreatedAt().Time) {
				labels := make([]string, 0, len(pr.Labels))
				for _, label := range pr.Labels {
					labels = append(labels, label.GetName())
				}

				allPRs = append(allPRs, models.PullRequest{
					Number:      pr.GetNumber(),
					Title:       pr.GetTitle(),
					Description: pr.GetBody(),
					Author:      pr.GetUser().GetLogin(),
					Labels:      labels,
					URL:         pr.GetHTMLURL(),
				})
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allPRs, nil
}

func (ghc *GitHubClient) GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error) {
	comparison, _, err := ghc.repoService.CompareCommits(ctx, ghc.owner, ghc.repo, previousTag, currentTag, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	contributorsMap := make(map[string]struct{})
	for _, commit := range comparison.Commits {
		if author := commit.GetAuthor(); author != nil {
			contributorsMap[author.GetLogin()] = struct{}{}
		}
	}

	contributors := make([]string, 0, len(contributorsMap))
	for contributor := range contributorsMap {
		contributors = append(contributors, contributor)
	}
	return contributors, nil
}

func (ghc *GitHubClient) GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error) {
	comparison, _, err := ghc.repoService.CompareCommits(ctx, ghc.owner, ghc.repo, previousTag, currentTag, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return nil, err
	}

	stats := &models.FileStatistics{
		FilesChanged: len(comparison.Files),
		Insertions:   0,
		Deletions:    0,
		TopFiles:     make([]models.FileChange, 0),
	}

	fileChanges := make([]models.FileChange, 0, len(comparison.Files))
	for _, file := range comparison.Files {
		stats.Insertions += file.GetAdditions()
		stats.Deletions += file.GetDeletions()

		fileChanges = append(fileChanges, models.FileChange{
			Path:      file.GetFilename(),
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
		})
	}

	sort.Slice(fileChanges, func(i, j int) bool {
		totalI := fileChanges[i].Additions + fileChanges[i].Deletions
		totalJ := fileChanges[j].Additions + fileChanges[j].Deletions
		return totalI > totalJ
	})

	if len(fileChanges) > 5 {
		stats.TopFiles = fileChanges[:5]
	} else {
		stats.TopFiles = fileChanges
	}
	return stats, nil
}

func (ghc *GitHubClient) GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error) {
	log := logger.FromContext(ctx)

	log.Debug("fetching github issue",
		"owner", ghc.owner,
		"repo", ghc.repo,
		"issue_number", issueNumber)

	issue, _, err := ghc.issuesService.Get(ctx, ghc.owner, ghc.repo, issueNumber)
	if err != nil {
		log.Error("failed to fetch github issue",
			"error", err,
			"owner", ghc.owner,
			"repo", ghc.repo,
			"issue_number", issueNumber)
		return nil, fmt.Errorf("error getting issue #%d: %w", issueNumber, err)
	}

	labels := make([]string, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		if label.Name != nil {
			labels = append(labels, label.GetName())
		}
	}

	var author string
	if issue.User != nil && issue.User.Login != nil {
		author = *issue.User.Login
	}

	var description string
	if issue.Body != nil {
		description = *issue.Body
	}

	var state string
	if issue.State != nil {
		state = *issue.State
	}

	var url string
	if issue.HTMLURL != nil {
		url = *issue.HTMLURL
	}

	criteria := extractAcceptanceCriteria(description)

	log.Debug("github issue fetched successfully",
		"issue_number", issueNumber,
		"title", issue.GetTitle(),
		"state", state,
		"labels_count", len(labels),
		"criteria_count", len(criteria))

	return &models.Issue{
		ID:          int(issue.GetID()),
		Number:      issue.GetNumber(),
		Title:       issue.GetTitle(),
		Description: description,
		State:       state,
		Labels:      labels,
		Author:      author,
		URL:         url,
		Criteria:    criteria,
	}, nil
}

func (ghc *GitHubClient) CreateIssue(ctx context.Context, title string, body string, labels []string, assignees []string) (*models.Issue, error) {
	log := logger.FromContext(ctx)

	log.Info("creating github issue",
		"owner", ghc.owner,
		"repo", ghc.repo,
		"title", title,
		"labels_count", len(labels),
		"assignees_count", len(assignees))

	if labels == nil {
		labels = []string{}
	}
	if assignees == nil {
		assignees = []string{}
	}

	issueRequest := &github.IssueRequest{
		Title:     github.Ptr(title),
		Body:      github.Ptr(body),
		Labels:    &labels,
		Assignees: &assignees,
	}

	ghIssue, resp, err := ghc.issuesService.Create(ctx, ghc.owner, ghc.repo, issueRequest)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnauthorized {
				return nil, domainErrors.ErrGitHubTokenInvalid.
					WithContext("operation", "create issue")
			}
			if resp.StatusCode == http.StatusNotFound {
				return nil, domainErrors.ErrRepositoryNotFound.
					WithContext("operation", "create issue").
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
		}
		log.Error("failed to create github issue",
			"error", err,
			"owner", ghc.owner,
			"repo", ghc.repo)
		return nil, fmt.Errorf("error creating issue: %w", err)
	}

	issue := &models.Issue{
		ID:          int(*ghIssue.ID),
		Number:      *ghIssue.Number,
		Title:       *ghIssue.Title,
		Description: getStringValue(ghIssue.Body),
		State:       *ghIssue.State,
		Author:      *ghIssue.User.Login,
		URL:         *ghIssue.HTMLURL,
		Labels:      make([]string, 0),
	}

	for _, label := range ghIssue.Labels {
		if label.Name != nil {
			issue.Labels = append(issue.Labels, label.GetName())
		}
	}

	log.Info("github issue created successfully",
		"issue_number", issue.Number,
		"issue_url", issue.URL)

	return issue, nil
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

func extractAcceptanceCriteria(body string) []string {
	var criteria []string
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		matches := regex.MarkdownCheckbox.FindStringSubmatch(line)
		if len(matches) > 2 {
			criterion := strings.TrimSpace(matches[2])
			if criterion != "" {
				criteria = append(criteria, criterion)
			}
		}
	}

	return criteria
}

func (ghc *GitHubClient) GetFileAtTag(ctx context.Context, tag, filepath string) (string, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: tag,
	}

	fileContent, _, _, err := ghc.repoService.GetContents(ctx, ghc.owner, ghc.repo, tag, opts)
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

func (ghc *GitHubClient) GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error) {
	issueNumbers := make(map[int]bool)
	for _, re := range []*regexp.Regexp{
		regex.BranchIssueSharp,
		regex.BranchIssueName,
		regex.BranchIssueStart,
		regex.BranchIssueFolder,
		regex.BranchIssueMid,
	} {
		if matches := re.FindStringSubmatch(branchName); len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				issueNumbers[num] = true
			}
		}
	}

	if prDescription != "" {
		matches := regex.GitHubClosedLink.FindAllStringSubmatch(prDescription, -1)
		for _, match := range matches {
			if len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					issueNumbers[num] = true
				}
			}
		}
		matchesSharp := regex.BranchIssueSharp.FindAllStringSubmatch(prDescription, -1)
		for _, match := range matchesSharp {
			if len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					issueNumbers[num] = true
				}
			}
		}
	}

	for _, commit := range commits {
		matches := regex.GitHubClosedLink.FindAllStringSubmatch(commit, -1)
		for _, match := range matches {
			if len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					issueNumbers[num] = true
				}
			}
		}
		matchesPR := regex.GitHubPR.FindAllStringSubmatch(commit, -1)
		for _, match := range matchesPR {
			if len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					issueNumbers[num] = true
				}
			}
		}
		matchesSharp := regex.BranchIssueSharp.FindAllStringSubmatch(commit, -1)
		for _, match := range matchesSharp {
			if len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					issueNumbers[num] = true
				}
			}
		}
	}

	var issues []models.Issue
	for issueNum := range issueNumbers {
		issue, err := ghc.GetIssue(ctx, issueNum)
		if err != nil {
			continue
		}
		issues = append(issues, *issue)
	}

	return issues, nil
}

func (ghc *GitHubClient) labelExists(existingLabels []string, target string) bool {
	for _, l := range existingLabels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}

func (ghc *GitHubClient) addLabelsToIssue(ctx context.Context, prNumber int, labels []string) error {
	_, _, err := ghc.issuesService.AddLabelsToIssue(ctx, ghc.owner, ghc.repo, prNumber, labels)
	if err != nil {
		return fmt.Errorf("failed to add labels to PR #%d: %w", prNumber, err)
	}
	return nil
}

func (ghc *GitHubClient) UpdateIssueChecklist(ctx context.Context, issueNumber int, checkedIndices []int) error {
	issue, _, err := ghc.issuesService.Get(ctx, ghc.owner, ghc.repo, issueNumber)
	if err != nil {
		return fmt.Errorf("error getting issue #%d: %w", issueNumber, err)
	}

	body := issue.GetBody()
	lines := strings.Split(body, "\n")
	var checklistLineIndices []int
	for i, line := range lines {
		if regex.MarkdownCheckboxUpdate.MatchString(line) {
			checklistLineIndices = append(checklistLineIndices, i)
		}
	}

	updated := false
	for _, idx := range checkedIndices {
		if idx >= 0 && idx < len(checklistLineIndices) {
			lineIdx := checklistLineIndices[idx]
			line := lines[lineIdx]

			if matches := regex.MarkdownCheckboxUpdate.FindStringSubmatch(line); len(matches) > 3 {
				if matches[2] == " " {
					lines[lineIdx] = matches[1] + "[x]" + matches[3]
					updated = true
				}
			}
		}
	}

	if !updated {
		return nil
	}

	newBody := strings.Join(lines, "\n")
	issueRequest := &github.IssueRequest{
		Body: github.Ptr(newBody),
	}

	_, _, err = ghc.issuesService.Edit(ctx, ghc.owner, ghc.repo, issueNumber, issueRequest)
	if err != nil {
		return fmt.Errorf("error updating issue body #%d: %w", issueNumber, err)
	}

	return nil
}

func (ghc *GitHubClient) ensureLabelsExist(ctx context.Context, existingLabels []string, requiredLabels []string) error {
	log := logger.FromContext(ctx)

	for _, label := range requiredLabels {
		if !ghc.labelExists(existingLabels, label) {
			meta := allowedLabels[label]

			description := labelDescriptions[label]
			if err := ghc.CreateLabel(ctx, label, meta.Color, description); err != nil {
				if !strings.Contains(err.Error(), "already_exists") && !strings.Contains(err.Error(), "422") {
					return fmt.Errorf("failed to create label '%s': %w", label, err)
				}
				log.Debug("label already exists, skipping creation",
					"label", label,
					"owner", ghc.owner,
					"repo", ghc.repo)
			}
		}
	}
	return nil
}

// getDiffFromCommits gets the combined diff of all commits when the total PR diff is too large
func (ghc *GitHubClient) getDiffFromCommits(ctx context.Context, commits []*github.RepositoryCommit) (string, error) {
	log := logger.FromContext(ctx)
	var combinedDiff strings.Builder

	log.Info("fetching diffs from commits",
		"commits_count", len(commits),
		"owner", ghc.owner,
		"repo", ghc.repo)

	for i, commit := range commits {
		sha := commit.GetSHA()
		log.Debug("processing commit",
			"current", i+1,
			"total", len(commits),
			"sha", sha[:8])
		fullCommit, _, err := ghc.repoService.GetCommit(ctx, ghc.owner, ghc.repo, sha, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get diff for commit %s: %w", sha[:8], err)
		}

		if fullCommit.GetStats().GetTotal() > 0 {
			combinedDiff.WriteString(fmt.Sprintf("\n# Commit: %s\n", sha[:8]))
			combinedDiff.WriteString(fmt.Sprintf("# Message: %s\n\n", strings.Split(commit.GetCommit().GetMessage(), "\n")[0]))

			for _, file := range fullCommit.Files {
				if file.Patch != nil {
					combinedDiff.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", file.GetFilename(), file.GetFilename()))
					combinedDiff.WriteString(*file.Patch)
					combinedDiff.WriteString("\n")
				}
			}
		}
	}

	return combinedDiff.String(), nil
}

func (ghc *GitHubClient) validateAndFilterLabels(labels []string) []string {
	var validLabels []string
	for _, label := range labels {
		cleaned := strings.ToLower(strings.TrimSpace(label))
		if cleaned != "" && ghc.isAllowedLabel(cleaned) {
			validLabels = append(validLabels, cleaned)
		}
	}
	return validLabels
}

func (ghc *GitHubClient) isAllowedLabel(label string) bool {
	_, exists := allowedLabels[label]
	return exists
}

func (ghc *GitHubClient) uploadBinaries(ctx context.Context, releaseID int64, version string, progressCh chan<- models.BuildProgress) error {
	tempDir, err := os.MkdirTemp("", "matecommit-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for build: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			return
		}
	}()

	commit, err := ghc.getCommitSHA(ctx)
	if err != nil {
		commit = "unknown"
	}
	date := time.Now().Format(time.RFC3339)

	log := logger.FromContext(ctx)

	log.Debug("creating binary builder",
		"repo", ghc.repo,
		"main_path", ghc.mainPath,
		"version", version)

	builderBinary := ghc.binaryBuilderFactory.NewBuilder(
		ghc.mainPath,
		ghc.repo,
		builder.WithVersion(version),
		builder.WithCommit(commit),
		builder.WithDate(date),
		builder.WithBuildDir(tempDir),
	)

	log.Info("compiling binaries for release",
		"version", version,
		"build_dir", tempDir)

	archives, err := builderBinary.BuildAndPackageAll(ctx, progressCh)
	if err != nil {
		return fmt.Errorf("failed to build binaries: %w", err)
	}

	log.Info("uploading binaries to release",
		"archives_count", len(archives),
		"release_id", releaseID,
		"version", version)

	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.UploadProgressStart,
			Total: len(archives),
		}
	}

	for i, archivePath := range archives {
		archiveName := filepath.Base(archivePath)

		if progressCh != nil {
			progressCh <- models.BuildProgress{
				Type:    models.UploadProgressAsset,
				Asset:   archiveName,
				Current: i + 1,
				Total:   len(archives),
			}
		}

		log.Info("uploading asset",
			"asset", archiveName,
			"progress", fmt.Sprintf("%d/%d", i+1, len(archives)))

		file, err := os.Open(archivePath)
		if err != nil {
			return fmt.Errorf("failed to open archive %s: %w", archivePath, err)
		}

		uploadOpts := &github.UploadOptions{
			Name:  archiveName,
			Label: archiveName,
		}

		_, _, err = ghc.releaseService.UploadReleaseAsset(ctx, ghc.owner, ghc.repo, releaseID, uploadOpts, file)
		_ = file.Close()
		if err != nil {
			return domainErrors.ErrUploadAsset.WithError(err).
				WithContext("asset_path", archivePath).
				WithContext("release_id", releaseID)
		}

		log.Info("asset uploaded successfully",
			"asset", archiveName,
			"progress", fmt.Sprintf("%d/%d", i+1, len(archives)))
	}

	if progressCh != nil {
		progressCh <- models.BuildProgress{
			Type:  models.UploadProgressComplete,
			Total: len(archives),
		}
	}

	return nil
}

func (ghc *GitHubClient) getCommitSHA(ctx context.Context) (string, error) {
	ref, _, err := ghc.repoService.GetCommit(ctx, ghc.owner, ghc.repo, "HEAD", nil)
	if err != nil {
		return "", err
	}
	return ref.GetSHA(), nil
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
