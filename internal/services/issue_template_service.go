package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"gopkg.in/yaml.v3"
)

var _ ports.IssueTemplateService = (*IssueTemplateService)(nil)

type IssueTemplateService struct {
	config *config.Config
	trans  *i18n.Translations
}

func NewIssueTemplateService(cfg *config.Config, trans *i18n.Translations) *IssueTemplateService {
	return &IssueTemplateService{
		config: cfg,
		trans:  trans,
	}
}

func (s *IssueTemplateService) GetTemplatesDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error obteniendo directorio actual: %w", err)
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
		return nil, fmt.Errorf("error leyendo directorio de templates: %w", err)
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
		return nil, fmt.Errorf("error leyendo archivo de template: %w", err)
	}
	return s.parseTemplate(string(content), filePath)
}

func (s *IssueTemplateService) parseTemplate(content string, filePath string) (*models.IssueTemplate, error) {
	template := &models.IssueTemplate{
		FilePath: filePath,
	}

	// Para archivos .yml, parseamos directamente todo el contenido como YAML
	if err := yaml.Unmarshal([]byte(content), template); err != nil {
		return nil, fmt.Errorf("error parseando template YAML: %w", err)
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

	return nil, fmt.Errorf("template '%s' no encontrado", name)
}

func (s *IssueTemplateService) InitializeTemplates(force bool) error {
	templatesDir, err := s.GetTemplatesDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("error creando directorio de templates: %w", err)
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
			return fmt.Errorf("error escribiendo template %s: %w", filePath, err)
		}
		created++
	}

	if created == 0 && skipped > 0 {
		return fmt.Errorf("los templates ya existen, usa --force para sobrescribirlas")
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
		"name":        s.trans.GetMessage("issue_template.bug_report_name", 0, nil),
		"description": s.trans.GetMessage("issue_template.bug_report_about", 0, nil),
		"title":       s.trans.GetMessage("issue_template.bug_report_title", 0, nil),
		"labels":      []string{"bug"},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": s.trans.GetMessage("issue_template.bug_report_intro", 0, nil),
				},
			},
			{
				"type": "textarea",
				"id":   "description",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.bug_description_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.bug_description_help", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.bug_description_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "steps",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.bug_steps_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.bug_steps_help", 0, nil),
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
					"label":       s.trans.GetMessage("issue_template.bug_expected_label", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.bug_expected_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "actual",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.bug_actual_label", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.bug_actual_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "input",
				"id":   "version",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.bug_version_label", 0, nil),
					"placeholder": "v1.0.0",
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.bug_additional_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.bug_additional_help", 0, nil),
				},
			},
		},
	}

	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildFeatureRequestTemplate() string {
	template := map[string]interface{}{
		"name":        s.trans.GetMessage("issue_template.feature_request_name", 0, nil),
		"description": s.trans.GetMessage("issue_template.feature_request_about", 0, nil),
		"title":       s.trans.GetMessage("issue_template.feature_request_title", 0, nil),
		"labels":      []string{"enhancement"},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": s.trans.GetMessage("issue_template.feature_intro", 0, nil),
				},
			},
			{
				"type": "textarea",
				"id":   "problem",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.feature_problem_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.feature_problem_help", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.feature_problem_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "solution",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.feature_solution_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.feature_solution_help", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.feature_solution_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "alternatives",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.feature_alternatives_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.feature_alternatives_help", 0, nil),
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.feature_additional_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.feature_additional_help", 0, nil),
				},
			},
		},
	}

	content, _ := yaml.Marshal(template)
	return string(content)
}

func (s *IssueTemplateService) buildCustomTemplate() string {
	template := map[string]interface{}{
		"name":        s.trans.GetMessage("issue_template.custom_issue_name", 0, nil),
		"description": s.trans.GetMessage("issue_template.custom_issue_about", 0, nil),
		"title":       s.trans.GetMessage("issue_template.custom_issue_title", 0, nil),
		"labels":      []string{},
		"body": []map[string]interface{}{
			{
				"type": "markdown",
				"attributes": map[string]string{
					"value": s.trans.GetMessage("issue_template.custom_intro", 0, nil),
				},
			},
			{
				"type": "textarea",
				"id":   "description",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.custom_description_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.custom_description_help", 0, nil),
					"placeholder": s.trans.GetMessage("issue_template.custom_description_placeholder", 0, nil),
				},
				"validations": map[string]bool{
					"required": true,
				},
			},
			{
				"type": "textarea",
				"id":   "additional",
				"attributes": map[string]interface{}{
					"label":       s.trans.GetMessage("issue_template.custom_additional_label", 0, nil),
					"description": s.trans.GetMessage("issue_template.custom_additional_help", 0, nil),
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

	// Solo para templates .md que tienen Body como string
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
