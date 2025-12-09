package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

var _ ports.ReleaseService = (*ReleaseService)(nil)

type ReleaseService struct {
	git      ports.GitService
	vcs      ports.VCSClient
	notesGen ports.ReleaseNotesGenerator
	trans    *i18n.Translations
}

func NewReleaseService(
	git ports.GitService,
	vcs ports.VCSClient,
	notesGen ports.ReleaseNotesGenerator,
	trans *i18n.Translations,
) *ReleaseService {
	return &ReleaseService{
		git:      git,
		vcs:      vcs,
		notesGen: notesGen,
		trans:    trans,
	}
}

func (s *ReleaseService) AnalyzeNextRelease(ctx context.Context) (*models.Release, error) {
	lastTag, err := s.git.GetLastTag(ctx)
	if err != nil {
		return nil, fmt.Errorf("error al obtener ultimo tag: %w", err)
	}

	if lastTag == "" {
		count, _ := s.git.GetCommitCount(ctx)
		if count == 0 {
			return nil, fmt.Errorf("no hay commits en el repositorio")
		}
		lastTag = "v0.0.0"
	}

	commits, err := s.git.GetCommitsSinceTag(ctx, lastTag)
	if err != nil {
		return nil, fmt.Errorf("error al obtener commits: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("no hay commits nuevos desde %s", lastTag)
	}

	release := &models.Release{
		PreviousVersion: lastTag,
		AllCommits:      commits,
	}

	s.categorizeCommits(release)

	newVersion, bump := s.calculateVersion(lastTag, release)
	release.Version = newVersion
	release.VersionBump = bump

	return release, nil
}

func (s *ReleaseService) GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	if s.notesGen == nil {
		return s.generateBasicNotes(release), nil
	}

	return s.notesGen.GenerateNotes(ctx, release)
}

func (s *ReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool) error {
	if s.vcs == nil {
		return fmt.Errorf("cliente VCS no configurado. Configura un proveedor VCS con 'matecommit config set-vcs'")
	}
	return s.vcs.CreateRelease(ctx, release, notes, draft)
}

func (s *ReleaseService) CreateTag(ctx context.Context, version, message string) error {
	return s.git.CreateTag(ctx, version, message)
}

func (s *ReleaseService) PushTag(ctx context.Context, version string) error {
	return s.git.PushTag(ctx, version)
}

// categorizeCommits categoriza los commits según conventional commits
func (s *ReleaseService) categorizeCommits(release *models.Release) {
	conventionalRegex := regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(([^)]+)\))?(!)?:\s*(.+)`)
	breakingRegex := regexp.MustCompile(`BREAKING[ -]CHANGE:\s*(.+)`)

	for _, commit := range release.AllCommits {
		msg := commit.Message
		lines := strings.Split(msg, "\n")
		firstLine := lines[0]

		prRegex := regexp.MustCompile(`\(#(\d+)\)`)
		prMatch := prRegex.FindStringSubmatch(firstLine)
		prNumber := ""
		if len(prMatch) > 1 {
			prNumber = prMatch[1]
		}

		hasBreaking := false
		for _, line := range lines[1:] {
			if breakingRegex.MatchString(line) {
				hasBreaking = true
				break
			}
		}

		matches := conventionalRegex.FindStringSubmatch(firstLine)
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

// calculateVersion calcula la nueva versión basándose en semantic versioning
func (s *ReleaseService) calculateVersion(currentTag string, release *models.Release) (string, models.VersionBump) {
	versionRegex := regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)
	matches := versionRegex.FindStringSubmatch(currentTag)

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
		// MINOR: nuevas features
		minor++
		patch = 0
		bump = models.MinorBump
	} else if len(release.BugFixes) > 0 || len(release.Improvements) > 0 {
		// PATCH: bug fixes o mejoras
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

// buildChangelog construye el changelog en formato markdown
func (s *ReleaseService) buildChangelog(release *models.Release) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", release.Version))

	if len(release.Breaking) > 0 {
		sb.WriteString("### ⚠️ BREAKING CHANGES\n\n")
		for _, item := range release.Breaking {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Features) > 0 {
		sb.WriteString("### ✨ New Features\n\n")
		for _, item := range release.Features {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.BugFixes) > 0 {
		sb.WriteString("### 🐛 Bug Fixes\n\n")
		for _, item := range release.BugFixes {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Improvements) > 0 {
		sb.WriteString("### 🔧 Improvements\n\n")
		for _, item := range release.Improvements {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Documentation) > 0 {
		sb.WriteString("### 📚 Documentation\n\n")
		for _, item := range release.Documentation {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *ReleaseService) formatReleaseItem(item models.ReleaseItem) string {
	line := "- "

	if item.Scope != "" {
		line += fmt.Sprintf("**%s**: ", item.Scope)
	}

	line += item.Description

	if item.PRNumber != "" {
		line += fmt.Sprintf(" (#%s)", item.PRNumber)
	}

	line += "\n"
	return line
}
