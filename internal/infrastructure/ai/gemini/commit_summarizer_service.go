package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

var _ ports.CommitSummarizer = (*GeminiCommitSummarizer)(nil)

type GeminiCommitSummarizer struct {
	*GeminiProvider
	wrapper *ai.CostAwareWrapper
	config  *config.Config
	trans   *i18n.Translations
}

type (
	CommitSuggestionJSON struct {
		Title        string            `json:"title"`
		Desc         string            `json:"desc"`
		Files        []string          `json:"files"`
		Analysis     *CodeAnalysisJSON `json:"analysis,omitempty"`
		Requirements *RequirementsJSON `json:"requirements,omitempty"`
	}

	CodeAnalysisJSON struct {
		OverView string `json:"overview"`
		Purpose  string `json:"purpose"`
		Impact   string `json:"impact"`
	}

	RequirementsJSON struct {
		Status      string   `json:"status"`
		Missing     []string `json:"missing"`
		Suggestions []string `json:"suggestions"`
	}
)

func NewGeminiCommitSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiCommitSummarizer, error) {
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

	service := &GeminiCommitSummarizer{
		GeminiProvider: NewGeminiProvider(client, modelName),
		config:         cfg,
		trans:          trans,
	}

	wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
		Provider:              service,
		BudgetDaily:           budgetDaily,
		Trans:                 trans,
		EstimatedOutputTokens: 800,
	})
	if err != nil {
		return nil, fmt.Errorf("error creando wrapper: %w", err)
	}

	service.wrapper = wrapper

	return service, nil
}

func (s *GeminiCommitSummarizer) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	if count <= 0 {
		msg := s.trans.GetMessage("error_invalid_suggestion_count", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	if len(info.Files) == 0 {
		msg := s.trans.GetMessage("error_no_files", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	prompt := s.generatePrompt(s.config.Language, info, count)

	generateFn := func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
		genConfig := GetGenerateConfig(mName, "application/json")

		resp, err := s.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
		if err != nil {
			return nil, nil, err
		}

		usage := extractUsage(resp)
		return resp, usage, nil
	}

	resp, usage, err := s.wrapper.WrapGenerate(ctx, "suggest-commits", prompt, generateFn)
	if err != nil {
		msg := s.trans.GetMessage("error_generating_content", 0, map[string]interface{}{
			"Error": err.Error(),
		})
		return nil, fmt.Errorf("%s", msg)
	}

	geminiResp := resp.(*genai.GenerateContentResponse)
	suggestions, err := s.parseSuggestionsJSON(geminiResp)
	if err != nil {
		rawResp := formatResponse(geminiResp)
		respLen := len(rawResp)
		preview := rawResp
		if respLen > 500 {
			preview = rawResp[:500] + "..."
		}
		return nil, fmt.Errorf("error al parsear respuesta JSON de la IA (longitud: %d caracteres): %w\nPrimeros caracteres: %s",
			respLen, err, preview)
	}
	if len(suggestions) == 0 {
		return nil, fmt.Errorf("la IA no generó ninguna sugerencia")
	}
	for i := range suggestions {
		suggestions[i].Usage = usage
	}
	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		suggestions = s.ensureIssueReference(suggestions, info.IssueInfo.Number)
	}
	return suggestions, nil
}

func (s *GeminiCommitSummarizer) parseSuggestionsJSON(resp *genai.GenerateContentResponse) ([]models.CommitSuggestion, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("respuesta vacía de la IA")
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return nil, fmt.Errorf("texto de respuesta vacío de la IA")
	}

	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var jsonSuggestions []CommitSuggestionJSON
	if err := json.Unmarshal([]byte(responseText), &jsonSuggestions); err != nil {
		return nil, fmt.Errorf("error al parsear JSON: %w", err)
	}

	suggestions := make([]models.CommitSuggestion, 0, len(jsonSuggestions))
	for _, js := range jsonSuggestions {
		suggestion := models.CommitSuggestion{
			CommitTitle: js.Title,
			Explanation: js.Desc,
			Files:       js.Files,
		}

		if js.Analysis != nil {
			suggestion.CodeAnalysis = models.CodeAnalysis{
				ChangesOverview: js.Analysis.OverView,
				PrimaryPurpose:  js.Analysis.Purpose,
				TechnicalImpact: js.Analysis.Impact,
			}
		}

		if js.Requirements != nil {
			suggestion.RequirementsAnalysis = models.RequirementsAnalysis{
				CriteriaStatus:         models.CriteriaStatus(js.Requirements.Status),
				MissingCriteria:        js.Requirements.Missing,
				ImprovementSuggestions: js.Requirements.Suggestions,
			}
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

func (s *GeminiCommitSummarizer) generatePrompt(locale string, info models.CommitInfo, count int) string {
	promptTemplate := ai.GetCommitPromptTemplate(locale, info.TicketInfo != nil &&
		info.TicketInfo.TicketTitle != "")

	filesFormatted := formatChanges(info.Files)

	diffFormatted := fmt.Sprintf("```diff\n%s\n```", info.Diff)

	ticketInfo := ""
	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		var titleLabel, descLabel, criteriaLabel string
		if locale == "es" {
			titleLabel = "**Título:**"
			descLabel = "**Descripción:**"
			criteriaLabel = "**Criterios de Aceptación:**"
		} else {
			titleLabel = "**Title:**"
			descLabel = "**Description:**"
			criteriaLabel = "**Acceptance Criteria:**"
		}

		ticketInfo = fmt.Sprintf(`%s %s
    %s %s
    %s
    %s`,
			titleLabel, info.TicketInfo.TicketTitle,
			descLabel, info.TicketInfo.TitleDesc,
			criteriaLabel,
			formatCriteria(info.TicketInfo.Criteria))
	}

	issueInstructions := ""
	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		num := info.IssueInfo.Number
		issueInstructions = fmt.Sprintf(ai.GetIssueReferenceInstructions(locale), num, num, num, num, num,
			num, num, num)
	} else {
		issueInstructions = ai.GetNoIssueReferenceInstruction(locale)
	}

	technicalAnalysis := ""
	if info.TicketInfo == nil || info.TicketInfo.TicketTitle == "" {
		technicalAnalysis = ai.GetTechnicalAnalysisInstruction(locale)
	}

	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		return fmt.Sprintf(promptTemplate,
			count,
			filesFormatted,
			diffFormatted,
			ticketInfo,
			issueInstructions,
			info.RecentHistory,
			count,
		)
	}

	return fmt.Sprintf(promptTemplate,
		count,
		filesFormatted,
		diffFormatted,
		issueInstructions,
		info.RecentHistory,
		technicalAnalysis,
		count,
	)
}

