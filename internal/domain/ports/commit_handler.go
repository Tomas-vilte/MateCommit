package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type CommitHandler interface {
	HandleSuggestions(ctx context.Context, suggestions []models.CommitSuggestion) error
}
