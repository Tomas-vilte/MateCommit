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
	// CreateRelease crea una nueva release en el repositorio
	// buildBinaries indica si se deben compilar y subir binarios (opcional, solo algunos proveedores lo soportan)
	CreateRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool) error
	// GetRelease obtiene la release del repositorio
	GetRelease(ctx context.Context, version string) (*models.VCSRelease, error)
	// UpdateRelease actualiza una release del repositorio
	UpdateRelease(ctx context.Context, version, body string) error
	// GetClosedIssuesBetweenTags obtiene issues cerrados entre dos tags
	GetClosedIssuesBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.Issue, error)
	// GetMergedPRsBetweenTags obtiene PRs mergeados entre dos tags
	GetMergedPRsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]models.PullRequest, error)
	// GetContributorsBetweenTags obtiene contributors entre dos tags
	GetContributorsBetweenTags(ctx context.Context, previousTag, currentTag string) ([]string, error)
	// GetFileStatsBetweenTags obtiene estadísticas de archivos entre dos tags
	GetFileStatsBetweenTags(ctx context.Context, previousTag, currentTag string) (*models.FileStatistics, error)
	// GetIssue obtiene información de un issue/ticket por su número
	GetIssue(ctx context.Context, issueNumber int) (*models.Issue, error)
	// GetFileAtTag obtiene el contenido de un archivo en un tag específico
	GetFileAtTag(ctx context.Context, tag, filepath string) (string, error)
	// GetPRIssues obtiene issues relacionadas con un PR basándose en branch name, commits y descripción
	GetPRIssues(ctx context.Context, branchName string, commits []string, prDescription string) ([]models.Issue, error)
	// UpdateIssueChecklist actualiza el checklist de un issue marcando elementos como completados
	UpdateIssueChecklist(ctx context.Context, issueNumber int, indices []int) error
	// CreateIssue crea una nueva issue en el repositorio
	CreateIssue(ctx context.Context, title string, body string, labels []string, assignees []string) (*models.Issue, error)
	// GetAuthenticatedUser obtiene el usuario autenticado actual
	GetAuthenticatedUser(ctx context.Context) (string, error)
}
