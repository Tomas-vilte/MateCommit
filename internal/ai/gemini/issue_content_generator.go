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

type GeminiIssueContentGenerator struct {
	*GeminiProvider
	wrapper    *ai.CostAwareWrapper
	generateFn ai.GenerateFunc
	config     *config.Config
}

var _ ports.IssueContentGenerator = (*GeminiIssueContentGenerator)(nil)

func NewGeminiIssueContentGenerator(ctx context.Context, cfg *config.Config, onConfirmation ai.ConfirmationCallback) (*GeminiIssueContentGenerator, error) {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists || providerCfg.APIKey == "" {
		return nil, domainErrors.ErrAPIKeyMissing
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  providerCfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "error creating AI client", err)
	}

	modelName := string(cfg.AIConfig.Models[config.AIGemini])

	budgetDaily := 0.0
	if cfg.AIConfig.BudgetDaily != nil {
		budgetDaily = *cfg.AIConfig.BudgetDaily
	}

	service := &GeminiIssueContentGenerator{
		GeminiProvider: NewGeminiProvider(client, modelName),
		config:         cfg,
	}

	wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
		Provider:              service,
		BudgetDaily:           budgetDaily,
		EstimatedOutputTokens: 600,
		OnConfirmation:        onConfirmation,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating wrapper: %w", err)
	}

	service.wrapper = wrapper
	service.generateFn = service.defaultGenerate

	return service, nil
}

func (s *GeminiIssueContentGenerator) defaultGenerate(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
	// Don't force JSON mode - let the model return structured text
	genConfig := GetGenerateConfig(mName, "")

	resp, err := s.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
	if err != nil {
		return nil, nil, err
	}

	usage := extractUsage(resp)
	return resp, usage, nil
}

// GenerateIssueContent generates issue content using Gemini AI.
func (s *GeminiIssueContentGenerator) GenerateIssueContent(ctx context.Context, request models.IssueGenerationRequest) (*models.IssueGenerationResult, error) {
	log := logger.FromContext(ctx)

	log.Info("generating issue content via gemini",
		"has_diff", request.Diff != "",
		"has_description", request.Description != "",
		"has_hint", request.Hint != "",
		"files_count", len(request.ChangedFiles))

	prompt := s.buildIssuePrompt(request)

	log.Debug("calling gemini API for issue content",
		"prompt_length", len(prompt))

	resp, usage, err := s.wrapper.WrapGenerate(ctx, "generate-issue", prompt, s.generateFn)
	if err != nil {
		log.Error("failed to generate issue content",
			"error", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "error generating issue content", err)
	}

	var responseText string
	if geminiResp, ok := resp.(*genai.GenerateContentResponse); ok {
		log.Debug("formatResponse received GenerateContentResponse",
			"candidates_count", len(geminiResp.Candidates))
		responseText = formatResponse(geminiResp)
		if len(responseText) > 0 {
			preview := responseText
			if len(responseText) > 100 {
				preview = responseText[:100]
			}
			log.Debug("formatResponse result",
				"response_length", len(responseText),
				"response_preview", preview)
		} else {
			log.Debug("formatResponse result empty")
		}
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
		log.Error("empty response from gemini AI after format")
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "empty response from AI", nil)
	}

	log.Debug("gemini response received",
		"response_length", len(responseText),
		"response_text", responseText)

	result, err := s.parseIssueResponse(responseText)
	if err != nil {
		log.Error("failed to parse issue response",
			"error", err)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "error parsing AI response", err)
	}

	result.Usage = usage

	log.Info("issue content generated successfully via gemini",
		"title", result.Title,
		"labels_count", len(result.Labels))

	return result, nil
}

