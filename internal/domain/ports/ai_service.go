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

// ReleaseNotesGenerator define la interfaz para generar notas de release.
type ReleaseNotesGenerator interface {
	GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error)
}

// IssueContentGenerator define la interfaz para generar contenido de issues con IA.
type IssueContentGenerator interface {
	// GenerateIssueContent genera el título, descripción y labels de una issue
	// basándose en el contexto proporcionado en el request.
	GenerateIssueContent(ctx context.Context, request models.IssueGenerationRequest) (*models.IssueGenerationResult, error)
}
