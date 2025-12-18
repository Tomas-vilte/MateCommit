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

type GeminiIssueContentGenerator struct {
	client *genai.Client
	config *config.Config
	trans  *i18n.Translations
}

var _ ports.IssueContentGenerator = (*GeminiIssueContentGenerator)(nil)

func NewGeminiIssueContentGenerator(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiIssueContentGenerator, error) {
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

	return &GeminiIssueContentGenerator{
		client: client,
		config: cfg,
		trans:  trans,
	}, nil
}

// GenerateIssueContent genera contenido de issue usando Gemini AI.
func (s *GeminiIssueContentGenerator) GenerateIssueContent(ctx context.Context, request models.IssueGenerationRequest) (*models.IssueGenerationResult, error) {
	prompt := s.buildIssuePrompt(request)
	modelName := string(s.config.AIConfig.Models[config.AIGemini])

	genConfig := &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.3),
		MaxOutputTokens:  int32(10000),
		ResponseMIMEType: "application/json",
		MediaResolution:  genai.MediaResolutionHigh,
	}

	resp, err := s.client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), genConfig)
	if err != nil {
		return nil, fmt.Errorf("error al generar contenido de la issue: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("ningún contenido generado por IA")
	}

	result, err := s.parseIssueResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("error al parsear la respuesta de la IA: %w", err)
	}

	result.Usage = extractUsage(resp)

	return result, nil
}

// buildIssuePrompt construye el prompt para generar contenido de issue.
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

	template := ai.GetIssuePromptTemplate(request.Language)
	return fmt.Sprintf(template, sb.String())
}

// parseIssueResponse parsea la respuesta JSON de Gemini.
func (s *GeminiIssueContentGenerator) parseIssueResponse(resp *genai.GenerateContentResponse) (*models.IssueGenerationResult, error) {
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from AI")
	}

	content := formatResponse(resp)

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

// cleanLabels limpia y valida las labels, mantiene solo las permitidas.
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