// buildIssuePrompt builds the prompt to generate issue content.
func (s *GeminiIssueContentGenerator) buildIssuePrompt(request models.IssueGenerationRequest) string {
	if request.Description != "" && request.Diff == "" && request.Hint == "" &&
		request.Template == nil && len(request.ChangedFiles) == 0 {
		return request.Description
	}

	var sb strings.Builder

	if request.Description != "" {
		sb.WriteString(fmt.Sprintf("Global Description: %s\n\n", request.Description))
	}

	if request.Diff != "" {
		sb.WriteString("Code Changes (git diff):\n\n")
		sb.WriteString("```diff\n")
		diff := request.Diff
		sb.WriteString(diff)
		sb.WriteString("\n```\n\n")

		if len(request.ChangedFiles) > 0 {
			sb.WriteString("Changed files:\n")
			for _, file := range request.ChangedFiles {
				sb.WriteString(fmt.Sprintf("- %s\n", file))
			}
			sb.WriteString("\n")
		}
	}

	if request.Hint != "" {
		sb.WriteString(fmt.Sprintf("User Hint: %s\n\n", request.Hint))
	}

	if request.Template != nil {
		lang := request.Language
		if lang == "" {
			lang = "en"
		}
		sb.WriteString(ai.FormatTemplateForPrompt(request.Template, lang, "issue"))
	}

	templateStr := ai.GetIssuePromptTemplate(request.Language)
	data := ai.PromptData{
		IssueInfo: sb.String(),
	}

	rendered, err := ai.RenderPrompt("issuePrompt", templateStr, data)
	if err != nil {
		return ""
	}

	if request.Template != nil {
		rendered += `

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🚨 FINAL REMINDER - CRITICAL OUTPUT REQUIREMENT 🚨

YOU MUST OUTPUT **ONLY** VALID JSON.

The template structure above should be used to FILL the "description" field with markdown content.

BUT your actual response MUST be a JSON object like this:
{
  "title": "string here",
  "description": "markdown content following the template structure",
  "labels": ["array", "of", "strings"]
}

❌ DO NOT output prose like "Here is a high-quality GitHub issue..."
❌ DO NOT output markdown text directly
❌ DO NOT output explanations

✅ ONLY output the JSON object
✅ Use the template to structure the markdown in the "description" field
✅ Return valid parseable JSON

BEGIN YOUR JSON OUTPUT NOW:`

		logger.Debug(context.Background(), "full prompt with template and final JSON reminder",
			"prompt_length", len(rendered),
			"prompt", rendered)
	}

	return rendered
}

// parseIssueResponse parses the Gemini JSON response.
func (s *GeminiIssueContentGenerator) parseIssueResponse(content string) (*models.IssueGenerationResult, error) {
	if content == "" {
		logger.Error(context.Background(), "received empty response from Gemini AI", nil)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "empty response from AI", nil)
	}

	if len(content) > 0 {
		preview := content
		if len(content) > 200 {
			preview = content[:200]
		}
		logger.Debug(context.Background(), "parsing Gemini response", "content_length", len(content), "content_preview", preview)
	}

	content = ExtractJSON(content)

	logger.Debug(context.Background(), "extracted JSON content",
		"content_length", len(content),
		"content", content)

	var jsonResult struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Labels      []string `json:"labels"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResult); err != nil {
		logger.Warn(context.Background(), "failed to unmarshal JSON, using fallback",
			"error", err.Error(),
			"content", content)
		return &models.IssueGenerationResult{
			Title:       "Generated Issue",
			Description: content,
			Labels:      []string{},
		}, nil
	}

	logger.Debug(context.Background(), "successfully parsed JSON",
		"title", jsonResult.Title,
		"description_length", len(jsonResult.Description),
		"labels_count", len(jsonResult.Labels))

	result := &models.IssueGenerationResult{
		Title:       strings.TrimSpace(jsonResult.Title),
		Description: strings.TrimSpace(jsonResult.Description),
		Labels:      s.cleanLabels(jsonResult.Labels),
	}

	if result.Title == "" {
		result.Title = "Generated Issue"
	}
	if result.Description == "" {
		result.Description = content
	}

	return result, nil
}

// cleanLabels cleans and validates labels, keeping only the allowed ones.
func (s *GeminiIssueContentGenerator) cleanLabels(labels []string) []string {
	allowedLabels := map[string]bool{
		"feature":  true,
		"fix":      true,
		"refactor": true,
		"docs":     true,
		"test":     true,
		"infra":    true,
	}

	cleaned := make([]string, 0)
	seen := make(map[string]bool)

	for _, label := range labels {
		trimmed := strings.TrimSpace(strings.ToLower(label))
		if trimmed != "" && allowedLabels[trimmed] && !seen[trimmed] {
			cleaned = append(cleaned, trimmed)
			seen[trimmed] = true
		}
	}

	return cleaned
}
