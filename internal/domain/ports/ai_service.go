package ports

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type AIProvider interface {
	GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]string, error)
}
