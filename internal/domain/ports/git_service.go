package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// GitService define los métodos para interactuar con el sistema de control de versiones Git
type GitService interface {

	// GetChangedFiles Obtiene los archivos modificados
	GetChangedFiles(ctx context.Context) ([]models.GitChange, error)

	// GetDiff Obtiene el diff completo
	GetDiff(ctx context.Context) (string, error)

	// HasStagedChanges Verifica si hay cambios en staging
	HasStagedChanges(ctx context.Context) bool

	// CreateCommit crea los commits
	CreateCommit(ctx context.Context, message string) error

	// AddFileToStaging agrega un archivo al área de staging
	AddFileToStaging(ctx context.Context, file string) error

	// GetCurrentBranch obtiene el nombre de la branch actual
	GetCurrentBranch(ctx context.Context) (string, error)

	// GetRepoInfo obtiene la informacion del repo
	GetRepoInfo(ctx context.Context) (string, string, string, error)

	GetLastTag(ctx context.Context) (string, error)
	GetCommitCount(ctx context.Context) (int, error)
	GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error)
	CreateTag(ctx context.Context, version, message string) error
	PushTag(ctx context.Context, version string) error
}
