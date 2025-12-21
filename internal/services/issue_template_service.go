package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
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

func (s *IssueTemplateService) GetTemplatesDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
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

	return templatesDir, nil
}

func (s *IssueTemplateService) ListTemplates() ([]models.TemplateMetadata, error) {
	templatesDir, err := s.GetTemplatesDir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		return []models.TemplateMetadata{}, nil
	}

	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read templates directory", err)
	}

	templates := make([]models.TemplateMetadata, 0)
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml")) {
			continue
		}

		filePath := filepath.Join(templatesDir, entry.Name())
		template, err := s.LoadTemplate(filePath)
		if err != nil {
			continue
		}

		templates = append(templates, models.TemplateMetadata{
			Name:     template.Name,
			About:    template.GetAbout(),
			FilePath: entry.Name(),
		})
	}

	return templates, nil
}

func (s *IssueTemplateService) LoadTemplate(filePath string) (*models.IssueTemplate, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to read template file: %s", filePath), err)
	}
	return s.parseTemplate(string(content), filePath)
}

func (s *IssueTemplateService) parseTemplate(content string, filePath string) (*models.IssueTemplate, error) {
	template := &models.IssueTemplate{
		FilePath: filePath,
	}

	// For .yml files, we parse all content directly as YAML
	if err := yaml.Unmarshal([]byte(content), template); err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("failed to parse YAML template: %s", filePath), err)
	}

	return template, nil
}

func (s *IssueTemplateService) GetTemplateByName(name string) (*models.IssueTemplate, error) {
	templatesDir, err := s.GetTemplatesDir()
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
			return s.LoadTemplate(path)
		}
	}

	return nil, domainErrors.NewAppError(domainErrors.TypeConfiguration, fmt.Sprintf("template '%s' not found", name), nil)
}

func (s *IssueTemplateService) InitializeTemplates(force bool) error {
	templatesDir, err := s.GetTemplatesDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(templatesDir, 0755); err != nil {
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
			skipped++
			continue
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return domainErrors.NewAppError(domainErrors.TypeInternal, fmt.Sprintf("failed to write template: %s", filePath), err)
		}
		created++
	}

	if created == 0 && skipped > 0 {
		return domainErrors.NewAppError(domainErrors.TypeConfiguration, "templates_already_exist", nil)
	}
	return nil
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
