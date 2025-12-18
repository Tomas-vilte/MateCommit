package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type ReleaseService interface {
	// AnalyzeNextRelease analiza commits y determina la siguiente versión
	AnalyzeNextRelease(ctx context.Context) (*models.Release, error)

	// GenerateReleaseNotes genera las notas del release con IA
	GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error)

	// PublishRelease crea el release en el VCS (GitHub, GitLab, etc.)
	// buildBinaries indica si se deben compilar y subir binarios al release
	PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool) error

	// CreateTag crea un tag git con la versión y mensaje especificados
	CreateTag(ctx context.Context, version, message string) error

	// PushTag sube el tag al repositorio remoto
	PushTag(ctx context.Context, version string) error

	// GetRelease obtiene una release del repositorio
	GetRelease(ctx context.Context, version string) (*models.VCSRelease, error)

	// UpdateRelease actualiza una release del repositorio
	UpdateRelease(ctx context.Context, version, body string) error

	// EnrichReleaseContext enriquece el release con información adicional de GitHub (issues, PRs, contributors, etc.)
	EnrichReleaseContext(ctx context.Context, release *models.Release) error
	UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error
	CommitChangelog(ctx context.Context, version string) error
	PushChanges(ctx context.Context) error
	UpdateAppVersion(version string) error
}
