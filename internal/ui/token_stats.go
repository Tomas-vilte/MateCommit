package ui

import (
	"fmt"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/fatih/color"
)

func PrintTokenUsage(usage *models.TokenUsage, t *i18n.Translations) {
	if usage == nil {
		return
	}
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	_, _ = cyan.Print("📊 ")
	fmt.Printf("%s: ", t.GetMessage("ui.token_usage", 0, nil))
	fmt.Printf("%s %d | ", t.GetMessage("ui.input", 0, nil), usage.InputTokens)
	fmt.Printf("%s %d | ", t.GetMessage("ui.output", 0, nil), usage.OutputTokens)
	fmt.Printf("%s %d\n", t.GetMessage("ui.total", 0, nil), usage.TotalTokens)
	if usage.CostUSD > 0 {
		_, _ = yellow.Print("💰 ")
		fmt.Printf("%s: ", t.GetMessage("ui.cost", 0, nil))
		_, _ = yellow.Printf("$%.4f USD\n", usage.CostUSD)
	}
	if usage.CacheHit {
		_, _ = green.Printf("✓ %s\n", t.GetMessage("cost.cache_hit", 0, nil))
	}
	if usage.DurationMs > 0 {
		fmt.Printf("⏱️  %s: %dms\n", t.GetMessage("ui.duration", 0, nil), usage.DurationMs)
	}
}
