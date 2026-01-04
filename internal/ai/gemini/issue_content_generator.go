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
	"google.golang.org/genai"
)

type GeminiIssueContentGenerator struct {
	*GeminiProvider
	wrapper    *ai.CostAwareWrapper
	generateFn ai.GenerateFunc
	config     *config.Config
}

var _ ai.IssueContentGenerator = (*GeminiIssueContentGenerator)(nil)

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

func getIssueSchema() *genai.Schema {
	return &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"title", "description", "labels"},
		Properties: map[string]*genai.Schema{
			"title": {
				Type:        genai.TypeString,
				Description: "The title of the issue",
			},
			"description": {
				Type:        genai.TypeString,
				Description: "The body of the issue in markdown format",
			},
			"labels": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "List of labels (e.g. bug, feature, refactor, good first issue)",
			},
		},
	}
}

func (s *GeminiIssueContentGenerator) defaultGenerate(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
	schema := getIssueSchema()
	genConfig := GetGenerateConfig(mName, "application/json", schema)
	log := logger.FromContext(ctx)

	resp, err := s.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
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

	result.Labels = CleanLabels(result.Labels, request.AvailableLabels)
	result.Usage = usage

	log.Info("issue content generated successfully via gemini",
		"title", result.Title,
		"labels_count", len(result.Labels))

	return result, nil
}

// buildIssuePrompt builds the prompt to generate issue content.
func (s *GeminiIssueContentGenerator) buildIssuePrompt(request models.IssueGenerationRequest) string {
	if request.Description != "" && request.Diff == "" && request.Hint == "" &&
		request.Template == nil && len(request.ChangedFiles) == 0 && len(request.AvailableLabels) == 0 {
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

	if len(request.AvailableLabels) > 0 {
		rendered += fmt.Sprintf("\n\nAvailable Labels (Select ONLY from this list):\n%s", strings.Join(request.AvailableLabels, ", "))
	}

	return rendered
}

// parseIssueResponse parses the Gemini JSON response.
func (s *GeminiIssueContentGenerator) parseIssueResponse(content string) (*models.IssueGenerationResult, error) {
	if content == "" {
		logger.Error(context.Background(), "received empty response from Gemini AI", nil)
		return nil, domainErrors.NewAppError(domainErrors.TypeAI, "empty response from AI", nil)
	}

	content = strings.TrimSpace(content)

	if len(content) > 0 {
		preview := content
		if len(content) > 200 {
			preview = content[:200]
		}
		logger.Debug(context.Background(), "parsing Gemini response", "content_length", len(content), "content_preview", preview)
	}

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
		Labels:      jsonResult.Labels,
	}

	if result.Title == "" {
		result.Title = "Generated Issue"
	}

	return result, nil
}
