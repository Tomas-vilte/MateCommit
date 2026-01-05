package gemini

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
	"google.golang.org/genai"
)

func TestNewReleaseNotesGenerator(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{"gemini": {APIKey: "test-api-key", Model: "gemini-2.5-flash", Temperature: 0.3, MaxTokens: 10000}},
		Language:    "en",
		AIConfig: config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIGemini: "gemini-pro",
			},
		},
	}

	// Act
	// Act
	generator, err := NewReleaseNotesGenerator(ctx, cfg, nil, "test-owner", "test-repo")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, generator)
	assert.Equal(t, "gemini-pro", generator.model)
	assert.Equal(t, "en", generator.lang)
}

func TestNewReleaseNotesGenerator_MissingKey(t *testing.T) {
	// Arrange
	ctx := context.Background()
	cfg := &config.Config{
		AIProviders: map[string]config.AIProviderConfig{},
	}
	// Act
	generator, err := NewReleaseNotesGenerator(ctx, cfg, nil, "test-owner", "test-repo")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, generator)
	assert.Contains(t, err.Error(), "API key is missing")
}

func TestBuildPrompt(t *testing.T) {
	t.Run("formats changes correctly in English", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "en"}
		release := &models.Release{
			Version:         "v1.0.0",
			PreviousVersion: "v0.9.0",
			VersionBump:     "major",
			Features:        []models.ReleaseItem{{Type: "feat", Description: "New feature"}},
			BugFixes:        []models.ReleaseItem{{Type: "fix", Description: "Bug fix"}},
			Breaking:        []models.ReleaseItem{{Type: "breaking", Description: "Breaking change"}},
			Improvements:    []models.ReleaseItem{{Type: "chore", Description: "Improvement"}},
		}

		// Act
		prompt := generator.buildPrompt(release)

		// Assert
		// Assert
		assert.Contains(t, prompt, "Versions: v0.9.0 -> v1.0.0 (major)")

		assert.Contains(t, prompt, "BREAKING CHANGES:")
		assert.Contains(t, prompt, "- breaking: Breaking change")
		assert.Contains(t, prompt, "NEW FEATURES:")
		assert.Contains(t, prompt, "- feat: New feature")
		assert.Contains(t, prompt, "BUG FIXES:")
		assert.Contains(t, prompt, "- fix: Bug fix")
		assert.Contains(t, prompt, "IMPROVEMENTS:")
		assert.Contains(t, prompt, "- chore: Improvement")
	})

	t.Run("formats changes correctly in Spanish", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "es"}
		release := &models.Release{
			Version: "v1.0.0",
		}

		// Act
		prompt := generator.buildPrompt(release)

		assert.Contains(t, prompt, "- Versiones:")
		assert.Contains(t, prompt, "->")
		assert.Contains(t, prompt, "(")
	})

	t.Run("handles empty changes", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "en"}
		release := &models.Release{Version: "v1.0.0"}

		// Act
		prompt := generator.buildPrompt(release)

		// Assert
		assert.NotContains(t, prompt, "BREAKING CHANGES:")
		assert.NotContains(t, prompt, "NEW FEATURES:")
		assert.NotContains(t, prompt, "BUG FIXES:")
		assert.NotContains(t, prompt, "IMPROVEMENTS:")
	})

	t.Run("formats complex release with all sections", func(t *testing.T) {
		// Arrange
		generator := &ReleaseNotesGenerator{lang: "en", owner: "owner", repo: "repo"}
		release := &models.Release{
			Version:         "v2.0.0",
			PreviousVersion: "v1.5.0",
			VersionBump:     "major",
			ClosedIssues: []models.Issue{
				{Number: 1, Title: "Issue 1", Author: "user1"},
			},
			MergedPRs: []models.PullRequest{
				{Number: 10, Title: "PR 10", Author: "user2", Description: "Long description\nwith multiple lines"},
			},
			Contributors:    []string{"user1", "user2"},
			NewContributors: []string{"user2"},
			FileStats: models.FileStatistics{
				FilesChanged: 5,
				Insertions:   100,
				Deletions:    20,
				TopFiles: []models.FileChange{
					{Path: "main.go", Additions: 50, Deletions: 10},
				},
			},
			Dependencies: []models.DependencyChange{
				{Name: "dep1", OldVersion: "1.0", NewVersion: "1.1", Type: "updated"},
				{Name: "dep2", NewVersion: "2.0", Type: "added"},
				{Name: "dep3", OldVersion: "0.5", Type: "removed"},
			},
		}

		// Act
		prompt := generator.buildPrompt(release)

		// Assert
		assert.Contains(t, prompt, "CLOSED ISSUES:")
		assert.Contains(t, prompt, "- #1: Issue 1 (by @user1)")
		assert.Contains(t, prompt, "MERGED PULL REQUESTS:")
		assert.Contains(t, prompt, "- #10: PR 10 (by @user2)")
		assert.Contains(t, prompt, "Description: Long description")
		assert.Contains(t, prompt, "CONTRIBUTORS (2 total):")
		assert.Contains(t, prompt, "New contributors: user2")
		assert.Contains(t, prompt, "FILE STATISTICS:")
		assert.Contains(t, prompt, "- Files changed: 5")
		assert.Contains(t, prompt, "- main.go (+50/-10)")
		assert.Contains(t, prompt, "DEPENDENCY UPDATES:")
		assert.Contains(t, prompt, "- dep1: 1.0 → 1.1")
		assert.Contains(t, prompt, "- Added: dep2 2.0")
		assert.Contains(t, prompt, "- Removed: dep3 0.5")
	})
}

