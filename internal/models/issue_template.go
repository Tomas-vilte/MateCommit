package models

// IssueTemplate represents an issue template with its metadata.
type IssueTemplate struct {
	// YAML frontmatter metadata
	Name        string   `yaml:"name"`
	About       string   `yaml:"about,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Title       string   `yaml:"title"`
	Labels      []string `yaml:"labels"`
	Assignees   []string `yaml:"assignees,omitempty"`

	// Template content
	// For .md: Markdown string
	// For .yml (GitHub Issue Forms): strict typed list of form items
	Body        []IssueFormItem `yaml:"body,omitempty"`
	BodyContent string          `yaml:"-"` // For backward compatibility with .md

	// Path to the template file
	FilePath string `yaml:"-"`
}

// GetAbout returns the template description (uses 'description' or 'about')
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
