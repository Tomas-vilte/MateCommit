package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ports"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/ui"
	"github.com/fatih/color"
)

// gitService is a minimal interface for testing purposes
type gitService interface {
	GetChangedFiles(ctx context.Context) ([]string, error)
	GetDiff(ctx context.Context) (string, error)
	HasStagedChanges(ctx context.Context) bool
	CreateCommit(ctx context.Context, message string) error
	AddFileToStaging(ctx context.Context, file string) error
	GetCurrentBranch(ctx context.Context) (string, error)
	GetRepoInfo(ctx context.Context) (string, string, string, error)
}

type SuggestionHandler struct {
	gitService gitService
	vcsClient  ports.VCSClient
	t          *i18n.Translations
}

func NewSuggestionHandler(gitSvc gitService, vcs ports.VCSClient, t *i18n.Translations) *SuggestionHandler {
	return &SuggestionHandler{
		gitService: gitSvc,
		vcsClient:  vcs,
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
		separator := color.New(color.FgCyan).Sprint("━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("\n%s\n", separator)

		suggestionHeader := color.New(color.FgMagenta, color.Bold).Sprint(h.t.GetMessage("ui_labels.suggestion_number", 0, map[string]interface{}{"Number": i + 1}))
		fmt.Printf("%s\n\n", suggestionHeader)

		_, _ = sectionColor.Println(h.t.GetMessage("ui_labels.code_analysis", 0, nil))
		printIndentedKeyValue(h.t.GetMessage("ui_labels.changes_overview", 0, nil), suggestion.CodeAnalysis.ChangesOverview)
		printIndentedKeyValue(h.t.GetMessage("ui_labels.primary_purpose", 0, nil), suggestion.CodeAnalysis.PrimaryPurpose)
		printIndentedKeyValue(h.t.GetMessage("ui_labels.technical_impact", 0, nil), suggestion.CodeAnalysis.TechnicalImpact)

		fmt.Println()
		fmt.Printf("%s\n", separator)

		fmt.Printf("%s %s\n\n",
			color.New(color.FgGreen, color.Bold).Sprint("✓ "+h.t.GetMessage("ui_labels.commit_label", 0, nil)),
			titleColor.Sprint(suggestion.CommitTitle),
		)

		_, _ = sectionColor.Println(h.t.GetMessage("ui_labels.modified_files", 0, nil))
		for _, file := range suggestion.Files {
			fmt.Printf("   %s %s\n", color.CyanString("•"), fileColor.Sprint(file))
		}

		fmt.Printf("\n%s\n", sectionColor.Sprint(h.t.GetMessage("ui_labels.explanation_label", 0, nil)))
		fmt.Printf("   %s\n", suggestion.Explanation)
		fmt.Println()

		if suggestion.RequirementsAnalysis.CriteriaStatus != "" {
			h.displayRequirementsAnalysis(suggestion.RequirementsAnalysis)
		} else {
			h.displayTechnicalAnalysis(suggestion.RequirementsAnalysis)
		}

		if i == 0 && suggestion.Usage != nil {
			fmt.Println()
			ui.PrintTokenUsage(suggestion.Usage, h.t)
		}

		fmt.Printf("%s\n", separator)
	}

	fmt.Println()
	ui.PrintInfo(h.t.GetMessage("ui_selection.select_option", 0, nil))
	fmt.Printf("   %s %s\n", color.GreenString("1-%d:", len(suggestions)), h.t.GetMessage("ui_selection.select_suggestion_range", 0, nil))
	fmt.Printf("   %s %s\n", color.RedString("0:"), h.t.GetMessage("ui_selection.cancel_operation", 0, nil))
	fmt.Println()
}

func (h *SuggestionHandler) displayRequirementsAnalysis(analysis models.RequirementsAnalysis) {
	reqColor := color.New(color.FgMagenta, color.Bold)

	fmt.Printf("%s\n", reqColor.Sprint(h.t.GetMessage("ui_labels.requirements_analysis", 0, nil)))

	statusText := h.getCriteriaStatusText(analysis.CriteriaStatus)
	statusEmoji := h.getCriteriaStatusEmoji(analysis.CriteriaStatus)
	statusColor := h.getCriteriaStatusColor(analysis.CriteriaStatus)

	fmt.Printf("   %s %s %s\n", statusEmoji, color.New(color.FgHiBlack).Sprint(h.t.GetMessage("ui_labels.status_label", 0, nil)),
		statusColor.Sprint(statusText))

	if len(analysis.MissingCriteria) > 0 {
		fmt.Printf("\n   %s %s\n", color.RedString("❌"), color.New(color.FgRed,
			color.Bold).Sprint(h.t.GetMessage("ui_labels.missing_criteria", 0, nil)))
		for _, criteria := range analysis.MissingCriteria {
			fmt.Printf("      %s %s\n", color.RedString("•"), criteria)
		}
	}

	if len(analysis.ImprovementSuggestions) > 0 {
		fmt.Printf("\n   %s %s\n", color.YellowString("💡"), color.New(color.FgYellow,
			color.Bold).Sprint(h.t.GetMessage("ui_labels.improvement_suggestions", 0, nil)))
		for _, improvement := range analysis.ImprovementSuggestions {
			fmt.Printf("      %s %s\n", color.YellowString("•"), improvement)
		}
		fmt.Println()
	}
}

func (h *SuggestionHandler) displayTechnicalAnalysis(analysis models.RequirementsAnalysis) {
	if len(analysis.ImprovementSuggestions) > 0 {
		techColor := color.New(color.FgBlue, color.Bold)
		fmt.Printf("%s\n", techColor.Sprint(h.t.GetMessage("ui_labels.technical_analysis", 0, nil)))
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
		msg = h.t.GetMessage("ai_service.criteria_fully_met_prefix", 0, nil)
		return msg
	case models.CriteriaPartiallyMet:
		msg = h.t.GetMessage("ai_service.criteria_partially_met_prefix", 0, nil)
		return msg
	case models.CriteriaNotMet:
		msg = h.t.GetMessage("ai_service.criteria_not_met_prefix", 0, nil)
		return msg
	default:
		msg = h.t.GetMessage("ai_service.criteria_unknown_prefix", 0, nil)
		return msg
	}
}

func (h *SuggestionHandler) handleCommitSelection(ctx context.Context, suggestions []models.CommitSuggestion) error {
	var selection int

	prompt := color.New(color.FgCyan, color.Bold).Sprint(h.t.GetMessage("ui_selection.select_option", 0, nil))
	fmt.Print(prompt + " ")

	if _, err := fmt.Scan(&selection); err != nil {
		msg := h.t.GetMessage("commit.error_reading_selection", 0, map[string]interface{}{"Error": err})
		ui.PrintError(os.Stdout, msg)
		return fmt.Errorf("%s", msg)
	}

	if selection == 0 {
		ui.PrintWarning(h.t.GetMessage("commit.operation_canceled", 0, nil))
		return nil
	}

	if selection < 1 || selection > len(suggestions) {
		msg := h.t.GetMessage("commit.invalid_selection", 0, map[string]interface{}{"Number": len(suggestions)})
		ui.PrintError(os.Stdout, msg)
		return fmt.Errorf("%s", msg)
	}

	return h.processCommit(ctx, suggestions[selection-1], h.gitService)
}

func (h *SuggestionHandler) processCommit(ctx context.Context, suggestion models.CommitSuggestion,
	gitSvc gitService) error {
	commitTitle := strings.TrimSpace(strings.TrimPrefix(suggestion.CommitTitle, "Commit: "))

	fmt.Println()
	ui.PrintInfo(h.t.GetMessage("ui_preview.commit_selected", 0, map[string]interface{}{
		"Title": commitTitle,
	}))

	treeHeader := h.t.GetMessage("ui_preview.modified_files_header", 0, nil)
	if err := ui.ShowFilesTree(suggestion.Files, treeHeader); err != nil {
		ui.PrintInfo(h.t.GetMessage("ui_preview.files_count", 0, map[string]interface{}{
			"Count": len(suggestion.Files),
		}))
	}

	statsHeader := h.t.GetMessage("ui_preview.changes_header", 0, nil)
	if err := ui.ShowDiffStats(suggestion.Files, statsHeader); err != nil {
		ui.PrintWarning(h.t.GetMessage("ui_preview.error_showing_stats", 0, map[string]interface{}{
			"Error": err,
		}))
	}

	if ui.AskConfirmation(h.t.GetMessage("ui_preview.ask_show_diff", 0, nil)) {
		fmt.Println()
		if err := ui.ShowDiff(suggestion.Files); err != nil {
			ui.PrintWarning(h.t.GetMessage("ui_preview.error_showing_diff", 0, map[string]interface{}{
				"Error": err,
			}))
		}
	}

	finalCommitMessage := commitTitle
	if ui.AskConfirmation(h.t.GetMessage("ui_preview.ask_edit_message", 0, nil)) {
		editorError := h.t.GetMessage("ui_preview.editor_error", 0, nil)
		editedMessage, err := ui.EditCommitMessage(commitTitle, editorError)
		if err != nil {
			ui.PrintError(os.Stdout, h.t.GetMessage("ui_preview.error_editing_message", 0, map[string]interface{}{
				"Error": err,
			}))
			return err
		}
		finalCommitMessage = editedMessage
		ui.PrintSuccess(os.Stdout, h.t.GetMessage("ui_preview.message_updated", 0, nil))
	}

	if !ui.AskConfirmation(h.t.GetMessage("ui_preview.ask_confirm_commit", 0, nil)) {
		ui.PrintWarning(h.t.GetMessage("ui_preview.commit_cancelled", 0, nil))
		return nil
	}

	spinner := ui.NewSmartSpinner(h.t.GetMessage("ui.adding_to_staging", 0, nil))
	spinner.Start()

	for _, file := range suggestion.Files {
		if err := gitSvc.AddFileToStaging(ctx, file); err != nil {
			spinner.Error(h.t.GetMessage("error_adding_file", 0, map[string]interface{}{
				"File": file,
			}))
			return fmt.Errorf("error adding %s: %w", file, err)
		}
	}

	spinner.Success(h.t.GetMessage("ui.files_added_to_staging", 0, map[string]interface{}{
		"Count": len(suggestion.Files),
	}))

	spinner = ui.NewSmartSpinner(h.t.GetMessage("ui.creating_commit", 0, nil))
	spinner.Start()

	if err := gitSvc.CreateCommit(ctx, finalCommitMessage); err != nil {
		spinner.Error(h.t.GetMessage("error_creating_commit", 0, nil))
		return fmt.Errorf("error creating commit: %w", err)
	}

	spinner.Success(h.t.GetMessage("ui.commit_created_successfully", 0, nil))
	fmt.Printf("\n   %s\n\n", finalCommitMessage)

	return nil
}

func printIndentedKeyValue(key, value string) {
	keyColored := color.New(color.FgHiBlack).Sprint(key + ":")
	valueColored := color.New(color.FgWhite).Sprint(value)
	fmt.Printf("   %s %s\n", keyColored, valueColored)
}
