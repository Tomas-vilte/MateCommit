package gemini

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/generative-ai-go/genai"
	"github.com/stretchr/testify/assert"
)

const (
	prResponseText = `## PR Title
Fix image loading error in gallery component

## Suggested Tags
fix,performance

## Key Changes
- Fixed memory leak in image loading process
- Optimized cache usage to improve performance
- Added error handling for network failures
`
)

func TestGeminiPRSummarizer(t *testing.T) {
	t.Run("NewGeminiPRSummarizer with empty API key", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "",
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)

		// Act
		summarizer, err := NewGeminiPRSummarizer(ctx, cfg, trans)

		// Assert
		assert.Nil(t, summarizer, "El summarizer no debería crearse con API key vacía")
		assert.Error(t, err, "Debería retornar un error con API key vacía")
	})

	t.Run("GeneratePRSummary with empty prompt", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "test-api-key",
			AIConfig: config.AIConfig{
				Models: map[config.AI]config.Model{
					config.AIGemini: "gemini-pro",
				},
			},
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)

		summarizer, err := NewGeminiPRSummarizer(ctx, cfg, trans)
		assert.NoError(t, err, "Error creando summarizer")

		// Act
		summary, err := summarizer.GeneratePRSummary(ctx, "")

		// Assert
		assert.Equal(t, models.PRSummary{}, summary, "No deberían generarse resúmenes con prompt vacío")
		assert.Error(t, err, "Debería retornar un error con prompt vacío")
	})

	t.Run("parseSummary correct format", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "test-api-key",
			Language:     "en",
		}

		trans, err := i18n.NewTranslations("en", "../../../i18n/locales/")
		assert.NoError(t, err)

		summarizer, err := NewGeminiPRSummarizer(ctx, cfg, trans)
		assert.NoError(t, err)

		// Act
		summary, err := summarizer.parseSummary(prResponseText)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Fix image loading error in gallery component", summary.Title)
		assert.ElementsMatch(t, []string{"fix", "performance"}, summary.Labels)
		assert.Contains(t, summary.Body, "Fixed memory leak in image loading process")
		assert.Contains(t, summary.Body, "Optimized cache usage to improve performance")
		assert.Contains(t, summary.Body, "Added error handling for network failures")
	})

	t.Run("parseSummary with missing title", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "test-api-key",
			Language:     "en",
		}

		trans, err := i18n.NewTranslations("en", "../../../i18n/locales/")
		assert.NoError(t, err)

		summarizer, err := NewGeminiPRSummarizer(ctx, cfg, trans)
		assert.NoError(t, err)

		missingTitleText := `## Suggested Tags
		fix,performance
		
		## Key Changes
		- Some change`

		// Act
		summary, err := summarizer.parseSummary(missingTitleText)

		// Assert
		assert.Error(t, err, "Debería retornar un error con título faltante")
		assert.Equal(t, "", summary.Title)
	})

	t.Run("generatePRPrompt should format correctly", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "test-api-key",
			Language:     "en",
		}

		trans, err := i18n.NewTranslations("en", "../../../i18n/locales/")
		assert.NoError(t, err)

		summarizer, err := NewGeminiPRSummarizer(ctx, cfg, trans)
		assert.NoError(t, err)

		prContent := "Some PR content to summarize"

		// Act
		prompt := summarizer.generatePRPrompt(prContent)

		// Assert
		assert.Contains(t, prompt, "Some PR content to summarize", "El prompt debe contener el contenido del PR")
		assert.Contains(t, prompt, "PR Title", "El prompt debe solicitar un título para el PR")
		assert.Contains(t, prompt, "Key Changes", "El prompt debe solicitar cambios clave")
		assert.Contains(t, prompt, "Suggested Tags", "El prompt debe solicitar etiquetas sugeridas")
	})

	t.Run("formatResponse", func(t *testing.T) {
		// Arrange
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []genai.Part{
							genai.Text("test content"),
						},
					},
				},
			},
		}

		// Act
		result := formatResponse(resp)

		// Assert
		expected := "test content"
		assert.Equal(t, expected, result, "formatResponse incorrecto")
	})

	t.Run("formatResponse with nil response", func(t *testing.T) {
		// Act
		result := formatResponse(nil)

		// Assert
		assert.Equal(t, "", result, "formatResponse con nil debería retornar string vacío")
	})

	t.Run("formatResponse with empty candidates", func(t *testing.T) {
		// Arrange
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		// Act
		result := formatResponse(resp)

		// Assert
		assert.Equal(t, "", result, "formatResponse con candidatos vacíos debería retornar string vacío")
	})
}
