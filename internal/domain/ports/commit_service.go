package ports

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type CommitService interface {
	GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error)
}
