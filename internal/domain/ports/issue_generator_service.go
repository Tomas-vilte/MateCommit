package ports

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// IssueGeneratorService define la interfaz para el servicio de generación de issues
type IssueGeneratorService interface {
	// GenerateFromDiff genera contenido para una issue a partir del diff actual
	GenerateFromDiff(ctx context.Context, hint string, skipLabels bool) (*models.IssueGenerationResult, error)

	// GenerateFromDescription genera contenido para una issue a partir de una descripción manual
	GenerateFromDescription(ctx context.Context, description string, skipLabels bool) (*models.IssueGenerationResult, error)

	// GenerateFromPR genera contenido para una issue a partir de un Pull Request existente
	GenerateFromPR(ctx context.Context, prNumber int, hint string, skipLabels bool) (*models.IssueGenerationResult, error)

	// GenerateWithTemplate genera contenido usando un template específico
	GenerateWithTemplate(ctx context.Context, templateName string, hint string, fromDiff bool, description string, skipLabels bool) (*models.IssueGenerationResult, error)

	// CreateIssue crea la issue en el sistema VCS
	CreateIssue(ctx context.Context, result *models.IssueGenerationResult, assignees []string) (*models.Issue, error)

	// GetAuthenticatedUser obtiene el usuario autenticado actual
	GetAuthenticatedUser(ctx context.Context) (string, error)

	// InferBranchName infiere un nombre de rama basado en el número de issue y las etiquetas
	InferBranchName(issueNumber int, labels []string) string

	// LinkIssueToPR vincula una issue a un Pull Request
	LinkIssueToPR(ctx context.Context, prNumber int, issueNumber int) error
}
