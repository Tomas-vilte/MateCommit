package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"golang.org/x/mod/semver"
)

func (s *ReleaseService) analyzeDependencyChanges(ctx context.Context, release *models.Release) ([]models.DependencyChange, error) {
	if s.vcsClient == nil {
		return []models.DependencyChange{}, nil
	}
	return s.depAnalyzer.AnalyzeAll(ctx, s.vcsClient, release.PreviousVersion, release.Version)
}

// categorizeCommits categorizes commits according to conventional commits
func (s *ReleaseService) categorizeCommits(release *models.Release) {
	for _, commit := range release.AllCommits {
		msg := commit.Message
		lines := strings.Split(msg, "\n")
		firstLine := lines[0]

		prMatch := regex.GitHubPR.FindStringSubmatch(firstLine)
		prNumber := ""
		if len(prMatch) > 1 {
			prNumber = prMatch[1]
		}

		hasBreaking := false
		for _, line := range lines[1:] {
			if regex.BreakingChange.MatchString(line) {
				hasBreaking = true
				break
			}
		}

		matches := regex.ConventionalCommit.FindStringSubmatch(firstLine)
		if len(matches) > 0 {
			commitType := matches[1]
			scope := matches[3]
			breaking := matches[4] == "!" || hasBreaking
			description := strings.TrimSpace(matches[5])

			item := models.ReleaseItem{
				Type:        commitType,
				Scope:       scope,
				Description: description,
				Breaking:    breaking,
				PRNumber:    prNumber,
			}

			switch commitType {
			case "feat":
				if breaking {
					release.Breaking = append(release.Breaking, item)
				} else {
					release.Features = append(release.Features, item)
				}
			case "fix":
				release.BugFixes = append(release.BugFixes, item)
			case "docs":
				release.Documentation = append(release.Documentation, item)
			case "perf", "refactor":
				release.Improvements = append(release.Improvements, item)
			default:
				release.Other = append(release.Other, item)
			}
		} else {
			item := models.ReleaseItem{
				Type:        "other",
				Description: firstLine,
				PRNumber:    prNumber,
			}
			release.Other = append(release.Other, item)
		}
	}
}

// calculateVersion calculates the new version based on semantic versioning
func (s *ReleaseService) calculateVersion(currentTag string, release *models.Release) (string, models.VersionBump) {
	matches := regex.SemVer.FindStringSubmatch(currentTag)

	major, minor, patch := 0, 0, 0
	if len(matches) >= 4 {
		major, _ = strconv.Atoi(matches[1])
		minor, _ = strconv.Atoi(matches[2])
		patch, _ = strconv.Atoi(matches[3])
	}

	bump := models.PatchBump

	if len(release.Breaking) > 0 {
		major++
		minor = 0
		patch = 0
		bump = models.MajorBump
	} else if len(release.Features) > 0 {
		// MINOR: new features
		minor++
		patch = 0
		bump = models.MinorBump
	} else if len(release.BugFixes) > 0 || len(release.Improvements) > 0 {
		// PATCH: bug fixes or improvements
		patch++
		bump = models.PatchBump
	}

	newVersion := fmt.Sprintf("v%d.%d.%d", major, minor, patch)
	return newVersion, bump
}

func (s *ReleaseService) generateBasicNotes(release *models.Release) *models.ReleaseNotes {
	title := fmt.Sprintf("Version %s", release.Version)

	summary := fmt.Sprintf("This release includes %d new features, %d bug fixes",
		len(release.Features), len(release.BugFixes))

	if len(release.Breaking) > 0 {
		summary += fmt.Sprintf(" and %d breaking changes", len(release.Breaking))
	}

	changelog := s.buildChangelog(release)

	return &models.ReleaseNotes{
		Title:       title,
		Summary:     summary,
		Changelog:   changelog,
		Recommended: release.VersionBump,
	}
}

// buildChangelogFromNotes formats the changelog using AI-generated highlights and semantic sections
func (s *ReleaseService) filterValidCommits(commits []models.Commit) []models.Commit {
	var valid []models.Commit
	for _, commit := range commits {
		if regex.ConventionalCommit.MatchString(commit.Message) {
			valid = append(valid, commit)
		}
	}
	return valid
}

func (s *ReleaseService) validateVersionIncrement(oldVersion, newVersion string) error {
	oldClean := strings.TrimPrefix(oldVersion, "v")
	newClean := strings.TrimPrefix(newVersion, "v")

	if semver.Compare("v"+oldClean, "v"+newClean) >= 0 {
		return fmt.Errorf("new version %s must be greater than previous version %s", newVersion, oldVersion)
	}
	return nil
}

func (s *ReleaseService) validateVersionString(versionStr string) error {
	cleanVersion := strings.TrimPrefix(versionStr, "v")

	if !semver.IsValid("v" + cleanVersion) {
		return fmt.Errorf("invalid semver format: %s", versionStr)
	}
	return nil
}
