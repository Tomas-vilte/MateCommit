package gemini

import (
	"context"
	"encoding/json"
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
	*GeminiProvider
	wrapper *ai.CostAwareWrapper
	trans   *i18n.Translations
	lang    string
	owner   string
	repo    string
}

type ReleaseNotesJSON struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Highlights      []string `json:"highlights"`
	BreakingChanges []string `json:"breaking_changes"`
	Contributors    string   `json:"contributors"`
}

func NewReleaseNotesGenerator(ctx context.Context, cfg *config.Config, trans *i18n.Translations, owner, repo string) (*ReleaseNotesGenerator, error) {
	providerCfg, exists := cfg.AIProviders["gemini"]
	if !exists || providerCfg.APIKey == "" {
		msg := trans.GetMessage("error_missing_api_key", 0, map[string]interface{}{
			"Provider": "gemini",
		})
		return nil, fmt.Errorf("%s", msg)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  providerCfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		msg := trans.GetMessage("ai_service.error_ai_client", 0, map[string]interface{}{
			"Error": err,
		})
		return nil, fmt.Errorf("%s", msg)
	}

	modelName := string(cfg.AIConfig.Models[config.AIGemini])

	budgetDaily := 0.0
	if cfg.AIConfig.BudgetDaily != nil {
		budgetDaily = *cfg.AIConfig.BudgetDaily
	}

	service := &ReleaseNotesGenerator{
		GeminiProvider: NewGeminiProvider(client, modelName),
		lang:           cfg.Language,
		trans:          trans,
		owner:          owner,
		repo:           repo,
	}

	wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
		Provider:              service,
		BudgetDaily:           budgetDaily,
		Trans:                 trans,
		EstimatedOutputTokens: 700,
	})
	if err != nil {
		return nil, fmt.Errorf("error creando wrapper: %w", err)
	}

	service.wrapper = wrapper

	return service, nil
}

func (g *ReleaseNotesGenerator) GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	prompt := g.buildPrompt(release)

	generateFn := func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
		genConfig := GetGenerateConfig(mName, "application/json")

		resp, err := g.Client.Models.GenerateContent(ctx, mName, genai.Text(p), genConfig)
		if err != nil {
			return nil, nil, err
		}

		usage := extractUsage(resp)
		return resp, usage, nil
	}

	resp, usage, err := g.wrapper.WrapGenerate(ctx, "generate-release", prompt, generateFn)
	if err != nil {
		return nil, fmt.Errorf("error generando release notes: %w", err)
	}

	geminiResp := resp.(*genai.GenerateContentResponse)

	if len(geminiResp.Candidates) == 0 {
		msg := g.trans.GetMessage("ai_service.error_no_ai_response", 0, nil)
		return nil, fmt.Errorf("%s", msg)
	}

	content := ""
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	notes, err := g.parseJSONResponse(content, release)
	if err != nil {
		return nil, fmt.Errorf("error al parsear respuesta JSON de release notes: %w", err)
	}

	notes.Usage = usage

	return notes, nil
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

	headers := ai.GetReleaseNotesSectionHeaders(g.lang)

	if len(release.Breaking) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["breaking"]))
		for _, item := range release.Breaking {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.Features) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["features"]))
		for _, item := range release.Features {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.BugFixes) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["fixes"]))
		for _, item := range release.BugFixes {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.Improvements) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["improvements"]))
		for _, item := range release.Improvements {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", item.Type, item.Description))
		}
		sb.WriteString("\n")
	}

	if len(release.ClosedIssues) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["closed_issues"]))
		for _, issue := range release.ClosedIssues {
			sb.WriteString(fmt.Sprintf("- #%d: %s (by @%s)\n", issue.Number, issue.Title, issue.Author))
		}
		sb.WriteString("\n")
	}

	if len(release.MergedPRs) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["merged_prs"]))
		for _, pr := range release.MergedPRs {
			sb.WriteString(fmt.Sprintf("- #%d: %s (by @%s)\n", pr.Number, pr.Title, pr.Author))
			if pr.Description != "" {
				lines := strings.Split(pr.Description, "\n")
				if len(lines) > 0 && lines[0] != "" {
					sb.WriteString(fmt.Sprintf("  Description: %s\n", lines[0]))
				}
			}
		}
		sb.WriteString("\n")
	}

	if len(release.Contributors) > 0 {
		sb.WriteString(fmt.Sprintf("%s (%d total):\n", headers["contributors"], len(release.Contributors)))
		for _, contributor := range release.Contributors {
			sb.WriteString(fmt.Sprintf("- @%s\n", contributor))
		}
		if len(release.NewContributors) > 0 {
			sb.WriteString(fmt.Sprintf("New contributors: %s\n", strings.Join(release.NewContributors, ", ")))
		}
		sb.WriteString("\n")
	}

	if release.FileStats.FilesChanged > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["file_stats"]))
		sb.WriteString(fmt.Sprintf("- Files changed: %d\n", release.FileStats.FilesChanged))
		sb.WriteString(fmt.Sprintf("- Insertions: +%d\n", release.FileStats.Insertions))
		sb.WriteString(fmt.Sprintf("- Deletions: -%d\n", release.FileStats.Deletions))
		if len(release.FileStats.TopFiles) > 0 {
			sb.WriteString("Top modified files:\n")
			for _, file := range release.FileStats.TopFiles {
				sb.WriteString(fmt.Sprintf("  - %s (+%d/-%d)\n", file.Path, file.Additions, file.Deletions))
			}
		}
		sb.WriteString("\n")
	}

	if len(release.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("%s\n", headers["deps"]))
		for _, dep := range release.Dependencies {
			switch dep.Type {
			case "updated":
				sb.WriteString(fmt.Sprintf("- %s: %s → %s\n", dep.Name, dep.OldVersion, dep.NewVersion))
			case "added":
				sb.WriteString(fmt.Sprintf("- Added: %s %s\n", dep.Name, dep.NewVersion))
			case "removed":
				sb.WriteString(fmt.Sprintf("- Removed: %s %s\n", dep.Name, dep.OldVersion))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *ReleaseNotesGenerator) parseJSONResponse(content string, release *models.Release) (*models.ReleaseNotes, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var jsonNotes ReleaseNotesJSON
	if err := json.Unmarshal([]byte(content), &jsonNotes); err != nil {
		return nil, fmt.Errorf("error al parsear JSON de release notes: %w", err)
	}

	notes := &models.ReleaseNotes{
		Title:           jsonNotes.Title,
		Summary:         jsonNotes.Summary,
		Highlights:      jsonNotes.Highlights,
		BreakingChanges: jsonNotes.BreakingChanges,
		Recommended:     release.VersionBump,
		Links:           make(map[string]string),
	}

	if jsonNotes.Contributors != "" && jsonNotes.Contributors != "N/A" {
		notes.Links["Contributors"] = jsonNotes.Contributors
	}

	return notes, nil
}
