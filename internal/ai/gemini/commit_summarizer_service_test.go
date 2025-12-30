package gemini

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
	"google.golang.org/genai"
)

const (
	responseJSON = `[
	{
		"title": "refactor: Mejoras en la presentación de sugerencias y configuración de Jira",
		"desc": "Se mejoró la salida de sugerencias y el manejo de errores en la configuración de Jira.",
		"files": [
			"cmd/main.go",
			"internal/cli/command/config/set_jira_config.go"
		],
		"analysis": {
			"overview": "Mejora en el manejo de la configuración de Jira y la presentación de sugerencias de commit.",
			"purpose": "Mejorar la experiencia del usuario al mostrar información más detallada.",
			"impact": "Se modifican varias partes del código para mejorar la estructura."
		},
		"requirements": {
			"status": "partially_met",
			"missing": [
				"Conexión a la API de Jira",
				"Extracción de Tickets"
			],
			"suggestions": [
				"Implementar manejo de errores para token expirado",
				"Agregar retry mechanism para API no disponible"
			]
		}
	}
]`
)

func TestGeminiCommitSummarizer(t *testing.T) {
	t.Run("NewGeminiCommitSummarizer with empty API key", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)

		// assert
		assert.Nil(t, service)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "API key is missing")
	})

	t.Run("GenerateSuggestions with invalid count", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		if err != nil {
			t.Fatalf("Error creando servicio: %v", err)
		}

		info := models.CommitInfo{
			Files: []string{"test.txt"},
			Diff:  "test diff",
		}

		// act
		suggestions, err := service.GenerateSuggestions(ctx, info, 0)

		// assert
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid suggestion count")
	})

	t.Run("GenerateSuggestions no files", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		if err != nil {
			t.Fatalf("Error creando servicio: %v", err)
		}

		info := models.CommitInfo{
			Files: []string{},
			Diff:  "test diff",
		}

		// act
		suggestions, err := service.GenerateSuggestions(ctx, info, 1)

		// assert
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no files to summarize")
	})

	t.Run("ParseSuggestionsJSON correct format", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		assert.NoError(t, err)

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: responseJSON},
						},
					},
				},
			},
		}

		// act
		suggestions, err := service.parseSuggestionsJSON(formatResponse(resp))

		// assert
		assert.NoError(t, err)
		assert.Equal(t, 1, len(suggestions), "Se esperaba 1 sugerencia")
		if len(suggestions) > 0 {
			suggestion := suggestions[0]
			assert.Equal(t, "refactor: Mejoras en la presentación de sugerencias y configuración de Jira", suggestion.CommitTitle)
			assert.Equal(t, 2, len(suggestion.Files), "Número incorrecto de archivos")
			assert.Contains(t, suggestion.Files, "cmd/main.go")
			assert.Contains(t, suggestion.Files, "internal/cli/command/config/set_jira_config.go")
			assert.Equal(t, "Se mejoró la salida de sugerencias y el manejo de errores en la configuración de Jira.", suggestion.Explanation)

			assert.Contains(t, suggestion.CodeAnalysis.ChangesOverview, "Mejora en el manejo de la configuración de Jira")
			assert.Contains(t, suggestion.CodeAnalysis.PrimaryPurpose, "Mejorar la experiencia del usuario")
			assert.Contains(t, suggestion.CodeAnalysis.TechnicalImpact, "Se modifican varias partes del código")

			assert.Equal(t, models.CriteriaPartiallyMet, suggestion.RequirementsAnalysis.CriteriaStatus)
			assert.Equal(t, 2, len(suggestion.RequirementsAnalysis.MissingCriteria))
			assert.Equal(t, 2, len(suggestion.RequirementsAnalysis.ImprovementSuggestions))
		}
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
		if result != expected {
			t.Errorf("formatResponse incorrecto. Esperado: %s, Obtenido: %s", expected, result)
		}
	})

	t.Run("generatePrompt with valid parameters", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:    "es",
			UseEmoji:    true,
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		assert.NoError(t, err)

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff contenido",
		}

		// act
		prompt := service.generatePrompt(cfg.Language, info, 3)

		// assert
		assert.Contains(t, prompt, "commit", "El prompt debería contener 'commit'")
		assert.Contains(t, prompt, "Archivos Modificados", "El prompt debería contener 'Archivos modificados'")
		assert.Contains(t, prompt, "feat", "El prompt debería contener tipos de commit")
		assert.Contains(t, prompt, "fix", "El prompt debería contener tipos de commit")
		assert.Contains(t, prompt, "refactor", "El prompt debería contener tipos de commit")
	})

	t.Run("generatePrompt with en locale", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:    "en",
			UseEmoji:    true,
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		assert.NoError(t, err)

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff content",
		}

		// act
		prompt := service.generatePrompt(cfg.Language, info, 3)

		// assert
		assert.Contains(t, prompt, "commit", "The prompt should contain 'commit'")
		assert.Contains(t, prompt, "Modified Files", "The prompt should contain 'Modified files'")
		assert.Contains(t, prompt, "feat", "The prompt should contain commit types")
		assert.Contains(t, prompt, "fix", "The prompt should contain commit types")
		assert.Contains(t, prompt, "refactor", "The prompt should contain commit types")
	})

	t.Run("generatePrompt with en locale", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:    "en",
			UseEmoji:    true,
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		// act
		service, err := NewGeminiCommitSummarizer(ctx, cfg, nil)
		assert.NoError(t, err)

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff content",
		}

		// act
		prompt := service.generatePrompt(cfg.Language, info, 3)

		t.Logf("Prompt generado:\n%s", prompt)

		// assert
		assert.Contains(t, prompt, "Generate 3 suggestions now", "The prompt should include the generation instruction")
		assert.Contains(t, prompt, "Modified Files", "Should include the modified files section")
		assert.Contains(t, prompt, "Code Changes", "Should include the diff section")
		assert.Contains(t, prompt, "technical analysis", "Should include the technical analysis section")
	})

	t.Run("parseSuggestionsJSON with nil response", func(t *testing.T) {
		// arrange
		service := &GeminiCommitSummarizer{}
		resp := (*genai.GenerateContentResponse)(nil)

		// act
		suggestions, err := service.parseSuggestionsJSON(formatResponse(resp))

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "empty response text from AI")
	})

	t.Run("parseSuggestionsJSON with empty candidates", func(t *testing.T) {
		// arrange
		service := &GeminiCommitSummarizer{}
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		// act
		suggestions, err := service.parseSuggestionsJSON(formatResponse(resp))

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "empty response text from AI")
	})

	t.Run("parseSuggestionsJSON with invalid JSON", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}
		// act
		service, _ := NewGeminiCommitSummarizer(ctx, cfg, nil)

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{Content: &genai.Content{Parts: []*genai.Part{{Text: "invalid json"}}}},
			},
		}

		// act
		suggestions, err := service.parseSuggestionsJSON(formatResponse(resp))

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "error parsing JSON")
	})

	t.Run("parseSuggestionsJSON status passthrough", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}}}
		// act
		service, _ := NewGeminiCommitSummarizer(ctx, cfg, nil)

		testCases := []struct {
			inputStatus    string
			expectedStatus models.CriteriaStatus
		}{
			{"full_met", models.CriteriaFullyMet},
			{"partially_met", models.CriteriaPartiallyMet},
			{"not_met", models.CriteriaNotMet},
			{"unknown_status", models.CriteriaStatus("unknown_status")},
		}

		for _, tc := range testCases {
			jsonStr := fmt.Sprintf(`[{
				"title": "test",
				"desc": "test",
				"files": ["test.go"],
				"requirements": {
					"status": "%s",
					"missing": [],
					"suggestions": []
				}
			}]`, tc.inputStatus)

			resp := &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: jsonStr}}}},
				},
			}

			// act
			suggestions, err := service.parseSuggestionsJSON(formatResponse(resp))

			// assert
			assert.NoError(t, err)
			assert.NotEmpty(t, suggestions)
			assert.Equal(t, tc.expectedStatus, suggestions[0].RequirementsAnalysis.CriteriaStatus, "Fallo passthrough para: %s", tc.inputStatus)
		}
	})

	t.Run("ensureIssueReference", func(t *testing.T) {
		service := &GeminiCommitSummarizer{}
		issueNum := 123
		suggestions := []models.CommitSuggestion{
			{CommitTitle: "feat: something"},
			{CommitTitle: "fix: bug (#123)"},
			{CommitTitle: "docs: update (#456)"},
			{CommitTitle: "refactor: code fixes #123"},
		}

		result := service.ensureIssueReference(suggestions, issueNum)

		assert.Equal(t, "feat: something (#123)", result[0].CommitTitle)
		assert.Equal(t, "fix: bug (#123)", result[1].CommitTitle)
		assert.Equal(t, "docs: update (#123)", result[2].CommitTitle)
		assert.Equal(t, "refactor: code fixes #123", result[3].CommitTitle)
	})
}

