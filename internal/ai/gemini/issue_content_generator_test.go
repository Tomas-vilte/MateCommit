package gemini

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
	"google.golang.org/genai"
)

func TestNewGeminiIssueContentGenerator(t *testing.T) {
	t.Run("should return error if API key is missing", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}
		gen, err := NewGeminiIssueContentGenerator(context.Background(), cfg, nil)
		assert.Error(t, err)
		assert.Nil(t, gen)
		assert.Contains(t, err.Error(), "API key is missing")
	})

	t.Run("should create generator if API key is present", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{
				"gemini": {APIKey: "fake-key"},
			},
		}
		gen, err := NewGeminiIssueContentGenerator(context.Background(), cfg, nil)
		assert.NoError(t, err)
		assert.NotNil(t, gen)
	})
}

func TestBuildIssuePrompt(t *testing.T) {
	cfg := &config.Config{}
	gen := &GeminiIssueContentGenerator{
		config: cfg,
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
			contains: []string{"user description"},
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

func TestBuildIssuePrompt_WithTemplate(t *testing.T) {
	cfg := &config.Config{}
	gen := &GeminiIssueContentGenerator{
		config: cfg,
	}

	t.Run("adds final JSON reminder when template is present", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "Bug Report",
			Title:       "Bug: {{title}}",
			BodyContent: "## Description\n{{description}}",
		}

		request := models.IssueGenerationRequest{
			Diff:     "test diff",
			Template: template,
			Language: "en",
		}

		prompt := gen.buildIssuePrompt(request)

		// Should contain the template
		assert.Contains(t, prompt, "Bug Report")

		// Should contain the final JSON reminder
		assert.Contains(t, prompt, "🚨 FINAL REMINDER - CRITICAL OUTPUT REQUIREMENT 🚨")
		assert.Contains(t, prompt, "YOU MUST OUTPUT **ONLY** VALID JSON")
		assert.Contains(t, prompt, "BEGIN YOUR JSON OUTPUT NOW:")

		// Should contain instructions about using template in description field
		assert.Contains(t, prompt, "The template structure above should be used to FILL the \"description\" field")

		// Should contain prohibitions
		assert.Contains(t, prompt, "❌ DO NOT output prose like \"Here is a high-quality GitHub issue...\"")
		assert.Contains(t, prompt, "❌ DO NOT output markdown text directly")

		// Verify the reminder is at the end
		lastIndex := len(prompt) - 500
		if lastIndex < 0 {
			lastIndex = 0
		}
		finalSection := prompt[lastIndex:]
		assert.Contains(t, finalSection, "BEGIN YOUR JSON OUTPUT NOW:")
	})

	t.Run("does NOT add final reminder when no template", func(t *testing.T) {
		request := models.IssueGenerationRequest{
			Diff:     "test diff",
			Template: nil,
			Language: "en",
		}

		prompt := gen.buildIssuePrompt(request)

		// Should NOT contain the final JSON reminder
		assert.NotContains(t, prompt, "🚨 FINAL REMINDER - CRITICAL OUTPUT REQUIREMENT 🚨")
		assert.NotContains(t, prompt, "BEGIN YOUR JSON OUTPUT NOW:")
	})

	t.Run("includes template in Spanish", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "Reporte de Bug",
			Title:       "Bug: {{title}}",
			BodyContent: "## Descripción\n{{description}}",
		}

		request := models.IssueGenerationRequest{
			Description: "descripción del problema",
			Template:    template,
			Language:    "es",
		}

		prompt := gen.buildIssuePrompt(request)

		// Should contain the template
		assert.Contains(t, prompt, "Reporte de Bug")

		// Should still contain the final JSON reminder (in English for consistency)
		assert.Contains(t, prompt, "🚨 FINAL REMINDER - CRITICAL OUTPUT REQUIREMENT 🚨")
		assert.Contains(t, prompt, "BEGIN YOUR JSON OUTPUT NOW:")
	})

	t.Run("handles template with all fields", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name:        "Feature Request",
			Title:       "Feature: {{title}}",
			BodyContent: "## Problem\n{{problem}}\n## Solution\n{{solution}}",
			Labels:      []string{"enhancement", "feature"},
		}

		request := models.IssueGenerationRequest{
			Diff:         "test diff",
			Template:     template,
			Language:     "en",
			ChangedFiles: []string{"main.go", "test.go"},
		}

		prompt := gen.buildIssuePrompt(request)

		// Should contain template information
		assert.Contains(t, prompt, "Feature Request")

		// Should contain changed files
		assert.Contains(t, prompt, "main.go")
		assert.Contains(t, prompt, "test.go")

		// Should contain the final reminder
		assert.Contains(t, prompt, "🚨 FINAL REMINDER - CRITICAL OUTPUT REQUIREMENT 🚨")
		assert.Contains(t, prompt, "BEGIN YOUR JSON OUTPUT NOW:")
	})

	t.Run("reminder contains complete JSON structure example", func(t *testing.T) {
		template := &models.IssueTemplate{
			Name: "Test Template",
		}

		request := models.IssueGenerationRequest{
			Template: template,
			Language: "en",
		}

		prompt := gen.buildIssuePrompt(request)

		// Should show the expected JSON structure
		assert.Contains(t, prompt, `"title": "string here"`)
		assert.Contains(t, prompt, `"description": "markdown content following the template structure"`)
		assert.Contains(t, prompt, `"labels": ["array", "of", "strings"]`)
	})
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

		result, err := gen.parseIssueResponse(formatResponse(resp))
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

		result, err := gen.parseIssueResponse(formatResponse(resp))
		assert.NoError(t, err)
		assert.Equal(t, "Generated Issue", result.Title)
		assert.Equal(t, "This is not JSON but raw text", result.Description)
	})

	t.Run("empty response", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{},
		}

		result, err := gen.parseIssueResponse(formatResponse(resp))
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "empty response from AI")
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

