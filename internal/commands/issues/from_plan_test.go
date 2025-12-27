package issues

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/urfave/cli/v3"
)

func TestFromPlanCommand(t *testing.T) {
	t.Run("should fail when file does not exist", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", "/nonexistent/file.md"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reading plan file")
	})

	t.Run("should fail when file is empty", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		tempFile := createTempPlanFile(t, "")
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", tempFile})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("should generate issue from plan file successfully", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		planContent := `# Implementation Plan

## Objective
Add user authentication feature

## Tasks
1. Create login endpoint
2. Implement JWT tokens
3. Add middleware for protected routes`

		tempFile := createTempPlanFile(t, planContent)
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		expectedResult := &models.IssueGenerationResult{
			Title:       "Add user authentication feature",
			Description: "Implementation plan:\n- Create login endpoint\n- Implement JWT tokens\n- Add middleware",
			Labels:      []string{"feature"},
		}

		mockGen.On("GenerateFromDescription", mock.Anything, planContent, false, false).
			Return(expectedResult, nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string(nil)).
			Return(&models.Issue{Number: 42, URL: "http://github.com/test/issues/42"}, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", tempFile})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
	})

	t.Run("should handle dry-run mode", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		planContent := `# Fix authentication bug`
		tempFile := createTempPlanFile(t, planContent)
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		expectedResult := &models.IssueGenerationResult{
			Title:       "Fix authentication bug",
			Description: "Fix the authentication issue",
			Labels:      []string{"bug"},
		}

		mockGen.On("GenerateFromDescription", mock.Anything, planContent, false, false).
			Return(expectedResult, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", tempFile, "--dry-run"})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
		mockGen.AssertNotCalled(t, "CreateIssue", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should assign to current user when --assign-me is set", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		planContent := `# Add feature X`
		tempFile := createTempPlanFile(t, planContent)
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		expectedResult := &models.IssueGenerationResult{
			Title:       "Add feature X",
			Description: "Feature implementation",
			Labels:      []string{"feature"},
		}

		mockGen.On("GenerateFromDescription", mock.Anything, planContent, false, false).
			Return(expectedResult, nil)
		mockGen.On("GetAuthenticatedUser", mock.Anything).
			Return("johndoe", nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string{"johndoe"}).
			Return(&models.Issue{Number: 1, URL: "http://test.com"}, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", tempFile, "--assign-me"})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
	})

	t.Run("should merge additional labels", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		planContent := `# Security improvement`
		tempFile := createTempPlanFile(t, planContent)
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		expectedResult := &models.IssueGenerationResult{
			Title:       "Security improvement",
			Description: "Improve security",
			Labels:      []string{"security"},
		}

		mockGen.On("GenerateFromDescription", mock.Anything, planContent, false, false).
			Return(expectedResult, nil)

		mockGen.On("CreateIssue", mock.Anything, mock.MatchedBy(func(r *models.IssueGenerationResult) bool {
			labelMap := make(map[string]bool)
			for _, l := range r.Labels {
				labelMap[l] = true
			}
			return labelMap["security"] && labelMap["high-priority"]
		}), []string(nil)).
			Return(&models.Issue{Number: 1, URL: "http://test.com"}, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{
			"test", "issue", "from-plan",
			"--file", tempFile,
			"--labels", "high-priority",
		})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
	})

	t.Run("should handle file with different encodings", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		planContent := `# Implementación de autenticación

Agregar funcionalidad de autenticación con JWT`

		tempFile := createTempPlanFile(t, planContent)
		defer func() {
			if err := os.Remove(tempFile); err != nil {
				t.Fatal(err)
			}
		}()

		expectedResult := &models.IssueGenerationResult{
			Title:       "Implementación de autenticación",
			Description: "Agregar funcionalidad de autenticación con JWT",
			Labels:      []string{"feature"},
		}

		mockGen.On("GenerateFromDescription", mock.Anything, planContent, false, false).
			Return(expectedResult, nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string(nil)).
			Return(&models.Issue{Number: 1, URL: "http://test.com"}, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "from-plan", "--file", tempFile})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
	})
}

func TestMergeUniqueLabels(t *testing.T) {
	tests := []struct {
		name       string
		existing   []string
		additional []string
		want       int
		contains   []string
	}{
		{
			name:       "merge without duplicates",
			existing:   []string{"bug", "feature"},
			additional: []string{"high-priority"},
			want:       3,
			contains:   []string{"bug", "feature", "high-priority"},
		},
		{
			name:       "merge with duplicates",
			existing:   []string{"bug", "security"},
			additional: []string{"bug", "urgent"},
			want:       3,
			contains:   []string{"bug", "security", "urgent"},
		},
		{
			name:       "empty existing",
			existing:   []string{},
			additional: []string{"new"},
			want:       1,
			contains:   []string{"new"},
		},
		{
			name:       "empty additional",
			existing:   []string{"existing"},
			additional: []string{},
			want:       1,
			contains:   []string{"existing"},
		},
		{
			name:       "filter empty strings",
			existing:   []string{"valid", ""},
			additional: []string{"", "another"},
			want:       2,
			contains:   []string{"valid", "another"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeUniqueLabels(tt.existing, tt.additional)
			assert.Equal(t, tt.want, len(result))

			resultMap := make(map[string]bool)
			for _, label := range result {
				resultMap[label] = true
			}

			for _, expected := range tt.contains {
				assert.True(t, resultMap[expected], "expected label %s not found", expected)
			}
		})
	}
}

func TestShowPreview(t *testing.T) {
	trans, err := i18n.NewTranslations("en", "../../i18n/locales")
	require.NoError(t, err)

	cfg := &config.Config{
		Language: "en",
		UseEmoji: true,
	}

	t.Run("should display complete preview", func(t *testing.T) {
		result := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "This is a test description",
			Labels:      []string{"bug", "urgent"},
		}

		err := showPreview(result, trans, cfg)
		assert.NoError(t, err)
	})

	t.Run("should handle empty description", func(t *testing.T) {
		result := &models.IssueGenerationResult{
			Title:  "Test Issue",
			Labels: []string{"feature"},
		}

		err := showPreview(result, trans, cfg)
		assert.NoError(t, err)
	})

	t.Run("should handle empty labels", func(t *testing.T) {
		result := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Description",
			Labels:      []string{},
		}

		err := showPreview(result, trans, cfg)
		assert.NoError(t, err)
	})

	t.Run("should respect emoji config", func(t *testing.T) {
		cfgNoEmoji := &config.Config{
			Language: "en",
			UseEmoji: false,
		}

		result := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Description",
		}

		err := showPreview(result, trans, cfgNoEmoji)
		assert.NoError(t, err)
	})
}

// createTempPlanFile creates a temporary file with the given content for testing.
func createTempPlanFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "plan.md")

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}
