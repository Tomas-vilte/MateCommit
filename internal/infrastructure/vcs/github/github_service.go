package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type PullRequestsService interface {
	Edit(ctx context.Context, owner, repo string, number int, pr *github.PullRequest) (*github.PullRequest, *github.Response, error)
	Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	GetRaw(ctx context.Context, owner, repo string, number int, opts github.RawOptions) (string, *github.Response, error)
}

type IssuesService interface {
	ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
}

type RepositoriesService interface {
	GetCommit(ctx context.Context, owner, repo, sha string) (*github.RepositoryCommit, *github.Response, error)
}

type GitHubClient struct {
	prService     PullRequestsService
	issuesService IssuesService
	repoService   RepositoriesService
	owner         string
	repo          string
	trans         *i18n.Translations
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
		prService:     client.PullRequests,
		issuesService: client.Issues,
		repoService:   client.Repositories,
		owner:         owner,
		repo:          repo,
		trans:         trans,
	}
}

func NewGitHubClientWithServices(
	prService PullRequestsService,
	issuesService IssuesService,
	repoService RepositoriesService,
	owner string,
	repo string,
	trans *i18n.Translations,
) *GitHubClient {
	return &GitHubClient{
		prService:     prService,
		issuesService: issuesService,
		repoService:   repoService,
		owner:         owner,
		repo:          repo,
		trans:         trans,
	}
}

func (ghc *GitHubClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	pr := &github.PullRequest{
		Title: github.String(summary.Title),
		Body:  github.String(summary.Body),
	}

	_, resp, err := ghc.prService.Edit(ctx, ghc.owner, ghc.repo, prNumber, pr)
	if err != nil {
		// Detectar error 403 de permisos insuficientes
		if resp != nil && resp.Response != nil && resp.Response.StatusCode == http.StatusForbidden {
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
		if resp != nil && resp.Response != nil && resp.Response.StatusCode == http.StatusNotAcceptable {
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
		ID:      prNumber,
		Creator: pr.GetUser().GetLogin(),
		Commits: prCommits,
		Diff:    diff,
	}, nil
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

		fullCommit, _, err := ghc.repoService.GetCommit(ctx, ghc.owner, ghc.repo, sha)
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

func (ghc *GitHubClient) addLabelsToIssue(ctx context.Context, prNumber int, labels []string) error {
	_, _, err := ghc.issuesService.AddLabelsToIssue(ctx, ghc.owner, ghc.repo, prNumber, labels)
	if err != nil {
		return fmt.Errorf("%s: %w", ghc.trans.GetMessage("error.add_labels", 0, map[string]interface{}{"pr_number": prNumber}), err)
	}
	return nil
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
		Name:        github.String(name),
		Color:       github.String(color),
		Description: github.String(description),
	})
	return err
}

func (ghc *GitHubClient) labelExists(existingLabels []string, target string) bool {
	for _, l := range existingLabels {
		if strings.EqualFold(l, target) {
			return true
		}
	}
	return false
}
