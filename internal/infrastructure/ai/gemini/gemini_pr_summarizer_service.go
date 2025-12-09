package gemini

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

type GeminiPRSummarizer struct {
	client *genai.Client
	config *config.Config
	trans  *i18n.Translations
}

var validLabels = map[string]bool{
	"feature":     true,
	"fix":         true,
	"refactor":    true,
	"docs":        true,
	"infra":       true,
	"test":        true,
	"performance": true,
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
	if prompt == "" {
		msg := gps.trans.GetMessage("gemini_service.error_empty_prompt", 0, nil)
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	formattedPrompt := gps.generatePRPrompt(prompt)
	modelName := string(gps.config.AIConfig.Models[config.AIGemini])

	resp, err := gps.client.Models.GenerateContent(ctx, modelName, genai.Text(formattedPrompt), nil)
	if err != nil {
		return models.PRSummary{}, err
	}

	rawSummary := formatResponse(resp)
	if rawSummary == "" {
		msg := gps.trans.GetMessage("gemini_service.response_empty", 0, nil)
		return models.PRSummary{}, fmt.Errorf("%s", msg)
	}

	return gps.parseSummary(rawSummary)
}

func (gps *GeminiPRSummarizer) generatePRPrompt(prContent string) string {
	template := ai.GetPRPromptTemplate(gps.config.Language)
	return fmt.Sprintf(template, prContent)
}

func (gps *GeminiPRSummarizer) cleanLabel(label string) string {
	cleaned := strings.TrimSpace(label)

	cleaned = strings.Trim(cleaned, `"'`)

	cleaned = strings.Trim(cleaned, "`")

	cleaned = strings.Trim(cleaned, "*_-~")

	cleaned = strings.ToLower(cleaned)

	reg := regexp.MustCompile(`[^a-z0-9\-_]`)
	cleaned = reg.ReplaceAllString(cleaned, "")

	return cleaned
}

func (gps *GeminiPRSummarizer) isValidLabel(label string) bool {
	return validLabels[label]
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
				labelText := lines[1]

				var labelParts []string
				if strings.Contains(labelText, ",") {
					labelParts = strings.Split(labelText, ",")
				} else {
					labelText = strings.ReplaceAll(labelText, "\n", " ")
					labelParts = strings.Fields(labelText)
				}

				for _, l := range labelParts {
					cleaned := gps.cleanLabel(l)

					if cleaned != "" && gps.isValidLabel(cleaned) {
						summary.Labels = append(summary.Labels, cleaned)
					}
				}
				break
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
		msg := gps.trans.GetMessage("gemini_service.title_not_found", 0, nil)
		return summary, fmt.Errorf("%s", msg)
	}

	if utf8.RuneCountInString(summary.Title) > 80 {
		summary.Title = string([]rune(summary.Title)[:77]) + "..."
	}

	return summary, nil
}
