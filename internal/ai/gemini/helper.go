package gemini

import (
	"encoding/json"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"google.golang.org/genai"
)

// extractUsage extracts usage metadata from the Gemini response
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

// GetGenerateConfig returns the optimal configuration for the model, enabling Thinking Mode if compatible.
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

// ExtractJSON attempts to extract a valid JSON block from text, handling Markdown code blocks
// and possible extra text that models with "Thinking" mode might generate.
func ExtractJSON(text string) string {
	text = strings.TrimSpace(text)

	matches := regex.MarkdownJSONBlock.FindAllStringSubmatch(text, -1)
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

	return SanitizeJSON(text)
}

// SanitizeJSON cleans malformed JSON that LLMs sometimes generate,
// such as unescaped newlines within String Literals.
func SanitizeJSON(s string) string {
	return regex.JSONString.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ReplaceAll(m, "\n", "\\n")
	})
}

func float32Ptr(f float32) *float32 {
	return &f
}

// extractTextFromMap extracts text from a cached response that was deserialized from JSON.
// The map structure mirrors genai.GenerateContentResponse but as map[string]interface{}.
// JSON tags use lowercase keys: "candidates", "content", "parts", "text", "thought"
func extractTextFromMap(respMap map[string]interface{}) string {
	var result strings.Builder

	// Navigate: respMap["candidates"] -> []interface{} of candidates
	candidates, ok := respMap["candidates"]
	if !ok {
		return ""
	}

	candidatesList, ok := candidates.([]interface{})
	if !ok {
		return ""
	}

	for _, cand := range candidatesList {
		candMap, ok := cand.(map[string]interface{})
		if !ok {
			continue
		}

		// Navigate: candMap["content"] -> map with parts
		content, ok := candMap["content"]
		if !ok {
			continue
		}

		contentMap, ok := content.(map[string]interface{})
		if !ok {
			continue
		}

		// Navigate: contentMap["parts"] -> []interface{} of parts
		parts, ok := contentMap["parts"]
		if !ok {
			continue
		}

		partsList, ok := parts.([]interface{})
		if !ok {
			continue
		}

		for _, part := range partsList {
			partMap, ok := part.(map[string]interface{})
			if !ok {
				continue
			}

			// Check if this is a thinking part (skip it)
			if thought, ok := partMap["thought"].(bool); ok && thought {
				continue
			}

			// Extract text from part
			if text, ok := partMap["text"].(string); ok && text != "" {
				result.WriteString(text)
			}
		}
	}

	return result.String()
}