func TestGenerateSuggestions_HappyPath(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "mate-commit-test-suggestions-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpHome); err != nil {
			return
		}
	}()
	oldHome := os.Getenv("HOME")
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test"}},
		AIConfig:    config.AIConfig{Models: map[config.AI]config.Model{config.AIGemini: "gemini-pro"}},
		Language:    "en",
	}
	_ = os.Setenv("HOME", tmpHome)
	defer func() {
		if err := os.Setenv("HOME", oldHome); err != nil {
			return
		}
	}()

	ctx := context.Background()
	// act
	service, _ := NewGeminiCommitSummarizer(ctx, cfg, nil)
	service.wrapper.SetSkipConfirmation(true)

	t.Run("successful suggestions generation", func(t *testing.T) {
		service.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: responseJSON}}}},
				},
			}, &models.TokenUsage{TotalTokens: 200}, nil
		}

		info := models.CommitInfo{
			Files: []string{"main.go"},
			Diff:  "some diff",
		}
		suggestions, err := service.GenerateSuggestions(ctx, info, 1)

		assert.NoError(t, err)
		assert.NotEmpty(t, suggestions)
		assert.Equal(t, 1, len(suggestions))
		assert.Contains(t, suggestions[0].CommitTitle, "Mejoras")
	})
}

