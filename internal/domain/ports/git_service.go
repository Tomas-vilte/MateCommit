package ports

import "github.com/Tomas-vilte/MateCommit/internal/domain/models"

type GitService interface {

	// GetChangedFiles Obtiene los archivos modificados
	GetChangedFiles() ([]models.GitChange, error)

	// GetDiff Obtiene el diff completo
	GetDiff() (string, error)
}
