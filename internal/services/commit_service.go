package services

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
)

var _ ports.CommitService = (*CommitService)(nil)

type CommitService struct {
	git           ports.GitService
	ai            ports.CommitSummarizer
	ticketManager ports.TickerManager
	vcsClient     ports.VCSClient
	config        *config.Config
	trans         *i18n.Translations
}

func NewCommitService(
	git ports.GitService,
	ai ports.CommitSummarizer,
	ticketManager ports.TickerManager,
	vcsClient ports.VCSClient,
	cfg *config.Config,
	trans *i18n.Translations) *CommitService {
	return &CommitService{
		git:           git,
		ai:            ai,
		ticketManager: ticketManager,
		vcsClient:     vcsClient,
		config:        cfg,
		trans:         trans,
	}
}

func (s *CommitService) GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error) {
	commitInfo, err := s.buildCommitInfo(ctx, 0)
	if err != nil {
		return nil, err
	}
	return s.ai.GenerateSuggestions(ctx, commitInfo, count)
}

func (s *CommitService) GenerateSuggestionsWithIssue(ctx context.Context, count int, issueNumber int) ([]models.CommitSuggestion, error) {
	commitInfo, err := s.buildCommitInfo(ctx, issueNumber)
	if err != nil {
		return nil, err
	}
	return s.ai.GenerateSuggestions(ctx, commitInfo, count)
}

func (s *CommitService) buildCommitInfo(ctx context.Context, issueNumber int) (models.CommitInfo, error) {
	var commitInfo models.CommitInfo

	if s.ai == nil {
		msg := s.trans.GetMessage("ai_missing_for_suggest", 0, nil)
		return commitInfo, fmt.Errorf("%s", msg)
	}

	changes, err := s.git.GetChangedFiles(ctx)
	if err != nil {
		return commitInfo, err
	}

	if len(changes) == 0 {
		msg := s.trans.GetMessage("commit_service.undetected_changes", 0, nil)
		return commitInfo, fmt.Errorf("%s", msg)
	}

	diff, err := s.git.GetDiff(ctx)
	if err != nil {
		msg := s.trans.GetMessage("commit_service.error_getting_diff", 0, map[string]interface{}{
			"Error": err,
		})
		return commitInfo, fmt.Errorf("%s", msg)
	}

	if diff == "" {
		msg := s.trans.GetMessage("commit_service.no_differences_detected", 0, nil)
		return commitInfo, fmt.Errorf("%s", msg)
	}

	files := make([]string, 0)
	for _, change := range changes {
		files = append(files, change.Path)
	}

	commitInfo = models.CommitInfo{
		Files: files,
		Diff:  diff,
	}

	if s.config.UseTicket {
		ticketID, err := s.getTicketIDFromBranch(ctx)
		if err != nil {
			msg := s.trans.GetMessage("commit_service.error_get_id_ticket", 0, map[string]interface{}{
				"Error": err,
			})
			return commitInfo, fmt.Errorf("%s", msg)
		}

		ticketInfo, err := s.ticketManager.GetTicketInfo(ticketID)
		if err != nil {
			msg := s.trans.GetMessage("commit_service.error_get_ticket_info", 0, map[string]interface{}{
				"Error": err,
			})
			return commitInfo, fmt.Errorf("%s", msg)
		}

		commitInfo.TicketInfo = ticketInfo
	}

	// Detección automática de issues (branch name o commits)
	detectedIssue := issueNumber
	if detectedIssue == 0 {
		detectedIssue = s.detectIssueNumber(ctx)
	}

	if detectedIssue > 0 {
		vcsClient, err := s.getOrCreateVCSClient(ctx)
		if err != nil {
			msg := s.trans.GetMessage("issue_vcs_init_error", 0, map[string]interface{}{
				"Error": err.Error(),
			})
			fmt.Println(msg)
		} else {
			issueInfo, err := vcsClient.GetIssue(ctx, detectedIssue)
			if err != nil {
				msg := s.trans.GetMessage("issue_fetch_error", 0, map[string]interface{}{
					"Number": detectedIssue,
					"Error":  err.Error(),
				})
				fmt.Println(msg)
			} else {
				// Feedback al usuario sobre detección automática
				var msg string
				if issueNumber == 0 {
					msg = s.trans.GetMessage("issue_detected_auto", 0, map[string]interface{}{
						"Number": detectedIssue,
						"Title":  issueInfo.Title,
					})
				} else {
					msg = s.trans.GetMessage("issue_using_manual", 0, map[string]interface{}{
						"Number": detectedIssue,
						"Title":  issueInfo.Title,
					})
				}
				fmt.Println(msg)
				commitInfo.IssueInfo = issueInfo
			}
		}
	}

	return commitInfo, nil
}