func TestGeneratePrompt_WithCriteria(t *testing.T) {
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test"}},
	}
	service := &GeminiCommitSummarizer{config: cfg}

	t.Run("formats criteria correctly in English", func(t *testing.T) {
		info := models.CommitInfo{
			Files: []string{"main.go"},
			Diff:  "diff",
			TicketInfo: &models.TicketInfo{
				TicketTitle: "Test Ticket",
				TitleDesc:   "Test Description",
				Criteria:    []string{"Crit 1", "Crit 2"},
			},
		}
		prompt := service.generatePrompt("en", info, 3)

		assert.Contains(t, prompt, "**Title:** Test Ticket")
		assert.Contains(t, prompt, "**Acceptance Criteria:**")
		assert.Contains(t, prompt, "- Crit 1")
		assert.Contains(t, prompt, "- Crit 2")
	})

	t.Run("formats criteria correctly in Spanish", func(t *testing.T) {
		info := models.CommitInfo{
			Files: []string{"main.go"},
			Diff:  "diff",
			TicketInfo: &models.TicketInfo{
				TicketTitle: "Ticket Test",
				TitleDesc:   "Desc Test",
				Criteria:    []string{"Crit 1"},
			},
		}
		prompt := service.generatePrompt("es", info, 3)

		assert.Contains(t, prompt, "**Título:** Ticket Test")
		assert.Contains(t, prompt, "**Criterios de Aceptación:**")
		assert.Contains(t, prompt, "- Crit 1")
	})
}
