package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
	"google.golang.org/genai"
)

var _ ports.ReleaseNotesGenerator = (*ReleaseNotesGenerator)(nil)

type ReleaseNotesGenerator struct {
	client *genai.Client
	trans  *i18n.Translations
	model  string
	lang   string
	owner  string
	repo   string
}

func NewReleaseNotesGenerator(ctx context.Context, cfg *config.Config, trans *i18n.Translations, owner, repo string) (*ReleaseNotesGenerator, error) {
	if cfg.GeminiAPIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		msg := trans.GetMessage("error_gemini_client", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	modelName := string(cfg.AIConfig.Models[config.AIGemini])
	return &ReleaseNotesGenerator{
		client: client,
		model:  modelName,
		lang:   cfg.Language,
		trans:  trans,
		owner:  owner,
		repo:   repo,
	}, nil
}

func (g *ReleaseNotesGenerator) GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	prompt := g.buildPrompt(release)

	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		msg := g.trans.GetMessage("error_generating_release_notes", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	if len(resp.Candidates) == 0 {
		msg := g.trans.GetMessage("error_no_ai_response", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	content := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		content += part.Text
	}
	parseResponse, _ := g.parseResponse(content, release)

	return parseResponse, nil
}

func (g *ReleaseNotesGenerator) buildPrompt(release *models.Release) string {
	template := ai.GetReleasePromptTemplate(g.lang)

	changes := g.formatChangesForPrompt(release)

	return fmt.Sprintf(template,
		g.owner,
		g.repo,
		release.PreviousVersion,
		release.Version,
		release.VersionBump,
		changes,
	)
}

func (g *ReleaseNotesGenerator) formatChangesForPrompt(release *models.Release) string {
	var sb strings.Builder

	if len(release.Breaking) > 0 {
		sb.WriteString("BREAKING CHANGES:\n")
		for _, item := range release.Breaking {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.Features) > 0 {
		sb.WriteString("NEW FEATURES:\n")
		for _, item := range release.Features {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.BugFixes) > 0 {
		sb.WriteString("BUG FIXES:\n")
		for _, item := range release.BugFixes {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.Improvements) > 0 {
		sb.WriteString("IMPROVEMENTS:\n")
		for _, item := range release.Improvements {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *ReleaseNotesGenerator) parseResponse(content string, release *models.Release) (*models.ReleaseNotes, error) {
	lines := strings.Split(content, "\n")

	notes := &models.ReleaseNotes{
		Recommended: release.VersionBump,
		Links:       make(map[string]string),
	}

	var (
		inHighlights      bool
		inSummary         bool
		inQuickStart      bool
		inBreakingChanges bool
		inExample         bool
		inComparison      bool
		inLinks           bool
		currentExample    *models.CodeExample
		currentComparison *models.Comparison
		quickStartLines   []string
		summaryLines      []string
	)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inExample && (strings.HasPrefix(trimmed, "TITLE:") || strings.HasPrefix(trimmed, "TÍTULO:")) {
			notes.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "TITLE:"))
			notes.Title = strings.TrimSpace(strings.TrimPrefix(notes.Title, "TÍTULO:"))
			inHighlights, inSummary, inQuickStart, inBreakingChanges, inExample, inComparison, inLinks = false, false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "SUMMARY:") || strings.HasPrefix(trimmed, "RESUMEN:") {
			inSummary = true
			inHighlights, inQuickStart, inBreakingChanges, inExample, inComparison, inLinks = false, false, false, false, false, false
			content := strings.TrimSpace(strings.TrimPrefix(trimmed, "SUMMARY:"))
			content = strings.TrimSpace(strings.TrimPrefix(content, "RESUMEN:"))
			if content != "" {
				summaryLines = append(summaryLines, content)
			}
		} else if strings.HasPrefix(trimmed, "HIGHLIGHTS:") {
			inHighlights = true
			inSummary, inQuickStart, inBreakingChanges, inExample, inComparison, inLinks = false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "QUICK_START:") {
			inQuickStart = true
			inHighlights, inSummary, inBreakingChanges, inExample, inComparison, inLinks = false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "EXAMPLES:") {
			inExample = true
			inHighlights, inSummary, inQuickStart, inBreakingChanges, inComparison, inLinks = false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "BREAKING_CHANGES:") {
			inBreakingChanges = true
			inHighlights, inSummary, inQuickStart, inExample, inComparison, inLinks = false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "COMPARISONS:") {
			inComparison = true
			inHighlights, inSummary, inQuickStart, inBreakingChanges, inExample, inLinks = false, false, false, false, false, false
		} else if strings.HasPrefix(trimmed, "LINKS:") {
			inLinks = true
			inHighlights, inSummary, inQuickStart, inBreakingChanges, inExample, inComparison = false, false, false, false, false, false
		} else if inSummary && trimmed != "" {
			summaryLines = append(summaryLines, trimmed)
		} else if inHighlights && strings.HasPrefix(trimmed, "-") {
			highlight := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if highlight != "" {
				notes.Highlights = append(notes.Highlights, highlight)
			}
		} else if inQuickStart && trimmed != "" {
			quickStartLines = append(quickStartLines, line)
		} else if inBreakingChanges && strings.HasPrefix(trimmed, "-") {
			bc := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if bc != "" && !strings.EqualFold(bc, "ninguno") && !strings.EqualFold(bc, "none") {
				notes.BreakingChanges = append(notes.BreakingChanges, bc)
			}
		} else if inExample {
			if strings.HasPrefix(trimmed, "EXAMPLE_") {
				if currentExample != nil {
					notes.Examples = append(notes.Examples, *currentExample)
				}
				currentExample = &models.CodeExample{}
			} else if currentExample != nil {
				if strings.HasPrefix(trimmed, "TITLE:") {
					currentExample.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "TITLE:"))
				} else if strings.HasPrefix(trimmed, "DESCRIPTION:") {
					currentExample.Description = strings.TrimSpace(strings.TrimPrefix(trimmed, "DESCRIPTION:"))
				} else if strings.HasPrefix(trimmed, "LANGUAGE:") {
					currentExample.Language = strings.TrimSpace(strings.TrimPrefix(trimmed, "LANGUAGE:"))
				} else if strings.HasPrefix(trimmed, "CODE:") {
					// Capturar el contenido después de CODE: (puede estar vacío)
					codeContent := strings.TrimSpace(strings.TrimPrefix(trimmed, "CODE:"))
					// Limpiar backticks de markdown si los hay
					if strings.HasPrefix(codeContent, "```") {
						codeContent = ""
					}
					currentExample.Code = codeContent
				} else if trimmed != "" && !strings.HasPrefix(trimmed, "EXAMPLE_") && !strings.HasPrefix(trimmed, "BREAKING_") && !strings.HasPrefix(trimmed, "COMPARISONS:") && !strings.HasPrefix(trimmed, "LINKS:") {
					// Capturar líneas adicionales de código (saltar backticks de markdown)
					if trimmed == "```" || strings.HasPrefix(trimmed, "```") {
						// Ignorar líneas de backticks
					} else if currentExample.Code == "" {
						currentExample.Code = line
					} else {
						currentExample.Code += "\n" + line
					}
				}
			}
		} else if inComparison {
			if strings.HasPrefix(trimmed, "COMPARISON_") {
				if currentComparison != nil {
					notes.Comparisons = append(notes.Comparisons, *currentComparison)
				}
				currentComparison = &models.Comparison{}
			} else if currentComparison != nil {
				if strings.HasPrefix(trimmed, "FEATURE:") {
					currentComparison.Feature = strings.TrimSpace(strings.TrimPrefix(trimmed, "FEATURE:"))
				} else if strings.HasPrefix(trimmed, "BEFORE:") {
					currentComparison.Before = strings.TrimSpace(strings.TrimPrefix(trimmed, "BEFORE:"))
				} else if strings.HasPrefix(trimmed, "AFTER:") {
					currentComparison.After = strings.TrimSpace(strings.TrimPrefix(trimmed, "AFTER:"))
				}
			}
		} else if inLinks {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if key != "" && value != "" {
					notes.Links[key] = value
				}
			}
		}
	}

	if currentExample != nil {
		notes.Examples = append(notes.Examples, *currentExample)
	}
	if currentComparison != nil {
		notes.Comparisons = append(notes.Comparisons, *currentComparison)
	}

	if len(quickStartLines) > 0 {
		notes.QuickStart = strings.Join(quickStartLines, "\n")
	}

	if len(summaryLines) > 0 {
		notes.Summary = strings.Join(summaryLines, " ")
	}

	if notes.Title == "" {
		notes.Title = fmt.Sprintf("Version %s", release.Version)
	}

	return notes, nil
}
