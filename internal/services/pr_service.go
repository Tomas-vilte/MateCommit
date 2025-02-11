package services

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

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

func (s *PRService) SummarizePR(ctx context.Context, prNumber int) (string, error) {
	// Obtener los datos del PR desde GitHub
	prData, err := s.vcsClient.GetPR(ctx, prNumber)
	if err != nil {
		msg := s.trans.GetMessage("error_fetching_pr", 0, map[string]interface{}{
			"PRNumber": prNumber,
			"Error":    err.Error(),
		})
		return "", fmt.Errorf("%s", msg)
	}

	// Construir el prompt para el modelo AI
	prompt := s.buildPRPrompt(prData)

	// Generar el resumen usando el servicio AI
	summary, err := s.aiService.GeneratePRSummary(ctx, prompt)
	if err != nil {
		msg := s.trans.GetMessage("error_generating_summary", 0, map[string]interface{}{
			"PRNumber": prNumber,
			"Error":    err.Error(),
		})
		return "", fmt.Errorf("%s", msg)
	}

	// Actualizar el PR con el resumen generado
	err = s.vcsClient.UpdatePR(ctx, prNumber, models.PRSummary{
		Title: s.generatePRTitle(prData, summary),
		Body:  s.formatPRBody(prData, summary),
	})
	if err != nil {
		msg := s.trans.GetMessage("error_updating_pr", 0, map[string]interface{}{
			"PRNumber": prNumber,
			"Error":    err.Error(),
		})
		return "", fmt.Errorf("%s", msg)
	}

	return summary, nil
}

func (s *PRService) buildPRPrompt(prData models.PRData) string {
	// Construir un prompt que incluya toda la información relevante del PR
	var prompt string

	prompt += fmt.Sprintf("PR #%d by %s\n\n", prData.ID, prData.Creator)

	// Agregar información de los commits
	prompt += "Commits:\n"
	for _, commit := range prData.Commits {
		prompt += fmt.Sprintf("- %s\n", commit.Message)
	}
	prompt += "\n"

	// Agregar el diff
	prompt += "Changes:\n"
	prompt += prData.Diff

	return prompt
}

func (s *PRService) generatePRTitle(prData models.PRData, summary string) string {
	// Por ahora mantenemos el título original si existe, o usamos un título genérico
	// En el futuro, podríamos extraer un título del resumen generado
	return fmt.Sprintf("[AI Summary] PR #%d", prData.ID)
}

func (s *PRService) formatPRBody(prData models.PRData, summary string) string {
	// Formatear el cuerpo del PR con el resumen y mantener el contenido original
	body := "## 🤖 AI Generated Summary\n\n"
	body += summary
	body += "\n\n---\n\n"
	body += "## Original Changes\n\n"

	// Agregar los commits originales
	body += "### Commits\n"
	for _, commit := range prData.Commits {
		body += fmt.Sprintf("- %s\n", commit.Message)
	}

	return body
}
