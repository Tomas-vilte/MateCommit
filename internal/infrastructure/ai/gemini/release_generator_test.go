package gemini

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func TestNewReleaseNotesGenerator(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		Language:    "en",
		AIConfig: config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIGemini: "gemini-pro",
			},
		},
	}

	// Act
	trans, err := i18n.NewTranslations("en", "")
	assert.NoError(t, err)

	generator, err := NewReleaseNotesGenerator(ctx, cfg, trans, "test-owner", "test-repo")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, generator)
	assert.Equal(t, "gemini-pro", generator.model)
	assert.Equal(t, "en", generator.lang)
}

func TestNewReleaseNotesGenerator_MissingKey(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{},
	}
	trans, err := i18n.NewTranslations("en", "")
	assert.NoError(t, err)

	// Act
	generator, err := NewReleaseNotesGenerator(ctx, cfg, trans, "test-owner", "test-repo")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, generator)
	assert.Contains(t, err.Error(), "The gemini API key is not configured")
}

func TestBuildPrompt(t *testing.T) {
	t.Run("formats changes correctly in English", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "en"}
		release := &models.Release{
			Version:         "v1.0.0",
			PreviousVersion: "v0.9.0",
			VersionBump:     "major",
			Features:        []models.ReleaseItem{{Type: "feat", Description: "New feature"}},
			BugFixes:        []models.ReleaseItem{{Type: "fix", Description: "Bug fix"}},
			Breaking:        []models.ReleaseItem{{Type: "breaking", Description: "Breaking change"}},
			Improvements:    []models.ReleaseItem{{Type: "chore", Description: "Improvement"}},
		}

		// Act
		prompt := generator.buildPrompt(release)

		// Assert
		// Assert
		assert.Contains(t, prompt, "Versions: v0.9.0 -> v1.0.0 (major)")

		assert.Contains(t, prompt, "BREAKING CHANGES:")
		assert.Contains(t, prompt, "- breaking: Breaking change")
		assert.Contains(t, prompt, "NEW FEATURES:")
		assert.Contains(t, prompt, "- feat: New feature")
		assert.Contains(t, prompt, "BUG FIXES:")
		assert.Contains(t, prompt, "- fix: Bug fix")
		assert.Contains(t, prompt, "IMPROVEMENTS:")
		assert.Contains(t, prompt, "- chore: Improvement")
	})

	t.Run("formats changes correctly in Spanish", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "es"}
		release := &models.Release{
			Version: "v1.0.0",
		}

		// Act
		prompt := generator.buildPrompt(release)

		assert.Contains(t, prompt, "- Versiones:")
		assert.Contains(t, prompt, "->")
		assert.Contains(t, prompt, "(")
	})

	t.Run("handles empty changes", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "en"}
		release := &models.Release{Version: "v1.0.0"}

		// Act
		prompt := generator.buildPrompt(release)

		// Assert
		assert.NotContains(t, prompt, "BREAKING CHANGES:")
		assert.NotContains(t, prompt, "NEW FEATURES:")
		assert.NotContains(t, prompt, "BUG FIXES:")
		assert.NotContains(t, prompt, "IMPROVEMENTS:")
	})
}

func TestParseJSONResponse(t *testing.T) {
	generator := &ReleaseNotesGenerator{}
	release := &models.Release{Version: "v1.0.0", VersionBump: "minor"}

	t.Run("parses JSON response correctly", func(t *testing.T) {
		// Arrange
		content := `{
			"title": "Release v1.0.0",
			"summary": "This is a summary.",
			"highlights": ["Highlight 1", "Highlight 2"],
			"breaking_changes": [],
			"contributors": ""
		}`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v1.0.0", notes.Title)
		assert.Equal(t, "This is a summary.", notes.Summary)
		assert.Equal(t, []string{"Highlight 1", "Highlight 2"}, notes.Highlights)
		assert.Equal(t, models.VersionBump("minor"), notes.Recommended)
	})

	t.Run("parses JSON with breaking changes", func(t *testing.T) {
		// Arrange
		content := `{
			"title": "Release v2.0.0",
			"summary": "Major release with breaking changes.",
			"highlights": ["New API", "Better performance"],
			"breaking_changes": ["Removed old API", "Changed config format"],
			"contributors": "https://github.com/test/repo/graphs/contributors"
		}`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v2.0.0", notes.Title)
		assert.Equal(t, "Major release with breaking changes.", notes.Summary)
		assert.Equal(t, []string{"New API", "Better performance"}, notes.Highlights)
		assert.Equal(t, []string{"Removed old API", "Changed config format"}, notes.BreakingChanges)
		assert.Equal(t, "https://github.com/test/repo/graphs/contributors", notes.Links["Contributors"])
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		// Arrange
		content := `invalid json`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, notes)
		assert.Contains(t, err.Error(), "parsear JSON")
	})

	t.Run("handles JSON with code fences", func(t *testing.T) {
		// Arrange
		content := "```json\n" + `{
			"title": "Test",
			"summary": "Summary",
			"highlights": [],
			"breaking_changes": [],
			"contributors": ""
		}` + "\n```"

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Test", notes.Title)
		assert.Equal(t, "Summary", notes.Summary)
	})
}
