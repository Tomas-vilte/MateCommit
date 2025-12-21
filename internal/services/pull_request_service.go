package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	domainErrors "github.com/Tomas-vilte/MateCommit/internal/domain/errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
)

// prVCSClient defines the methods needed by PRService from a VCS provider.
type prVCSClient interface {
	GetPR(ctx context.Context, prNumber int) (models.PRData, error)
	GetPRIssues(ctx context.Context, branchName string, commitMessages []string, description string) ([]models.Issue, error)
	UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error
}

// prAIProvider defines the methods needed by PRService from an AI provider.
type prAIProvider interface {
	GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error)
}

type PRService struct {
	vcsClient prVCSClient
	aiService prAIProvider
	config    *config.Config
}

type PROption func(*PRService)

func WithPRVCSClient(vcs prVCSClient) PROption {
	return func(s *PRService) {
		s.vcsClient = vcs
	}
}

func WithPRAIProvider(ai prAIProvider) PROption {
	return func(s *PRService) {
		s.aiService = ai
	}
}

func WithPRConfig(cfg *config.Config) PROption {
	return func(s *PRService) {
		s.config = cfg
	}
}

func NewPRService(opts ...PROption) *PRService {
	s := &PRService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *PRService) SummarizePR(ctx context.Context, prNumber int, progress func(models.ProgressEvent)) (models.PRSummary, error) {
	if s.aiService == nil {
		return models.PRSummary{}, domainErrors.ErrAPIKeyMissing
	}

	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		return models.PRSummary{}, domainErrors.NewAppError(domainErrors.TypeVCS, "error getting PR", err)
	}

	var commitMessages []string
	for _, commit := range prData.Commits {
		commitMessages = append(commitMessages, commit.Message)
	}

	issues, err := s.vcsClient.GetPRIssues(ctx, prData.BranchName, commitMessages, prData.Description)
	if err == nil && len(issues) > 0 {
		prData.RelatedIssues = issues

		issueNums := make([]string, len(issues))
		for i, issue := range issues {
			issueNums[i] = fmt.Sprintf("#%d", issue.Number)
		}

		if progress != nil {
			progress(models.ProgressEvent{
				Type: models.ProgressIssuesDetected,
				Data: map[string]interface{}{
					"PRNumber": prNumber,
					"Issues":   issueNums,
				},
			})
		}
	}

	prompt := s.buildPRPrompt(prData)

	summary, err := s.aiService.GeneratePRSummary(ctx, prompt)
	if err != nil {
		return models.PRSummary{}, domainErrors.NewAppError(domainErrors.TypeAI, "error generating PR summary", err)
	}

	if len(prData.RelatedIssues) > 0 {
		summary = s.ensurePRIssueReferences(summary, prData.RelatedIssues)
		if progress != nil {
			progress(models.ProgressEvent{
				Type: models.ProgressIssuesClosing,
				Data: map[string]interface{}{
					"Count": len(prData.RelatedIssues),
				},
			})
		}
	}

	breakingChanges := s.detectBreakingChanges(prData.Commits)
	if len(breakingChanges) > 0 {
		if progress != nil {
			progress(models.ProgressEvent{
				Type: models.ProgressBreakingChanges,
				Data: map[string]interface{}{
					"Count": len(breakingChanges),
				},
			})
		}
		summary = s.addBreakingChangesToSummary(summary, breakingChanges)
	}

	testPlan := s.generateTestPlan(prData)
	if testPlan != "" {
		if progress != nil {
			progress(models.ProgressEvent{
				Type: models.ProgressTestPlan,
			})
		}
		summary.Body += testPlan
	}

	err = s.vcsClient.UpdatePR(ctx, prNumber, summary)
	if err != nil {
		return models.PRSummary{}, domainErrors.NewAppError(domainErrors.TypeVCS, "error updating PR", err)
	}

	return summary, nil
}

