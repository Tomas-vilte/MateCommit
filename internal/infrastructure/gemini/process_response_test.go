package gemini

import (
	"encoding/json"
	"os"
	"testing"
)

type CustomResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"Text"`
			} `json:"Parts"`
			Role string `json:"Role"`
		} `json:"Content"`
		FinishReason  int           `json:"FinishReason"`
		SafetyRatings []interface{} `json:"SafetyRatings"`
		TokenCount    int           `json:"TokenCount"`
	} `json:"Candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"PromptTokenCount"`
		CandidatesTokenCount int `json:"CandidatesTokenCount"`
		TotalTokenCount      int `json:"TotalTokenCount"`
	} `json:"UsageMetadata"`
}

func TestProcessResponse(t *testing.T) {
	t.Run("Test con 2 sugerencias", func(t *testing.T) {
		fileContent, err := os.ReadFile("response_mock.json")
		if err != nil {
			t.Fatalf("Error al leer el archivo JSON: %v", err)
		}
		var resp CustomResponse
		if err := json.Unmarshal(fileContent, &resp); err != nil {
			t.Fatalf("Error al unmarshal JSON: %v", err)
		}

		var responseText string
		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
			responseText = resp.Candidates[0].Content.Parts[0].Text
		}

		suggestions := processResponse(responseText)

		expected := []string{
			"Sugerencia 1: feat: Refactoriza la CLI para permitir múltiples sugerencias\nExplicación: Este commit refactoriza la CLI para permitir a los usuarios generar múltiples sugerencias de mensajes de commit utilizando la opción `--count`, también se añadió la opción `--format` para cambiar el formato de los mensajes (conventional, gitmoji). Además, se ajusta el servicio `CommitService` y el provider `GeminiService` para que puedan manejar la generación de múltiples sugerencias.",
			"Sugerencia 2: feat: Implementa nueva CLI con opciones de sugerencias y formato\nExplicación: Este commit añade la biblioteca `urfave/cli/v3` para implementar una nueva CLI, permitiendo a los usuarios solicitar multiples sugerencias de mensajes de commit, usando la opción `--count` y cambiar el formato con la opción `--format`. Se añadió el soporte para diferentes formatos de commit (conventional y gitmoji) en el servicio `GeminiService`.",
			"Sugerencia 3: feat: Mejora la CLI para generación de sugerencias de commit\nExplicación: Este commit introduce una nueva CLI que utiliza `urfave/cli/v3` para generar sugerencias de commit, se añadió soporte para diferentes formatos de commit (conventional y gitmoji). Se implementó la lógica para generar una cantidad configurable de sugerencias y se mejoró la generación en el servicio `GeminiService`.",
		}

		if !equal(suggestions, expected) {
			t.Errorf("processResponse() = %v, want %v", suggestions, expected)
		}
	})
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
