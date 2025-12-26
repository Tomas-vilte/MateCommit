package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
)

func TestNewCommitSummarizer_TypedNilCheck(t *testing.T) {
	cfg := &config.Config{
		AIConfig: config.AIConfig{
			ActiveAI: "gemini",
		},
		AIProviders: map[string]config.AIProviderConfig{
			"gemini": {APIKey: ""}, // Trigger error
		},
	}

	summarizer, err := NewCommitSummarizer(context.Background(), cfg, nil)

	assert.Error(t, err)
	// Check if the interface itself is nil
	assert.True(t, summarizer == nil, "Summarizer interface should be truly nil, not a typed nil")
}

func TestNewPRSummarizer_TypedNilCheck(t *testing.T) {
	cfg := &config.Config{
		AIConfig: config.AIConfig{
			ActiveAI: "gemini",
		},
		AIProviders: map[string]config.AIProviderConfig{
			"gemini": {APIKey: ""},
		},
	}

	summarizer, err := NewPRSummarizer(context.Background(), cfg, nil)

	assert.Error(t, err)
	assert.True(t, summarizer == nil)
}

func TestNewIssueContentGenerator_TypedNilCheck(t *testing.T) {
	cfg := &config.Config{
		AIConfig: config.AIConfig{
			ActiveAI: "gemini",
		},
		AIProviders: map[string]config.AIProviderConfig{
			"gemini": {APIKey: ""},
		},
	}

	generator, err := NewIssueContentGenerator(context.Background(), cfg, nil)

	assert.Error(t, err)
	assert.True(t, generator == nil)
}
