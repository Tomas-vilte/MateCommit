package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestIssueGeneratorService_GenerateFromDiff(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Language: "en"}

	t.Run("Success - Basic diff with feature", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		realDiff := `diff --git a/internal/services/user_service.go b/internal/services/user_service.go
index 1234567..abcdefg 100644
--- a/internal/services/user_service.go
+++ b/internal/services/user_service.go
@@ -10,6 +10,12 @@ type UserService struct {
 	db *sql.DB
 }
 
+func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
+	// Implementation for creating a new user
+	return s.db.Create(user)
+}
+
 func (s *UserService) GetUser(ctx context.Context, id int) (*models.User, error) {
 	return s.db.FindByID(id)
 }`

		mockGit.On("GetDiff", ctx).Return(realDiff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"internal/services/user_service.go"}, nil)

		expectedRequest := models.IssueGenerationRequest{
			Diff:         realDiff,
			ChangedFiles: []string{"internal/services/user_service.go"},
			Hint:         "Add user creation functionality",
			Language:     "en",
		}
		expectedResult := &models.IssueGenerationResult{
			Title:       "Add CreateUser method to UserService",
			Description: "Implement user creation functionality in the UserService to support new user registration flow.",
			Labels:      []string{"feature", "enhancement"},
		}

		mockAI.On("GenerateIssueContent", ctx, expectedRequest).Return(expectedResult, nil)

		result, err := service.GenerateFromDiff(ctx, "Add user creation functionality", false, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Add CreateUser method to UserService", result.Title)
		assert.Contains(t, result.Labels, "feature")
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Bug fix with test files", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		bugFixDiff := `diff --git a/internal/auth/validator.go b/internal/auth/validator.go
index 1234567..abcdefg 100644
--- a/internal/auth/validator.go
+++ b/internal/auth/validator.go
@@ -15,7 +15,7 @@ func ValidateToken(token string) error {
 		return errors.New("invalid token format")
 	}
 	
-	if len(token) < 10 {
+	if len(token) < 32 {
 		return errors.New("token too short")
 	}
 	
diff --git a/internal/auth/validator_test.go b/internal/auth/validator_test.go
index 1234567..abcdefg 100644
--- a/internal/auth/validator_test.go
+++ b/internal/auth/validator_test.go
@@ -20,6 +20,11 @@ func TestValidateToken(t *testing.T) {
 		err := ValidateToken("short")
 		assert.Error(t, err)
 	})
+	
+	t.Run("token with correct length", func(t *testing.T) {
+		err := ValidateToken("a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6")
+		assert.NoError(t, err)
+	})
 }`

		mockGit.On("GetDiff", ctx).Return(bugFixDiff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{
			"internal/auth/validator.go",
			"internal/auth/validator_test.go",
		}, nil)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Fix token validation length requirement",
			Description: "Updated minimum token length from 10 to 32 characters to improve security. Added test coverage for the new validation logic.",
			Labels:      []string{"bug", "security"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Fix token validation length requirement", result.Title)
		assert.Contains(t, result.Labels, "test")
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Error - No changes in diff", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		mockGit.On("GetDiff", ctx).Return("", nil)

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domainErrors.ErrNoChanges, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Error - Git diff fails", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		gitError := errors.New("git: not a git repository")
		mockGit.On("GetDiff", ctx).Return("", gitError)

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockGit.AssertExpectations(t)
	})

	t.Run("Error - AI service not configured", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewIssueGeneratorService(mockGit, nil, WithIssueConfig(cfg))

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domainErrors.ErrAPIKeyMissing, err)
	})

	t.Run("Error - AI generation fails", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		mockGit.On("GetDiff", ctx).Return("some diff", nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"file.go"}, nil)

		aiError := errors.New("AI service rate limit exceeded")
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(nil, aiError)

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Skip labels", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueConfig(cfg))

		mockGit.On("GetDiff", ctx).Return("diff content", nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"main.go"}, nil)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Test Description",
			Labels:      []string{"feature"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromDiff(ctx, "", true, false, nil)

		assert.NoError(t, err)
		assert.NotContains(t, result.Labels, "test")
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_GenerateFromDescription(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Language: "en"}

	t.Run("Success - Feature request", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		description := "We need to add OAuth2 authentication support for third-party integrations"
		expectedRequest := models.IssueGenerationRequest{
			Description: description,
			Language:    "en",
		}
		expectedResult := &models.IssueGenerationResult{
			Title:       "Add OAuth2 authentication support",
			Description: "Implement OAuth2 authentication flow to enable third-party integrations with popular services like Google, GitHub, and Microsoft.",
			Labels:      []string{"feature", "authentication"},
		}

		mockAI.On("GenerateIssueContent", ctx, expectedRequest).Return(expectedResult, nil)

		result, err := service.GenerateFromDescription(ctx, description, false, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Add OAuth2 authentication support", result.Title)
		assert.Contains(t, result.Labels, "feature")
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Bug report", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		description := "Users are experiencing timeout errors when uploading files larger than 10MB"
		expectedResult := &models.IssueGenerationResult{
			Title:       "Fix file upload timeout for large files",
			Description: "Users report timeout errors when uploading files larger than 10MB. Need to increase timeout limits and implement chunked upload for better reliability.",
			Labels:      []string{"bug", "upload"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromDescription(ctx, description, false, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Fix file upload timeout for large files", result.Title)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Skip labels", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		expectedResult := &models.IssueGenerationResult{
			Title:       "Manual Issue",
			Description: "Manual Description",
			Labels:      []string{"feature"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromDescription(ctx, "manual description", true, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Manual Issue", result.Title)
		assert.Empty(t, result.Labels)
		mockAI.AssertExpectations(t)
	})

	t.Run("Error - Empty description", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		result, err := service.GenerateFromDescription(ctx, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - AI service not configured", func(t *testing.T) {
		service := NewIssueGeneratorService(nil, nil, WithIssueConfig(cfg))

		result, err := service.GenerateFromDescription(ctx, "some description", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domainErrors.ErrAPIKeyMissing, err)
	})

	t.Run("Error - AI generation fails", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		aiError := errors.New("AI service unavailable")
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(nil, aiError)

		result, err := service.GenerateFromDescription(ctx, "description", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockAI.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_GenerateFromPR(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Language: "en"}

	t.Run("Success - Feature PR", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockVCS := new(MockVCSClient)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueVCSClient(mockVCS), WithIssueConfig(cfg))

		prData := models.PRData{
			Title:       "Add user profile page",
			Description: "This PR implements a new user profile page with avatar upload and bio editing capabilities.",
			Commits: []models.Commit{
				{Message: "feat: add profile page component"},
				{Message: "feat: implement avatar upload"},
				{Message: "test: add profile page tests"},
			},
			Diff: `diff --git a/frontend/pages/profile.tsx b/frontend/pages/profile.tsx
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/frontend/pages/profile.tsx
@@ -0,0 +1,50 @@
+import React from 'react';
+
+export const ProfilePage = () => {
+  return <div>Profile Page</div>;
+};`,
		}
		mockVCS.On("GetPR", ctx, 42).Return(prData, nil)
		mockVCS.On("GetRepoLabels", ctx).Return([]string(nil), nil)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Implement user profile page",
			Description: "Add comprehensive user profile page with avatar upload and bio editing features.",
			Labels:      []string{"feature", "frontend"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromPR(ctx, 42, "", false, false, nil)

		assert.NoError(t, err)
		assert.Equal(t, "Implement user profile page", result.Title)
		assert.Contains(t, result.Description, "Related PR: #42")
		assert.Contains(t, result.Labels, "feature")
		mockVCS.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Bug fix PR with hint", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockVCS := new(MockVCSClient)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueVCSClient(mockVCS), WithIssueConfig(cfg))

		prData := models.PRData{
			Title:       "Fix memory leak in cache service",
			Description: "Resolves memory leak caused by unbounded cache growth",
			Commits: []models.Commit{
				{Message: "fix: add cache eviction policy"},
			},
			Diff: "diff --git a/cache/service.go b/cache/service.go\n...",
		}
		mockVCS.On("GetPR", ctx, 123).Return(prData, nil)
		mockVCS.On("GetRepoLabels", ctx).Return([]string(nil), nil)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Memory leak in cache service",
			Description: "Cache service experiences memory leak due to unbounded growth. Implement LRU eviction policy.",
			Labels:      []string{"bug", "performance"},
		}

		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(expectedResult, nil)

		result, err := service.GenerateFromPR(ctx, 123, "Focus on performance impact", false, false, nil)

		assert.NoError(t, err)
		assert.Contains(t, result.Description, "Related PR: #123")
		mockVCS.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Error - VCS client not configured", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueConfig(cfg))

		result, err := service.GenerateFromPR(ctx, 1, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domainErrors.ErrConfigMissing, err)
	})

	t.Run("Error - AI service not configured", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewIssueGeneratorService(nil, nil, WithIssueVCSClient(mockVCS), WithIssueConfig(cfg))

		result, err := service.GenerateFromPR(ctx, 1, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domainErrors.ErrAPIKeyMissing, err)
	})

	t.Run("Error - PR not found", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockVCS := new(MockVCSClient)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueVCSClient(mockVCS), WithIssueConfig(cfg))

		prError := errors.New("PR #999 not found")
		mockVCS.On("GetPR", ctx, 999).Return(models.PRData{}, prError)

		result, err := service.GenerateFromPR(ctx, 999, "", false, false, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockVCS.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_GenerateWithTemplate(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Language: "en"}

	t.Run("Success - Bug report template with diff", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockAI := new(MockIssueContentGenerator)
		mockTemplate := new(MockIssueTemplateService)
		service := NewIssueGeneratorService(mockGit, mockAI, WithIssueTemplateService(mockTemplate), WithIssueConfig(cfg))

		template := &models.IssueTemplate{
			Name:      "bug_report",
			Title:     "[BUG] ",
			Labels:    []string{"bug"},
			Assignees: []string{"maintainer"},
		}
		generated := &models.IssueGenerationResult{
			Title:       "Application crashes on startup",
			Description: "The application crashes when launched with invalid config",
			Labels:      []string{"crash"},
		}
		merged := &models.IssueGenerationResult{
			Title:       "[BUG] Application crashes on startup",
			Description: "The application crashes when launched with invalid config\n---\n## Bug Report Template\nPlease provide details...",
			Labels:      []string{"bug", "crash"},
			Assignees:   []string{"maintainer"},
		}

		mockTemplate.On("GetTemplateByName", ctx, "bug_report").Return(template, nil)
		mockGit.On("GetDiff", ctx).Return("some changes", nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"config/loader.go"}, nil)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(generated, nil)
		mockTemplate.On("MergeWithGeneratedContent", template, mock.Anything).Return(merged)

		result, err := service.GenerateWithTemplate(ctx, "bug_report", "", true, "", false)

		assert.NoError(t, err)
		assert.Equal(t, "[BUG] Application crashes on startup", result.Title)
		assert.Contains(t, result.Labels, "bug")
		assert.Contains(t, result.Assignees, "maintainer")
		mockTemplate.AssertExpectations(t)
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Feature request template with description", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockTemplate := new(MockIssueTemplateService)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueTemplateService(mockTemplate), WithIssueConfig(cfg))

		template := &models.IssueTemplate{
			Name:   "feature_request",
			Title:  "[FEATURE] ",
			Labels: []string{"enhancement"},
		}
		generated := &models.IssueGenerationResult{
			Title:       "Add dark mode support",
			Description: "Users want dark mode for better night-time viewing",
			Labels:      []string{"ui"},
		}
		merged := &models.IssueGenerationResult{
			Title:       "[FEATURE] Add dark mode support",
			Description: "Users want dark mode for better night-time viewing\n---\n## Feature Request",
			Labels:      []string{"enhancement", "ui"},
		}

		mockTemplate.On("GetTemplateByName", ctx, "feature_request").Return(template, nil)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(generated, nil)
		mockTemplate.On("MergeWithGeneratedContent", template, mock.Anything).Return(merged)

		result, err := service.GenerateWithTemplate(ctx, "feature_request", "", false, "Add dark mode", false)

		assert.NoError(t, err)
		assert.Equal(t, "[FEATURE] Add dark mode support", result.Title)
		assert.Contains(t, result.Labels, "enhancement")
		mockTemplate.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Error - Template not found", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockTemplate := new(MockIssueTemplateService)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueTemplateService(mockTemplate), WithIssueConfig(cfg))

		templateError := errors.New("template not found")
		mockTemplate.On("GetTemplateByName", ctx, "nonexistent").Return(nil, templateError)

		result, err := service.GenerateWithTemplate(ctx, "nonexistent", "", false, "desc", false)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockTemplate.AssertExpectations(t)
	})

	t.Run("Error - No input provided", func(t *testing.T) {
		mockAI := new(MockIssueContentGenerator)
		mockTemplate := new(MockIssueTemplateService)
		service := NewIssueGeneratorService(nil, mockAI, WithIssueTemplateService(mockTemplate), WithIssueConfig(cfg))

		template := &models.IssueTemplate{Name: "test"}
		mockTemplate.On("GetTemplateByName", ctx, "test").Return(template, nil)

		result, err := service.GenerateWithTemplate(ctx, "test", "", false, "", false)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockTemplate.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_InferBranchName(t *testing.T) {
	service := &IssueGeneratorService{}

	tests := []struct {
		name        string
		issueNumber int
		labels      []string
		expected    string
	}{
		{
			name:        "fix label has highest priority",
			issueNumber: 123,
			labels:      []string{"feature", "fix", "docs"},
			expected:    "fix/issue-123",
		},
		{
			name:        "refactor has priority over docs",
			issueNumber: 456,
			labels:      []string{"docs", "refactor", "test"},
			expected:    "refactor/issue-456",
		},
		{
			name:        "feature is default when present",
			issueNumber: 789,
			labels:      []string{"feature"},
			expected:    "feature/issue-789",
		},
		{
			name:        "no labels defaults to feature",
			issueNumber: 999,
			labels:      []string{},
			expected:    "feature/issue-999",
		},
		{
			name:        "test label",
			issueNumber: 111,
			labels:      []string{"test"},
			expected:    "test/issue-111",
		},
		{
			name:        "docs has priority over infra",
			issueNumber: 222,
			labels:      []string{"infra", "docs"},
			expected:    "docs/issue-222",
		},
		{
			name:        "unknown labels default to feature",
			issueNumber: 333,
			labels:      []string{"custom", "unknown"},
			expected:    "feature/issue-333",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.InferBranchName(tt.issueNumber, tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIssueGeneratorService_CreateIssue(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		result := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Test Description",
			Labels:      []string{"bug"},
		}
		assignees := []string{"user1", "user2"}

		expectedIssue := &models.Issue{
			Number:      42,
			Title:       "Test Issue",
			Description: "Test Description",
			Labels:      []string{"bug"},
			URL:         "https://github.com/owner/repo/issues/42",
		}

		mockVCS.On("CreateIssue", ctx, "Test Issue", "Test Description", []string{"bug"}, assignees).
			Return(expectedIssue, nil)

		issue, err := service.CreateIssue(ctx, result, assignees)

		assert.NoError(t, err)
		assert.Equal(t, 42, issue.Number)
		assert.Equal(t, "Test Issue", issue.Title)
		mockVCS.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_GetAuthenticatedUser(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		mockVCS.On("GetAuthenticatedUser", ctx).Return("testuser", nil)

		username, err := service.GetAuthenticatedUser(ctx)

		assert.NoError(t, err)
		assert.Equal(t, "testuser", username)
		mockVCS.AssertExpectations(t)
	})

	t.Run("Error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		vcsError := errors.New("authentication failed")
		mockVCS.On("GetAuthenticatedUser", ctx).Return("", vcsError)

		username, err := service.GetAuthenticatedUser(ctx)

		assert.Error(t, err)
		assert.Empty(t, username)
		mockVCS.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_LinkIssueToPR(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		prData := models.PRData{
			Title:       "Fix bug",
			Description: "This fixes the bug",
		}

		mockVCS.On("GetPR", ctx, 10).Return(prData, nil)
		mockVCS.On("UpdatePR", ctx, 10, models.PRSummary{
			Title: "Fix bug",
			Body:  "This fixes the bug\n\nCloses #42",
		}).Return(nil)

		err := service.LinkIssueToPR(ctx, 10, 42)

		assert.NoError(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("Error - VCS client not configured", func(t *testing.T) {
		service := &IssueGeneratorService{}

		err := service.LinkIssueToPR(ctx, 10, 42)

		assert.Error(t, err)
		assert.Equal(t, domainErrors.ErrConfigMissing, err)
	})

	t.Run("Error - Get PR fails", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		prError := errors.New("PR not found")
		mockVCS.On("GetPR", ctx, 10).Return(models.PRData{}, prError)

		err := service.LinkIssueToPR(ctx, 10, 42)

		assert.Error(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("Error - Update PR fails", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := &IssueGeneratorService{vcsClient: mockVCS}

		prData := models.PRData{
			Title:       "Fix bug",
			Description: "This fixes the bug",
		}

		updateError := errors.New("update failed")
		mockVCS.On("GetPR", ctx, 10).Return(prData, nil)
		mockVCS.On("UpdatePR", ctx, 10, mock.Anything).Return(updateError)

		err := service.LinkIssueToPR(ctx, 10, 42)

		assert.Error(t, err)
		mockVCS.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_SuggestTemplates(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mockTemplate := new(MockIssueTemplateService)
		service := &IssueGeneratorService{templateService: mockTemplate}

		expectedTemplates := []models.TemplateMetadata{
			{Name: "bug_report", FilePath: ".github/ISSUE_TEMPLATE/bug_report.md"},
			{Name: "feature_request", FilePath: ".github/ISSUE_TEMPLATE/feature_request.md"},
		}

		mockTemplate.On("ListTemplates", ctx).Return(expectedTemplates, nil)

		templates, err := service.SuggestTemplates(ctx)

		assert.NoError(t, err)
		assert.Len(t, templates, 2)
		assert.Equal(t, "bug_report", templates[0].Name)
		mockTemplate.AssertExpectations(t)
	})

	t.Run("No template service", func(t *testing.T) {
		service := &IssueGeneratorService{}

		templates, err := service.SuggestTemplates(ctx)

		assert.NoError(t, err)
		assert.Empty(t, templates)
	})

	t.Run("Error - List templates fails", func(t *testing.T) {
		mockTemplate := new(MockIssueTemplateService)
		service := &IssueGeneratorService{templateService: mockTemplate}

		templateError := errors.New("failed to list templates")
		mockTemplate.On("ListTemplates", ctx).Return([]models.TemplateMetadata{}, templateError)

		templates, err := service.SuggestTemplates(ctx)

		assert.NoError(t, err)
		assert.Empty(t, templates)
		mockTemplate.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_ExtractFilesFromDiff(t *testing.T) {
	service := &IssueGeneratorService{}

	t.Run("Extract files from standard diff", func(t *testing.T) {
		diff := `diff --git a/internal/services/user.go b/internal/services/user.go
index 1234567..abcdefg 100644
--- a/internal/services/user.go
+++ b/internal/services/user.go
@@ -10,6 +10,12 @@
diff --git a/internal/models/user.go b/internal/models/user.go
index 9876543..fedcba9 100644
--- a/internal/models/user.go
+++ b/internal/models/user.go`

		files := service.extractFilesFromDiff(diff)

		assert.Len(t, files, 2)
		assert.Contains(t, files, "internal/services/user.go")
		assert.Contains(t, files, "internal/models/user.go")
	})

	t.Run("Exclude /dev/null", func(t *testing.T) {
		diff := `diff --git a/deleted.go b/deleted.go
deleted file mode 100644
--- a/deleted.go
+++ /dev/null`

		files := service.extractFilesFromDiff(diff)

		assert.NotContains(t, files, "/dev/null")
	})

	t.Run("No duplicates", func(t *testing.T) {
		diff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
--- a/file.go
+++ b/file.go`

		files := service.extractFilesFromDiff(diff)

		assert.Len(t, files, 1)
		assert.Equal(t, "file.go", files[0])
	})
}
