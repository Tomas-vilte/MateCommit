package github

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/http"
)

type GitHubClient struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubClient(owner, repo, token string) *GitHubClient {
	var httpClient *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	gc := github.NewClient(httpClient)

	return &GitHubClient{
		client: gc,
		owner:  owner,
		repo:   repo,
	}
}

// UpdatePR actualiza una Pull Request (título, body y etiquetas) en GitHub.
func (ghc *GitHubClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	pr, _, err := ghc.client.PullRequests.Get(ctx, ghc.owner, ghc.repo, prNumber)
	if err != nil {
		return fmt.Errorf("error al obtener el PR %d de Github: %w", prNumber, err)
	}

	if pr == nil {
		return fmt.Errorf("PR %d no encontrado en Github", prNumber)
	}

	prTitle := summary.Title
	prBody := summary.Body
	prInput := &github.PullRequest{
		Title: &prTitle,
		Body:  &prBody,
	}

	_, _, err = ghc.client.PullRequests.Edit(ctx, ghc.owner, ghc.repo, prNumber, prInput)
	if err != nil {
		return fmt.Errorf("error al actualizar el PR %d en GitHub: %w", prNumber, err)
	}

	return nil
}

// GetPR obtiene los datos de PR (por ejemplo, para extraer commits, diff, etc.) de GitHub.
func (ghc *GitHubClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	githubPR, _, err := ghc.client.PullRequests.Get(ctx, ghc.owner, ghc.repo, prNumber)
	if err != nil {
		return models.PRData{}, fmt.Errorf("error al obtener el PR %d de Github: %w", prNumber, err)
	}

	if githubPR == nil {
		return models.PRData{}, fmt.Errorf("PR %d no encontrado en Github", prNumber)
	}

	githubCommits, _, err := ghc.client.PullRequests.ListCommits(ctx, ghc.owner, ghc.repo, prNumber, nil)
	if err != nil {
		return models.PRData{}, fmt.Errorf("error al obtener los commits del PR %d de Github: %w", prNumber, err)
	}

	diff, _, err := ghc.client.PullRequests.GetRaw(ctx, ghc.owner, ghc.repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		return models.PRData{}, fmt.Errorf("error al obtener el diff del PR %d de Github: %w", prNumber, err)
	}

	prData := models.PRData{
		ID:      prNumber,
		Creator: githubPR.GetUser().GetLogin(),
		Diff:    diff,
	}

	for _, githubCommit := range githubCommits {
		if githubCommit.Commit != nil && githubCommit.Commit.Message != nil {
			prData.Commits = append(prData.Commits, models.Commit{
				Message: *githubCommit.Commit.Message,
			})
		}
	}

	return prData, nil
}
