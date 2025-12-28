package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/ai"
	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ports"
	"google.golang.org/genai"
)

var _ ports.PRSummarizer = (*GeminiPRSummarizer)(nil)

type GeminiPRSummarizer struct {
	*GeminiProvider
	wrapper    *ai.CostAwareWrapper
	generateFn ai.GenerateFunc
	config     *config.Config
}

type PRSummaryJSON struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}

func NewGeminiPRSummarizer(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback) (*GeminiPRSummarizer, error) {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists || providerCfg.APIKey == "" {
		return nil, domainErrors.ErrAPIKeyMissing
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  providerCfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "invalid") ||
			strings.Contains(errMsg, "unauthorized") ||
			strings.Contains(errMsg, "api key") ||
			strings.Contains(errMsg, "authentication") {
			return nil, domainErrors.ErrGeminiAPIKeyInvalid.WithError(err)
		}
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "error creating AI client", err)
	}
	modelName := string(cfg.AIConfig.Models[config.AIGemini])

	budgetDaily := 0.0
	if cfg.AIConfig.BudgetDaily != nil {
		budgetDaily = *cfg.AIConfig.BudgetDaily
	}

	service := &GeminiPRSummarizer{
		GeminiProvider: NewGeminiProvider(client, modelName),
		config:         cfg,
	}

	wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
		Provider:              service,
		BudgetDaily:           budgetDaily,
		EstimatedOutputTokens: 500,
		OnConfirmation:        onConfirmation,
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
	log := logger.FromContext(ctx)

	resp, err := gps.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
	if err != nil {
		log.Error("gemini API call failed",
			"error", err,
			"model", mName)

		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "quota") ||
			strings.Contains(errMsg, "rate limit") ||
			strings.Contains(errMsg, "resource exhausted") {
			return nil, nil, domainErrors.ErrGeminiQuotaExceeded.WithError(err)
		}

		if strings.Contains(errMsg, "invalid") ||
			strings.Contains(errMsg, "unauthorized") ||
			strings.Contains(errMsg, "api key") {
			return nil, nil, domainErrors.ErrGeminiAPIKeyInvalid.WithError(err)
		}

		return nil, nil, domainErrors.ErrAIGeneration.WithError(err)
	}

	usage := extractUsage(resp)
	return resp, usage, nil
}

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prContent string) (models.PRSummary, error) {
	log := logger.FromContext(ctx)

	log.Info("generating PR summary via gemini",
		"content_length", len(prContent))

	prompt := gps.generatePRPrompt(prContent)

	log.Debug("calling gemini API for PR summary",
		"prompt_length", len(prompt))

	resp, usage, err := gps.wrapper.WrapGenerate(ctx, "summarize-pr", prompt, gps.generateFn)
	if err != nil {
		log.Error("failed to generate PR summary",
			"error", err)
		return models.PRSummary{}, err
	}

	var responseText string
	if geminiResp, ok := resp.(*genai.GenerateContentResponse); ok {
		log.Debug("formatResponse received GenerateContentResponse",
			"candidates_count", len(geminiResp.Candidates))
		responseText = formatResponse(geminiResp)
	} else if str, ok := resp.(string); ok {
		responseText = str
		log.Debug("received string response", "length", len(str))
	} else if respMap, ok := resp.(map[string]interface{}); ok {
		log.Debug("received map response from cache, extracting text")
		responseText = extractTextFromMap(respMap)
		log.Debug("extracted text from map", "length", len(responseText))
	} else {
		log.Warn("unexpected response type", "type", fmt.Sprintf("%T", resp))
	}

	if responseText == "" {
		return models.PRSummary{}, domainErrors.ErrInvalidAIOutput.
			WithContext("reason", "empty response from AI").
			WithContext("operation", "summarize PR")
	}

	responseText = ExtractJSON(responseText)
	var jsonSummary PRSummaryJSON
	if err := json.Unmarshal([]byte(responseText), &jsonSummary); err != nil {
		respLen := len(responseText)
		preview := responseText
		if respLen > 500 {
			preview = responseText[:500] + "..."
		}
		return models.PRSummary{}, domainErrors.ErrInvalidAIOutput.
			WithContext("reason", "failed to parse JSON").
			WithContext("response_length", respLen).
			WithContext("preview", preview).
			WithError(err)
	}
	if strings.TrimSpace(jsonSummary.Title) == "" {
		respLen := len(responseText)
		preview := responseText
		if respLen > 500 {
			preview = responseText[:500] + "..."
		}
		log.Warn("AI generated no PR title",
			"response_length", respLen)
		return models.PRSummary{}, domainErrors.ErrInvalidAIOutput.
			WithContext("reason", "AI generated no PR title").
			WithContext("response_length", respLen).
			WithContext("preview", preview)
	}

	log.Info("PR summary generated successfully via gemini",
		"labels_count", len(jsonSummary.Labels))

	return models.PRSummary{
		Title:  jsonSummary.Title,
		Body:   jsonSummary.Body,
		Labels: jsonSummary.Labels,
		Usage:  usage,
	}, nil
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	templateStr := ai.GetPRPromptTemplate(gps.config.Language)
	data := ai.PromptData{
		PRContent: prContent,
	}

	rendered, err := ai.RenderPrompt("prPrompt", templateStr, data)
	if err != nil {
		return ""
	}

	return rendered
}
