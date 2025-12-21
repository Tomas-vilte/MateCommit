package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	domainErrors "github.com/Tomas-vilte/MateCommit/internal/domain/errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
)

// commitGitService defines only the methods needed by CommitService.
type commitGitService interface {
	GetChangedFiles(ctx context.Context) ([]string, error)
	GetDiff(ctx context.Context) (string, error)
	GetRecentCommitMessages(ctx context.Context, limit int) ([]string, error)
	GetRepoInfo(ctx context.Context) (string, string, string, error)
	GetCurrentBranch(ctx context.Context) (string, error)
}

type CommitService struct {
	git           commitGitService
	ai            ports.CommitSummarizer
	ticketManager ports.TickerManager
	vcsClient     ports.VCSClient
	config        *config.Config
}

type Option func(*CommitService)

func WithTicketManager(tm ports.TickerManager) Option {
	return func(s *CommitService) {
		s.ticketManager = tm
	}
}

func WithVCSClient(vcs ports.VCSClient) Option {
	return func(s *CommitService) {
		s.vcsClient = vcs
	}
}

func WithConfig(cfg *config.Config) Option {
	return func(s *CommitService) {
		s.config = cfg
	}
}

func NewCommitService(gitSvc commitGitService, ai ports.CommitSummarizer, opts ...Option) *CommitService {
	s := &CommitService{
		git: gitSvc,
		ai:  ai,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *CommitService) GenerateSuggestions(ctx context.Context, count int, issueNumber int, progress func(models.ProgressEvent)) ([]models.CommitSuggestion, error) {
	commitInfo, err := s.buildCommitInfo(ctx, issueNumber, progress)
	if err != nil {
		return nil, err
	}
	return s.ai.GenerateSuggestions(ctx, commitInfo, count)
}

func (s *CommitService) buildCommitInfo(ctx context.Context, issueNumber int, progress func(models.ProgressEvent)) (models.CommitInfo, error) {
	var commitInfo models.CommitInfo

	if s.ai == nil {
		return commitInfo, domainErrors.ErrAPIKeyMissing
	}

	changes, err := s.git.GetChangedFiles(ctx)
	if err != nil {
		return commitInfo, err
	}

	if len(changes) == 0 {
		return commitInfo, domainErrors.ErrNoChanges
	}

	diff, err := s.git.GetDiff(ctx)
	if err != nil {
		return commitInfo, domainErrors.NewAppError(domainErrors.TypeGit, "error getting git diff", err)
	}

	if diff == "" {
		return commitInfo, domainErrors.ErrNoChanges
	}

	recentHistory, _ := s.git.GetRecentCommitMessages(ctx, 10)

	commitInfo = models.CommitInfo{
		Files:         changes,
		Diff:          diff,
		RecentHistory: strings.Join(recentHistory, "\n"),
	}

	if s.config != nil && s.config.UseTicket {
		ticketID, err := s.getTicketIDFromBranch(ctx)
		if err != nil {
			return commitInfo, domainErrors.NewAppError(domainErrors.TypeGit, "error getting ticket ID from branch", err)
		}

		ticketInfo, err := s.ticketManager.GetTicketInfo(ticketID)
		if err != nil {
			return commitInfo, domainErrors.NewAppError(domainErrors.TypeInternal, "error getting ticket info", err)
		}

		commitInfo.TicketInfo = ticketInfo
	}

	detectedIssue := issueNumber
	if detectedIssue == 0 {
		detectedIssue = s.detectIssueNumber(ctx)
	}

	if detectedIssue > 0 {
		vcsClient, err := s.getOrCreateVCSClient(ctx)
		if err != nil {
			if progress != nil {
				progress(models.ProgressEvent{
					Type:    models.ProgressGeneric,
					Message: fmt.Sprintf("issue_vcs_init_error: %v", err),
				})
			}
		} else {
			issueInfo, err := vcsClient.GetIssue(ctx, detectedIssue)
			if err != nil {
				if progress != nil {
					progress(models.ProgressEvent{
						Type: models.ProgressGeneric,
						Data: map[string]interface{}{
							"Error":   err.Error(),
							"IssueID": detectedIssue,
						},
					})
				}
			} else {
				if progress != nil {
					progress(models.ProgressEvent{
						Type: models.ProgressIssuesDetected,
						Data: map[string]interface{}{
							"IssueID": detectedIssue,
							"Title":   issueInfo.Title,
							"IsAuto":  issueNumber == 0,
						},
					})
				}
				commitInfo.IssueInfo = issueInfo

				if len(issueInfo.Criteria) > 0 && commitInfo.TicketInfo == nil {
					commitInfo.TicketInfo = &models.TicketInfo{
						TicketID:    fmt.Sprintf("#%d", issueInfo.Number),
						TicketTitle: issueInfo.Title,
						TitleDesc:   issueInfo.Description,
						Criteria:    issueInfo.Criteria,
					}
				}
			}
		}
	}

	return commitInfo, nil
}

func (s *CommitService) getOrCreateVCSClient(ctx context.Context) (ports.VCSClient, error) {
	if s.vcsClient != nil {
		return s.vcsClient, nil
	}

	if s.config == nil {
		return nil, domainErrors.ErrConfigMissing
	}

	owner, repo, provider, err := s.git.GetRepoInfo(ctx)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting repo info", err)
	}

	vcsConfig, exists := s.config.VCSConfigs[provider]
	if !exists {
		if s.config.ActiveVCSProvider != "" {
			vcsConfig, exists = s.config.VCSConfigs[s.config.ActiveVCSProvider]
			if !exists {
				return nil, domainErrors.ErrConfigMissing
			}
			provider = s.config.ActiveVCSProvider
		} else {
			return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("VCS provider '%s' not configured", provider), nil)
		}
	}

	switch provider {
	case "github":
		// Removing 'nil' for translations as per decoupling plan (assuming GitHubClient is updated)
		return github.NewGitHubClient(owner, repo, vcsConfig.Token), nil
	default:
		return nil, domainErrors.ErrVCSNotSupported
	}
}

