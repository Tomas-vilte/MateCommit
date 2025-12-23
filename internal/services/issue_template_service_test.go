package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestIssueTemplateService_GetTemplatesDir(t *testing.T) {
	cwd, _ := os.Getwd()

	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{
			name:     "GitHub provider",
			provider: "github",
			expected: filepath.Join(cwd, ".github", "ISSUE_TEMPLATE"),
		},
		{
			name:     "GitLab provider",
			provider: "gitlab",
			expected: filepath.Join(cwd, ".gitlab", "issue_templates"),
		},
		{
			name:     "Default provider",
			provider: "",
			expected: filepath.Join(cwd, ".github", "ISSUE_TEMPLATE"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{ActiveVCSProvider: tt.provider}
			service := NewIssueTemplateService(WithTemplateConfig(cfg))
			dir, err := service.GetTemplatesDir(context.Background())

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, dir)
		})
	}
}

func TestIssueTemplateService_ParseTemplate(t *testing.T) {
	service := &IssueTemplateService{}

	t.Run("Valid GitHub Issue Form YAML", func(t *testing.T) {
		content := `name: Bug Report
description: Create a report to help us improve
title: '[BUG] '
labels:
  - bug
  - fix
assignees:
  - user1
  - user2
body:
  - type: markdown
    attributes:
      value: "Thanks for reporting!"
  - type: textarea
    id: description
    attributes:
      label: "What happened?"
    validations:
      required: true`
		template, err := service.parseTemplate(context.Background(), content, "test.yml")

		require.NoError(t, err)
		assert.Equal(t, "Bug Report", template.Name)
		assert.Equal(t, "Create a report to help us improve", template.Description)
		assert.Equal(t, "[BUG] ", template.Title)
		assert.Equal(t, []string{"bug", "fix"}, template.Labels)
		assert.Equal(t, []string{"user1", "user2"}, template.Assignees)
		assert.NotNil(t, template.Body)
	})

	t.Run("Legacy markdown template with frontmatter", func(t *testing.T) {
		content := `name: Feature Request
about: Suggest an idea
title: '[FEATURE] '
labels:
  - enhancement`
		template, err := service.parseTemplate(context.Background(), content, "test.yml")

		require.NoError(t, err)
		assert.Equal(t, "Feature Request", template.Name)
		assert.Equal(t, "Suggest an idea", template.About)
		assert.Equal(t, "[FEATURE] ", template.Title)
		assert.Equal(t, []string{"enhancement"}, template.Labels)
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		content := `name: : invalid
title: [unclosed`
		_, err := service.parseTemplate(context.Background(), content, "test.yml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIGURATION: failed to parse YAML template")
	})
}

