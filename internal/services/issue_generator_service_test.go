package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestIssueGeneratorService(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Language: "en"}

	t.Run("GenerateFromDiff - Success", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		mockGit.On("GetDiff", ctx).Return("diff content", nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"main.go"}, nil)

		expectedRequest := models.IssueGenerationRequest{
			Diff:         "diff content",
			ChangedFiles: []string{"main.go"},
			Hint:         "test hint",
			Language:     "en",
		}
		expectedResult := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Test Description",
			Labels:      []string{"feature"},
		}

		mockAI.On("GenerateIssueContent", ctx, expectedRequest).Return(expectedResult, nil)

		result, err := service.GenerateFromDiff(ctx, "test hint", false)

		assert.NoError(t, err)
		assert.Equal(t, "Test Issue", result.Title)
		assert.Contains(t, result.Labels, "feature")
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("GenerateFromDescription - Success", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		expectedRequest := models.IssueGenerationRequest{
			Description: "manual description",
			Language:    "en",
		}
		expectedResult := &models.IssueGenerationResult{
			Title:       "Manual Issue",
			Description: "Manual Description",
		}

		mockAI.On("GenerateIssueContent", ctx, expectedRequest).Return(expectedResult, nil)

		result, err := service.GenerateFromDescription(ctx, "manual description", true)

		assert.NoError(t, err)
		assert.Equal(t, "Manual Issue", result.Title)
		assert.Empty(t, result.Labels)
		mockAI.AssertExpectations(t)
	})

	t.Run("GenerateFromPR - Success", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockVCS := new(MockVCSClient)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueVCSClient(mockVCS), WithIssueConfig(cfg))

		prData := models.PRData{
			Title:       "PR Title",
			Description: "PR Description",
			Commits:     []models.Commit{{Message: "commit 1"}},
			Diff:        "diff --git a/file.go b/file.go\n...",
		}
		mockVCS.On("GetPR", ctx, 1).Return(prData, nil)

		expectedResult := &models.IssueGenerationResult{
			Title:       "PR based Issue",
			Description: "PR based Description",
			Labels:      []string{"fix"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromPR(ctx, 1, "", false)

		assert.NoError(t, err)
		assert.Equal(t, "PR based Issue", result.Title)
		assert.Contains(t, result.Description, "Related PR: #1")
		mockVCS.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("GenerateWithTemplate - Success", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		mockTemplate := new(MockIssueTemplateService)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueTemplateService(mockTemplate), WithIssueConfig(cfg))

		template := &models.IssueTemplate{
			Name:      "bug_report",
			Title:     "[BUG] ",
			Labels:    []string{"bug"},
			Assignees: []string{"tester"},
		}
		generated := &models.IssueGenerationResult{
			Title:       "something broke",
			Description: "it fails",
			Labels:      []string{"fix"},
		}
		merged := &models.IssueGenerationResult{
			Title:       "[BUG] something broke",
			Description: "it fails\n---\nTemplate body",
			Labels:      []string{"bug", "fix"},
			Assignees:   []string{"tester"},
		}

		mockTemplate.On("GetTemplateByName", ctx, "bug_report").Return(template, nil)
		mockGit.On("GetDiff", ctx).Return("some changes", nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"file.go"}, nil)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(generated, nil)
		mockTemplate.On("MergeWithGeneratedContent", template, mock.Anything).Return(merged)

		result, err := service.GenerateWithTemplate(ctx, "bug_report", "", true, "", false)

		assert.NoError(t, err)
		assert.Equal(t, "[BUG] something broke", result.Title)
		assert.Contains(t, result.Labels, "bug")
		assert.Contains(t, result.Assignees, "tester")
	})
}

func TestInferBranchName(t *testing.T) {
	service := &IssueGeneratorService{}

	tests := []struct {
		name        string
		issueNumber int
		labels      []string
		expected    string
	}{
		{
			name:        "fix label has priority",
			issueNumber: 123,
			labels:      []string{"feature", "fix", "docs"},
			expected:    "fix/issue-123",
		},
		{
			name:        "feature is default",
			issueNumber: 456,
			labels:      []string{"feature"},
			expected:    "feature/issue-456",
		},
		{
			name:        "refactor has higher priority than docs",
			issueNumber: 789,
			labels:      []string{"docs", "refactor"},
			expected:    "refactor/issue-789",
		},
		{
			name:        "no labels defaults to feature",
			issueNumber: 999,
			labels:      []string{},
			expected:    "feature/issue-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.InferBranchName(tt.issueNumber, tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}