func TestParseJSONResponse(t *testing.T) {
	generator := &ReleaseNotesGenerator{}
	release := &models.Release{Version: "v1.0.0", VersionBump: "minor"}

	t.Run("parses JSON response correctly", func(t *testing.T) {
		// Arrange
		content := `{
			"title": "Release v1.0.0",
			"summary": "This is a summary.",
			"highlights": ["Highlight 1", "Highlight 2"],
			"breaking_changes": [],
			"contributors": ""
		}`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v1.0.0", notes.Title)
		assert.Equal(t, "This is a summary.", notes.Summary)
		assert.Equal(t, []string{"Highlight 1", "Highlight 2"}, notes.Highlights)
		assert.Equal(t, models.VersionBump("minor"), notes.Recommended)
	})

	t.Run("parses JSON with breaking changes", func(t *testing.T) {
		// Arrange
		content := `{
			"title": "Release v2.0.0",
			"summary": "Major release with breaking changes.",
			"highlights": ["New API", "Better performance"],
			"breaking_changes": ["Removed old API", "Changed config format"],
			"contributors": "https://github.com/test/repo/graphs/contributors"
		}`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v2.0.0", notes.Title)
		assert.Equal(t, "Major release with breaking changes.", notes.Summary)
		assert.Equal(t, []string{"New API", "Better performance"}, notes.Highlights)
		assert.Equal(t, []string{"Removed old API", "Changed config format"}, notes.BreakingChanges)
		assert.Equal(t, "https://github.com/test/repo/graphs/contributors", notes.Links["Contributors"])
	})

	t.Run("parses JSON with semantic sections", func(t *testing.T) {
		// Arrange
		content := `{
			"title": "Release v3.0.0",
			"summary": "Semantic release",
			"sections": [
				{
					"title": "🎨 UI Improvements",
					"items": ["Dark Mode", "New Icons"]
				},
				{
					"title": "🐛 Fixes",
					"items": ["Crash on login"]
				}
			],
			"highlights": [],
			"breaking_changes": []
		}`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, notes.Sections, 2)
		assert.Equal(t, "🎨 UI Improvements", notes.Sections[0].Title)
		assert.Equal(t, []string{"Dark Mode", "New Icons"}, notes.Sections[0].Items)
		assert.Equal(t, "🐛 Fixes", notes.Sections[1].Title)
		assert.Equal(t, []string{"Crash on login"}, notes.Sections[1].Items)
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		// Arrange
		content := `invalid json`

		// Act
		notes, err := generator.parseJSONResponse(content, release)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, notes)
		assert.Contains(t, err.Error(), "error parsing AI JSON response")
	})

	t.Run("handles N/A contributors", func(t *testing.T) {
		content := `{"title": "T", "summary": "S", "highlights": [], "breaking_changes": [], "contributors": "N/A"}`
		notes, err := generator.parseJSONResponse(content, release)
		assert.NoError(t, err)
		assert.Empty(t, notes.Links["Contributors"])
	})
}

func TestGenerateNotes(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "matecommit-test-gen-notes-*")
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
	// act
	generator, _ := NewReleaseNotesGenerator(ctx, cfg, nil, "owner", "repo")
	generator.wrapper.SetSkipConfirmation(true)

	t.Run("successful generation", func(t *testing.T) {
		// Arrange
		expectedJSON := `{"title": "Release v1.0.0", "summary": "Summary", "highlights": ["H1"], "breaking_changes": []}`
		generator.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: expectedJSON}}}},
				},
				UsageMetadata: &genai.GenerateContentResponseUsageMetadata{TotalTokenCount: 100},
			}, &models.TokenUsage{TotalTokens: 100}, nil
		}

		// Act
		notes, err := generator.GenerateNotes(ctx, &models.Release{Version: "v1.0.0"})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "Release v1.0.0", notes.Title)
		assert.Equal(t, 100, notes.Usage.TotalTokens)
	})

	t.Run("AI returns error", func(t *testing.T) {
		// Arrange
		generator.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return nil, nil, fmt.Errorf("AI error")
		}

		// Act
		notes, err := generator.GenerateNotes(ctx, &models.Release{})

		// Assert
		assert.Error(t, err)
		assert.Nil(t, notes)
		assert.Contains(t, err.Error(), "AI error")
	})

	t.Run("no candidates from AI", func(t *testing.T) {
		// Arrange
		generator.generateFn = func(ctx context.Context, mName string, p string) (interface{}, *models.TokenUsage, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{},
			}, &models.TokenUsage{}, nil
		}

		// Act
		_, err := generator.GenerateNotes(ctx, &models.Release{})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid AI output format")
	})
}
