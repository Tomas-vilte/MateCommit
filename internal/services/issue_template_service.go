package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"gopkg.in/yaml.v3"
)

type IssueTemplateService struct {
	config *config.Config
}

type IssueOption func(*IssueTemplateService)

func WithTemplateConfig(cfg *config.Config) IssueOption {
	return func(s *IssueTemplateService) {
		s.config = cfg
	}
}

func NewIssueTemplateService(opts ...IssueOption) *IssueTemplateService {
	s := &IssueTemplateService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *IssueTemplateService) GetTemplatesDir(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error(ctx, "failed to get current working directory", err)
		return "", domainErrors.NewAppError(domainErrors.TypeInternal, "failed to get current working directory", err)
	}

	provider := strings.ToLower(s.config.ActiveVCSProvider)
	var templatesDir string

	switch provider {
	case "gitlab":
		templatesDir = filepath.Join(cwd, ".gitlab", "issue_templates")
	case "github":
		fallthrough
	default:
		templatesDir = filepath.Join(cwd, ".github", "ISSUE_TEMPLATE")
	}

	logger.Debug(ctx, "identified templates directory", "provider", provider, "path", templatesDir)
	return templatesDir, nil
}

func (s *IssueTemplateService) ListTemplates(ctx context.Context) ([]models.TemplateMetadata, error) {
	templatesDir, err := s.GetTemplatesDir(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		logger.Debug(ctx, "templates directory does not exist, returning empty list", "path", templatesDir)
		return []models.TemplateMetadata{}, nil
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		logger.Error(ctx, "failed to read templates directory", err, "path", templatesDir)
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read templates directory", err)
	}

	templates := make([]models.TemplateMetadata, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yml") &&
			!strings.HasSuffix(entry.Name(), ".yaml") &&
			!strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(templatesDir, entry.Name())
		var template *models.IssueTemplate

		if strings.HasSuffix(entry.Name(), ".md") {
			template, err = s.LoadMarkdownTemplate(ctx, filePath)
		} else {
			template, err = s.LoadTemplate(ctx, filePath)
		}
		if err != nil {
			logger.Warn(ctx, "skipping invalid template", "path", filePath, "error", err)
			continue
		}

		templates = append(templates, models.TemplateMetadata{
			Name:     template.Name,
			About:    template.GetAbout(),
			FilePath: entry.Name(),
		})
	}

	logger.Debug(ctx, "listed templates", "count", len(templates))
	return templates, nil
}

func (s *IssueTemplateService) LoadTemplate(ctx context.Context, filePath string) (*models.IssueTemplate, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error(ctx, "failed to read template file", err, "path", filePath)
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to read template file: %s", filePath), err)
	}
	return s.parseTemplate(ctx, string(content), filePath)
}

func (s *IssueTemplateService) parseTemplate(ctx context.Context, content string, filePath string) (*models.IssueTemplate, error) {
	template := &models.IssueTemplate{
		FilePath: filePath,
	}

	if err := yaml.Unmarshal([]byte(content), template); err != nil {
		logger.Error(ctx, "failed to parse YAML template", err, "path", filePath)
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to parse YAML template: %s", filePath), err)
	}

	logger.Debug(ctx, "successfully loaded/parsed template", "name", template.Name, "path", filePath)
	return template, nil
}

func (s *IssueTemplateService) GetTemplateByName(ctx context.Context, name string) (*models.IssueTemplate, error) {
	templatesDir, err := s.GetTemplatesDir(ctx)
	if err != nil {
		return nil, err
	}

	possiblePaths := []string{
		filepath.Join(templatesDir, name+".yml"),
		filepath.Join(templatesDir, name+".yaml"),
		filepath.Join(templatesDir, name),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return s.LoadTemplate(ctx, path)
		}
	}

	logger.Warn(ctx, "template not found by name", "name", name, "searched_paths", possiblePaths)
	return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("template '%s' not found", name), nil)
}

