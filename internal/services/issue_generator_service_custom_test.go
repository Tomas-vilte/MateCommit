package services

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestIssueGeneratorService_SelectTemplateWithAI(t *testing.T) {
	t.Run("Success - AI selects Bug Report", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
			{Name: "Feature Request", FilePath: "feat.yml", About: "New features"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockTemplateSvc.On("GetTemplateByName", ctx, "bug").Return(&models.IssueTemplate{
			Name:     "Bug Report",
			FilePath: "bug.yml",
		}, nil)

		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return strings.Contains(req.Description, "Available templates") &&
				strings.Contains(req.Description, "Based on the context, select")
		})).Return(&models.IssueGenerationResult{
			Title: "Bug Report",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Fixing a panic in main.go", nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, "Bug Report", tmpl.Name)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Returns nil when no templates available", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return([]models.TemplateMetadata{}, nil)

		mockAI := new(MockIssueContentGenerator)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil not error when no templates
		assert.Nil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
	})

	t.Run("Returns nil when ListTemplates fails", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return([]models.TemplateMetadata{}, assert.AnError)

		mockAI := new(MockIssueContentGenerator)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil not error
		assert.Nil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
	})

	t.Run("Returns nil when no template service configured", func(t *testing.T) {
		ctx := context.Background()

		mockAI := new(MockIssueContentGenerator)

		service := NewIssueGeneratorService(nil, mockAI)

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil not error
		assert.Nil(t, tmpl)
	})

	t.Run("Returns nil when AI returns empty title", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(&models.IssueGenerationResult{
			Title: "",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil when no match found
		assert.Nil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Returns nil when AI generation fails", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(nil, assert.AnError)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil not error when AI fails
		assert.Nil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Returns nil when selected template not found in list", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
			{Name: "Feature Request", FilePath: "feature_request.yml", About: "New features"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		// AI selects a template that doesn't match any in the list
		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(&models.IssueGenerationResult{
			Title: "Non Existent Template",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Some description", nil, nil)

		assert.NoError(t, err) // Returns nil,nil when no match found
		assert.Nil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Strips extension from FilePath", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Security Issue", FilePath: "security.yaml", About: "Security vulnerabilities"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		// GetTemplateByName should be called with "security", not "security.yaml"
		mockTemplateSvc.On("GetTemplateByName", ctx, "security").Return(&models.IssueTemplate{
			Name:     "Security Issue",
			FilePath: "security.yaml",
		}, nil)

		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(&models.IssueGenerationResult{
			Title: "Security Issue",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "SQL injection vulnerability", nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, "Security Issue", tmpl.Name)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Matches with different casing", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Feature Request", FilePath: "feature_request.yml", About: "New features"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockTemplateSvc.On("GetTemplateByName", ctx, "feature_request").Return(&models.IssueTemplate{
			Name:     "Feature Request",
			FilePath: "feature_request.yml",
		}, nil)

		mockAI := new(MockIssueContentGenerator)
		// AI returns lowercase version
		mockAI.On("GenerateIssueContent", ctx, mock.Anything).Return(&models.IssueGenerationResult{
			Title: "feature request",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "", "Add dark mode", nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Equal(t, "Feature Request", tmpl.Name)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Uses title and description in prompt", func(t *testing.T) {
		ctx := context.Background()

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockTemplateSvc.On("GetTemplateByName", ctx, "bug").Return(&models.IssueTemplate{
			Name:     "Bug Report",
			FilePath: "bug.yml",
		}, nil)

		mockAI := new(MockIssueContentGenerator)
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return strings.Contains(req.Description, "Test Title") &&
				strings.Contains(req.Description, "Test Description") &&
				strings.Contains(req.Description, "main.go")
		})).Return(&models.IssueGenerationResult{
			Title: "Bug Report",
		}, nil)

		service := NewIssueGeneratorService(nil, mockAI,
			WithIssueTemplateService(mockTemplateSvc))

		tmpl, err := service.SelectTemplateWithAI(ctx, "Test Title", "Test Description", []string{"main.go"}, []string{"bug", "urgent"})

		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})
}

func TestIssueGeneratorService_GenerateFromDiff_WithAutoTemplate(t *testing.T) {
	t.Run("Success - Auto-selects template and generates issue", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{Language: "en"}

		diff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,4 @@ func main() {
+    // Fix panic when user is nil
+    if user != nil {`

		mockGit := new(MockGitService)
		mockGit.On("GetDiff", ctx).Return(diff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"main.go"}, nil)

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml", About: "Fix bugs"},
			{Name: "Feature Request", FilePath: "feat.yml", About: "New features"},
		}

		bugTemplate := &models.IssueTemplate{
			Name:        "Bug Report",
			FilePath:    "bug.yml",
			Title:       "Bug: {{title}}",
			BodyContent: "## Description\n{{description}}",
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)
		mockTemplateSvc.On("GetTemplateByName", ctx, "bug").Return(bugTemplate, nil)
		mockTemplateSvc.On("MergeWithGeneratedContent", bugTemplate, mock.AnythingOfType("*models.IssueGenerationResult")).
			Return(&models.IssueGenerationResult{
				Title:       "Fix null pointer panic in user handler",
				Description: "## Description\nFixed panic when user is nil",
				Labels:      []string{"bug", "fix"},
			}, nil)

		mockAI := new(MockIssueContentGenerator)
		// First call: template selection
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return strings.Contains(req.Description, "Available templates")
		})).Return(&models.IssueGenerationResult{
			Title: "Bug Report",
		}, nil).Once()

		// Second call: actual issue generation with template
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return req.Template != nil &&
				req.Template.Name == "Bug Report" &&
				strings.Contains(req.Diff, "Fix panic")
		})).Return(&models.IssueGenerationResult{
			Title:       "Fix null pointer panic in user handler",
			Description: "## Description\nFixed panic when user is nil",
			Labels:      []string{"bug", "fix"},
		}, nil).Once()

		service := NewIssueGeneratorService(mockGit, mockAI,
			WithIssueConfig(cfg),
			WithIssueTemplateService(mockTemplateSvc))

		result, err := service.GenerateFromDiff(ctx, "fix panic", false, true, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Fix null pointer panic in user handler", result.Title)
		assert.Contains(t, result.Labels, "bug")
		mockGit.AssertExpectations(t)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Auto-template disabled, no template used", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{Language: "en"}

		diff := `diff --git a/feature.go b/feature.go
index 1234567..abcdefg 100644
--- a/feature.go
+++ b/feature.go
@@ -1,0 +1,5 @@
+func NewFeature() {`

		mockGit := new(MockGitService)
		mockGit.On("GetDiff", ctx).Return(diff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"feature.go"}, nil)

		mockAI := new(MockIssueContentGenerator)
		// Only one call: direct issue generation without template
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return req.Template == nil &&
				strings.Contains(req.Diff, "NewFeature")
		})).Return(&models.IssueGenerationResult{
			Title:       "Add new feature",
			Description: "Implemented new feature",
			Labels:      []string{"feature"},
		}, nil)

		service := NewIssueGeneratorService(mockGit, mockAI,
			WithIssueConfig(cfg))

		result, err := service.GenerateFromDiff(ctx, "", false, false, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Add new feature", result.Title)
		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Template selection returns nil, falls back to no template", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{Language: "en"}

		diff := `diff --git a/test.go b/test.go`

		mockGit := new(MockGitService)
		mockGit.On("GetDiff", ctx).Return(diff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"test.go"}, nil)

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return([]models.TemplateMetadata{}, nil)

		mockAI := new(MockIssueContentGenerator)
		// Only one call: direct generation because template selection failed
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return req.Template == nil
		})).Return(&models.IssueGenerationResult{
			Title:       "Test change",
			Description: "Updated test",
			Labels:      []string{"test"},
		}, nil)

		service := NewIssueGeneratorService(mockGit, mockAI,
			WithIssueConfig(cfg),
			WithIssueTemplateService(mockTemplateSvc))

		result, err := service.GenerateFromDiff(ctx, "", false, true, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test change", result.Title)
		mockGit.AssertExpectations(t)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("Success - Template selection fails, continues without template", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{Language: "en"}

		diff := `diff --git a/code.go b/code.go`

		mockGit := new(MockGitService)
		mockGit.On("GetDiff", ctx).Return(diff, nil)
		mockGit.On("GetChangedFiles", ctx).Return([]string{"code.go"}, nil)

		mockTemplateInfo := []models.TemplateMetadata{
			{Name: "Bug Report", FilePath: "bug.yml"},
		}

		mockTemplateSvc := new(MockIssueTemplateService)
		mockTemplateSvc.On("ListTemplates", ctx).Return(mockTemplateInfo, nil)

		mockAI := new(MockIssueContentGenerator)
		// First call: template selection - AI fails
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return strings.Contains(req.Description, "Available templates")
		})).Return(nil, assert.AnError).Once()

		// Second call: continues with no template
		mockAI.On("GenerateIssueContent", ctx, mock.MatchedBy(func(req models.IssueGenerationRequest) bool {
			return req.Template == nil
		})).Return(&models.IssueGenerationResult{
			Title:       "Code update",
			Description: "Updated code",
			Labels:      []string{"refactor"},
		}, nil).Once()

		service := NewIssueGeneratorService(mockGit, mockAI,
			WithIssueConfig(cfg),
			WithIssueTemplateService(mockTemplateSvc))

		result, err := service.GenerateFromDiff(ctx, "", false, true, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Code update", result.Title)
		mockGit.AssertExpectations(t)
		mockTemplateSvc.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})
}
