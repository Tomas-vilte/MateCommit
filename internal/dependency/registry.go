package dependency

import (
	"context"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

type AnalyzerRegistry struct {
	analyzers []vcs.DependencyAnalyzer
}

func NewAnalyzerRegistry() *AnalyzerRegistry {
	return &AnalyzerRegistry{
		analyzers: []vcs.DependencyAnalyzer{
			NewGoModAnalyzer(),
			NewPackageJsonAnalyzer(),
		},
	}
}

// AnalyzeAll executes all applicable analyzers and combines the results
func (r *AnalyzerRegistry) AnalyzeAll(ctx context.Context, vcsClient vcs.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
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
