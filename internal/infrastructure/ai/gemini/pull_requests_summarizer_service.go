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
	*GeminiProvider
	wrapper    *ai.CostAwareWrapper
	generateFn ai.GenerateFunc
	config     *config.Config
	trans      *i18n.Translations
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
	modelName := string(cfg.AIConfig.Models[config.AIGemini])

	budgetDaily := 0.0
	if cfg.AIConfig.BudgetDaily != nil {
		budgetDaily = *cfg.AIConfig.BudgetDaily
	}

	service := &GeminiPRSummarizer{
		GeminiProvider: NewGeminiProvider(client, modelName),
		config:         cfg,
		trans:          trans,
	}

	wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
		Provider:              service,
		BudgetDaily:           budgetDaily,
		Trans:                 trans,
		EstimatedOutputTokens: 500,
	})
	if err != nil {
		return nil, fmt.Errorf("error creando wrapper: %w", err)
	}

	service.wrapper = wrapper
	service.generateFn = service.defaultGenerate

	return service, nil
}

func (gps *GeminiPRSummarizer) defaultGenerate(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
	genConfig := GetGenerateConfig(mName, "application/json")

	resp, err := gps.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
	if err != nil {
		return nil, nil, err
	}

	usage := extractUsage(resp)
	return resp, usage, nil
}

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prContent string) (models.PRSummary, error) {
	prompt := gps.generatePRPrompt(prContent)

	resp, usage, err := gps.wrapper.WrapGenerate(ctx, "summarize-pr", prompt, gps.generateFn)
	if err != nil {
		return models.PRSummary{}, fmt.Errorf("error al generar resumen de PR: %w", err)
	}

	var responseText string
	if geminiResp, ok := resp.(*genai.GenerateContentResponse); ok {
		responseText = formatResponse(geminiResp)
	} else if s, ok := resp.(string); ok {
		responseText = s
	}
	if responseText == "" {
		return models.PRSummary{}, fmt.Errorf("respuesta vacía de la IA")
	}
	responseText = ExtractJSON(responseText)
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