func TestExtractTextFromMap(t *testing.T) {
	t.Run("extracts text from valid map structure", func(t *testing.T) {
		respMap := map[string]interface{}{
			"candidates": []interface{}{
				map[string]interface{}{
					"content": map[string]interface{}{
						"parts": []interface{}{
							map[string]interface{}{
								"text":    "This is the response",
								"thought": false,
							},
						},
					},
				},
			},
		}

		result := extractTextFromMap(respMap)
		assert.Equal(t, "This is the response", result)
	})

	t.Run("skips thinking parts", func(t *testing.T) {
		respMap := map[string]interface{}{
			"candidates": []interface{}{
				map[string]interface{}{
					"content": map[string]interface{}{
						"parts": []interface{}{
							map[string]interface{}{
								"text":    "This is thinking",
								"thought": true,
							},
							map[string]interface{}{
								"text":    "This is the actual response",
								"thought": false,
							},
						},
					},
				},
			},
		}

		result := extractTextFromMap(respMap)
		assert.Equal(t, "This is the actual response", result)
	})

	t.Run("handles multiple candidates and parts", func(t *testing.T) {
		respMap := map[string]interface{}{
			"candidates": []interface{}{
				map[string]interface{}{
					"content": map[string]interface{}{
						"parts": []interface{}{
							map[string]interface{}{
								"text": "Part 1",
							},
							map[string]interface{}{
								"text": " Part 2",
							},
						},
					},
				},
			},
		}

		result := extractTextFromMap(respMap)
		assert.Equal(t, "Part 1 Part 2", result)
	})

	t.Run("returns empty string for missing candidates key", func(t *testing.T) {
		respMap := map[string]interface{}{}
		result := extractTextFromMap(respMap)
		assert.Equal(t, "", result)
	})

	t.Run("returns empty string for invalid structure", func(t *testing.T) {
		respMap := map[string]interface{}{
			"candidates": "not an array",
		}
		result := extractTextFromMap(respMap)
		assert.Equal(t, "", result)
	})

	t.Run("skips malformed candidates gracefully", func(t *testing.T) {
		respMap := map[string]interface{}{
			"candidates": []interface{}{
				"invalid candidate",
				map[string]interface{}{
					"content": map[string]interface{}{
						"parts": []interface{}{
							map[string]interface{}{
								"text": "Valid text",
							},
						},
					},
				},
			},
		}

		result := extractTextFromMap(respMap)
		assert.Equal(t, "Valid text", result)
	})
}

func TestGenerateIssueContent_HappyPath(t *testing.T) {
	// Setup temp home
	tmpHome, err := os.MkdirTemp("", "mate-commit-test-issue-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpHome); err != nil {
			return
		}
	}()
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() {
		if err := os.Setenv("HOME", oldHome); err != nil {
			return
		}
	}()

	ctx := context.Background()
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test"}},
		AIConfig:    config.AIConfig{Models: map[config.AI]config.Model{config.AIGemini: "gemini-pro"}},
	}
	gen, _ := NewGeminiIssueContentGenerator(ctx, cfg, nil)
	gen.wrapper.SetSkipConfirmation(true)

	t.Run("successful issue content generation", func(t *testing.T) {
		expectedJSON := `{"title": "Issue Title", "description": "Issue Description", "labels": ["fix"]}`
		gen.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: expectedJSON}}}},
				},
			}, &models.TokenUsage{TotalTokens: 30}, nil
		}

		result, err := gen.GenerateIssueContent(ctx, models.IssueGenerationRequest{})

		assert.NoError(t, err)
		assert.Equal(t, "Issue Title", result.Title)
		assert.Equal(t, "Issue Description", result.Description)
		assert.Contains(t, result.Labels, "fix")
	})
}
