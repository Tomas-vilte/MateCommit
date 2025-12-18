package models

// IssueTemplate representa un template de issue con su metadata.
type IssueTemplate struct {
	// Metadata del frontmatter YAML
	Name        string   `yaml:"name"`
	About       string   `yaml:"about,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Title       string   `yaml:"title"`
	Labels      []string `yaml:"labels"`
	Assignees   []string `yaml:"assignees,omitempty"`

	// Contenido del template
	// Para .md: string con markdown
	// Para .yml (GitHub Issue Forms): array de campos del formulario
	Body        interface{} `yaml:"body,omitempty"`
	BodyContent string      `yaml:"-"` // Para retrocompatibilidad con .md

	// Path al archivo del template
	FilePath string `yaml:"-"`
}

// GetAbout retorna la descripción del template (usa 'description' o 'about')
func (t *IssueTemplate) GetAbout() string {
	if t.Description != "" {
		return t.Description
	}
	return t.About
}

type TemplateMetadata struct {
	Name     string
	About    string
	FilePath string
}
