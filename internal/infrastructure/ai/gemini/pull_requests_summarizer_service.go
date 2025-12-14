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
	if cfg.GeminiAPIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		msg := trans.GetMessage("error_gemini_client", 0, map[string]interface{}{
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

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error) {
	modelName := string(gps.config.AIConfig.Models[config.AIGemini])

	genConfig := &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.3),
		MaxOutputTokens:  int32(10000),
		ResponseMIMEType: "application/json",
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

	return models.PRSummary{
		Title:  jsonSummary.Title,
		Body:   jsonSummary.Body,
		Labels: jsonSummary.Labels,
	}, nil
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	template := ai.GetPRPromptTemplate(gps.config.Language)
	return fmt.Sprintf(template, prContent)
}
