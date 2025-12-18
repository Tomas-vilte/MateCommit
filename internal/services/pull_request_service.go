package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
)

var _ ports.PRService = (*PRService)(nil)

type PRService struct {
	vcsClient ports.VCSClient
	aiService ports.PRSummarizer
	trans     *i18n.Translations
	config    *config.Config
}

func NewPRService(vcsClient ports.VCSClient, aiService ports.PRSummarizer, trans *i18n.Translations,
	cfg *config.Config) *PRService {
	return &PRService{
		vcsClient: vcsClient,
		aiService: aiService,
		trans:     trans,
		config:    cfg,
	}
}

func (s *PRService) SummarizePR(ctx context.Context, prNumber int, progress func(string)) (models.PRSummary, error) {
	if s.aiService == nil {
		msg := s.trans.GetMessage("ai_missing.ai_missing_for_pr", 0, nil)
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		msg := s.trans.GetMessage("pr_service.error_get_pr", 0, map[string]interface{}{
			"Error": err,
		})
		return models.PRSummary{}, fmt.Errorf("%s", msg)
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
		msg := s.trans.GetMessage("pr_detected_issues", 0, map[string]interface{}{
			"Number": prNumber,
			"Issues": strings.Join(issueNums, ", "),
		})
		progress(msg)
	}

	prompt := s.buildPRPrompt(prData)

	summary, err := s.aiService.GeneratePRSummary(ctx, prompt)
	if err != nil {
		msg := s.trans.GetMessage("pr_service.error_create_summary_pr", 0, map[string]interface{}{
			"Error": err,
		})
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	if len(prData.RelatedIssues) > 0 {
		summary = s.ensurePRIssueReferences(summary, prData.RelatedIssues)

		msg := s.trans.GetMessage("pr_issues_will_close_on_merge", 0, map[string]interface{}{
			"Count": len(prData.RelatedIssues),
		})
		progress(msg)
	}

	breakingChanges := s.detectBreakingChanges(prData.Commits)
	if len(breakingChanges) > 0 {
		msg := s.trans.GetMessage("pr_breaking_changes_detected", 0, map[string]interface{}{
			"Count": len(breakingChanges),
		})
		progress(msg)
		summary = s.addBreakingChangesToSummary(summary, breakingChanges)
	}

	testPlan := s.generateTestPlan(prData)
	if testPlan != "" {
		progress(s.trans.GetMessage("pr_test_plan_generated", 0, nil))
		summary.Body += testPlan
	}

	err = s.vcsClient.UpdatePR(ctx, prNumber, summary)
	if err != nil {
		msg := s.trans.GetMessage("pr_service.error_update_pr", 0, map[string]interface{}{
			"Error": err,
		})
		return models.PRSummary{}, fmt.Errorf("%s", msg)
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

	prompt += "📊 **Métricas:**\n"
	prompt += fmt.Sprintf("- %d commits\n", commitCount)
	prompt += fmt.Sprintf("- %d archivos cambiados\n", filesChanged)
	prompt += fmt.Sprintf("- ~%d líneas en diff\n\n", diffLines)

	breakingChanges := s.detectBreakingChanges(prData.Commits)
	if len(breakingChanges) > 0 {
		prompt += "⚠️ **Breaking Changes detectados:**\n"
		for _, bc := range breakingChanges {
			prompt += fmt.Sprintf("- %s\n", bc)
		}
		prompt += "\n"
	}

	if len(prData.RelatedIssues) > 0 {
		locale := s.config.Language
		issuesFormatted := ai.FormatIssuesForPrompt(prData.RelatedIssues, locale)
		issueContext := fmt.Sprintf(ai.GetPRIssueContextInstructions(locale), issuesFormatted)
		prompt += issueContext + "\n\n"
	}

	prompt += "Commits:\n"
	for _, commit := range prData.Commits {
		prompt += fmt.Sprintf("- %s\n", commit.Message)
	}
	prompt += "\n"

	prompt += "Archivos principales modificados:\n"
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
		testPlan.WriteString(fmt.Sprintf("- [ ] Verificar que #%d esté resuelto\n", issue.Number))
	}

	testPlan.WriteString("- [ ] Ejecutar tests existentes\n")
	testPlan.WriteString("- [ ] Verificar que no hay regresiones\n")

	return testPlan.String()
}

// addBreakingChangesToSummary agrega sección de breaking changes al resumen
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
