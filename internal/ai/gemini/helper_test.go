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
		cfg := GetGenerateConfig("gemini-1.5-flash", "")
		assert.NotNil(t, cfg)
		assert.Equal(t, float32(0.3), *cfg.Temperature)
		assert.Empty(t, cfg.ResponseMIMEType)
		assert.Nil(t, cfg.ThinkingConfig)
	})

	t.Run("json response type", func(t *testing.T) {
		cfg := GetGenerateConfig("gemini-1.5-flash", "application/json")
		assert.Equal(t, "application/json", cfg.ResponseMIMEType)
	})

	t.Run("Thinking Mode for gemini-3", func(t *testing.T) {
		cfg := GetGenerateConfig("gemini-3-flash-preview", "")
		assert.NotNil(t, cfg.ThinkingConfig)
		assert.True(t, cfg.ThinkingConfig.IncludeThoughts)
		assert.Equal(t, genai.ThinkingLevelHigh, cfg.ThinkingConfig.ThinkingLevel)
	})
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pure JSON object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "pure JSON array",
			input:    `[{"key": "value"}]`,
			expected: `[{"key": "value"}]`,
		},
		{
			name:     "markdown code block with json tag",
			input:    "Sure, here is the JSON:\n```json\n{\"key\": \"value\"}\n```\nHope it helps!",
			expected: `{"key": "value"}`,
		},
		{
			name:     "markdown code block without tag",
			input:    "```\n[1, 2, 3]\n```",
			expected: `[1, 2, 3]`,
		},
		{
			name:     "text before and after JSON object",
			input:    "Some thinking content... {\"title\": \"fix bug\"} more text",
			expected: `{"title": "fix bug"}`,
		},
		{
			name:     "text before and after JSON array",
			input:    "Reasoning: ... [{\"title\": \"feat\"}] end",
			expected: `[{"title": "feat"}]`,
		},
		{
			name:     "balanced matching with stray brackets in prose",
			input:    "Thoughts [about stuff]: {\"key\": \"value\"} More text [end]",
			expected: `{"key": "value"}`,
		},
		{
			name: "unescaped newlines in JSON string",
			input: `{"desc": "This is a
multi-line
description"}`,
			expected: `{"desc": "This is a\nmulti-line\ndescription"}`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
