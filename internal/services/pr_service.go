package services

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

type PRService struct {
	vcsClient ports.VCSClient
	aiService ports.PRSummarizer
}

func NewPRService(vcsClient ports.VCSClient, aiService ports.PRSummarizer) *PRService {
	return &PRService{
		vcsClient: vcsClient,
		aiService: aiService,
	}
}

func (s *PRService) SummarizePR(ctx context.Context, prNumber int) (models.PRSummary, error) {
	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		return models.PRSummary{}, fmt.Errorf("hubo un error al obtener el pr: %w", err)
	}

	prompt := s.buildPRPrompt(prData)

	summary, err := s.aiService.GeneratePRSummary(ctx, prompt)
	if err != nil {
		return models.PRSummary{}, fmt.Errorf("hubo un error al crear el resumen del pull requests: %w", err)
	}

	err = s.vcsClient.UpdatePR(ctx, prNumber, summary)
	if err != nil {
		return models.PRSummary{}, fmt.Errorf("hubo un error al actualizar el pull requests: %w", err)
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
