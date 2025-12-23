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
		"bug_report.yml":      s.buildTemplateContent("bug_report"),
		"feature_request.yml": s.buildTemplateContent("feature_request"),
		"custom.yml":          s.buildTemplateContent("custom"),
	}

	created := 0
	skipped := 0

	for filename, content := range templates {
		filePath := filepath.Join(templatesDir, filename)

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
	template := map[string]interface{}{
		"name":        "Bug report",
		"description": "Create a report to help us improve",
		"title":       "[BUG] ",
		"labels":      []string{"bug"},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": "Thank you for reporting a bug!",
				},
			},
			{
				"type": "textarea",
				"id":   "description",
				"attributes": map[string]interface{}{
					"label":       "Description",
					"description": "A clear and concise description of what the bug is.",
					"placeholder": "Enter bug description",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "steps",
				"attributes": map[string]interface{}{
					"label":       "Steps to reproduce",
					"description": "Explain how you encountered the bug.",
					"placeholder": "1. \n2. \n3. ",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "expected",
				"attributes": map[string]interface{}{
					"label":       "Expected behavior",
					"placeholder": "What did you expect to happen?",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "actual",
				"attributes": map[string]interface{}{
					"label":       "Actual behavior",
					"placeholder": "What actually happened?",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "input",
				"id":   "version",
				"attributes": map[string]interface{}{
					"label":       "Version",
					"placeholder": "v1.0.0",
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       "Additional information",
					"description": "Add any other context about the problem here.",
				},
			},
		},
	}

	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildFeatureRequestTemplate() string {
	template := map[string]interface{}{
		"name":        "Feature request",
		"description": "Suggest an idea for this project",
		"title":       "[FEATURE] ",
		"labels":      []string{"enhancement"},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": "Thank you for suggesting a feature!",
				},
			},
			{
				"type": "textarea",
				"id":   "problem",
				"attributes": map[string]interface{}{
					"label":       "Problem description",
					"description": "A clear and concise description of what the problem is.",
					"placeholder": "I'm always frustrated when...",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "solution",
				"attributes": map[string]interface{}{
					"label":       "Proposed solution",
					"description": "A clear and concise description of what you want to happen.",
					"placeholder": "I would like to see...",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "alternatives",
				"attributes": map[string]interface{}{
					"label":       "Alternatives considered",
					"description": "A clear and concise description of any alternative solutions.",
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       "Additional information",
					"description": "Add any other context or screenshots about the feature request here.",
				},
			},
		},
	}

	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildCustomTemplate() string {
	template := map[string]interface{}{
		"name":        "Custom issue",
		"description": "File a custom issue",
		"title":       "[ISSUE] ",
		"labels":      []string{},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": "Open a custom issue.",
				},
			},
			{
				"type": "textarea",
				"id":   "description",
				"attributes": map[string]interface{}{
					"label":       "Description",
					"description": "Enter the issue description.",
					"placeholder": "Describe your issue here",
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       "Additional information",
					"description": "Any additional context.",
				},
			},
		},
	}

	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult {
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

	// Only for .md templates that have Body as a string
	if template.BodyContent != "" {
		descBuilder.WriteString("---\n\n")
		descBuilder.WriteString(template.BodyContent)
	}

	result.Description = descBuilder.String()

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
