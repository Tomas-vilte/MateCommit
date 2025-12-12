package gemini

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

type GeminiService struct {
	client *genai.Client
	config *config.Config
	trans  *i18n.Translations
}

func NewGeminiService(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiService, error) {
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

	return &GeminiService{
		client: client,
		config: cfg,
		trans:  trans,
	}, nil
}

func (s *GeminiService) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	if count <= 0 {
		msg := s.trans.GetMessage("error_invalid_suggestion_count", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	if len(info.Files) == 0 {
		msg := s.trans.GetMessage("error_no_files", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	prompt := s.generatePrompt(s.config.Language, info, count)
	modelName := string(s.config.AIConfig.Models[config.AIGemini])

	resp, err := s.client.Models.GenerateContent(ctx, modelName, genai.Text(prompt), nil)
	if err != nil {
		msg := s.trans.GetMessage("error_generating_content", 0, map[string]interface{}{
			"Error": err.Error(),
		})
		return nil, fmt.Errorf("%s", msg)
	}

	suggestions := s.parseSuggestions(resp)
	if len(suggestions) == 0 {
		msg := s.trans.GetMessage("error_no_suggestions", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		suggestions = s.ensureIssueReference(suggestions, info.IssueInfo.Number)
	}

	return suggestions, nil
}

func (s *GeminiService) generatePrompt(locale string, info models.CommitInfo, count int) string {
	promptTemplate := ai.GetCommitPromptTemplate(locale, info.TicketInfo != nil && info.TicketInfo.TicketTitle != "")

	ticketInfo := ""
	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		ticketInfo = fmt.Sprintf("\nTicket Title: %s\nTicket Description: %s\nAcceptance Criteria: %s",
			info.TicketInfo.TicketTitle,
			info.TicketInfo.TitleDesc,
			strings.Join(info.TicketInfo.Criteria, ", "))
	}

	issueInstructions := ""
	if info.IssueInfo != nil && info.IssueInfo.Number > 0 {
		num := info.IssueInfo.Number
		issueInstructions = fmt.Sprintf(ai.GetIssueReferenceInstructions(locale), num, num, num, num, num, num, num, num)
	} else {
		issueInstructions = "No hay issue asociado, no incluyas referencias de issues en el título."
	}

	return fmt.Sprintf(promptTemplate,
		count,
		count,
		formatChanges(info.Files),
		info.Diff,
		ticketInfo,
		issueInstructions,
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

func (s *GeminiService) getSuggestionDelimiter() string {
	return s.trans.GetMessage("gemini_service.suggestion_prefix", 0, nil)
}

func (s *GeminiService) parseSuggestions(resp *genai.GenerateContentResponse) []models.CommitSuggestion {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return nil
	}

	delimiter := s.getSuggestionDelimiter()
	re := regexp.MustCompile(delimiter)
	parts := re.Split(responseText, -1)
	suggestions := make([]models.CommitSuggestion, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		suggestion := s.parseSuggestionPart(part)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

func (s *GeminiService) parseSuggestionPart(part string) *models.CommitSuggestion {
	lines := strings.Split(strings.TrimSpace(part), "\n")
	if len(lines) < 3 {
		return nil
	}

	suggestion := &models.CommitSuggestion{
		CodeAnalysis:         models.CodeAnalysis{},
		RequirementsAnalysis: models.RequirementsAnalysis{},
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "Commit:") {
			suggestion.CommitTitle = strings.TrimSpace(strings.TrimPrefix(line, "Commit:"))
			break
		}
	}

	var collectingFiles bool
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "📄") {
			collectingFiles = true
			continue
		}

		if collectingFiles {
			if strings.HasPrefix(trimmedLine, "-") {
				file := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "-"))
				if strings.Contains(file, "->") {
					parts := strings.Split(file, "->")
					if len(parts) > 1 {
						file = strings.TrimSpace(parts[len(parts)-1])
					}
				}
				suggestion.Files = append(suggestion.Files, file)
			} else if trimmedLine == "" || strings.HasPrefix(trimmedLine, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil)) {
				collectingFiles = false
			}
		}
	}

	var explanation strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil)) {
			explanation.WriteString(strings.TrimSpace(strings.TrimPrefix(line, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil))))
			explanation.WriteString("\n")
		}
	}
	suggestion.Explanation = strings.TrimSpace(explanation.String())

	for i, line := range lines {
		if strings.HasPrefix(line, s.trans.GetMessage("gemini_service.code_analysis_prefix", 0, nil)) {
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], s.trans.GetMessage("gemini_service.changes_overview_prefix", 0, nil)) {
				suggestion.CodeAnalysis.ChangesOverview = strings.TrimSpace(strings.TrimPrefix(lines[i+1], s.trans.GetMessage("gemini_service.changes_overview_prefix", 0, nil)))
			}
			if i+2 < len(lines) && strings.HasPrefix(lines[i+2], s.trans.GetMessage("gemini_service.primary_purpose_prefix", 0, nil)) {
				suggestion.CodeAnalysis.PrimaryPurpose = strings.TrimSpace(strings.TrimPrefix(lines[i+2], s.trans.GetMessage("gemini_service.primary_purpose_prefix", 0, nil)))
			}
			if i+3 < len(lines) && strings.HasPrefix(lines[i+3], s.trans.GetMessage("gemini_service.technical_impact_prefix", 0, nil)) {
				suggestion.CodeAnalysis.TechnicalImpact = strings.TrimSpace(strings.TrimPrefix(lines[i+3], s.trans.GetMessage("gemini_service.technical_impact_prefix", 0, nil)))
			}
			break
		}
	}

	var (
		collectingMissingCriteria bool
		collectingImprovements    bool
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "⚠️") {
			switch {
			case strings.Contains(trimmedLine, s.trans.GetMessage("gemini_service.criteria_fully_met_prefix", 0, nil)):
				suggestion.RequirementsAnalysis.CriteriaStatus = models.CriteriaFullyMet
			case strings.Contains(trimmedLine, s.trans.GetMessage("gemini_service.criteria_partially_met_prefix", 0, nil)):
				suggestion.RequirementsAnalysis.CriteriaStatus = models.CriteriaPartiallyMet
			default:
				suggestion.RequirementsAnalysis.CriteriaStatus = models.CriteriaNotMet
			}
			continue
		}

		if strings.HasPrefix(trimmedLine, "❌") {
			collectingMissingCriteria = true
			collectingImprovements = false
			continue
		}

		if strings.HasPrefix(trimmedLine, "💡") {
			collectingMissingCriteria = false
			collectingImprovements = true
			continue
		}

		if collectingMissingCriteria && strings.HasPrefix(trimmedLine, "-") {
			criteria := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "-"))
			suggestion.RequirementsAnalysis.MissingCriteria = append(
				suggestion.RequirementsAnalysis.MissingCriteria,
				criteria,
			)
		}

		if collectingImprovements && strings.HasPrefix(trimmedLine, "-") {
			improvement := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "-"))
			suggestion.RequirementsAnalysis.ImprovementSuggestions = append(
				suggestion.RequirementsAnalysis.ImprovementSuggestions,
				improvement,
			)
		}

		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "📊") ||
			strings.HasPrefix(trimmedLine, "📝") {
			collectingMissingCriteria = false
			collectingImprovements = false
		}
	}

	return suggestion
}

// ensureIssueReference asegura que todas las sugerencias incluyan la referencia al issue correcta
func (s *GeminiService) ensureIssueReference(suggestions []models.CommitSuggestion, issueNumber int) []models.CommitSuggestion {
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
