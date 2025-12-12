package dependency

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

var _ ports.DependencyAnalyzer = (*PackageJsonAnalyzer)(nil)

type PackageJsonAnalyzer struct{}

func NewPackageJsonAnalyzer() *PackageJsonAnalyzer {
	return &PackageJsonAnalyzer{}
}

func (p *PackageJsonAnalyzer) CanHandle(ctx context.Context, vcsClient ports.VCSClient, _, currentTag string) bool {
	content, err := p.getFileContent(ctx, vcsClient, currentTag, "package.json")
	return err == nil && content != ""
}

func (p *PackageJsonAnalyzer) AnalyzeChanges(ctx context.Context, vcsClient ports.VCSClient, previousTag, currentTag string) ([]models.DependencyChange, error) {
	oldContent, err := p.getFileContent(ctx, vcsClient, previousTag, "package.json")
	if err != nil {
		return nil, fmt.Errorf("error al leer el package.json antiguo: %w", err)
	}

	newContent, err := p.getFileContent(ctx, vcsClient, currentTag, "package.json")
	if err != nil {
		return nil, fmt.Errorf("error al leer el nuevo package.json: %w", err)
	}

	oldDeps, err := p.parsePackageJson(oldContent)
	if err != nil {
		return nil, fmt.Errorf("error al parsear viejo package.json: %w", err)
	}

	newDeps, err := p.parsePackageJson(newContent)
	if err != nil {
		return nil, fmt.Errorf("error al parsear nuevo package.json: %w", err)
	}

	return p.computeChanges(oldDeps, newDeps), nil

}

func (p *PackageJsonAnalyzer) Name() string {
	return "package.json"
}

func (p *PackageJsonAnalyzer) getFileContent(ctx context.Context, vcsClient ports.VCSClient, tag, filepath string) (string, error) {
	if vcsClient == nil {
		return "", fmt.Errorf("vcsClient is nil")
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
					Severity:   p.calculateSeverity(oldDep.version, newDep.version),
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

// cleanVersion remueve prefijos como ^, ~, >=, etc de versiones npm
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

// calculateSeverity determina la severidad del cambio basado en semver
func (p *PackageJsonAnalyzer) calculateSeverity(oldVersion, newVersion string) models.ChangeSeverity {
	oldClean := p.cleanVersion(oldVersion)
	newClean := p.cleanVersion(newVersion)

	oldParts := p.parseVersion(oldClean)
	newParts := p.parseVersion(newClean)

	if len(oldParts) < 3 || len(newParts) < 3 {
		return models.UnknownChange
	}

	if newParts[0] > oldParts[0] {
		return models.MajorChange
	}

	if newParts[1] > oldParts[1] {
		return models.MinorChange
	}

	if newParts[2] > oldParts[2] {
		return models.PatchChange
	}

	return models.UnknownChange
}

// parseVersion extrae [major, minor, patch] de una version string
func (p *PackageJsonAnalyzer) parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, 3)

	for i := 0; i < 3 && i < len(parts); i++ {
		numStr := parts[i]
		if idx := strings.IndexAny(numStr, "-+"); idx != -1 {
			numStr = numStr[:idx]
		}

		num, err := strconv.Atoi(numStr)
		if err != nil {
			return []int{}
		}
		result = append(result, num)
	}

	return result
}