func (s *IssueTemplateService) InitializeTemplates(ctx context.Context, force bool) error {
	templatesDir, err := s.GetTemplatesDir(ctx)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		logger.Error(ctx, "failed to create templates directory", err, "path", templatesDir)
		return domainErrors.NewAppError(domainErrors.TypeInternal, "failed to create templates directory", err)
	}

	templates := map[string]string{
		"bug_report.yml":           s.buildTemplateContent("bug_report"),
		"feature_request.yml":      s.buildTemplateContent("feature_request"),
		"custom.yml":               s.buildTemplateContent("custom"),
		"performance.yml":          s.buildPerformanceTemplate(),
		"documentation.yml":        s.buildDocumentationTemplate(),
		"security.yml":             s.buildSecurityTemplate(),
		"tech_debt.yml":            s.buildTechDebtTemplate(),
		"question.yml":             s.buildQuestionTemplate(),
		"dependency.yml":           s.buildDependencyTemplate(),
		"PULL_REQUEST_TEMPLATE.md": s.buildDefaultPRTemplate(),
	}

	created := 0
	skipped := 0

	for filename, content := range templates {
		filePath := filepath.Join(templatesDir, filename)
		if filename == "PULL_REQUEST_TEMPLATE.md" && strings.HasSuffix(templatesDir, "ISSUE_TEMPLATE") {
			filePath = filepath.Join(filepath.Dir(templatesDir), filename)
		}

		if _, err := os.Stat(filePath); err == nil && !force {
			logger.Debug(ctx, "template already exists, skipping", "path", filePath)
			skipped++
			continue
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			logger.Error(ctx, "failed to write template file during initialization", err, "path", filePath)
			return domainErrors.NewAppError(domainErrors.TypeInternal, fmt.Sprintf("failed to write template: %s", filePath), err)
		}
		logger.Info(ctx, "successfully created template", "path", filePath)
		created++
	}

	logger.Info(ctx, "template initialization complete", "created", created, "skipped", skipped)
	if created == 0 && skipped > 0 {
		return domainErrors.NewAppError(domainErrors.TypeConfiguration, "templates_already_exist", nil)
	}
	return nil
}

// GetPRTemplatesDir returns the directory where PR templates are stored
func (s *IssueTemplateService) GetPRTemplatesDir(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error(ctx, "failed to get current working directory", err)
		return "", domainErrors.NewAppError(domainErrors.TypeInternal, "failed to get current working directory", err)
	}
	provider := strings.ToLower(s.config.ActiveVCSProvider)
	var templatesDir string
	switch provider {
	case "gitlab":
		templatesDir = filepath.Join(cwd, ".gitlab", "merge_request_templates")
	case "github":
		fallthrough
	default:
		templatesDir = filepath.Join(cwd, ".github", "PULL_REQUEST_TEMPLATE")
	}
	logger.Debug(ctx, "identified PR templates directory", "provider", provider, "path", templatesDir)
	return templatesDir, nil
}

func (s *IssueTemplateService) ListPRTemplates(ctx context.Context) ([]models.TemplateMetadata, error) {
	templatesDir, err := s.GetPRTemplatesDir(ctx)
	if err != nil {
		return nil, err
	}

	templates := make([]models.TemplateMetadata, 0)

	singleTemplatePath := filepath.Join(filepath.Dir(templatesDir), "PULL_REQUEST_TEMPLATE.md")
	if _, err := os.Stat(singleTemplatePath); err == nil {
		template, err := s.LoadMarkdownTemplate(ctx, singleTemplatePath)
		if err != nil {
			logger.Warn(ctx, "failed to load single PR template", "path", singleTemplatePath, "error", err)
		} else {
			templates = append(templates, models.TemplateMetadata{
				Name:     "Default PR Template",
				About:    template.GetAbout(),
				FilePath: "PULL_REQUEST_TEMPLATE.md",
			})
		}
	}

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		logger.Debug(ctx, "PR templates directory does not exist", "path", templatesDir)
		return templates, nil
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		logger.Error(ctx, "failed to read PR templates directory", err, "path", templatesDir)
		return templates, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".md") &&
			!strings.HasSuffix(entry.Name(), ".yml") &&
			!strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		filePath := filepath.Join(templatesDir, entry.Name())
		var template *models.IssueTemplate
		var err error

		if strings.HasSuffix(entry.Name(), ".md") {
			template, err = s.LoadMarkdownTemplate(ctx, filePath)
		} else {
			template, err = s.LoadTemplate(ctx, filePath)
		}
		if err != nil {
			logger.Warn(ctx, "skipping invalid PR template", "path", filePath, "error", err)
			continue
		}

		templates = append(templates, models.TemplateMetadata{
			Name:     entry.Name(),
			About:    template.GetAbout(),
			FilePath: entry.Name(),
		})
	}

	logger.Debug(ctx, "listed PR templates",
		"count", len(templates),
	)
	return templates, nil
}

