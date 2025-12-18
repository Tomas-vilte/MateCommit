package ports

import (
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// IssueTemplateService define la interfaz para el servicio de gestión de templates de issues
type IssueTemplateService interface {
	// GetTemplatesDir obtiene el directorio donde se almacenan los templates
	GetTemplatesDir() (string, error)

	// ListTemplates lista todos los templates disponibles
	ListTemplates() ([]models.TemplateMetadata, error)

	// LoadTemplate carga un template desde un archivo
	LoadTemplate(filePath string) (*models.IssueTemplate, error)

	// GetTemplateByName obtiene un template por su nombre
	GetTemplateByName(name string) (*models.IssueTemplate, error)

	// InitializeTemplates inicializa los templates predefinidos
	InitializeTemplates(force bool) error

	// MergeWithGeneratedContent combina un template con contenido generado por IA
	MergeWithGeneratedContent(template *models.IssueTemplate, generated *models.IssueGenerationResult) *models.IssueGenerationResult
}
