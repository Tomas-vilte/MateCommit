package ports

import "github.com/Tomas-vilte/MateCommit/internal/domain/models"

type CommitHandler interface {
	HandleSuggestions(suggestions []models.CommitSuggestion) error
}
