package models

// IssueFormItem represents an item within a GitHub Issue Form (YAML).
type IssueFormItem struct {
	Type        string          `yaml:"type"`
	ID          string          `yaml:"id,omitempty"`
	Attributes  FormAttributes  `yaml:"attributes,omitempty"`
	Validations FormValidations `yaml:"validations,omitempty"`
}

// FormAttributes contains the visual and behavioral attributes of the field.
type FormAttributes struct {
	Label       string   `yaml:"label"`
	Description string   `yaml:"description,omitempty"`
	Placeholder string   `yaml:"placeholder,omitempty"`
	Value       string   `yaml:"value,omitempty"`   // For 'markdown' type elements
	Options     []string `yaml:"options,omitempty"` // For dropdowns
	Multiple    bool     `yaml:"multiple,omitempty"`
}

// FormValidations defines validation rules.
type FormValidations struct {
	Required bool `yaml:"required,omitempty"`
}
