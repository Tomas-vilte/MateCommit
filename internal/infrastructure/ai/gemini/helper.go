package gemini

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"google.golang.org/genai"
)

// extractUsage extrae los metadatos de uso de la respuesta de Gemini
func extractUsage(resp *genai.GenerateContentResponse) *models.TokenUsage {
	if resp == nil || resp.UsageMetadata == nil {
		return nil
	}
	return &models.TokenUsage{
		InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
		OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		TotalTokens:  int(resp.UsageMetadata.TotalTokenCount),
	}
}

// GetGenerateConfig retorna la configuración óptima para el modelo, activando Thinking Mode si es compatible.
func GetGenerateConfig(modelName string, responseType string) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature:     float32Ptr(0.3),
		MaxOutputTokens: int32(10000),
		MediaResolution: genai.MediaResolutionHigh,
	}

	if responseType == "application/json" {
		config.ResponseMIMEType = "application/json"
	}

	if strings.HasPrefix(modelName, "gemini-3") {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingLevel:   genai.ThinkingLevelHigh,
		}
	}

	return config
}

// ExtractJSON intenta extraer un bloque JSON válido de un texto, manejando bloques de código markdown
// y posible texto extra que los modelos con "Thinking" pueden generar.
func ExtractJSON(text string) string {
	text = strings.TrimSpace(text)

	// 1. Intentar encontrar JSON en bloques de código markdown
	re := regexp.MustCompile("(?s)```(?:json)?\n?(.*?)```")
	matches := re.FindAllStringSubmatch(text, -1)
	var bestMarkdown string
	for _, m := range matches {
		if len(m) > 1 {
			content := strings.TrimSpace(m[1])
			sanitized := SanitizeJSON(content)
			if json.Valid([]byte(sanitized)) {
				if len(sanitized) > len(bestMarkdown) {
					bestMarkdown = sanitized
				}
			}
		}
	}
	if bestMarkdown != "" {
		return bestMarkdown
	}

	// 2. Buscar bloques balanceados ({...} o [...]) y quedarse con el más largo que sea JSON válido
	var bestBlock string
	for i := 0; i < len(text); {
		startIdx := strings.IndexAny(text[i:], "{[")
		if startIdx == -1 {
			break
		}
		startIdx += i

		opener := text[startIdx]
		var closer byte
		if opener == '{' {
			closer = '}'
		} else {
			closer = ']'
		}

		count := 0
		inString := false
		escaped := false
		foundEnd := false
		endIdx := -1

		for j := startIdx; j < len(text); j++ {
			char := text[j]
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = !inString
				continue
			}

			if !inString {
				if char == opener {
					count++
				} else if char == closer {
					count--
					if count == 0 {
						foundEnd = true
						endIdx = j
						break
					}
				}
			}
		}

		if foundEnd {
			block := text[startIdx : endIdx+1]
			sanitized := SanitizeJSON(block)
			if json.Valid([]byte(sanitized)) {
				if len(sanitized) > len(bestBlock) {
					bestBlock = sanitized
				}
			}
			i = endIdx + 1
		} else {
			i = startIdx + 1
		}
	}

	if bestBlock != "" {
		return bestBlock
	}

	// Fallback: si nada funcionó, sanear y devolver el texto original
	return SanitizeJSON(text)
}

var jsonStringRegex = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)

// SanitizeJSON limpia el JSON malformado que a veces generan los LLMs,
// como saltos de línea sin escapar dentro de Literales de Cadena.
func SanitizeJSON(s string) string {
	// Reemplazar saltos de línea crudos dentro de los strings JSON por \n escapados
	return jsonStringRegex.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ReplaceAll(m, "\n", "\\n")
	})
}

func float32Ptr(f float32) *float32 {
	return &f
}
