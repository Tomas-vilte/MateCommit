package dependency

import (
	"context"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

type AnalyzerRegistry struct {
	analyzers []ports.DependencyAnalyzer
}

func NewAnalyzerRegistry() *AnalyzerRegistry {
	return &AnalyzerRegistry{
		analyzers: []ports.DependencyAnalyzer{
			NewGoModAnalyzer(),
			NewPackageJsonAnalyzer(),
		},
	}
}

// RegisterAnalyzer agrega un analyzer personalizado
func (r *AnalyzerRegistry) RegisterAnalyzer(analyzer ports.DependencyAnalyzer) {
	r.analyzers = append(r.analyzers, analyzer)
}

// AnalyzeAll ejecuta todos los analyzers aplicables y combina los resultados
func (r *AnalyzerRegistry) AnalyzeAll(ctx context.Context, vcsClient ports.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
	var allChanges []models.DependencyChange

	for _, analyzer := range r.analyzers {
		if analyzer.CanHandle(ctx, vcsClient, previousTag, currentTag) {
			changes, err := analyzer.AnalyzeChanges(ctx, vcsClient, previousTag, currentTag)
			if err != nil {
				continue
			}
			allChanges = append(allChanges, changes...)
		}
	}
	return allChanges, nil
}

// GetSupportedAnalyzers retorna lista de analyzers detectados
func (r *AnalyzerRegistry) GetSupportedAnalyzers(ctx context.Context, vcsClient ports.VCSClient, previousTag, currentTag string) []string {
	var supported []string

	for _, analyzer := range r.analyzers {
		if analyzer.CanHandle(ctx, vcsClient, previousTag, currentTag) {
			supported = append(supported, analyzer.Name())
		}
	}
	return supported
}
