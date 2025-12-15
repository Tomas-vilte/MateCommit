package gemini

import (
	"context"
	"fmt"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
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

func TestGeminiService(t *testing.T) {
	t.Run("NewGeminiService with empty API key", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)

		// act
		service, err := NewGeminiService(ctx, cfg, trans)

		// assert
		if service != nil {
			t.Error("El servicio no deberia crearse con API key vacia")
		}

		if err == nil {
			t.Error("Deberia retornar un error con API key vacia")
		}
	})

	t.Run("GenerateSuggestions with invalid count", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		if err != nil {
			t.Fatalf("Error al crear el traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, cfg, trans)
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
		if suggestions != nil {
			t.Error("No deberian generarse sugerencias con count <= 0")
		}

		if err == nil {
			t.Error("Deberia retornar un error con count <= 0")
		}
	})

	t.Run("GenerateSuggestions no files", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		if err != nil {
			t.Fatalf("Error creando traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, cfg, trans)
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
		if suggestions != nil {
			t.Error("No deberian generarse sugerencias sin archivos")
		}
		if err == nil {
			t.Error("Deberia retornar un error sin archivos")
		}
	})

	t.Run("ParseSuggestionsJSON correct format", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
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
		suggestions, err := service.parseSuggestionsJSON(resp)

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

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
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
		assert.Contains(t, prompt, "explicación", "El prompt debería contener 'Explicación'")
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

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
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
		assert.Contains(t, prompt, "explanation", "The prompt should contain 'Explanation'")
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

		trans, err := i18n.NewTranslations("es", "../../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
		assert.NoError(t, err)

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff content",
		}

		// act
		prompt := service.generatePrompt(cfg.Language, info, 3)

		t.Logf("Prompt generado:\n%s", prompt)

		// assert
		assert.Contains(t, prompt, "Generate 3 commit message suggestions", "El prompt debe incluir la instrucción de generación")
		assert.Contains(t, prompt, "Modified Files", "Debe incluir la sección de archivos modificados")
		assert.Contains(t, prompt, "Code Changes", "Debe incluir la sección de diff")
		assert.Contains(t, prompt, "technical analysis", "Debe incluir la sección de análisis técnico")
	})

	t.Run("parseSuggestionsJSON with nil response", func(t *testing.T) {
		// arrange
		service := &GeminiService{}
		resp := (*genai.GenerateContentResponse)(nil)

		// act
		suggestions, err := service.parseSuggestionsJSON(resp)

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "respuesta vacía")
	})

	t.Run("parseSuggestionsJSON with empty candidates", func(t *testing.T) {
		// arrange
		service := &GeminiService{}
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		// act
		suggestions, err := service.parseSuggestionsJSON(resp)

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "respuesta vacía")
	})

	t.Run("parseSuggestionsJSON with invalid JSON", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		}
		trans, _ := i18n.NewTranslations("es", "../../../i18n/locales/")
		service, _ := NewGeminiService(ctx, cfg, trans)

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{Content: &genai.Content{Parts: []*genai.Part{{Text: "invalid json"}}}},
			},
		}

		// act
		suggestions, err := service.parseSuggestionsJSON(resp)

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "parsear JSON")
	})

	t.Run("parseSuggestionsJSON status passthrough", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}}}
		trans, _ := i18n.NewTranslations("es", "../../../i18n/locales/")
		service, _ := NewGeminiService(ctx, cfg, trans)

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
			suggestions, err := service.parseSuggestionsJSON(resp)

			// assert
			assert.NoError(t, err)
			assert.NotEmpty(t, suggestions)
			assert.Equal(t, tc.expectedStatus, suggestions[0].RequirementsAnalysis.CriteriaStatus, "Fallo passthrough para: %s", tc.inputStatus)
		}
	})
}