func (s *PRService) buildPRPrompt(prData models.PRData) string {
	var prompt string

	prompt += fmt.Sprintf("PR #%d by %s\n", prData.ID, prData.Creator)
	prompt += fmt.Sprintf("Branch: %s\n\n", prData.BranchName)

	commitCount := len(prData.Commits)
	diffLines := strings.Count(prData.Diff, "\n")
	filesChanged := strings.Count(prData.Diff, "diff --git")

	prompt += fmt.Sprintf("Stats: %d commits, %d files, ~%d lines\n\n", commitCount, filesChanged, diffLines)
	breakingChanges := s.detectBreakingChanges(prData.Commits)
	if len(breakingChanges) > 0 {
		prompt += "⚠️ Breaking Changes:\n"
		for _, bc := range breakingChanges {
			prompt += fmt.Sprintf("- %s\n", bc)
		}
		prompt += "\n"
	}

	if len(prData.RelatedIssues) > 0 {
		locale := s.config.Language
		issuesFormatted := ai.FormatIssuesForPrompt(prData.RelatedIssues, locale)
		data := ai.PromptData{RelatedIssues: issuesFormatted}
		issueContext, _ := ai.RenderPrompt("prIssueContext", ai.GetPRIssueContextInstructions(locale), data)
		prompt += issueContext + "\n\n"
	}

	prompt += "Commits:\n"
	for _, commit := range prData.Commits {
		prompt += fmt.Sprintf("- %s\n", commit.Message)
	}
	prompt += "\n"

	prompt += "Main files modified:\n"
	lines := strings.Split(prData.Diff, "\n")
	fileCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				file := strings.TrimPrefix(parts[2], "a/")
				prompt += fmt.Sprintf("- %s\n", file)
				fileCount++
				if fileCount >= 20 {
					break
				}
			}
		}
	}
	prompt += "\n"

	prompt += "Changes (diff completo):\n"
	prompt += prData.Diff

	return prompt
}

func (s *PRService) ensurePRIssueReferences(summary models.PRSummary, issues []models.Issue) models.PRSummary {
	if len(issues) == 0 {
		return summary
	}

	var closingRefs []string
	for _, issue := range issues {
		keyword := "Closes"
		issueTitleLower := strings.ToLower(issue.Title)
		if strings.Contains(issueTitleLower, "bug") ||
			strings.Contains(issueTitleLower, "fix") ||
			strings.Contains(issueTitleLower, "error") {
			keyword = "Fixes"
		}
		closingRefs = append(closingRefs, fmt.Sprintf("%s #%d", keyword, issue.Number))
	}

	closingLine := strings.Join(closingRefs, ", ")

	bodyLower := strings.ToLower(summary.Body)
	hasClosingRefs := false
	for _, issue := range issues {
		if strings.Contains(bodyLower, fmt.Sprintf("closes #%d", issue.Number)) ||
			strings.Contains(bodyLower, fmt.Sprintf("fixes #%d", issue.Number)) ||
			strings.Contains(bodyLower, fmt.Sprintf("resolves #%d", issue.Number)) {
			hasClosingRefs = true
			break
		}
	}

	if !hasClosingRefs {
		summary.Body = closingLine + "\n\n" + summary.Body
	}

	return summary
}

func (s *PRService) detectBreakingChanges(commits []models.Commit) []string {
	var breaking []string
	for _, commit := range commits {
		msgLower := strings.ToLower(commit.Message)
		if strings.Contains(msgLower, "breaking change") ||
			strings.Contains(msgLower, "breaking:") ||
			strings.Contains(msgLower, "!:") {
			breaking = append(breaking, commit.Message)
		}
	}
	return breaking
}

func (s *PRService) generateTestPlan(prData models.PRData) string {
	if len(prData.RelatedIssues) == 0 {
		return ""
	}

	var testPlan strings.Builder
	testPlan.WriteString("\n\n## Test Plan\n\n")

	for _, issue := range prData.RelatedIssues {
		testPlan.WriteString(fmt.Sprintf("- [ ] Verify #%d is resolved\n", issue.Number))
	}

	testPlan.WriteString("- [ ] Run existing tests\n")
	testPlan.WriteString("- [ ] Verify no regressions\n")

	return testPlan.String()
}

func (s *PRService) addBreakingChangesToSummary(summary models.PRSummary, breakingChanges []string) models.PRSummary {
	if len(breakingChanges) == 0 {
		return summary
	}

	var breakingSection strings.Builder
	breakingSection.WriteString("\n\n## ⚠️ Breaking Changes\n\n")

	for _, change := range breakingChanges {
		breakingSection.WriteString(fmt.Sprintf("- %s\n", change))
	}

	summary.Body += breakingSection.String()
	return summary
}
