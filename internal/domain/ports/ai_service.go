package ports

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type AIProvider interface {
	GenerateCommitMessage(ctx context.Context, info models.CommitInfo) (string, error)
}
