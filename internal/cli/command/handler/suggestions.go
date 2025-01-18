package handler

import (
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"strings"
)

func HandleSuggestions(suggestions []models.CommitSuggestion, gitService *git.GitService, t *i18n.Translations) error {
	displaySuggestions(suggestions, t)
	return handleCommitSelection(suggestions, gitService, t)
}

func displaySuggestions(suggestions []models.CommitSuggestion, t *i18n.Translations) {
	fmt.Printf("%s\n", t.GetMessage("commit.header_message", 0, nil))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")

	for _, suggestion := range suggestions {
		fmt.Printf("%s\n", suggestion.CommitTitle)
		fmt.Println(t.GetMessage("commit.file_list_header", 0, nil))
		for _, file := range suggestion.Files {
			fmt.Printf("   - %s\n", file)
		}
		fmt.Printf("%s\n", suggestion.Explanation)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	}

	fmt.Println(t.GetMessage("commit.select_option_prompt", 0, nil))
	fmt.Println(t.GetMessage("commit.option_commit", 0, nil))
	fmt.Println(t.GetMessage("commit.option_exit", 0, nil))
}

func handleCommitSelection(suggestions []models.CommitSuggestion, gitService *git.GitService, t *i18n.Translations) error {
	var selection int
	fmt.Print(t.GetMessage("commit.prompt_selection", 0, nil))
	if _, err := fmt.Scan(&selection); err != nil {
		msg := t.GetMessage("commit.error_reading_selection", 0, map[string]interface{}{"Error": err})
		return fmt.Errorf("%s", msg)
	}

	if selection == 0 {
		fmt.Println(t.GetMessage("commit.operation_canceled", 0, nil))
		return nil
	}

	if selection < 1 || selection > len(suggestions) {
		msg := t.GetMessage("commit.invalid_selection", 0, map[string]interface{}{"Number": len(suggestions)})
		return fmt.Errorf("%s", msg)
	}

	return processCommit(suggestions[selection-1], gitService, t)
}

func processCommit(suggestion models.CommitSuggestion, gitService *git.GitService, t *i18n.Translations) error {
	commitTitle := strings.TrimSpace(strings.TrimPrefix(suggestion.CommitTitle, "Commit: "))

	for _, file := range suggestion.Files {
		if err := gitService.AddFileToStaging(file); err != nil {
			msg := t.GetMessage("commit.error_add_file_staging", 0, map[string]interface{}{
				"File":  file,
				"Error": err,
			})
			return fmt.Errorf("%s", msg)
		}
		msg := t.GetMessage("commit.add_file_to_staging", 0, map[string]interface{}{"File": file})
		fmt.Printf("%s", msg)
	}

	if err := gitService.CreateCommit(commitTitle); err != nil {
		msg := t.GetMessage("commit.error_creating_commit", 0, map[string]interface{}{
			"Commit": commitTitle,
			"Error":  err,
		})
		return fmt.Errorf("%s", msg)
	}

	fmt.Printf("%s\n", t.GetMessage("commit.commit_successful", 0, map[string]interface{}{"CommitTitle": commitTitle}))
	return nil
}