// LoadMarkdownTemplate loads a Markdown template file
func (s *IssueTemplateService) LoadMarkdownTemplate(ctx context.Context, filePath string) (*models.IssueTemplate, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		logger.Error(ctx, "failed to read markdown template file", err, "path", filePath)
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to read template file: %s", filePath), err)
	}

	return s.parseMarkdownTemplate(ctx, string(content), filePath)
}

// parseMarkdownTemplate parses a Markdown template, extracting YAML frontmatter if present
func (s *IssueTemplateService) parseMarkdownTemplate(ctx context.Context, content string, filePath string) (*models.IssueTemplate, error) {
	template := &models.IssueTemplate{
		FilePath: filePath,
	}

	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content, "---\n", 3)
		if len(parts) >= 3 {
			frontmatter := parts[1]
			if err := yaml.Unmarshal([]byte(frontmatter), template); err != nil {
				logger.Warn(ctx, "failed to parse YAML frontmatter, using as plain markdown", "path", filePath, "error", err)
			}
			template.BodyContent = strings.TrimSpace(parts[2])
		} else {
			template.BodyContent = content
		}
	} else {
		template.BodyContent = content
	}

	if template.Name == "" {
		template.Name = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	logger.Debug(ctx, "successfully loaded/parsed markdown template", "name", template.Name, "path", filePath)
	return template, nil
}

// GetPRTemplate loads a specific PR template by name
func (s *IssueTemplateService) GetPRTemplate(ctx context.Context, name string) (*models.IssueTemplate, error) {
	templatesDir, err := s.GetPRTemplatesDir(ctx)
	if err != nil {
		return nil, err
	}
	singleTemplatePath := filepath.Join(filepath.Dir(templatesDir), "PULL_REQUEST_TEMPLATE.md")
	if name == "" || name == "PULL_REQUEST_TEMPLATE.md" {
		if _, err := os.Stat(singleTemplatePath); err == nil {
			return s.LoadMarkdownTemplate(ctx, singleTemplatePath)
		}
	}
	possiblePaths := []string{
		filepath.Join(templatesDir, name+".md"),
		filepath.Join(templatesDir, name+".yml"),
		filepath.Join(templatesDir, name+".yaml"),
		filepath.Join(templatesDir, name),
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			if strings.HasSuffix(path, ".md") {
				return s.LoadMarkdownTemplate(ctx, path)
			}
			return s.LoadTemplate(ctx, path)
		}
	}
	logger.Warn(ctx, "PR template not found by name", "name", name, "searched_paths", possiblePaths)
	return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("PR template '%s' not found", name), nil)
}

func (s *IssueTemplateService) buildTemplateContent(templateType string) string {
	switch templateType {
	case "bug_report":
		return s.buildBugReportTemplate()
	case "feature_request":
		return s.buildFeatureRequestTemplate()
	case "custom":
		return s.buildCustomTemplate()
	default:
		return ""
	}
}

