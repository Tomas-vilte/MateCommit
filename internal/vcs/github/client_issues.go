package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v80/github"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
)

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

// ListOpenIssues fetches currently open issues, most recently updated
// first, capped to a single page so a duplicate check stays cheap and fast.
func (ghc *GitHubClient) ListOpenIssues(ctx context.Context) ([]models.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	issues, _, err := ghc.issuesService.ListByRepo(ctx, ghc.owner, ghc.repo, opts)
	if err != nil {
		return nil, err
	}

	result := make([]models.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.PullRequestLinks != nil {
			continue
		}
		labels := make([]string, 0, len(issue.Labels))
		for _, label := range issue.Labels {
			labels = append(labels, label.GetName())
		}
		result = append(result, models.Issue{
			Number: issue.GetNumber(),
			Title:  issue.GetTitle(),
			Labels: labels,
			Author: issue.GetUser().GetLogin(),
			URL:    issue.GetHTMLURL(),
		})
	}

	return result, nil
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
