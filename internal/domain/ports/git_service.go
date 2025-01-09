package ports

import "github.com/Tomas-vilte/MateCommit/internal/domain/models"

type GitService interface {
	GetChangedFiles() ([]models.GitChange, error)
	GetDiff() (string, error)
	CreateCommit(message string) error
}
