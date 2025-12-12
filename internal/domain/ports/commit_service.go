package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

type CommitService interface {
	// GenerateSuggestions genera sugerencias de commit basadas en los cambios detectados
	GenerateSuggestions(ctx context.Context, count int) ([]models.CommitSuggestion, error)
	// GenerateSuggestionsWithIssue genera sugerencias considerando un issue específico
	GenerateSuggestionsWithIssue(ctx context.Context, count int, issueNumber int) ([]models.CommitSuggestion, error)
}
