package gemini

import (
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"google.golang.org/genai"
)

// extractUsage extrae los metadatos de uso de la respuesta de Gemini
func extractUsage(resp *genai.GenerateContentResponse) *models.UsageMetadata {
	if resp.UsageMetadata == nil {
		return nil
	}
	return &models.UsageMetadata{
		InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
		OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		TotalTokens:  int(resp.UsageMetadata.TotalTokenCount),
	}
}
