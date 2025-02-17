package ports

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// CommitSummarizer es una interfaz que define el servicio para generar sugerencias de commits.
type CommitSummarizer interface {
	//GenerateSuggestions genera una lista de sugerencias de mensajes de commit.
	GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error)
}

// PRSummarizer define la interfaz para los servicios que resumen Pull Requests.
type PRSummarizer interface {
	// GeneratePRSummary genera un resumen de un Pull Request dado un prompt.
	GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error)
}