func (s *IssueTemplateService) buildBugReportTemplate() string {
	template := models.IssueTemplate{
		Name:        "Bug report",
		Description: "Create a report to help us improve",
		Title:       "[BUG] ",
		Labels:      []string{"bug"},
		Body: []models.IssueFormItem{
			{
				Type: "markdown",
				Attributes: models.FormAttributes{
					Value: "Thank you for reporting a bug!",
				},
			},
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Description",
					Description: "A clear and concise description of what the bug is.",
					Placeholder: "Enter bug description",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "steps",
				Attributes: models.FormAttributes{
					Label:       "Steps to reproduce",
					Description: "Explain how you encountered the bug.",
					Placeholder: "1. \n2. \n3. ",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "expected",
				Attributes: models.FormAttributes{
					Label:       "Expected behavior",
					Placeholder: "What did you expect to happen?",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "actual",
				Attributes: models.FormAttributes{
					Label:       "Actual behavior",
					Placeholder: "What actually happened?",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "input",
				ID:   "version",
				Attributes: models.FormAttributes{
					Label:       "Version",
					Placeholder: "v1.0.0",
				},
			},
			{
				Type: "textarea",
				ID:   "additional",
				Attributes: models.FormAttributes{
					Label:       "Additional information",
					Description: "Add any other context about the problem here.",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildFeatureRequestTemplate() string {
	template := models.IssueTemplate{
		Name:        "Feature request",
		Description: "Suggest an idea for this project",
		Title:       "[FEATURE] ",
		Labels:      []string{"enhancement"},
		Body: []models.IssueFormItem{
			{
				Type: "markdown",
				Attributes: models.FormAttributes{
					Value: "Thank you for suggesting a feature!",
				},
			},
			{
				Type: "textarea",
				ID:   "problem",
				Attributes: models.FormAttributes{
					Label:       "Problem description",
					Description: "A clear and concise description of what the problem is.",
					Placeholder: "I'm always frustrated when...",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "solution",
				Attributes: models.FormAttributes{
					Label:       "Proposed solution",
					Description: "A clear and concise description of what you want to happen.",
					Placeholder: "I would like to see...",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "alternatives",
				Attributes: models.FormAttributes{
					Label:       "Alternatives considered",
					Description: "A clear and concise description of any alternative solutions.",
				},
			},
			{
				Type: "textarea",
				ID:   "additional",
				Attributes: models.FormAttributes{
					Label:       "Additional information",
					Description: "Add any other context or screenshots about the feature request here.",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildCustomTemplate() string {
	template := models.IssueTemplate{
		Name:        "Custom issue",
		Description: "File a custom issue",
		Title:       "[ISSUE] ",
		Labels:      []string{},
		Body: []models.IssueFormItem{
			{
				Type: "markdown",
				Attributes: models.FormAttributes{
					Value: "Open a custom issue.",
				},
			},
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Description",
					Description: "Enter the issue description.",
					Placeholder: "Describe your issue here",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "additional",
				Attributes: models.FormAttributes{
					Label:       "Additional information",
					Description: "Any additional context.",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildPerformanceTemplate() string {
	template := models.IssueTemplate{
		Name:        "Performance Issue",
		Description: "Report a performance issue or inefficiency",
		Title:       "[PERF] ",
		Labels:      []string{"performance", "optimization"},
		Body: []models.IssueFormItem{
			{
				Type: "markdown",
				Attributes: models.FormAttributes{
					Value: "Thanks for helping us make things faster! Please describe the performance issue in detail.",
				},
			},
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Description",
					Description: "What is slow or inefficient?",
					Placeholder: "The dashboard takes 5 seconds to load...",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "input",
				ID:   "metric",
				Attributes: models.FormAttributes{
					Label:       "Metric (optional)",
					Description: "e.g., Response time, CPU usage, Memory",
					Placeholder: "500ms -> 2s",
				},
			},
			{
				Type: "textarea",
				ID:   "repro",
				Attributes: models.FormAttributes{
					Label:       "Steps to reproduce",
					Description: "How can we observe this?",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}
func (s *IssueTemplateService) buildDocumentationTemplate() string {
	template := models.IssueTemplate{
		Name:        "Documentation",
		Description: "Improvements or additions to documentation",
		Title:       "[DOCS] ",
		Labels:      []string{"documentation"},
		Body: []models.IssueFormItem{
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Description",
					Description: "What needs to be documented or improved?",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "location",
				Attributes: models.FormAttributes{
					Label:       "Relevant files/sections",
					Description: "Where should this check go?",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}
func (s *IssueTemplateService) buildSecurityTemplate() string {
	template := models.IssueTemplate{
		Name:        "Security Vulnerability",
		Description: "Report a security vulnerability",
		Title:       "[SECURITY] ",
		Labels:      []string{"security", "critical"},
		Body: []models.IssueFormItem{
			{
				Type: "markdown",
				Attributes: models.FormAttributes{
					Value: "**IMPORTANT:** Please do not disclose security vulnerabilities publicly until they have been addressed.",
				},
			},
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Vulnerability Description",
					Description: "Describe the security issue.",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "impact",
				Attributes: models.FormAttributes{
					Label:       "Impact",
					Description: "What is the potential impact of this vulnerability?",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}
func (s *IssueTemplateService) buildTechDebtTemplate() string {
	template := models.IssueTemplate{
		Name:        "Tech Debt / Refactor",
		Description: "Propose a refactoring or technical improvement",
		Title:       "[REFACTOR] ",
		Labels:      []string{"refactor", "tech-debt"},
		Body: []models.IssueFormItem{
			{
				Type: "textarea",
				ID:   "description",
				Attributes: models.FormAttributes{
					Label:       "Description",
					Description: "What code needs refactoring?",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "reason",
				Attributes: models.FormAttributes{
					Label:       "Reason",
					Description: "Why should we do this? (e.g. readability, maintainability)",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}
func (s *IssueTemplateService) buildQuestionTemplate() string {
	template := models.IssueTemplate{
		Name:        "Question",
		Description: "Ask a question about the project",
		Title:       "[QUESTION] ",
		Labels:      []string{"question"},
		Body: []models.IssueFormItem{
			{
				Type: "textarea",
				ID:   "question",
				Attributes: models.FormAttributes{
					Label:       "Question",
					Description: "What would you like to know?",
				},
				Validations: models.FormValidations{Required: true},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}
func (s *IssueTemplateService) buildDependencyTemplate() string {
	template := models.IssueTemplate{
		Name:        "Dependency Update",
		Description: "Update a project dependency",
		Title:       "[DEPENDENCY] ",
		Labels:      []string{"dependencies"},
		Body: []models.IssueFormItem{
			{
				Type: "input",
				ID:   "package",
				Attributes: models.FormAttributes{
					Label: "Package Name",
				},
				Validations: models.FormValidations{Required: true},
			},
			{
				Type: "textarea",
				ID:   "reason",
				Attributes: models.FormAttributes{
					Label:       "Reason for update",
					Description: "Security fix, new features, etc.",
				},
			},
		},
	}
	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult {
	ctx := context.Background()

	logger.Debug(ctx, "merging template with generated content",
		"template_title", template.Title,
		"generated_title", generated.Title,
		"generated_description_length", len(generated.Description))

	result := &models.IssueGenerationResult{
		Labels:    make([]string, 0),
		Assignees: make([]string, 0),
	}

	if template.Title != "" {
		result.Title = template.Title + generated.Title
	} else {
		result.Title = generated.Title
	}

	var descBuilder strings.Builder

	descBuilder.WriteString(generated.Description)
	descBuilder.WriteString("\n\n")

	if template.BodyContent != "" {
		descBuilder.WriteString("---\n\n")
		descBuilder.WriteString(template.BodyContent)
	}

	result.Description = descBuilder.String()

	logger.Debug(ctx, "merge result",
		"result_title", result.Title,
		"result_description_length", len(result.Description))

	labelMap := make(map[string]bool)
	for _, label := range template.Labels {
		if label != "" {
			labelMap[label] = true
		}
	}
	for _, label := range generated.Labels {
		if label != "" {
			labelMap[label] = true
		}
	}

	for label := range labelMap {
		result.Labels = append(result.Labels, label)
	}

	assigneeMap := make(map[string]bool)
	for _, assignee := range template.Assignees {
		if assignee != "" {
			assigneeMap[assignee] = true
		}
	}
	for _, assignee := range generated.Assignees {
		if assignee != "" {
			assigneeMap[assignee] = true
		}
	}

	for assignee := range assigneeMap {
		result.Assignees = append(result.Assignees, assignee)
	}

	return result
}

func (s *IssueTemplateService) buildDefaultPRTemplate() string {
	return `## Description
<!-- Describe your changes in detail here. code reference, etc -->
## Related Issues
<!-- Closes #1, Fixes #2 -->
## Type of Change
<!-- Check the relevant option -->
- [ ] 🐛 Bug fix (non-breaking change which fixes an issue)
- [ ] ✨ New feature (non-breaking change which adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] 📝 Documentation update
- [ ] 🎨 Style/Refactor (non-breaking change which improves code quality)
## Checklist
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
`
}
