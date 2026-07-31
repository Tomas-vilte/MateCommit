package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v80/github"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
)

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

// CreatePR opens a new Pull Request from headBranch into baseBranch.
func (ghc *GitHubClient) CreatePR(ctx context.Context, title, body, headBranch, baseBranch string) (*models.CreatedPR, error) {
	pr, resp, err := ghc.prService.Create(ctx, ghc.owner, ghc.repo, &github.NewPullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Head:  github.Ptr(headBranch),
		Base:  github.Ptr(baseBranch),
	})
	if err != nil {
		if resp != nil {
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, domainErrors.ErrGitHubRateLimit.
					WithContext("retry_after", resp.Header.Get("Retry-After")).
					WithContext("operation", "create PR")
			}
			if resp.StatusCode == http.StatusForbidden {
				return nil, domainErrors.ErrGitHubInsufficientPerms.
					WithContext("operation", "create PR").
					WithContext("repo", fmt.Sprintf("%s/%s", ghc.owner, ghc.repo))
			}
		}
		return nil, fmt.Errorf("failed to create PR from %s to %s: %w", headBranch, baseBranch, err)
	}

	return &models.CreatedPR{
		Number: pr.GetNumber(),
		URL:    pr.GetHTMLURL(),
	}, nil
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

// addIssueNumberMatches finds every match of re in text and adds the
// captured issue number (submatch group 1) to dest.
func addIssueNumberMatches(re *regexp.Regexp, text string, dest map[int]bool) {
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				dest[num] = true
			}
		}
	}
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
		addIssueNumberMatches(re, branchName, issueNumbers)
	}

	if prDescription != "" {
		addIssueNumberMatches(regex.GitHubClosedLink, prDescription, issueNumbers)
		addIssueNumberMatches(regex.BranchIssueSharp, prDescription, issueNumbers)
	}

	for _, commit := range commits {
		addIssueNumberMatches(regex.GitHubClosedLink, commit, issueNumbers)
		addIssueNumberMatches(regex.GitHubPR, commit, issueNumbers)
		addIssueNumberMatches(regex.BranchIssueSharp, commit, issueNumbers)
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
			fmt.Fprintf(&combinedDiff, "\n# Commit: %s\n", sha[:8])
			fmt.Fprintf(&combinedDiff, "# Message: %s\n\n", strings.Split(commit.GetCommit().GetMessage(), "\n")[0])

			for _, file := range fullCommit.Files {
				if file.Patch != nil {
					fmt.Fprintf(&combinedDiff, "diff --git a/%s b/%s\n", file.GetFilename(), file.GetFilename())
					combinedDiff.WriteString(*file.Patch)
					combinedDiff.WriteString("\n")
				}
			}
		}
	}

	return combinedDiff.String(), nil
}