func TestIssueTemplateService_FilesystemOps(t *testing.T) {
	tmpDir := t.TempDir()

	origCwd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(origCwd); err != nil {
			panic(err)
		}
	}()

	cfg := &config.Config{ActiveVCSProvider: "github"}
	service := NewIssueTemplateService(WithTemplateConfig(cfg))

	t.Run("InitializeTemplates", func(t *testing.T) {
		err := service.InitializeTemplates(context.Background(), false)
		assert.NoError(t, err)

		templatesDir := filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE")
		assert.DirExists(t, templatesDir)
		assert.FileExists(t, filepath.Join(templatesDir, "bug_report.yml"))
		assert.FileExists(t, filepath.Join(templatesDir, "feature_request.yml"))
		assert.FileExists(t, filepath.Join(templatesDir, "custom.yml"))
	})

	t.Run("InitializeTemplates - Already exists", func(t *testing.T) {
		err := service.InitializeTemplates(context.Background(), false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIGURATION: templates_already_exist")

		err = service.InitializeTemplates(context.Background(), true)
		assert.NoError(t, err)
	})

	t.Run("ListTemplates", func(t *testing.T) {
		templates, err := service.ListTemplates(context.Background())
		assert.NoError(t, err)
		assert.Len(t, templates, 3)

		err = os.WriteFile(filepath.Join(tmpDir, ".github", "ISSUE_TEMPLATE", "test.txt"), []byte("..."), 0644)
		require.NoError(t, err)

		templates, err = service.ListTemplates(context.Background())
		assert.NoError(t, err)
		assert.Len(t, templates, 3)
	})

	t.Run("GetTemplateByName", func(t *testing.T) {
		template, err := service.GetTemplateByName(context.Background(), "bug_report")
		assert.NoError(t, err)
		assert.NotNil(t, template)

		template, err = service.GetTemplateByName(context.Background(), "bug_report.yml")
		assert.NoError(t, err)
		assert.NotNil(t, template)

		_, err = service.GetTemplateByName(context.Background(), "non_existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestIssueTemplateService_MergeWithGeneratedContent(t *testing.T) {
	service := &IssueTemplateService{}

	t.Run("Merge with markdown template (legacy)", func(t *testing.T) {
		template := &models.IssueTemplate{
			Title:       "[TEMPLATE] ",
			BodyContent: "Template body",
			Labels:      []string{"bug", "critical"},
			Assignees:   []string{"dev1"},
		}

		generated := &models.IssueGenerationResult{
			Title:       "Something failed",
			Description: "Detailed error info",
			Labels:      []string{"fix", "critical"},
			Assignees:   []string{"dev1", "dev2"},
		}

		result := service.MergeWithGeneratedContent(template, generated)

		assert.Equal(t, "[TEMPLATE] Something failed", result.Title)
		assert.Contains(t, result.Description, "Detailed error info")
		assert.Contains(t, result.Description, "Template body")
		assert.Contains(t, result.Labels, "bug")
		assert.Contains(t, result.Labels, "fix")
		assert.Contains(t, result.Labels, "critical")
		assert.Len(t, result.Labels, 3)

		assert.Contains(t, result.Assignees, "dev1")
		assert.Contains(t, result.Assignees, "dev2")
		assert.Len(t, result.Assignees, 2)
	})

	t.Run("Merge with GitHub Issue Form template", func(t *testing.T) {
		template := &models.IssueTemplate{
			Title:  "[BUG] ",
			Labels: []string{"bug"},
			Body: []interface{}{
				map[string]interface{}{
					"type": "markdown",
					"attributes": map[string]interface{}{
						"value": "Thanks for reporting!",
					},
				},
			},
		}

		generated := &models.IssueGenerationResult{
			Title:       "Application crashes",
			Description: "The app crashes when clicking submit",
			Labels:      []string{"needs-triage"},
		}

		result := service.MergeWithGeneratedContent(template, generated)

		assert.Equal(t, "[BUG] Application crashes", result.Title)
		assert.Contains(t, result.Description, "The app crashes when clicking submit")
		assert.Contains(t, result.Labels, "bug")
		assert.Contains(t, result.Labels, "needs-triage")
		assert.Len(t, result.Labels, 2)
	})

	t.Run("Merge without template title", func(t *testing.T) {
		template := &models.IssueTemplate{
			Title:  "",
			Labels: []string{"enhancement"},
		}

		generated := &models.IssueGenerationResult{
			Title:       "Add dark mode",
			Description: "Users want dark mode",
			Labels:      []string{"ui"},
		}

		result := service.MergeWithGeneratedContent(template, generated)

		assert.Equal(t, "Add dark mode", result.Title)
		assert.Contains(t, result.Description, "Users want dark mode")
		assert.Contains(t, result.Labels, "enhancement")
		assert.Contains(t, result.Labels, "ui")
	})
}
func TestIssueTemplateService_GetTemplateByName_NotFound(t *testing.T) {
	cfg := &config.Config{ActiveVCSProvider: "github"}
	service := NewIssueTemplateService(WithTemplateConfig(cfg))
	_, err := service.GetTemplateByName(context.Background(), "ghost")
	assert.Error(t, err)
}

func TestIssueTemplateService_MergeWithGeneratedContent_Realistic(t *testing.T) {
	cfg := &config.Config{}
	service := NewIssueTemplateService(WithTemplateConfig(cfg))
	template := &models.IssueTemplate{
		Title: "[BUG] ",
		Body: []interface{}{
			map[string]interface{}{
				"type": "textarea",
				"id":   "repro",
				"attributes": map[string]interface{}{
					"label": "Steps to reproduce",
				},
			},
		},
	}
	generated := &models.IssueGenerationResult{
		Title:       "Server error 500",
		Description: "- Go to /home\n- Click login",
	}

	result := service.MergeWithGeneratedContent(template, generated)
	assert.Equal(t, "[BUG] Server error 500", result.Title)
	assert.Contains(t, result.Description, "- Go to /home")
}

func TestIssueTemplateService_PRTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	origCwd, _ := os.Getwd()
	err := os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origCwd) }()

	cfg := &config.Config{ActiveVCSProvider: "github"}
	service := NewIssueTemplateService(WithTemplateConfig(cfg))

	t.Run("GetPRTemplatesDir", func(t *testing.T) {
		dir, err := service.GetPRTemplatesDir(context.Background())
		assert.NoError(t, err)
		assert.Contains(t, dir, ".github/PULL_REQUEST_TEMPLATE")
	})

	t.Run("ListPRTemplates - Single File", func(t *testing.T) {
		_ = os.MkdirAll(filepath.Join(tmpDir, ".github"), 0755)
		err := os.WriteFile(filepath.Join(tmpDir, ".github", "PULL_REQUEST_TEMPLATE.md"), []byte("## Template"), 0644)
		require.NoError(t, err)

		templates, err := service.ListPRTemplates(context.Background())
		assert.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "PULL_REQUEST_TEMPLATE.md", templates[0].FilePath)
	})

	t.Run("ListPRTemplates - Directory", func(t *testing.T) {
		_ = os.Remove(filepath.Join(tmpDir, ".github", "PULL_REQUEST_TEMPLATE.md"))

		targetDir := filepath.Join(tmpDir, ".github", "PULL_REQUEST_TEMPLATE")
		_ = os.MkdirAll(targetDir, 0755)

		err := os.WriteFile(filepath.Join(targetDir, "custom.md"), []byte("## Custom"), 0644)
		require.NoError(t, err)

		templates, err := service.ListPRTemplates(context.Background())
		assert.NoError(t, err)
		assert.Len(t, templates, 1)
		assert.Equal(t, "custom.md", templates[0].FilePath)
	})

	t.Run("GetPRTemplate - Single File", func(t *testing.T) {
		_ = os.WriteFile(filepath.Join(tmpDir, ".github", "PULL_REQUEST_TEMPLATE.md"), []byte("## Main\nContent"), 0644)

		tmpl, err := service.GetPRTemplate(context.Background(), "")
		assert.NoError(t, err)
		assert.Contains(t, tmpl.BodyContent, "## Main")

		tmpl, err = service.GetPRTemplate(context.Background(), "PULL_REQUEST_TEMPLATE.md")
		assert.NoError(t, err)
		assert.Contains(t, tmpl.BodyContent, "Content")
	})
}
