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
	genConfig := GetGenerateConfig(mName, "application/json")

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
		responseText = formatResponse(geminiResp)
	} else if str, ok := resp.(string); ok {
		responseText = str
	}

	if responseText == "" {
		log.Error("empty response from gemini AI")
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "empty response from AI", nil)
	}

	log.Debug("gemini response received",
		"response_length", len(responseText))

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

	return rendered
}

// parseIssueResponse parses the Gemini JSON response.
func (s *GeminiIssueContentGenerator) parseIssueResponse(content string) (*models.IssueGenerationResult, error) {
	if content == "" {
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "empty response from AI", nil)
	}

	content = ExtractJSON(content)

	var jsonResult struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Labels      []string `json:"labels"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResult); err != nil {
		return &models.IssueGenerationResult{
			Title:       "Generated Issue",
			Description: content,
			Labels:      []string{},
		}, nil
	}

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
