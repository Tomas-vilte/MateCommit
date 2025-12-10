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
		GeminiAPIKey: "test-api-key",
		Language:     "en",
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
		GeminiAPIKey: "",
	}
	trans, err := i18n.NewTranslations("en", "")
	assert.NoError(t, err)

	// Act
	generator, err := NewReleaseNotesGenerator(ctx, cfg, trans, "test-owner", "test-repo")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, generator)
	assert.Contains(t, err.Error(), "The GEMINI_API_KEY is not configured")
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
		assert.Contains(t, prompt, "Previous version: v0.9.0")
		assert.Contains(t, prompt, "New version: v1.0.0")
		assert.Contains(t, prompt, "Bump type: major")

		// Check for sections
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

		assert.Contains(t, prompt, "Versión anterior:")
		assert.Contains(t, prompt, "Nueva versión:")
		assert.Contains(t, prompt, "Tipo de bump:")
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

func TestParseResponse(t *testing.T) {
	generator := &ReleaseNotesGenerator{}
	release := &models.Release{Version: "v1.0.0", VersionBump: "minor"}

	t.Run("parses English response correctly", func(t *testing.T) {
		// Arrange
		content := `TITLE: Release v1.0.0
SUMMARY: This is a summary.
HIGHLIGHTS:
- Highlight 1
- Highlight 2`

		// Act
		notes, err := generator.parseResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v1.0.0", notes.Title)
		assert.Equal(t, "This is a summary.", notes.Summary)
		assert.Equal(t, []string{"Highlight 1", "Highlight 2"}, notes.Highlights)
		assert.Equal(t, models.VersionBump("minor"), notes.Recommended)
	})

	t.Run("parses Spanish response correctly", func(t *testing.T) {
		// Arrange
		content := `TÍTULO: Lanzamiento v1.0.0
RESUMEN: Este es un resumen.
HIGHLIGHTS:
- Destacado 1
- Destacado 2`

		// Act
		notes, err := generator.parseResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Lanzamiento v1.0.0", notes.Title)
		assert.Equal(t, "Este es un resumen.", notes.Summary)
		assert.Equal(t, []string{"Destacado 1", "Destacado 2"}, notes.Highlights)
	})

	t.Run("handles missing title by using default", func(t *testing.T) {
		// Arrange
		content := `SUMMARY: Just a summary.`

		// Act
		notes, err := generator.parseResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Version v1.0.0", notes.Title) // Default behavior
		assert.Equal(t, "Just a summary.", notes.Summary)
	})

	t.Run("handles extra whitespace", func(t *testing.T) {
		// Arrange
		content := `
TITLE:   Release v1.0.0  
SUMMARY:  Summary with spaces.  
HIGHLIGHTS:
-   Highlight 1   
`

		// Act
		notes, err := generator.parseResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v1.0.0", notes.Title)
		assert.Equal(t, "Summary with spaces.", notes.Summary)
		assert.Equal(t, []string{"Highlight 1"}, notes.Highlights)
	})
}
