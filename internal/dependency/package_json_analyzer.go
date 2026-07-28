package dependency

import (
	"context"
	"encoding/json"
	"strings"

	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/vcs"
)

var _ vcs.DependencyAnalyzer = (*PackageJsonAnalyzer)(nil)

type PackageJsonAnalyzer struct{}

func NewPackageJsonAnalyzer() *PackageJsonAnalyzer {
	return &PackageJsonAnalyzer{}
}

func (p *PackageJsonAnalyzer) CanHandle(ctx context.Context, vcsClient vcs.VCSClient, _, currentTag string) bool {
	content, err := p.getFileContent(ctx, vcsClient, currentTag, "package.json")
	return err == nil && content != ""
}

func (p *PackageJsonAnalyzer) AnalyzeChanges(ctx context.Context, vcsClient vcs.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
	oldContent, err := p.getFileContent(ctx, vcsClient, previousTag, "package.json")
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read old package.json", err)
	}

	newContent, err := p.getFileContent(ctx, vcsClient, currentTag, "package.json")
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to read new package.json", err)
	}

	oldDeps, err := p.parsePackageJson(oldContent)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to parse old package.json", err)
	}

	newDeps, err := p.parsePackageJson(newContent)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to parse new package.json", err)
	}

	return p.computeChanges(oldDeps, newDeps), nil
}

func (p *PackageJsonAnalyzer) Name() string {
	return "package.json"
}

func (p *PackageJsonAnalyzer) getFileContent(ctx context.Context, vcsClient vcs.VCSClient, tag, filepath string) (string, error) {
	if vcsClient == nil {
		return "", domainErrors.NewAppError(domainErrors.TypeInternal, "vcsClient is nil", nil)
	}
	return vcsClient.GetFileAtTag(ctx, tag, filepath)
}

type packageJson struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type npmDep struct {
	version string
	isDev   bool
}

func (p *PackageJsonAnalyzer) parsePackageJson(content string) (map[string]npmDep, error) {
	var pkg packageJson
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil, err
	}

	deps := make(map[string]npmDep)

	for name, version := range pkg.Dependencies {
		deps[name] = npmDep{
			version: version,
			isDev:   false,
		}
	}

	for name, version := range pkg.DevDependencies {
		deps[name] = npmDep{
			version: version,
			isDev:   true,
		}
	}

	return deps, nil
}

func (p *PackageJsonAnalyzer) computeChanges(oldDeps, newDeps map[string]npmDep) []models.DependencyChange {
	var changes []models.DependencyChange

	for name, newDep := range newDeps {
		if oldDep, exists := oldDeps[name]; exists {
			if oldDep.version != newDep.version {
				changes = append(changes, models.DependencyChange{
					Name:       name,
					OldVersion: p.cleanVersion(oldDep.version),
					NewVersion: p.cleanVersion(newDep.version),
					Type:       models.DependencyUpdated,
					Manager:    "package.json",
					Severity:   calculateSemverSeverity(p.cleanVersion(oldDep.version), p.cleanVersion(newDep.version)),
					IsDirect:   !newDep.isDev,
				})
			}
		} else {
			changes = append(changes, models.DependencyChange{
				Name:       name,
				NewVersion: p.cleanVersion(newDep.version),
				Type:       models.DependencyAdded,
				Manager:    "package.json",
				Severity:   models.UnknownChange,
				IsDirect:   !newDep.isDev,
			})
		}
	}

	for name, oldDep := range oldDeps {
		if _, exists := newDeps[name]; !exists {
			changes = append(changes, models.DependencyChange{
				Name:       name,
				OldVersion: p.cleanVersion(oldDep.version),
				Type:       models.DependencyRemoved,
				Manager:    "package.json",
				Severity:   models.UnknownChange,
				IsDirect:   !oldDep.isDev,
			})
		}
	}

	return changes
}

// cleanVersion removes prefixes like ^, ~, >=, etc from npm versions
func (p *PackageJsonAnalyzer) cleanVersion(version string) string {
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")
	return strings.TrimSpace(version)
}

