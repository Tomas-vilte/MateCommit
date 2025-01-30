package handler

import (
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"strings"
)

type SuggestionHandler struct {
	gitService ports.GitService
	t          *i18n.Translations
}

func NewSuggestionHandler(git ports.GitService, t *i18n.Translations) *SuggestionHandler {
	return &SuggestionHandler{
		gitService: git,
		t:          t,
	}
}

func (h *SuggestionHandler) HandleSuggestions(suggestions []models.CommitSuggestion) error {
	h.displaySuggestions(suggestions)
	return h.handleCommitSelection(suggestions)
}

func (h *SuggestionHandler) displaySuggestions(suggestions []models.CommitSuggestion) {
	fmt.Printf("%s\n", h.t.GetMessage("commit.header_message", 0, nil))

	for i, suggestion := range suggestions {
		fmt.Printf("\n=========[ Sugerencia %d ]=========\n", i+1)

		// Mostrar análisis de código
		fmt.Println("\n📊 Análisis de Código:")
		fmt.Printf("- Resumen de Cambios: %s\n", suggestion.CodeAnalysis.ChangesOverview)
		fmt.Printf("- Propósito Principal: %s\n", suggestion.CodeAnalysis.PrimaryPurpose)
		fmt.Printf("- Impacto Técnico: %s\n", suggestion.CodeAnalysis.TechnicalImpact)

		// Mostrar sugerencia de commit
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Commit: %s\n", suggestion.CommitTitle)

		// Mostrar archivos modificados
		fmt.Println("📄 Archivos modificados:")
		for _, file := range suggestion.Files {
			fmt.Printf("   - %s\n", file)
		}
		fmt.Printf("Explicación: %s\n", suggestion.Explanation)

		// Mostrar análisis de requerimientos
		fmt.Println("\n🎯 Análisis de Requerimientos:")
		fmt.Printf("⚠️  Estado de los Criterios: %s\n", h.getCriteriaStatusText(suggestion.RequirementsAnalysis.CriteriaStatus))

		if len(suggestion.RequirementsAnalysis.MissingCriteria) > 0 {
			fmt.Println("\n❌ Criterios Faltantes:")
			for _, criteria := range suggestion.RequirementsAnalysis.MissingCriteria {
				fmt.Printf("   - %s\n", criteria)
			}
		}

		if len(suggestion.RequirementsAnalysis.ImprovementSuggestions) > 0 {
			fmt.Println("\n💡 Sugerencias de Mejora:")
			for _, improvement := range suggestion.RequirementsAnalysis.ImprovementSuggestions {
				fmt.Printf("   - %s\n", improvement)
			}
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	}

	fmt.Println(h.t.GetMessage("commit.select_option_prompt", 0, nil))
	fmt.Println(h.t.GetMessage("commit.option_commit", 0, nil))
	fmt.Println(h.t.GetMessage("commit.option_exit", 0, nil))
}

func (h *SuggestionHandler) getCriteriaStatusText(status models.CriteriaStatus) string {
	var msg string
	switch status {
	case models.CriteriaFullyMet:
		msg = h.t.GetMessage("gemini_service.criteria_fully_met_prefix", 0, nil)
		return msg
	case models.CriteriaPartiallyMet:
		msg = h.t.GetMessage("gemini_service.criteria_partially_met_prefix", 0, nil)
		return msg
	case models.CriteriaNotMet:
		msg = h.t.GetMessage("gemini_service.criteria_not_met_prefix", 0, nil)
		return msg
	default:
		msg = h.t.GetMessage("gemini_service.criteria_unknown_prefix", 0, nil)
		return msg
	}
}

func (h *SuggestionHandler) handleCommitSelection(suggestions []models.CommitSuggestion) error {
	var selection int
	fmt.Print(h.t.GetMessage("commit.prompt_selection", 0, nil))
	if _, err := fmt.Scan(&selection); err != nil {
		msg := h.t.GetMessage("commit.error_reading_selection", 0, map[string]interface{}{"Error": err})
		return fmt.Errorf("%s", msg)
	}

	if selection == 0 {
		fmt.Println(h.t.GetMessage("commit.operation_canceled", 0, nil))
		return nil
	}

	if selection < 1 || selection > len(suggestions) {
		msg := h.t.GetMessage("commit.invalid_selection", 0, map[string]interface{}{"Number": len(suggestions)})
		return fmt.Errorf("%s", msg)
	}

	return h.processCommit(suggestions[selection-1], h.gitService)
}

func (h *SuggestionHandler) processCommit(suggestion models.CommitSuggestion, gitService ports.GitService) error {
	commitTitle := strings.TrimSpace(strings.TrimPrefix(suggestion.CommitTitle, "Commit: "))

	for _, file := range suggestion.Files {
		if err := gitService.AddFileToStaging(file); err != nil {
			msg := h.t.GetMessage("commit.error_add_file_staging", 0, map[string]interface{}{
				"File":  file,
				"Error": err,
			})
			return fmt.Errorf("%s", msg)
		}
		msg := h.t.GetMessage("commit.add_file_to_staging", 0, map[string]interface{}{"File": file})
		fmt.Printf("%s", msg)
	}

	if err := gitService.CreateCommit(commitTitle); err != nil {
		msg := h.t.GetMessage("commit.error_creating_commit", 0, map[string]interface{}{
			"Commit": commitTitle,
			"Error":  err,
		})
		return fmt.Errorf("%s", msg)
	}

	fmt.Printf("%s\n", h.t.GetMessage("commit.commit_successful", 0, map[string]interface{}{"CommitTitle": commitTitle}))
	return nil
}
