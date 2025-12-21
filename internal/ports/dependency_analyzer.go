package ports

import (
	"context"

	"github.com/thomas-vilte/matecommit/internal/models"
)

// DependencyAnalyzer defines the interface to analyze dependencies for different languages
type DependencyAnalyzer interface {
	// CanHandle detects if this analyzer can handle the project
	CanHandle(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) bool

	// AnalyzeChanges analyzes dependency changes between two versions
	AnalyzeChanges(ctx context.Context, vcsClient VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error)

	// Name returns the name of the dependency manager
	Name() string
}
