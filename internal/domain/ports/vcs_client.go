package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// VCSClient define los métodos comunes para interactuar con las APIs de los sistemas de control de versiones.
type VCSClient interface {
	// UpdatePR actualiza una Pull Request (título, body y etiquetas) en el proveedor.
	UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error
	// GetPR obtiene los datos de PR (por ejemplo, para extraer commits, diff, etc.).
	GetPR(ctx context.Context, prNumber int) (models.PRData, error)
	// GetRepoLabels obtiene todas las labels disponibles en el repositorio
	GetRepoLabels(ctx context.Context) ([]string, error)
	// CreateLabel crea una nueva label en el repositorio
	CreateLabel(ctx context.Context, name string, color string, description string) error
	// AddLabelsToPR agrega labels específicas a un PR
	AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error
	CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error
}