// detectIssueNumber attempts to automatically detect the issue number
// Priority: 1) Branch name, 2) Recent commits
func (s *CommitService) detectIssueNumber(ctx context.Context) int {
	if issueNum := s.detectIssueFromBranch(ctx); issueNum > 0 {
		return issueNum
	}

	if issueNum := s.detectIssueFromCommits(ctx); issueNum > 0 {
		return issueNum
	}

	return 0
}

// detectIssueFromBranch detects issue number from the branch name
// Supported patterns: 123-desc, feature/123-desc, #123, issue-123, issue/123
func (s *CommitService) detectIssueFromBranch(ctx context.Context) int {
	branchName, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		return 0
	}

	patterns := []string{
		`#(\d+)`,         // #123
		`issue[/-](\d+)`, // issue-123, issue/123
		`^(\d+)-`,        // 123-feature
		`/(\d+)-`,        // feature/123-desc
		`-(\d+)-`,        // bugfix-123-description
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(branchName); len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				return num
			}
		}
	}

	return 0
}

// detectIssueFromCommits detects issue number from recent commits
// Search for GitHub keywords: fixes, closes, resolves followed by #123
func (s *CommitService) detectIssueFromCommits(ctx context.Context) int {
	commitMessages, err := s.git.GetRecentCommitMessages(ctx, 5)
	if err != nil {
		return 0
	}

	// https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue
	keywords := []string{
		"fix", "fixes", "fixed",
		"close", "closes", "closed",
		"resolve", "resolves", "resolved",
	}

	for _, keyword := range keywords {
		pattern := fmt.Sprintf(`(?i)\b%s\s+#(\d+)\b`, keyword)
		re := regexp.MustCompile(pattern)
		for _, msg := range commitMessages {
			if match := re.FindStringSubmatch(msg); len(match) > 1 {
				if num, err := strconv.Atoi(match[1]); err == nil {
					return num
				}
			}
		}
	}

	return 0
}

func (s *CommitService) getTicketIDFromBranch(ctx context.Context) (string, error) {
	branchName, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		return "", domainErrors.NewAppError(domainErrors.TypeGit, "error getting branch name", err)
	}

	re := regexp.MustCompile(`([A-Za-z]+-\d+)`)
	match := re.FindString(branchName)
	if match == "" {
		return "", domainErrors.NewAppError(domainErrors.TypeGit, "ticket ID not found in branch", nil)
	}

	return match, nil
}
