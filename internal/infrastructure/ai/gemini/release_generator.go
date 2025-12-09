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
}

func NewReleaseNotesGenerator(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*ReleaseNotesGenerator, error) {
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

	return g.parseResponse(content, release)
}

func (g *ReleaseNotesGenerator) buildPrompt(release *models.Release) string {
	template := ai.GetReleasePromptTemplate(g.lang)

	changes := g.formatChangesForPrompt(release)

	return fmt.Sprintf(template,
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
	}

	var inHighlights bool
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "TITLE:") || strings.HasPrefix(line, "TÍTULO:") {
			notes.Title = strings.TrimSpace(strings.TrimPrefix(line, "TITLE:"))
			notes.Title = strings.TrimSpace(strings.TrimPrefix(notes.Title, "TÍTULO:"))
		} else if strings.HasPrefix(line, "SUMMARY:") || strings.HasPrefix(line, "RESUMEN:") {
			notes.Summary = strings.TrimSpace(strings.TrimPrefix(line, "SUMMARY:"))
			notes.Summary = strings.TrimSpace(strings.TrimPrefix(notes.Summary, "RESUMEN:"))
		} else if strings.HasPrefix(line, "HIGHLIGHTS:") {
			inHighlights = true
		} else if inHighlights && strings.HasPrefix(line, "-") {
			highlight := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			if highlight != "" {
				notes.Highlights = append(notes.Highlights, highlight)
			}
		}
	}

	if notes.Title == "" {
		notes.Title = fmt.Sprintf("Version %s", release.Version)
	}

	return notes, nil

}
