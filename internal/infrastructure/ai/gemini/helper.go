package gemini

import (
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"google.golang.org/genai"
)

// extractUsage extrae los metadatos de uso de la respuesta de Gemini
func extractUsage(resp *genai.GenerateContentResponse) *models.TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return nil
	}
	return &models.TokenUsage{
		InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
		OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		TotalTokens:  int(resp.UsageMetadata.TotalTokenCount),
	}
}

// GetGenerateConfig retorna la configuración óptima para el modelo, activando Thinking Mode si es compatible.
func GetGenerateConfig(modelName string, responseType string) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature:     float32Ptr(0.3),
		MaxOutputTokens: int32(10000),
		MediaResolution: genai.MediaResolutionHigh,
	}

	if responseType == "application/json" {
		config.ResponseMIMEType = "application/json"
	}

	if strings.HasPrefix(modelName, "gemini-3") {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingLevel:   genai.ThinkingLevelHigh,
		}
	}

	return config
}

func float32Ptr(f float32) *float32 {
	return &f
}
