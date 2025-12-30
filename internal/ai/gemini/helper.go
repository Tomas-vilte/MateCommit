package gemini

import (
	"strings"

	"github.com/thomas-vilte/matecommit/internal/models"
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
func GetGenerateConfig(modelName string, responseType string, schema *genai.Schema) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{
		Temperature:     float32Ptr(0.3),
		MaxOutputTokens: int32(10000),
		MediaResolution: genai.MediaResolutionHigh,
	}

	if responseType == "application/json" {
		config.ResponseMIMEType = "application/json"
		if schema != nil {
			config.ResponseJsonSchema = schema
		}
	}

	if strings.HasPrefix(modelName, "gemini-3") {
		config.ThinkingConfig = &genai.ThinkingConfig{
			IncludeThoughts: true,
			ThinkingLevel:   genai.ThinkingLevelHigh,
		}
	}

	return config
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
