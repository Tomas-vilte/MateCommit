package gemini

import (
	"strings"
)

// CleanLabels cleans and validates labels, keeping only the allowed ones.
// It accepts a list of labels to clean and a list of available labels from the repository.
// If availableLabels is empty, it falls back to a default list of common labels.
func CleanLabels(labels []string, availableLabels []string) []string {
	allowedLabels := make(map[string]bool)

	if len(availableLabels) > 0 {
		for _, l := range availableLabels {
			allowedLabels[strings.ToLower(l)] = true
		}
	} else {
		// Fallback to default list if no repo labels provided
		defaultLabels := []string{
			"feature", "fix", "refactor", "docs", "test", "infra",
			"enhancement", "bug", "good first issue", "help wanted",
			"chore", "performance", "security", "tech-debt", "breaking-change",
		}
		for _, l := range defaultLabels {
			allowedLabels[l] = true
		}
	}

	cleaned := make([]string, 0)
	seen := make(map[string]bool)

	for _, label := range labels {
		trimmed := strings.TrimSpace(strings.ToLower(label))
		if trimmed != "" && allowedLabels[trimmed] && !seen[trimmed] {
			cleaned = append(cleaned, trimmed)
			seen[trimmed] = true
		}
	}

	return cleaned
}
