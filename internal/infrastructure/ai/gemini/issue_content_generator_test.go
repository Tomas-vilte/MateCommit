package gemini

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

func TestNewGeminiIssueContentGenerator(t *testing.T) {
	trans, err := i18n.NewTranslations("en", "../../../i18n/locales/")
	require.NoError(t, err)

	t.Run("should return error if API key is missing", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}
		gen, err := NewGeminiIssueContentGenerator(context.Background(), cfg, trans)
		assert.Error(t, err)
		assert.Nil(t, gen)
		assert.Contains(t, err.Error(), "API key is not configured")
	})

	t.Run("should create generator if API key is present", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{
				"gemini": {APIKey: "fake-key"},
			},
		}
		gen, err := NewGeminiIssueContentGenerator(context.Background(), cfg, trans)
		assert.NoError(t, err)
		assert.NotNil(t, gen)
	})
}

func TestBuildIssuePrompt(t *testing.T) {
	trans, _ := i18n.NewTranslations("en", "../../../i18n/locales/")
	cfg := &config.Config{}
	gen := &GeminiIssueContentGenerator{
		config: cfg,
		trans:  trans,
	}

	tests := []struct {
		name     string
		request  models.IssueGenerationRequest
		contains []string
	}{
		{
			name: "from diff only",
			request: models.IssueGenerationRequest{
				Diff:     "test diff",
				Language: "en",
			},
			contains: []string{"Code Changes (git diff)", "test diff"},
		},
		{
			name: "from description only",
			request: models.IssueGenerationRequest{
				Description: "user description",
				Language:    "en",
			},
			contains: []string{"Global Description: user description"},
		},
		{
			name: "full request",
			request: models.IssueGenerationRequest{
				Diff:        "test diff",
				Description: "user description",
				Hint:        "special hint",
				Language:    "es",
			},
			contains: []string{"Code Changes (git diff)", "user description", "special hint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := gen.buildIssuePrompt(tt.request)
			for _, c := range tt.contains {
				assert.Contains(t, prompt, c)
			}
		})
	}
}

func TestParseIssueResponse(t *testing.T) {
	gen := &GeminiIssueContentGenerator{}

	t.Run("valid JSON response", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: `{"title": "Bug Fix", "description": "Fixed a bug", "labels": ["fix", "test"]}`},
						},
					},
				},
			},
		}

		result, err := gen.parseIssueResponse(resp)
		assert.NoError(t, err)
		assert.Equal(t, "Bug Fix", result.Title)
		assert.Equal(t, "Fixed a bug", result.Description)
		assert.ElementsMatch(t, []string{"fix", "test"}, result.Labels)
	})

	t.Run("invalid JSON response - fallback", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "This is not JSON but raw text"},
						},
					},
				},
			},
		}

		result, err := gen.parseIssueResponse(resp)
		assert.NoError(t, err)
		assert.Equal(t, "Generated Issue", result.Title)
		assert.Equal(t, "This is not JSON but raw text", result.Description)
	})

	t.Run("empty response", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		result, err := gen.parseIssueResponse(resp)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestCleanLabels(t *testing.T) {
	gen := &GeminiIssueContentGenerator{}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "only allowed labels",
			input:    []string{"fix", "feature", "bug", "invalid"},
			expected: []string{"fix", "feature"},
		},
		{
			name:     "mixed case and spaces",
			input:    []string{"  Fix ", "FEATURE", "test"},
			expected: []string{"fix", "feature", "test"},
		},
		{
			name:     "duplicates",
			input:    []string{"fix", "fix", "FIX"},
			expected: []string{"fix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.cleanLabels(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
