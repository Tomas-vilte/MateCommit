package ports

import "github.com/Tomas-vilte/MateCommit/internal/domain/models"

// GitService define los métodos para interactuar con el sistema de control de versiones Git
type GitService interface {

	// GetChangedFiles Obtiene los archivos modificados
	GetChangedFiles() ([]models.GitChange, error)

	// GetDiff Obtiene el diff completo
	GetDiff() (string, error)

	// HasStagedChanges Verifica si hay cambios en staging
	HasStagedChanges() bool

	// CreateCommit crea los commits
	CreateCommit(message string) error

	// AddFileToStaging agrega un archivo al área de staging
	AddFileToStaging(file string) error

	// GetCurrentBranch obtiene el nombre de la branch actual
	GetCurrentBranch() (string, error)

	// GetRepoInfo obtiene la informacion del repo
	GetRepoInfo() (string, string, string, error)
}
