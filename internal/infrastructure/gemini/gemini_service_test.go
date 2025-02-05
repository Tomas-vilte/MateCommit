package gemini

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/generative-ai-go/genai"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	responseText = `📊 Análisis de Código:
- Resumen de Cambios: Mejora en el manejo de la configuración de Jira y la presentación de sugerencias de commit.
- Propósito Principal: Mejorar la experiencia del usuario al mostrar información más detallada.
- Impacto Técnico: Se modifican varias partes del código para mejorar la estructura.

📝 Sugerencias:
Commit: refactor: Mejoras en la presentación de sugerencias y configuración de Jira
📄 Archivos modificados:
   - cmd/main.go
   - internal/cli/command/config/set_jira_config.go
Explicación: Se mejoró la salida de sugerencias y el manejo de errores en la configuración de Jira.

🎯 Análisis de Criterios de Aceptación:
⚠️ Estado de los Criterios: Cumplimiento Parcial
❌ Criterios Faltantes:
   - Conexión a la API de Jira
   - Extracción de Tickets
💡 Sugerencias de Mejora:
   - Implementar manejo de errores para token expirado
   - Agregar retry mechanism para API no disponible`
)

func TestGeminiService(t *testing.T) {
	t.Run("NewGeminiService with empty API key", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "",
		}

		trans, err := i18n.NewTranslations("es", "../../i18n/locales/")
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
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("es", "../../i18n/locales/")
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
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("es", "../../i18n/locales/")
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

	t.Run("ParseSuggestions correct format", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("es", "../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
		assert.NoError(t, err)

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []genai.Part{
							genai.Text(responseText),
						},
					},
				},
			},
		}

		// act
		suggestions := service.parseSuggestions(resp)

		// assert
		assert.Equal(t, 1, len(suggestions), "Se esperaba 1 sugerencia")
		if len(suggestions) > 0 {
			suggestion := suggestions[0]
			assert.Equal(t, "refactor: Mejoras en la presentación de sugerencias y configuración de Jira", suggestion.CommitTitle)
			assert.Equal(t, 2, len(suggestion.Files), "Número incorrecto de archivos")
			assert.Contains(t, suggestion.Files, "cmd/main.go")
			assert.Contains(t, suggestion.Files, "internal/cli/command/config/set_jira_config.go")
			assert.Equal(t, "Se mejoró la salida de sugerencias y el manejo de errores en la configuración de Jira.", suggestion.Explanation)

			// Verificar análisis de código
			assert.Contains(t, suggestion.CodeAnalysis.ChangesOverview, "Mejora en el manejo de la configuración de Jira")
			assert.Contains(t, suggestion.CodeAnalysis.PrimaryPurpose, "Mejorar la experiencia del usuario")
			assert.Contains(t, suggestion.CodeAnalysis.TechnicalImpact, "Se modifican varias partes del código")

			// Verificar análisis de requisitos
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
		if result != expected {
			t.Errorf("formatResponse incorrecto. Esperado: %s, Obtenido: %s", expected, result)
		}
	})

	t.Run("generatePrompt with valid parameters", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:     "es",
			UseEmoji:     true,
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("es", "../../i18n/locales/")
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
		assert.Contains(t, prompt, "Archivos modificados", "El prompt debería contener 'Archivos modificados'")
		assert.Contains(t, prompt, "Explicación", "El prompt debería contener 'Explicación'")
		assert.Contains(t, prompt, "🔍", "El prompt debería contener el emoji de análisis")
		assert.Contains(t, prompt, "feat:", "El prompt debería contener tipos de commit")
		assert.Contains(t, prompt, "fix:", "El prompt debería contener tipos de commit")
		assert.Contains(t, prompt, "refactor:", "El prompt debería contener tipos de commit")
	})

	t.Run("generatePrompt with en locale", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:     "en",
			UseEmoji:     true,
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("en", "../../i18n/locales/")
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
		assert.Contains(t, prompt, "Modified files", "The prompt should contain 'Modified files'")
		assert.Contains(t, prompt, "Explanation", "The prompt should contain 'Explanation'")
		assert.Contains(t, prompt, "🔍", "The prompt should contain the analysis emoji")
		assert.Contains(t, prompt, "feat:", "The prompt should contain commit types")
		assert.Contains(t, prompt, "fix:", "The prompt should contain commit types")
		assert.Contains(t, prompt, "refactor:", "The prompt should contain commit types")
	})

	t.Run("generatePrompt with en locale", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.Config{
			Language:     "en",
			UseEmoji:     true, // Emoji activado, pero opcional en inglés
			GeminiAPIKey: "test-api-key",
		}

		trans, err := i18n.NewTranslations("en", "../../i18n/locales/")
		assert.NoError(t, err)
		service, err := NewGeminiService(ctx, cfg, trans)
		assert.NoError(t, err)

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff content",
		}

		// act
		prompt := service.generatePrompt(cfg.Language, info, 3)

		// Imprimir el prompt generado para depuración
		t.Logf("Prompt generado:\n%s", prompt)

		// assert
		assert.Contains(t, prompt, "Generate 3 commit message suggestions", "El prompt debe incluir la instrucción de generación")
		assert.Contains(t, prompt, "Modified files:", "Debe incluir la sección de archivos modificados")
		assert.Contains(t, prompt, "Diff:", "Debe incluir la sección de diff")
		assert.Contains(t, prompt, "Technical Analysis:", "Debe incluir la sección de análisis técnico")

		// Verificación opcional de emojis
		if cfg.UseEmoji {
			assert.Contains(t, prompt, "🔍", "El prompt debería contener el emoji de análisis si está activado")
		}
	})

	t.Run("parseSuggestions with nil response", func(t *testing.T) {
		// arrange
		service := &GeminiService{}
		resp := (*genai.GenerateContentResponse)(nil)

		// act
		suggestions := service.parseSuggestions(resp)

		// assert
		if suggestions != nil {
			t.Errorf("Expected nil, got: %v", suggestions)
		}
	})

	t.Run("parseSuggestions with empty candidates", func(t *testing.T) {
		// arrange
		service := &GeminiService{}
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		// act
		suggestions := service.parseSuggestions(resp)
		// assert
		if suggestions != nil {
			t.Errorf("Expected nil, got: %v", suggestions)
		}
	})
}