func formatChanges(files []string) string {
	if len(files) == 0 {
		return ""
	}
	formattedFiles := make([]string, len(files))
	for i, file := range files {
		formattedFiles[i] = fmt.Sprintf("- %s", file)
	}
	return strings.Join(formattedFiles, "\n")
}

func formatCriteria(criteria []string) string {
	if len(criteria) == 0 {
		return ""
	}
	formattedCriteria := make([]string, len(criteria))
	for i, criterion := range criteria {
		formattedCriteria[i] = fmt.Sprintf("  - %s", criterion)
	}
	return strings.Join(formattedCriteria, "\n")
}

// formatResponse formatea la respuesta de la API de Gemini en una cadena.
func formatResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}

	var formattedContent strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					formattedContent.WriteString(part.Text)
				}
			}
		}
	}
	return formattedContent.String()
}

// ensureIssueReference asegura que todas las sugerencias incluyan la referencia al issue correcta
func (s *GeminiCommitSummarizer) ensureIssueReference(suggestions []models.CommitSuggestion, issueNumber int) []models.CommitSuggestion {
	issuePattern := regexp.MustCompile(`\(#\d+\)`)

	for i := range suggestions {
		title := suggestions[i].CommitTitle
		title = strings.TrimSpace(title)

		if strings.Contains(title, fmt.Sprintf("(#%d)", issueNumber)) ||
			strings.Contains(title, fmt.Sprintf("fixes #%d", issueNumber)) ||
			strings.Contains(title, fmt.Sprintf("closes #%d", issueNumber)) {
			continue
		}

		if issuePattern.MatchString(title) {
			title = issuePattern.ReplaceAllString(title, fmt.Sprintf("(#%d)", issueNumber))
			suggestions[i].CommitTitle = title
			continue
		}

		suggestions[i].CommitTitle = fmt.Sprintf("%s (#%d)", title, issueNumber)
	}

	return suggestions
}
