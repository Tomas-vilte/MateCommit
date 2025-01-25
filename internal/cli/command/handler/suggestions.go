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
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")

	for _, suggestion := range suggestions {
		fmt.Printf("%s\n", suggestion.CommitTitle)
		fmt.Println(h.t.GetMessage("commit.file_list_header", 0, nil))
		for _, file := range suggestion.Files {
			fmt.Printf("   - %s\n", file)
		}
		fmt.Printf("%s\n", suggestion.Explanation)

		// Formatear CriteriaStatus
		switch suggestion.CriteriaStatus {
		case models.CriteriaFullyMet:
			fmt.Println(h.t.GetMessage("commit.criteria_fully_met", 0, nil))
		case models.CriteriaPartiallyMet:
			fmt.Println(h.t.GetMessage("commit.criteria_partially_met", 0, nil))
		case models.CriteriaNotMet:
			fmt.Println(h.t.GetMessage("commit.criteria_not_met", 0, nil))
		default:
			fmt.Println(h.t.GetMessage("commit.criteria_unknown", 0, nil))
		}

		// Formatear MissingCriteria
		if len(suggestion.MissingCriteria) > 0 {
			fmt.Printf("%s: %s\n", h.t.GetMessage("commit.missing_criteria_header", 0, nil), strings.Join(suggestion.MissingCriteria, ", "))
		} else {
			fmt.Println(h.t.GetMessage("commit.missing_criteria_none", 0, nil))
		}

		// Formatear ImprovementSuggestions
		if len(suggestion.ImprovementSuggestions) > 0 {
			fmt.Printf("%s: %s\n", h.t.GetMessage("commit.improvement_suggestions_header", 0, nil), strings.Join(suggestion.ImprovementSuggestions, ", "))
		} else {
			fmt.Println(h.t.GetMessage("commit.improvement_suggestions_none", 0, nil))
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	}

	fmt.Println(h.t.GetMessage("commit.select_option_prompt", 0, nil))
	fmt.Println(h.t.GetMessage("commit.option_commit", 0, nil))
	fmt.Println(h.t.GetMessage("commit.option_exit", 0, nil))
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
