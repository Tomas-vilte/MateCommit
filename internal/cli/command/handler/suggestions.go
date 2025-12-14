package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/ui"
	"github.com/fatih/color"
)

var _ ports.CommitHandler = (*SuggestionHandler)(nil)

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

func (h *SuggestionHandler) HandleSuggestions(ctx context.Context, suggestions []models.CommitSuggestion) error {
	h.displaySuggestions(suggestions)
	return h.handleCommitSelection(ctx, suggestions)
}

func (h *SuggestionHandler) displaySuggestions(suggestions []models.CommitSuggestion) {
	titleColor := color.New(color.FgCyan, color.Bold)
	sectionColor := color.New(color.FgYellow, color.Bold)
	fileColor := color.New(color.FgHiBlack)

	fmt.Printf("\n%s\n", h.t.GetMessage("commit.header_message", 0, nil))

	for i, suggestion := range suggestions {
		// Separador visual
		separator := color.New(color.FgCyan).Sprint("━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("\n%s\n", separator)

		// Header con número
		suggestionHeader := color.New(color.FgMagenta, color.Bold).Sprintf("📝 Sugerencia #%d", i+1)
		fmt.Printf("%s\n\n", suggestionHeader)

		// Análisis de Código
		sectionColor.Println("📊 Análisis de Código:")
		ui.PrintKeyValue("Resumen de Cambios", suggestion.CodeAnalysis.ChangesOverview)
		ui.PrintKeyValue("Propósito Principal", suggestion.CodeAnalysis.PrimaryPurpose)
		ui.PrintKeyValue("Impacto Técnico", suggestion.CodeAnalysis.TechnicalImpact)

		fmt.Println()
		fmt.Printf("%s\n", separator)

		// Commit title destacado
		fmt.Printf("%s %s\n\n",
			color.New(color.FgGreen, color.Bold).Sprint("✓ Commit:"),
			titleColor.Sprint(suggestion.CommitTitle),
		)

		// Archivos modificados con icono
		sectionColor.Println("📄 Archivos modificados:")
		for _, file := range suggestion.Files {
			fmt.Printf("   %s %s\n", color.CyanString("•"), fileColor.Sprint(file))
		}

		// Explicación
		fmt.Printf("\n%s %s\n",
			sectionColor.Sprint("💬 Explicación:"),
			suggestion.Explanation,
		)
		fmt.Println() // Espacio extra

		if suggestion.RequirementsAnalysis.CriteriaStatus != "" {
			h.displayRequirementsAnalysis(suggestion.RequirementsAnalysis)
		} else {
			h.displayTechnicalAnalysis(suggestion.RequirementsAnalysis)
		}

		fmt.Printf("%s\n", separator)
	}

	// Opciones de selección con estilo
	fmt.Println()
	ui.PrintInfo(h.t.GetMessage("ui_selection.select_option", 0, nil))
	fmt.Printf("   %s %s\n", color.GreenString("1-%d:", len(suggestions)), h.t.GetMessage("ui_selection.select_suggestion_range", 0, nil))
	fmt.Printf("   %s %s\n", color.RedString("0:"), h.t.GetMessage("ui_selection.cancel_operation", 0, nil))
	fmt.Println()
}

func (h *SuggestionHandler) displayRequirementsAnalysis(analysis models.RequirementsAnalysis) {
	reqColor := color.New(color.FgMagenta, color.Bold)

	fmt.Printf("%s\n", reqColor.Sprint("🎯 Análisis de Requerimientos:"))

	// Status con emoji según el estado
	statusText := h.getCriteriaStatusText(analysis.CriteriaStatus)
	statusEmoji := h.getCriteriaStatusEmoji(analysis.CriteriaStatus)
	statusColor := h.getCriteriaStatusColor(analysis.CriteriaStatus)

	fmt.Printf("   %s %s %s\n", statusEmoji, color.New(color.FgHiBlack).Sprint("Estado:"),
		statusColor.Sprint(statusText))

	if len(analysis.MissingCriteria) > 0 {
		fmt.Printf("\n   %s %s\n", color.RedString("❌"), color.New(color.FgRed,
			color.Bold).Sprint("Criterios Faltantes:"))
		for _, criteria := range analysis.MissingCriteria {
			fmt.Printf("      %s %s\n", color.RedString("•"), criteria)
		}
	}

	if len(analysis.ImprovementSuggestions) > 0 {
		fmt.Printf("\n   %s %s\n", color.YellowString("💡"), color.New(color.FgYellow,
			color.Bold).Sprint("Sugerencias de Mejora:"))
		for _, improvement := range analysis.ImprovementSuggestions {
			fmt.Printf("      %s %s\n", color.YellowString("•"), improvement)
		}
		fmt.Println()
	}
}

func (h *SuggestionHandler) displayTechnicalAnalysis(analysis models.RequirementsAnalysis) {
	if len(analysis.ImprovementSuggestions) > 0 {
		techColor := color.New(color.FgBlue, color.Bold)
		fmt.Printf("%s\n", techColor.Sprint("🔧 Análisis Técnico:"))
		for _, improvement := range analysis.ImprovementSuggestions {
			fmt.Printf("   %s %s\n", color.CyanString("•"), improvement)
		}
		fmt.Println()
	}
}

func (h *SuggestionHandler) getCriteriaStatusEmoji(status models.CriteriaStatus) string {
	switch status {
	case models.CriteriaFullyMet:
		return "✅"
	case models.CriteriaPartiallyMet:
		return "⚠️"
	case models.CriteriaNotMet:
		return "❌"
	default:
		return "❓"
	}
}

func (h *SuggestionHandler) getCriteriaStatusColor(status models.CriteriaStatus) *color.Color {
	switch status {
	case models.CriteriaFullyMet:
		return color.New(color.FgGreen, color.Bold)
	case models.CriteriaPartiallyMet:
		return color.New(color.FgYellow, color.Bold)
	case models.CriteriaNotMet:
		return color.New(color.FgRed, color.Bold)
	default:
		return color.New(color.FgHiBlack)
	}
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

func (h *SuggestionHandler) handleCommitSelection(ctx context.Context, suggestions []models.CommitSuggestion) error {
	var selection int

	// Prompt con color
	prompt := color.New(color.FgCyan, color.Bold).Sprint("Selecciona una opción: ")
	fmt.Print(prompt)

	if _, err := fmt.Scan(&selection); err != nil {
		msg := h.t.GetMessage("commit.error_reading_selection", 0, map[string]interface{}{"Error": err})
		ui.PrintError(msg)
		return fmt.Errorf("%s", msg)
	}

	if selection == 0 {
		ui.PrintWarning(h.t.GetMessage("commit.operation_canceled", 0, nil))
		return nil
	}

	if selection < 1 || selection > len(suggestions) {
		msg := h.t.GetMessage("commit.invalid_selection", 0, map[string]interface{}{"Number": len(suggestions)})
		ui.PrintError(msg)
		return fmt.Errorf("%s", msg)
	}

	return h.processCommit(ctx, suggestions[selection-1], h.gitService)
}

func (h *SuggestionHandler) processCommit(ctx context.Context, suggestion models.CommitSuggestion,
	gitService ports.GitService) error {
	commitTitle := strings.TrimSpace(strings.TrimPrefix(suggestion.CommitTitle, "Commit: "))

	// Mostrar resumen del commit seleccionado
	fmt.Println()
	ui.PrintInfo(h.t.GetMessage("ui_preview.commit_selected", 0, map[string]interface{}{
		"Title": commitTitle,
	}))
	ui.PrintInfo(h.t.GetMessage("ui_preview.files_count", 0, map[string]interface{}{
		"Count": len(suggestion.Files),
	}))

	// Preguntar si quiere ver diff
	if ui.AskConfirmation(h.t.GetMessage("ui_preview.ask_show_diff", 0, nil)) {
		fmt.Println()
		if err := ui.ShowDiff(suggestion.Files); err != nil {
			ui.PrintWarning(h.t.GetMessage("ui_preview.error_showing_diff", 0, map[string]interface{}{
				"Error": err,
			}))
		}
	}

	// Confirmación final
	if !ui.AskConfirmation(h.t.GetMessage("ui_preview.ask_confirm_commit", 0, nil)) {
		ui.PrintWarning(h.t.GetMessage("ui_preview.commit_cancelled", 0, nil))
		return nil
	}

	// Spinner para staging
	spinner := ui.NewSmartSpinner(h.t.GetMessage("ui.adding_to_staging", 0, nil))
	spinner.Start()

	for _, file := range suggestion.Files {
		if err := gitService.AddFileToStaging(ctx, file); err != nil {
			spinner.Error(fmt.Sprintf("Error al agregar %s", file))
			msg := h.t.GetMessage("commit.error_add_file_staging", 0, map[string]interface{}{
				"File":  file,
				"Error": err,
			})
			return fmt.Errorf("%s", msg)
		}
	}

	spinner.Success(h.t.GetMessage("ui.files_added_to_staging", 0, map[string]interface{}{
		"Count": len(suggestion.Files),
	}))

	// Spinner para commit
	commitSpinner := ui.NewSmartSpinner(h.t.GetMessage("ui.creating_commit", 0, nil))
	commitSpinner.Start()

	if err := gitService.CreateCommit(ctx, commitTitle); err != nil {
		commitSpinner.Error("Error al crear el commit")
		msg := h.t.GetMessage("commit.error_creating_commit", 0, map[string]interface{}{
			"Commit": commitTitle,
			"Error":  err,
		})
		return fmt.Errorf("%s", msg)
	}

	commitSpinner.Stop()

	// Mensaje de éxito con estilo
	ui.PrintSuccess(h.t.GetMessage("ui.commit_created_successfully", 0, nil))
	fmt.Printf("\n   %s\n\n", color.New(color.FgCyan).Sprint(commitTitle))

	return nil
}
