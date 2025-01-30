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
	default:
		promptTemplate = promptTemplateEN
	}

	// Determinar la sección de análisis de requerimientos
	var reqAnalysisTemplate string
	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		if locale == "es" {
			reqAnalysisTemplate = `⚠️ Estado de los Criterios: [completamente_cumplidos/parcialmente_cumplidos/no_cumplidos]
			❌ Criterios Faltantes:
			   - [Lista detallada de criterios no implementados o parcialmente implementados]
			💡 Sugerencias de Mejora:
			   - [Lista de mejoras específicas para cumplir los criterios]`
		} else {
			reqAnalysisTemplate = `⚠️ Criteria Status: [fully_met/partially_met/not_met]
			❌ Missing Criteria:
			   - [Detailed list of non-implemented or partially implemented criteria]
			💡 Improvement Suggestions:
			   - [List of specific improvements to meet criteria]`
		}
	} else {
		if locale == "es" {
			reqAnalysisTemplate = `💭 Análisis Técnico:
			- Calidad del Código: [Evaluación de la calidad y claridad del código]
			- Mejores Prácticas: [Análisis de adherencia a mejores prácticas]
			💡 Sugerencias de Mejora:
			   - [Lista de mejoras técnicas recomendadas]`
		} else {
			reqAnalysisTemplate = `💭 Technical Analysis:
			- Code Quality: [Evaluation of code quality and clarity]
			- Best Practices: [Analysis of adherence to best practices]
			💡 Improvement Suggestions:
			   - [List of recommended technical improvements]`
		}
	}

	// Si hay información del ticket, la agregamos al prompt
	ticketInfo := ""
	if info.TicketInfo != nil && info.TicketInfo.TicketTitle != "" {
		ticketInfo = fmt.Sprintf("\nTicket Title: %s\nTicket Description: %s\nAcceptance Criteria: %s",
			info.TicketInfo.TicketTitle, info.TicketInfo.TitleDesc, strings.Join(info.TicketInfo.Criteria, ", "))
	}

	return fmt.Sprintf(promptTemplate,
		count,
		formatChanges(info.Files),
		info.Diff,
		reqAnalysisTemplate,
		ticketInfo,
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

	suggestion := &models.CommitSuggestion{
		CodeAnalysis:         models.CodeAnalysis{},
		RequirementsAnalysis: models.RequirementsAnalysis{},
	}

	// Extraer el título del commit
	for _, line := range lines {
		if strings.HasPrefix(line, "Commit:") {
			suggestion.CommitTitle = strings.TrimSpace(strings.TrimPrefix(line, "Commit:"))
			break
		}
	}

	// Extraer los archivos modificados
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
				suggestion.Files = append(suggestion.Files, file)
			} else if trimmedLine == "" || strings.HasPrefix(trimmedLine, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil)) {
				collectingFiles = false
			}
		}
	}

	// Extraer la explicación
	var explanation strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil)) {
			explanation.WriteString(strings.TrimSpace(strings.TrimPrefix(line, s.trans.GetMessage("gemini_service.explanation_prefix", 0, nil))))
			explanation.WriteString("\n")
		}
	}
	suggestion.Explanation = strings.TrimSpace(explanation.String())

	// Extraer el análisis de código
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

	// Extraer el análisis de requisitos
	var (
		collectingMissingCriteria bool
		collectingImprovements    bool
	)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Procesar estado de criterios
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

		// Procesar criterios faltantes
		if strings.HasPrefix(trimmedLine, "❌") {
			collectingMissingCriteria = true
			collectingImprovements = false
			continue
		}

		// Procesar sugerencias de mejora
		if strings.HasPrefix(trimmedLine, "💡") {
			collectingMissingCriteria = false
			collectingImprovements = true
			continue
		}

		// Recolectar criterios faltantes
		if collectingMissingCriteria && strings.HasPrefix(trimmedLine, "-") {
			criteria := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "-"))
			suggestion.RequirementsAnalysis.MissingCriteria = append(
				suggestion.RequirementsAnalysis.MissingCriteria,
				criteria,
			)
		}

		// Recolectar sugerencias de mejora
		if collectingImprovements && strings.HasPrefix(trimmedLine, "-") {
			improvement := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "-"))
			suggestion.RequirementsAnalysis.ImprovementSuggestions = append(
				suggestion.RequirementsAnalysis.ImprovementSuggestions,
				improvement,
			)
		}

		// Detener la recolección si encontramos una línea vacía o un nuevo encabezado
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "📊") ||
			strings.HasPrefix(trimmedLine, "📝") {
			collectingMissingCriteria = false
			collectingImprovements = false
		}
	}

	return suggestion
}
