package ui

import (
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

func PrintTokenUsage(usage *models.UsageMetadata, trans *i18n.Translations) {
	if usage == nil {
		return
	}

	header := trans.GetMessage("token_usage.header", 0, nil)
	inputLabel := trans.GetMessage("token_usage.input", 0, nil)
	outputLabel := trans.GetMessage("token_usage.output", 0, nil)
	totalLabel := trans.GetMessage("token_usage.total", 0, nil)

	fmt.Println()
	PrintSectionBanner(header)

	PrintKeyValue(inputLabel, fmt.Sprintf("%d", usage.InputTokens))
	PrintKeyValue(outputLabel, fmt.Sprintf("%d", usage.OutputTokens))
	PrintKeyValue(totalLabel, fmt.Sprintf("%d", usage.TotalTokens))
	fmt.Println()
}