func (s *CommitService) getOrCreateVCSClient(ctx context.Context) (ports.VCSClient, error) {
	if s.vcsClient != nil {
		return s.vcsClient, nil
	}

	owner, repo, provider, err := s.git.GetRepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("error al obtener información del repositorio: %w", err)
	}

	vcsConfig, exists := s.config.VCSConfigs[provider]
	if !exists {
		if s.config.ActiveVCSProvider != "" {
			vcsConfig, exists = s.config.VCSConfigs[s.config.ActiveVCSProvider]
			if !exists {
				return nil, fmt.Errorf("configuración para el proveedor de VCS '%s' no encontrada", s.config.ActiveVCSProvider)
			}
			provider = s.config.ActiveVCSProvider
		} else {
			return nil, fmt.Errorf("proveedor de VCS '%s' detectado automáticamente pero no configurado", provider)
		}
	}

	switch provider {
	case "github":
		return github.NewGitHubClient(owner, repo, vcsConfig.Token, s.trans), nil
	default:
		return nil, fmt.Errorf("proveedor de VCS no compatible: %s", provider)
	}
}

// detectIssueNumber intenta detectar automáticamente el número de issue
// Prioridad: 1) Branch name, 2) Commits recientes
func (s *CommitService) detectIssueNumber(ctx context.Context) int {
	// Primero intentar desde branch name
	if issueNum := s.detectIssueFromBranch(ctx); issueNum > 0 {
		return issueNum
	}

	// Luego intentar desde commits recientes
	if issueNum := s.detectIssueFromCommits(ctx); issueNum > 0 {
		return issueNum
	}

	return 0
}

// detectIssueFromBranch detecta issue number desde el nombre de la rama
// Patrones soportados: 123-desc, feature/123-desc, #123, issue-123, issue/123
func (s *CommitService) detectIssueFromBranch(ctx context.Context) int {
	branchName, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		return 0
	}

	patterns := []string{
		`#(\d+)`,          // #123
		`issue[/-](\d+)`,  // issue-123, issue/123
		`^(\d+)-`,         // 123-feature
		`/(\d+)-`,         // feature/123-desc
		`-(\d+)-`,         // bugfix-123-description
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

// detectIssueFromCommits detecta issue number desde commits recientes
// Busca keywords de GitHub: fixes, closes, resolves seguido de #123
func (s *CommitService) detectIssueFromCommits(ctx context.Context) int {
	// Obtener últimos 5 commits de la rama actual
	cmd := exec.CommandContext(ctx, "git", "log", "-5", "--pretty=format:%s %b")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	commitMessages := string(output)

	// Patrones de GitHub keywords
	// https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue
	keywords := []string{
		"fix", "fixes", "fixed",
		"close", "closes", "closed",
		"resolve", "resolves", "resolved",
	}

	for _, keyword := range keywords {
		// Buscar "fixes #123", "closes #456", etc.
		pattern := fmt.Sprintf(`(?i)\b%s\s+#(\d+)\b`, keyword)
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(commitMessages); len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				return num
			}
		}
	}

	// También buscar referencias simples como "#123" en commits
	simplePattern := regexp.MustCompile(`#(\d+)`)
	matches := simplePattern.FindAllStringSubmatch(commitMessages, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				return num
			}
		}
	}

	return 0
}

func (s *CommitService) getTicketIDFromBranch(ctx context.Context) (string, error) {
	branchName, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		msg := s.trans.GetMessage("commit_service.error_get_name_from_branch", 0, map[string]interface{}{
			"Error": err,
		})
		return "", fmt.Errorf("%s", msg)
	}

	re := regexp.MustCompile(`([A-Za-z]+-\d+)`)
	match := re.FindString(branchName)
	if match == "" {
		msg := s.trans.GetMessage("commit_service.ticket_id_not_found_branch", 0, nil)
		return "", fmt.Errorf("%s", msg)
	}

	return match, nil
}
