package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/httpclient"
	"github.com/google/go-github/v80/github"
	"golang.org/x/oauth2"
)

var _ ports.VCSClient = (*GitHubClient)(nil)

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
}

type GitHubClient struct {
	prService      PullRequestsService
	issuesService  IssuesService
	repoService    RepositoriesService
	releaseService ReleasesService
	owner          string
	repo           string
	trans          *i18n.Translations
	token          string
	httpClient     httpclient.HTTPClient
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

func NewGitHubClient(owner, repo, token string, trans *i18n.Translations) *GitHubClient {
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(httpClient)
	return &GitHubClient{
		prService:      client.PullRequests,
		issuesService:  client.Issues,
		repoService:    client.Repositories,
		releaseService: client.Repositories,
		owner:          owner,
		repo:           repo,
		trans:          trans,
		token:          token,
		httpClient:     httpClient,
	}
}

func NewGitHubClientWithServices(
	prService PullRequestsService,
	issuesService IssuesService,
	repoService RepositoriesService,
	releaseService ReleasesService,
	owner string,
	repo string,
	trans *i18n.Translations,
) *GitHubClient {
	return &GitHubClient{
		prService:      prService,
		issuesService:  issuesService,
		repoService:    repoService,
		releaseService: releaseService,
		owner:          owner,
		repo:           repo,
		trans:          trans,
		token:          "",
		httpClient:     &http.Client{},
	}
}

func (ghc *GitHubClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	pr := &github.PullRequest{
		Title: github.Ptr(summary.Title),
		Body:  github.Ptr(summary.Body),
	}

	_, resp, err := ghc.prService.Edit(ctx, ghc.owner, ghc.repo, prNumber, pr)
	if err != nil {
		// Detectar error 403 de permisos insuficientes
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("%s\n\n%s",
				ghc.trans.GetMessage("error.insufficient_permissions", 0, map[string]interface{}{
					"pr_number": prNumber,
					"owner":     ghc.owner,
					"repo":      ghc.repo,
				}),
				ghc.trans.GetMessage("error.token_scopes_help", 0, nil))
		}
		return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.update_pr", 0, map[string]interface{}{
			"pr_number": prNumber,
		}), err)
	}

	if len(summary.Labels) > 0 {
		if err := ghc.AddLabelsToPR(ctx, prNumber, summary.Labels); err != nil {
			return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.add_labels", 0, map[string]interface{}{
				"pr_number": prNumber,
			}), err)
		}
	}

	return nil
}

func (ghc *GitHubClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	pr, _, err := ghc.prService.Get(ctx, ghc.owner, ghc.repo, prNumber)
	if err != nil {
		return models.PRData{}, fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_pr", 0, map[string]interface{}{"pr_number": prNumber}), err)
	}

	commits, _, err := ghc.prService.ListCommits(ctx, ghc.owner, ghc.repo, prNumber, &github.ListOptions{})
	if err != nil {
		return models.PRData{}, fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_commits", 0, map[string]interface{}{"pr_number": prNumber}), err)
	}

	prCommits := make([]models.Commit, len(commits))
	for i, commit := range commits {
		prCommits[i] = models.Commit{
			Message: commit.GetCommit().GetMessage(),
		}
	}

	diff, resp, err := ghc.prService.GetRaw(ctx, ghc.owner, ghc.repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		// Si es error 406 (diff demasiado grande), usar fallback commit por commit
		if resp != nil && resp.StatusCode == http.StatusNotAcceptable {
			fmt.Printf("%s\n", ghc.trans.GetMessage("warning.pr_too_large", 0, map[string]interface{}{
				"pr_number": prNumber,
			}))
			diff, err = ghc.getDiffFromCommits(ctx, commits)
			if err != nil {
				return models.PRData{}, fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_diff_from_commits", 0, map[string]interface{}{
					"pr_number": prNumber,
				}), err)
			}
		} else {
			return models.PRData{}, fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_diff", 0, map[string]interface{}{"pr_number": prNumber}), err)
		}
	}

	return models.PRData{
		ID:            prNumber,
		Creator:       pr.GetUser().GetLogin(),
		Commits:       prCommits,
		Diff:          diff,
		BranchName:    pr.GetHead().GetRef(),
		PRDescription: pr.GetBody(),
	}, nil
}

func (ghc *GitHubClient) AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error {
	validLabels := ghc.validateAndFilterLabels(labels)
	if len(validLabels) == 0 {
		return nil
	}

	existingLabels, err := ghc.GetRepoLabels(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_labels", 0, nil), err)
	}

	if err := ghc.ensureLabelsExist(ctx, existingLabels, validLabels); err != nil {
		return err
	}

	return ghc.addLabelsToIssue(ctx, prNumber, validLabels)
}

