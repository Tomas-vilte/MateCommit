package services

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

var _ ports.PRService = (*PRService)(nil)

type PRService struct {
	vcsClient ports.VCSClient
	aiService ports.PRSummarizer
	trans     *i18n.Translations
}

func NewPRService(vcsClient ports.VCSClient, aiService ports.PRSummarizer, trans *i18n.Translations) *PRService {
	return &PRService{
		vcsClient: vcsClient,
		aiService: aiService,
		trans:     trans,
	}
}

func (s *PRService) SummarizePR(ctx context.Context, prNumber int) (models.PRSummary, error) {
	if s.aiService == nil {
		msg := s.trans.GetMessage("ai_missing_for_pr", 0, nil)
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}
	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		msg := s.trans.GetMessage("pr_service.error_get_pr", 0, map[string]interface{}{
			"Error": err,
		})
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	prompt := s.buildPRPrompt(prData)

	summary, err := s.aiService.GeneratePRSummary(ctx, prompt)
	if err != nil {
		msg := s.trans.GetMessage("pr_service.error_create_summary_pr", 0, map[string]interface{}{
			"Error": err,
		})
		return models.PRSummary{}, fmt.Errorf("%s", msg)
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

	prompt += fmt.Sprintf("PR #%d by %s\n\n", prData.ID, prData.Creator)

	prompt += "Commits:\n"
	for _, commit := range prData.Commits {
		prompt += fmt.Sprintf("- %s\n", commit.Message)
	}
	prompt += "\n"

	prompt += "Changes:\n"
	prompt += prData.Diff

	return prompt
}
