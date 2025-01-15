package gemini

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/generative-ai-go/genai"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

const (
	responseText = "=========[ Sugerencia ]=========\n1. Primera sugerencia:\nCommit: refactor: mejorar el manejo de errores de GitService\nArchivos: internal/infrastructure/git/git_service.go, internal/infrastructure/git/git_service_test.go\nExplicación: Manejo de errores mejorado en GitService, específicamente para los métodos HasStaggedChanges y GetDiff. Se agregaron pruebas más sólidas."
)

func TestGeminiService(t *testing.T) {
	t.Run("NewGeminiService with empty API key", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.CommitConfig{}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("es")
		assert.NoError(t, err)

		// act
		service, err := NewGeminiService(ctx, "", cfg, trans)

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
		cfg := &config.CommitConfig{}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("es")
		service, err := NewGeminiService(ctx, "test-api-key", cfg, trans)
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
		cfg := &config.CommitConfig{}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("es")
		if err != nil {
			t.Fatalf("Error creando traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, "test-api-key", cfg, trans)
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
		cfg := &config.CommitConfig{}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("es")
		if err != nil {
			t.Fatalf("Error al crear el traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, "test-api-key", cfg, trans)
		if err != nil {
			t.Fatalf("Error al crear el servicio: %v", err)
		}

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
		if len(suggestions) != 1 {
			t.Errorf("Se esperaba 1 sugerencia, se obtuvieron %d", len(suggestions))
		}

		if len(suggestions) > 0 {
			suggestion := suggestions[0]
			if suggestion.CommitTitle != "refactor: mejorar el manejo de errores de GitService" {
				t.Errorf("Titulo incorrecto: %s", suggestion.CommitTitle)
			}
			if len(suggestion.Files) != 2 {
				t.Errorf("Numero incorrecto de archivos: %d", len(suggestion.Files))
			}
			if suggestion.Explanation == "" {
				t.Error("La explicación no debería estar vacía")
			}
		}
	})

	t.Run("formatChanges", func(t *testing.T) {
		// arrange
		files := []string{"test.txt", "main.go"}

		// Act
		result := formatChanges(files)

		// Assert
		expected := "- test.txt\n- main.go"
		if result != expected {
			t.Errorf("formatChanges incorrecto. Esperado: %s, Obtenido: %s", expected, result)
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
		cfg := &config.CommitConfig{
			Locale:   config.CommitLocale{Lang: "es"},
			UseEmoji: true,
		}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("es")
		if err != nil {
			t.Fatalf("Error creando traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, "test-api-key", cfg, trans)
		if err != nil {
			t.Fatalf("Error creando servicio: %v", err)
		}

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff contenido",
		}

		// act
		prompt := service.generatePrompt(cfg.Locale, info, 3)

		// assert
		if !strings.Contains(prompt, "commit") || !strings.Contains(prompt, "Archivos") || !strings.Contains(prompt, "Explicación") {
			t.Errorf("generatePrompt incorrecto. El prompt no contiene los elementos esenciales.")
		}

		// Verificar que se utiliza el emoji en el mensaje del commit
		if strings.Contains(prompt, "[emoji]") {
			t.Log("Emojis utilizados correctamente")
		} else {
			t.Error("Se esperaba que se usaran emojis en el prompt")
		}
	})

	t.Run("generatePrompt with en locale", func(t *testing.T) {
		// arrange
		ctx := context.Background()
		cfg := &config.CommitConfig{
			Locale:   config.CommitLocale{Lang: "en"},
			UseEmoji: true,
		}
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		if err := os.Chdir("../../.."); err != nil {
			t.Fatalf("Error al cambiar de directorio: %v", err)
		}
		trans, err := i18n.NewTranslations("en")
		if err != nil {
			t.Fatalf("Error al crear el traductor: %v", err)
		}
		service, err := NewGeminiService(ctx, "test-api-key", cfg, trans)
		if err != nil {
			t.Fatalf("Error creating service: %v", err)
		}

		info := models.CommitInfo{
			Files: []string{"test.txt", "main.go"},
			Diff:  "diff content",
		}

		// act
		prompt := service.generatePrompt(cfg.Locale, info, 3)

		if !strings.Contains(prompt, "commit") || !strings.Contains(prompt, "Files") || !strings.Contains(prompt, "Explanation") {
			t.Errorf("generatePrompt incorrecto. Prompt no contiene elementos esenciales en inglés.")
		}

		if strings.Contains(prompt, "[emoji]") {
			t.Log("Emojis usados correctamente en el mensaje en inglés")
		} else {
			t.Error("Se espera que los emojis se utilicen en el mensaje en inglés")
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
