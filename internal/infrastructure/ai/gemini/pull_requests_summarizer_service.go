package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

var _ ports.PRSummarizer = (*GeminiPRSummarizer)(nil)

type GeminiPRSummarizer struct {
	client *genai.Client
	config *config.Config
	trans  *i18n.Translations
}

type PRSummaryJSON struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

func NewGeminiPRSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiPRSummarizer, error) {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists || providerCfg.APIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, map[string]interface{}{"Provider": "gemini"})
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  providerCfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		msg := trans.GetMessage("ai_service.error_ai_client", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	return &GeminiPRSummarizer{
		client: client,
		config: cfg,
		trans:  trans,
	}, nil
}

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prContent string) (models.PRSummary, error) {
	modelName := string(gps.config.AIConfig.Models[config.AIGemini])

	prompt := gps.generatePRPrompt(prContent)

	genConfig := &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.3),
		MaxOutputTokens:  int32(10000),
		ResponseMIMEType: "application/json",
		MediaResolution:  genai.MediaResolutionHigh,
	}

	resp, err := gps.client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), genConfig)
	if err != nil {
		return models.PRSummary{}, fmt.Errorf("error al generar resumen de PR: %w", err)
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return models.PRSummary{}, fmt.Errorf("respuesta vacía de la IA")
	}

	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var jsonSummary PRSummaryJSON
	if err := json.Unmarshal([]byte(responseText), &jsonSummary); err != nil {
		return models.PRSummary{}, fmt.Errorf("error al parsear JSON de PR: %w", err)
	}

	if strings.TrimSpace(jsonSummary.Title) == "" {
		respLen := len(responseText)
		preview := responseText
		if respLen > 500 {
			preview = responseText[:500] + "..."
		}
		return models.PRSummary{}, fmt.Errorf("la IA no generó un título para el PR. Respuesta (longitud: %d): %s", respLen, preview)
	}

	usage := extractUsage(resp)

	return models.PRSummary{
		Title:  jsonSummary.Title,
		Body:   jsonSummary.Body,
		Labels: jsonSummary.Labels,
		Usage:  usage,
	}, nil
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	template := ai.GetPRPromptTemplate(gps.config.Language)
	return fmt.Sprintf(template, prContent)
}
