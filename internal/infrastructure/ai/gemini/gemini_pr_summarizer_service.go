package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
	"unicode/utf8"
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

func (gps *GeminiPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error) {
	if prompt == "" {
		msg := gps.trans.GetMessage("error_empty_prompt", 0, nil)
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	formattedPrompt := gps.generatePRPrompt(prompt)

	resp, err := gps.model.GenerateContent(ctx, genai.Text(formattedPrompt))
	if err != nil {
		return models.PRSummary{}, err
	}

	rawSummary := formatResponse(resp)
	if rawSummary == "" {
		return models.PRSummary{}, fmt.Errorf("respuesta vacía de la IA")
	}

	return gps.parseSummary(rawSummary)
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	template := ai.GetPRPromptTemplate(gps.config.Language)
	return fmt.Sprintf(template, prContent)
}

func (gps *GeminiPRSummarizer) parseSummary(raw string) (models.PRSummary, error) {
	summary := models.PRSummary{}
	raw = strings.ReplaceAll(raw, "## ", "##")
	sections := strings.Split(raw, "##")
	titleKey := gps.trans.GetMessage("gemini_service.pr_title_section", 0, nil)
	labelsKey := gps.trans.GetMessage("gemini_service.pr_labels_section", 0, nil)
	changesKey := gps.trans.GetMessage("gemini_service.pr_changes_section", 0, nil)

	for _, sec := range sections {
		if strings.HasPrefix(sec, titleKey) {
			lines := strings.SplitN(sec, "\n", 2)
			if len(lines) > 1 {
				summary.Title = strings.TrimSpace(lines[1])
				break
			}
		}
	}

	for _, sec := range sections {
		if strings.HasPrefix(sec, labelsKey) {
			lines := strings.SplitN(sec, "\n", 2)
			if len(lines) > 1 {
				labels := strings.Split(lines[1], ",")
				for _, l := range labels {
					cleaned := strings.TrimSpace(l)
					if cleaned != "" {
						summary.Labels = append(summary.Labels, cleaned)
					}
				}
			}
		}
	}

	var bodyParts []string
	for _, sec := range sections {
		if strings.HasPrefix(sec, changesKey) {
			lines := strings.SplitN(sec, "\n", 2)
			if len(lines) > 1 {
				bodyParts = append(bodyParts, strings.TrimSpace(lines[1]))
			}
		}
	}
	summary.Body = strings.Join(bodyParts, "\n\n")

	if summary.Title == "" {
		return summary, fmt.Errorf("no se encontró el título en la respuesta")
	}

	if utf8.RuneCountInString(summary.Title) > 80 {
		summary.Title = string([]rune(summary.Title)[:77]) + "..."
	}

	return summary, nil
}
