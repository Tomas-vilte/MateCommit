package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
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
	fmt.Printf("%s\n", h.t.GetMessage("commit.header_message", 0, nil))

	for i, suggestion := range suggestions {
		suggestionHeader := h.t.GetMessage("suggestion_header", 0, map[string]interface{}{"Number": i + 1})
		fmt.Printf("\n%s\n", suggestionHeader)

		fmt.Printf("\n%s\n", h.t.GetMessage("gemini_service.code_analysis_prefix", 0, nil))
		fmt.Printf("%s %s\n", h.t.GetMessage("gemini_service.changes_overview_prefix", 0, nil), suggestion.CodeAnalysis.ChangesOverview)
		fmt.Printf("%s %s\n", h.t.GetMessage("gemini_service.primary_purpose_prefix", 0, nil), suggestion.CodeAnalysis.PrimaryPurpose)
		fmt.Printf("%s %s\n", h.t.GetMessage("gemini_service.technical_impact_prefix", 0, nil), suggestion.CodeAnalysis.TechnicalImpact)

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Commit: %s\n", suggestion.CommitTitle)

		fmt.Println(h.t.GetMessage("gemini_service.modified_files_prefix", 0, nil))
		for _, file := range suggestion.Files {
			fmt.Printf("   - %s\n", file)
		}
		fmt.Printf("%s %s\n", h.t.GetMessage("gemini_service.explanation_prefix", 0, nil), suggestion.Explanation)
		fmt.Println()

		if suggestion.RequirementsAnalysis.CriteriaStatus != "" {
			fmt.Printf("%s\n", h.t.GetMessage("gemini_service.requirements_analysis_prefix", 0, nil))
			statusMsg := h.t.GetMessage("gemini_service.criteria_status_full", 0, map[string]interface{}{
				"Status": h.getCriteriaStatusText(suggestion.RequirementsAnalysis.CriteriaStatus),
			})
			fmt.Printf("%s\n", statusMsg)
			fmt.Println()

			if len(suggestion.RequirementsAnalysis.MissingCriteria) > 0 {
				fmt.Printf("\n%s", h.t.GetMessage("gemini_service.missing_criteria_prefix", 0, nil))
				for _, criteria := range suggestion.RequirementsAnalysis.MissingCriteria {
					fmt.Printf("\n   - %s\n", criteria)
				}
			}

			if len(suggestion.RequirementsAnalysis.ImprovementSuggestions) > 0 {
				fmt.Printf("\n%s", h.t.GetMessage("gemini_service.improvement_suggestions_prefix", 0, nil))
				for _, improvement := range suggestion.RequirementsAnalysis.ImprovementSuggestions {
					fmt.Printf("\n   - %s", improvement)
				}
				fmt.Println()
			}
		} else {
			fmt.Printf("%s\n", h.t.GetMessage("gemini_service.technical_analysis_section", 0, nil))
			if len(suggestion.RequirementsAnalysis.ImprovementSuggestions) > 0 {
				fmt.Println(h.t.GetMessage("gemini_service.improvement_suggestions_label", 0, nil))
				for _, improvement := range suggestion.RequirementsAnalysis.ImprovementSuggestions {
					fmt.Printf("   - %s\n", improvement)
				}
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

func (h *SuggestionHandler) handleCommitSelection(ctx context.Context, suggestions []models.CommitSuggestion) error {
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

	return h.processCommit(ctx, suggestions[selection-1], h.gitService)
}

func (h *SuggestionHandler) processCommit(ctx context.Context, suggestion models.CommitSuggestion, gitService ports.GitService) error {
	commitTitle := strings.TrimSpace(strings.TrimPrefix(suggestion.CommitTitle, "Commit: "))

	for _, file := range suggestion.Files {
		if err := gitService.AddFileToStaging(ctx, file); err != nil {
			msg := h.t.GetMessage("commit.error_add_file_staging", 0, map[string]interface{}{
				"File":  file,
				"Error": err,
			})
			return fmt.Errorf("%s", msg)
		}
		msg := h.t.GetMessage("commit.add_file_to_staging", 0, map[string]interface{}{"File": file})
		fmt.Printf("%s", msg)
	}

	if err := gitService.CreateCommit(ctx, commitTitle); err != nil {
		msg := h.t.GetMessage("commit.error_creating_commit", 0, map[string]interface{}{
			"Commit": commitTitle,
			"Error":  err,
		})
		return fmt.Errorf("%s", msg)
	}

	fmt.Printf("%s\n", h.t.GetMessage("commit.commit_successful", 0, map[string]interface{}{"CommitTitle": commitTitle}))
	return nil
}
