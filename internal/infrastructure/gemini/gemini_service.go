package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
)

type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config *config.Config
	trans  *i18n.Translations
}

func NewGeminiService(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiService, error) {
	if cfg.GeminiAPIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	return &GeminiService{
		client: client,
		model:  model,
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
	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
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

	return suggestions, nil
}

func (s *GeminiService) generatePrompt(locale string, info models.CommitInfo, count int) string {
	var promptTemplate string
	switch locale {
	case "es":
		promptTemplate = promptTemplateES
	case "en":
		promptTemplate = promptTemplateEN
	default:
		promptTemplate = promptTemplateEN
	}

	if s.config.UseEmoji {
		promptTemplate = strings.Replace(promptTemplate, "Commit: [type]: [message]\n", "Commit: ✨ [type]: [message]\n", 1)
		promptTemplate = strings.Replace(promptTemplate, "Commit: fix:", "Commit: 🐛 fix:", 1)
		promptTemplate = strings.Replace(promptTemplate, "Commit: docs:", "Commit: 📚 docs:", 1)
	}

	// Si hay información del ticket, la agregamos al prompt
	ticketInfo := ""
	if info.TicketTitle != "" {
		ticketInfo = fmt.Sprintf("\nTicket Title: %s\nTicket Description: %s\nAcceptance Criteria: %s",
			info.TicketTitle, info.TicketDesc, strings.Join(info.Criteria, ", "))
	}

	return fmt.Sprintf(promptTemplate,
		count,
		count,
		ticketInfo,
		formatChanges(info.Files),
		info.Diff,
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

func formatResponse(resp *genai.GenerateContentResponse) string {
	if resp == nil || resp.Candidates == nil {
		return ""
	}

	var formattedContent strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				formattedContent.WriteString(fmt.Sprintf("%v", part))
			}
		}
	}
	return formattedContent.String()
}

func (s *GeminiService) getSuggestionDelimiter() string {
	return s.trans.GetMessage("suggestion_delimiter", 0, nil)
}

func (s *GeminiService) getFilesPrefix() string {
	return s.trans.GetMessage("files_prefix", 0, nil)
}

func (s *GeminiService) parseSuggestions(resp *genai.GenerateContentResponse) []models.CommitSuggestion {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}

	responseText := formatResponse(resp)
	if responseText == "" {
		return nil
	}

	suggestions := make([]models.CommitSuggestion, 0)
	delimiter := s.getSuggestionDelimiter()
	parts := strings.Split(responseText, delimiter)

	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
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

	suggestion := &models.CommitSuggestion{}

	// Extraer el título del commit
	suggestion.CommitTitle = strings.TrimSpace(lines[1])

	// Extraer los archivos modificados
	prefixFiles := s.getFilesPrefix()
	if filesPart := strings.TrimPrefix(lines[2], prefixFiles); filesPart != "" {
		files := strings.Split(filesPart, ",")
		suggestion.Files = make([]string, 0, len(files))
		for _, file := range files {
			if trimmed := strings.TrimSpace(file); trimmed != "" {
				suggestion.Files = append(suggestion.Files, trimmed)
			}
		}
	}

	// Extraer la explicación
	var explanation strings.Builder
	for i := 3; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], s.trans.GetMessage("commit.criteria_status_prefix", 0, nil)) ||
			strings.HasPrefix(lines[i], s.trans.GetMessage("commit.missing_criteria_prefix", 0, nil)) ||
			strings.HasPrefix(lines[i], s.trans.GetMessage("commit.improvement_suggestions_prefix", 0, nil)) {
			break // Detener si encontramos los campos adicionales
		}
		explanation.WriteString(lines[i])
		explanation.WriteString("\n")
	}
	suggestion.Explanation = strings.TrimSpace(explanation.String())

	// Extraer el estado de los criterios
	criteriaStatusPrefix := s.trans.GetMessage("commit.criteria_status_prefix", 0, nil)
	for _, line := range lines {
		if strings.HasPrefix(line, criteriaStatusPrefix) {
			criteriaStatus := strings.TrimSpace(strings.TrimPrefix(line, criteriaStatusPrefix))
			switch criteriaStatus {
			case s.trans.GetMessage("commit.criteria_fully_met_prefix", 0, nil):
				suggestion.CriteriaStatus = models.CriteriaFullyMet
			case s.trans.GetMessage("commit.criteria_partially_met_prefix", 0, nil):
				suggestion.CriteriaStatus = models.CriteriaPartiallyMet
			case s.trans.GetMessage("commit.criteria_not_met_prefix", 0, nil):
				suggestion.CriteriaStatus = models.CriteriaNotMet
			default:
				suggestion.CriteriaStatus = models.CriteriaNotMet
			}
		}
	}

	// Extraer los criterios faltantes
	missingCriteriaPrefix := s.trans.GetMessage("commit.missing_criteria_prefix", 0, nil)
	for _, line := range lines {
		if strings.HasPrefix(line, missingCriteriaPrefix) {
			missingCriteria := strings.TrimSpace(strings.TrimPrefix(line, missingCriteriaPrefix))
			if missingCriteria != s.trans.GetMessage("commit.missing_criteria_none", 0, nil) {
				suggestion.MissingCriteria = strings.Split(missingCriteria, ",")
			}
		}
	}

	// Extraer las sugerencias de mejora
	improvementSuggestionsPrefix := s.trans.GetMessage("commit.improvement_suggestions_prefix", 0, nil)
	for _, line := range lines {
		if strings.HasPrefix(line, improvementSuggestionsPrefix) {
			improvementSuggestions := strings.TrimSpace(strings.TrimPrefix(line, improvementSuggestionsPrefix))
			if improvementSuggestions != s.trans.GetMessage("commit.improvement_suggestions_none", 0, nil) {
				suggestion.ImprovementSuggestions = strings.Split(improvementSuggestions, ",")
			}
		}
	}

	return suggestion
}
