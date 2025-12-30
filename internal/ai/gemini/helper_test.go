package gemini

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestExtractUsage(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		assert.Nil(t, extractUsage(nil))
	})

	t.Run("nil UsageMetadata", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{}
		assert.Nil(t, extractUsage(resp))
	})

	t.Run("valid UsageMetadata", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
		}
		usage := extractUsage(resp)
		assert.NotNil(t, usage)
		assert.Equal(t, 10, usage.InputTokens)
		assert.Equal(t, 20, usage.OutputTokens)
		assert.Equal(t, 30, usage.TotalTokens)
	})
}

func TestGetGenerateConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cfg := GetGenerateConfig("gemini-1.5-flash", "", nil)
		assert.NotNil(t, cfg)
		assert.Equal(t, float32(0.3), *cfg.Temperature)
		assert.Empty(t, cfg.ResponseMIMEType)
		assert.Nil(t, cfg.ThinkingConfig)
	})

	t.Run("json response type", func(t *testing.T) {
		cfg := GetGenerateConfig("gemini-1.5-flash", "application/json", nil)
		assert.Equal(t, "application/json", cfg.ResponseMIMEType)
	})

	t.Run("Thinking Mode for gemini-3", func(t *testing.T) {
		cfg := GetGenerateConfig("gemini-3-flash-preview", "", nil)
		assert.NotNil(t, cfg.ThinkingConfig)
		assert.True(t, cfg.ThinkingConfig.IncludeThoughts)
		assert.Equal(t, genai.ThinkingLevelHigh, cfg.ThinkingConfig.ThinkingLevel)
	})
}
