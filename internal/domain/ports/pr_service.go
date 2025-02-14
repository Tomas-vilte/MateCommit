package ports

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// PRService define la interfaz para el servicio de resumen de Pull Requests.
type PRService interface {
	SummarizePR(ctx context.Context, prNumber int, contextAdditional string) (models.PRSummary, error)
}
