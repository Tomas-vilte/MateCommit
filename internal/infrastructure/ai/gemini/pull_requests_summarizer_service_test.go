package gemini

import (
	"context"
	"os"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestGeminiPRSummarizer(t *testing.T) {
	t.Run("NewGeminiPRSummarizer with empty API key", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
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
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
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

	t.Run("generatePRPrompt should format correctly", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
			Language:    "en",
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
		assert.Contains(t, prompt, "Catchy but descriptive", "El prompt debe solicitar un título descriptivo")
		assert.Contains(t, prompt, "Key Changes", "El prompt debe solicitar cambios clave")
		assert.Contains(t, prompt, "Labels: Choose wisely", "El prompt debe solicitar etiquetas con criterio")
	})

	t.Run("formatResponse", func(t *testing.T) {
		// Arrange
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "test content"},
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

func TestGeneratePRSummary_HappyPath(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "mate-commit-test-pr-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpHome); err != nil {
			return
		}
	}()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() {
		if err := os.Setenv("HOME", oldHome); err != nil {
			return
		}
	}()

	ctx := context.Background()
	trans, _ := i18n.NewTranslations("en", "../../../i18n/locales/")
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test"}},
		AIConfig:    config.AIConfig{Models: map[config.AI]config.Model{config.AIGemini: "gemini-pro"}},
	}
	summarizer, _ := NewGeminiPRSummarizer(ctx, cfg, trans)
	summarizer.wrapper.SetSkipConfirmation(true)

	t.Run("successful PR summary", func(t *testing.T) {
		expectedJSON := `{"title": "Awesome Feature", "body": "This PR adds awesome feature", "labels": ["feature"]}`
		summarizer.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: expectedJSON}}}},
				},
			}, &models.TokenUsage{TotalTokens: 50}, nil
		}

		summary, err := summarizer.GeneratePRSummary(ctx, "content")

		assert.NoError(t, err)
		assert.Equal(t, "Awesome Feature", summary.Title)
		assert.Equal(t, "This PR adds awesome feature", summary.Body)
		assert.Contains(t, summary.Labels, "feature")
	})

	t.Run("empty title error", func(t *testing.T) {
		expectedJSON := `{"title": "", "body": "no title", "labels": []}`
		summarizer.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: expectedJSON}}}},
				},
			}, &models.TokenUsage{}, nil
		}

		summary, err := summarizer.GeneratePRSummary(ctx, "content")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "respuesta vacía")
		assert.Empty(t, summary.Title)
	})
}
