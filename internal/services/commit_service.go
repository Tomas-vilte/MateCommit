package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/github"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ports"
	"github.com/thomas-vilte/matecommit/internal/regex"
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
	ticketManager ports.TicketManager
	vcsClient     ports.VCSClient
	config        *config.Config
}

type Option func(*CommitService)

func WithTicketManager(tm ports.TicketManager) Option {
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
	log := logger.FromContext(ctx)

	log.Info("generating commit suggestions",
		"count", count,
		"issue_number", issueNumber,
	)
	commitInfo, err := s.buildCommitInfo(ctx, issueNumber, progress)
	if err != nil {
		log.Error("failed to build commit info",
			"error", err,
			"issue_number", issueNumber,
		)
		return nil, err
	}
	log.Debug("commit info built successfully",
		"files_changed", len(commitInfo.Files),
		"has_diff", len(commitInfo.Diff) > 0,
		"has_issue_info", commitInfo.IssueInfo != nil,
	)

	suggestions, err := s.ai.GenerateSuggestions(ctx, commitInfo, count)
	if err != nil {
		log.Error("failed to generate suggestions",
			"error", err,
			"commit_info_size", len(commitInfo.Diff))
		return nil, err
	}

	log.Info("suggestions generated successfully",
		"count", len(suggestions))

	return suggestions, nil
}

func (s *CommitService) buildCommitInfo(ctx context.Context, issueNumber int, progress func(models.ProgressEvent)) (models.CommitInfo, error) {
	log := logger.FromContext(ctx)

	log.Debug("building commit info",
		"issue_number", issueNumber,
	)

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

	detectedIssueNumber := issueNumber
	if detectedIssueNumber == 0 {
		detectedIssueNumber = s.detectIssueNumber(ctx)
	}

	if detectedIssueNumber > 0 {
		log.Debug("issue number detected",
			"issue_number", detectedIssueNumber,
			"source", "branch_or_commits",
		)

		vcsClient, err := s.getOrCreateVCSClient(ctx)
		if err != nil {
			if progress != nil {
				progress(models.ProgressEvent{
					Type:    models.ProgressGeneric,
					Message: fmt.Sprintf("issue_vcs_init_error: %v", err),
				})
			}
		} else {
			issueInfo, err := vcsClient.GetIssue(ctx, detectedIssueNumber)
			if err != nil {
				if progress != nil {
					progress(models.ProgressEvent{
						Type: models.ProgressGeneric,
						Data: &models.ProgressData{
							Error:  err.Error(),
							Number: detectedIssueNumber,
						},
					})
				}
			} else {
				if progress != nil {
					progress(models.ProgressEvent{
						Type: models.ProgressIssuesDetected,
						Data: &models.ProgressData{
							Number: detectedIssueNumber,
							Title:  issueInfo.Title,
							IsAuto: issueNumber == 0,
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

	log := logger.FromContext(ctx)
	log.Debug("creating VCS client")

	if s.config == nil {
		return nil, domainErrors.ErrConfigMissing
	}

	owner, repo, provider, err := s.git.GetRepoInfo(ctx)
	if err != nil {
		logger.Error(ctx, "failed to get repo info for VCS client", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting repo info", err)
	}

	log.Debug("repo info retrieved", "owner", owner, "repo", repo, "provider", provider)

	vcsConfig, exists := s.config.VCSConfigs[provider]
	if !exists {
		if s.config.ActiveVCSProvider != "" {
			vcsConfig, exists = s.config.VCSConfigs[s.config.ActiveVCSProvider]
			if !exists {
				return nil, domainErrors.ErrConfigMissing
			}
			provider = s.config.ActiveVCSProvider
			log.Debug("using active VCS provider", "provider", provider)
		} else {
			return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("VCS provider '%s' not configured", provider), nil)
		}
	}

	switch provider {
	case "github":
		if vcsConfig.Token == "" {
			return nil, domainErrors.ErrTokenMissing
		}
		log.Debug("VCS client created successfully", "provider", "github")
		return github.NewGitHubClient(owner, repo, vcsConfig.Token), nil
	default:
		return nil, domainErrors.ErrVCSNotSupported
	}
}

// detectIssueNumber attempts to automatically detect the issue number
// Priority: 1) Branch name, 2) Recent commits
func (s *CommitService) detectIssueNumber(ctx context.Context) int {
	log := logger.FromContext(ctx)
	log.Debug("attempting to detect issue number")

	if issueNum := s.detectIssueFromBranch(ctx); issueNum > 0 {
		log.Debug("issue number detected from branch", "issue_number", issueNum)
		return issueNum
	}

	if issueNum := s.detectIssueFromCommits(ctx); issueNum > 0 {
		log.Debug("issue number detected from commits", "issue_number", issueNum)
		return issueNum
	}

	log.Debug("no issue number detected")
	return 0
}

// detectIssueFromBranch detects issue number from the branch name
// Supported patterns: 123-desc, feature/123-desc, #123, issue-123, issue/123
func (s *CommitService) detectIssueFromBranch(ctx context.Context) int {
	branchName, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		logger.Debug(ctx, "failed to get current branch for issue detection", "error", err.Error())
		return 0
	}

	logger.Debug(ctx, "checking branch name for issue number", "branch", branchName)

	for _, re := range []*regexp.Regexp{
		regex.BranchIssueSharp,
		regex.BranchIssueName,
		regex.BranchIssueStart,
		regex.BranchIssueFolder,
		regex.BranchIssueMid,
	} {
		if match := re.FindStringSubmatch(branchName); len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				logger.Debug(ctx, "issue number found in branch name", "issue_number", num, "branch", branchName)
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
		logger.Debug(ctx, "failed to get recent commits for issue detection", "error", err.Error())
		return 0
	}

	logger.Debug(ctx, "checking recent commits for issue references", "commit_count", len(commitMessages))

	// https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue
	for _, msg := range commitMessages {
		if match := regex.GitHubClosedLink.FindStringSubmatch(msg); len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				logger.Debug(ctx, "issue number found in commit message", "issue_number", num)
				return num
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

	match := regex.JiraTicket.FindString(branchName)
	if match == "" {
		return "", domainErrors.NewAppError(domainErrors.TypeGit, "ticket ID not found in branch", nil)
	}

	return match, nil
}
