package dependency

import (
	"context"
	"strings"

	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

var _ vcs.DependencyAnalyzer = (*GoModAnalyzer)(nil)

type GoModAnalyzer struct{}

func NewGoModAnalyzer() *GoModAnalyzer {
	return &GoModAnalyzer{}
}

func (g *GoModAnalyzer) Name() string {
	return "go.mod"
}

func (g *GoModAnalyzer) CanHandle(ctx context.Context, vcsClient vcs.VCSClient, _, currentTag string) bool {
	content, err := g.getFileContent(ctx, vcsClient, currentTag, "go.mod")
	return err == nil && content != ""
}

func (g *GoModAnalyzer) AnalyzeChanges(ctx context.Context, vcsClient vcs.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
	oldContent, err := g.getFileContent(ctx, vcsClient, previousTag, "go.mod")
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read old go.mod", err)
	}

	newContent, err := g.getFileContent(ctx, vcsClient, currentTag, "go.mod")
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read new go.mod", err)
	}

	oldDeps := g.parseGoMod(oldContent)
	newDeps := g.parseGoMod(newContent)

	return g.computeChanges(oldDeps, newDeps), nil
}

func (g *GoModAnalyzer) getFileContent(ctx context.Context, vcsClient vcs.VCSClient, tag, filepath string) (string, error) {
	if vcsClient == nil {
		return "", domainErrors.NewAppError(domainErrors.TypeInternal, "vcsClient is nil", nil)
	}
	return vcsClient.GetFileAtTag(ctx, tag, filepath)
}

type goDep struct {
	version  string
	indirect bool
}

func (g *GoModAnalyzer) parseGoMod(content string) map[string]goDep {
	deps := make(map[string]goDep)

	inRequire := false
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.HasPrefix(trimmedLine, "require (") {
			inRequire = true
			continue
		}

		if trimmedLine == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			matches := regex.GoModRequireBlock.FindStringSubmatch(line)
			if len(matches) >= 3 {
				module := matches[1]
				version := matches[2]
				indirect := len(matches) > 3 && matches[3] != ""

				deps[module] = goDep{
					version:  version,
					indirect: indirect,
				}
			}
		} else if strings.HasPrefix(trimmedLine, "require ") {
			matches := regex.GoModRequireSingle.FindStringSubmatch(trimmedLine)
			if len(matches) >= 3 {
				module := matches[1]
				version := matches[2]
				indirect := len(matches) > 3 && matches[3] != ""

				deps[module] = goDep{
					version:  version,
					indirect: indirect,
				}
			}
		}
	}
	return deps
}

func (g *GoModAnalyzer) computeChanges(oldDeps, newDeps map[string]goDep) []models.DependencyChange {
	var changes []models.DependencyChange

	for module, newDep := range newDeps {
		if oldDep, exists := oldDeps[module]; exists {
			if oldDep.version != newDep.version {
				changes = append(changes, models.DependencyChange{
					Name:       module,
					OldVersion: oldDep.version,
					NewVersion: newDep.version,
					Type:       models.DependencyUpdated,
					Manager:    "go.mod",
					Severity:   calculateSemverSeverity(oldDep.version, newDep.version),
					IsDirect:   !newDep.indirect,
				})
			}
		} else {
			changes = append(changes, models.DependencyChange{
				Name:       module,
				NewVersion: newDep.version,
				Type:       models.DependencyAdded,
				Manager:    "go.mod",
				Severity:   models.UnknownChange,
				IsDirect:   !newDep.indirect,
			})
		}
	}

	for module, oldDep := range oldDeps {
		if _, exists := newDeps[module]; !exists {
			changes = append(changes, models.DependencyChange{
				Name:       module,
				OldVersion: oldDep.version,
				Type:       models.DependencyRemoved,
				Manager:    "go.mod",
				Severity:   models.UnknownChange,
				IsDirect:   !oldDep.indirect,
			})
		}
	}
	return changes
}

