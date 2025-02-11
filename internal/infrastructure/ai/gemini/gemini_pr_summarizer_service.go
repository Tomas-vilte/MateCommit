package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiPRSummarizer struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config *config.Config
	trans  *i18n.Translations
}

func NewGeminiPRSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiPRSummarizer, error) {
	if cfg.GeminiAPIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	modelName := string(cfg.AIConfig.Models[config.AIGemini])
	model := client.GenerativeModel(modelName)

	return &GeminiPRSummarizer{
		client: client,
		model:  model,
		config: cfg,
		trans:  trans,
	}, nil
}

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		msg := gps.trans.GetMessage("error_empty_prompt", 0, nil)
		return "", fmt.Errorf("%s", msg)
	}

	formattedPrompt := gps.generatePRPrompt(prompt)

	resp, err := gps.model.GenerateContent(ctx, genai.Text(formattedPrompt))
	if err != nil {
		return "", err
	}

	summary := formatResponse(resp)
	if summary == "" {
		return "", fmt.Errorf("error vacio")
	}

	return summary, nil
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	template := ai.GetPRPromptTemplate(gps.config.Language)
	return fmt.Sprintf(template, prContent)
}