func (ghc *GitHubClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	labels, _, err := ghc.issuesService.ListLabels(ctx, ghc.owner, ghc.repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.get_repo_labels", 0, nil), err)
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

func (ghc *GitHubClient) CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error {
	body := notes.Changelog
	if body == "" {
		body = fmt.Sprintf("# %s\n\n%s\n\n", notes.Title, notes.Summary)
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

	_, resp, err := ghc.releaseService.CreateRelease(ctx, ghc.owner, ghc.repo, releaseRequest)
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusUnprocessableEntity {
				return fmt.Errorf("%s", ghc.trans.GetMessage("error.release_already_exists", 0, map[string]interface{}{"Version": release.Version}))
			}
			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("%s", ghc.trans.GetMessage("error.repo_or_tag_not_found", 0, map[string]interface{}{"Version": release.Version}))
			}
		}
		return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.create_release", 0, nil), err)
	}
	return nil
}

func (ghc *GitHubClient) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	release, resp, err := ghc.releaseService.GetReleaseByTag(ctx, ghc.owner, ghc.repo, version)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, fmt.Errorf("%s", ghc.trans.GetMessage("error.repo_or_tag_not_found", 0, map[string]interface{}{
				"Version": version,
			}))
		}
		return nil, err
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
		if resp != nil && resp.StatusCode == 404 {
			return fmt.Errorf("%s", ghc.trans.GetMessage("error.repo_or_tag_not_found", 0, map[string]interface{}{
				"Version": version,
			}))
		}
		return err
	}

	releaseUpdate := &github.RepositoryRelease{
		Body: github.Ptr(body),
	}

	_, _, err = ghc.releaseService.EditRelease(ctx, ghc.owner, ghc.repo, release.GetID(), releaseUpdate)
	if err != nil {
		return err
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
	issue, _, err := ghc.issuesService.Get(ctx, ghc.owner, ghc.repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo issue #%d: %w", issueNumber, err)
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

	// Extract acceptance criteria from body markdown
	criteria := extractAcceptanceCriteria(description)

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

func extractAcceptanceCriteria(body string) []string {
	var criteria []string
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") {
			criterion := strings.TrimSpace(trimmed[5:])
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
		return "", fmt.Errorf("archivo no encontrado: %s en %s", filepath, tag)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("error decodificando contenido del archivo: %w", err)
	}

	return content, nil
}

func (ghc *GitHubClient) GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error) {
	issueNumbers := make(map[int]bool)

	branchPatterns := []string{
		`#(\d+)`,
		`issue[/-](\d+)`,
		`^(\d+)-`,
		`/(\d+)-`,
		`-(\d+)-`,
	}

	for _, pattern := range branchPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(branchName); len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				issueNumbers[num] = true
			}
		}
	}

	if prDescription != "" {
		descPatterns := []string{
			`(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)`,
			`#(\d+)`,
		}

		for _, pattern := range descPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringSubmatch(prDescription, -1)
			for _, match := range matches {
				if len(match) > 1 {
					if num, err := strconv.Atoi(match[1]); err == nil {
						issueNumbers[num] = true
					}
				}
			}
		}
	}

	commitPatterns := []string{
		`(?i)(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)`,
		`\(#(\d+)\)`,
		`#(\d+)`,
	}

	for _, commit := range commits {
		for _, pattern := range commitPatterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllStringSubmatch(commit, -1)
			for _, match := range matches {
				if len(match) > 1 {
					if num, err := strconv.Atoi(match[1]); err == nil {
						issueNumbers[num] = true
					}
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
		return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.add_labels", 0, map[string]interface{}{"pr_number": prNumber}), err)
	}
	return nil
}

func (ghc *GitHubClient) ensureLabelsExist(ctx context.Context, existingLabels []string, requiredLabels []string) error {
	for _, label := range requiredLabels {
		if !ghc.labelExists(existingLabels, label) {
			meta := allowedLabels[label]

			description := ghc.trans.GetMessage(meta.Key, 0, map[string]interface{}{"label": label})
			if err := ghc.CreateLabel(ctx, label, meta.Color, description); err != nil {
				if !strings.Contains(err.Error(), "already_exists") && !strings.Contains(err.Error(), "422") {
					return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.create_label", 0, map[string]interface{}{"label": label}), err)
				}
				fmt.Printf("Label '%s' already exists, continuing...\n", label)
			}
		}
	}
	return nil
}

// getDiffFromCommits obtiene el diff combinado de todos los commits cuando el diff completo del PR es demasiado grande
func (ghc *GitHubClient) getDiffFromCommits(ctx context.Context, commits []*github.RepositoryCommit) (string, error) {
	var combinedDiff strings.Builder

	fmt.Printf("%s\n", ghc.trans.GetMessage("info.fetching_commit_diffs", 0, map[string]interface{}{
		"total": len(commits),
	}))

	for i, commit := range commits {
		sha := commit.GetSHA()
		fmt.Printf("%s (%d/%d)\n", ghc.trans.GetMessage("info.processing_commit", 0, map[string]interface{}{
			"current": i + 1,
			"total":   len(commits),
			"sha":     sha[:8],
		}), i+1, len(commits))
		fullCommit, _, err := ghc.repoService.GetCommit(ctx, ghc.owner, ghc.repo, sha, nil)
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", ghc.trans.GetMessage("error.get_commit_diff", 0, nil), sha[:8], err)
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
