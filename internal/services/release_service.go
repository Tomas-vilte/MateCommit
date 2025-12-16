package services

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/dependency"
)

var (
	conventionalRegex = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(([^)]+)\))?(!)?:\s*(.+)`)
	breakingRegex     = regexp.MustCompile(`BREAKING[ -]CHANGE:\s*(.+)`)
	versionRegex      = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)
)

var _ ports.ReleaseService = (*ReleaseService)(nil)

type ReleaseService struct {
	git         ports.GitService
	vcsClient   ports.VCSClient
	notesGen    ports.ReleaseNotesGenerator
	trans       *i18n.Translations
	depAnalyzer *dependency.AnalyzerRegistry
	config      *config.Config
}

func NewReleaseService(
	git ports.GitService,
	vcsClient ports.VCSClient,
	notesGen ports.ReleaseNotesGenerator,
	trans *i18n.Translations,
	config *config.Config,
) *ReleaseService {
	return &ReleaseService{
		git:         git,
		vcsClient:   vcsClient,
		notesGen:    notesGen,
		trans:       trans,
		depAnalyzer: dependency.NewAnalyzerRegistry(),
		config:      config,
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
	if s.vcsClient == nil {
		return fmt.Errorf("cliente VCS no configurado. Configura un proveedor VCS con 'matecommit config set-vcsClient'")
	}
	return s.vcsClient.CreateRelease(ctx, release, notes, draft)
}

func (s *ReleaseService) CreateTag(ctx context.Context, version, message string) error {
	return s.git.CreateTag(ctx, version, message)
}

func (s *ReleaseService) PushTag(ctx context.Context, version string) error {
	return s.git.PushTag(ctx, version)
}

func (s *ReleaseService) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	if s.vcsClient == nil {
		return nil, fmt.Errorf("%s", s.trans.GetMessage("error.vcs_provider_not_configured", 0, map[string]interface{}{
			"Provider": "VCS",
		}))
	}
	return s.vcsClient.GetRelease(ctx, version)
}

func (s *ReleaseService) UpdateRelease(ctx context.Context, version, body string) error {
	if s.vcsClient == nil {
		return fmt.Errorf("%s", s.trans.GetMessage("error.vcs_provider_not_configured", 0, map[string]interface{}{
			"Provider": "VCS",
		}))
	}
	return s.vcsClient.UpdateRelease(ctx, version, body)
}

func (s *ReleaseService) EnrichReleaseContext(ctx context.Context, release *models.Release) error {
	if s.vcsClient == nil {
		return fmt.Errorf("%s", s.trans.GetMessage("error.vcs_provider_not_configured", 0, map[string]interface{}{
			"Provider": "VCS",
		}))
	}

	if issues, err := s.vcsClient.GetClosedIssuesBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.ClosedIssues = issues
	}

	if prs, err := s.vcsClient.GetMergedPRsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.MergedPRs = prs
	}

	if contributors, err := s.vcsClient.GetContributorsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.Contributors = contributors
		release.NewContributors = contributors
	}

	if stats, err := s.vcsClient.GetFileStatsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.FileStats = *stats
	}

	if deps, err := s.analyzeDependencyChanges(ctx, release); err == nil {
		release.Dependencies = deps
	}

	return nil
}

func (s *ReleaseService) UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error {
	const changelogFile = "CHANGELOG.md"

	newContent := notes.Changelog
	if newContent == "" {
		newContent = s.buildChangelog(release)
	}

	versionHeader := fmt.Sprintf("## %s (%s)", release.Version, time.Now().Format("2006-01-02"))

	if !strings.HasPrefix(strings.TrimSpace(newContent), "## "+release.Version) {
		newContent = fmt.Sprintf("%s\n\n%s\n\n", versionHeader, strings.TrimSpace(newContent))
	} else {
		newContent = newContent + "\n\n"
	}

	fmt.Println(s.trans.GetMessage("release.changelog_update_started", 0, nil))

	return s.prependToChangelog(changelogFile, newContent)
}

func (s *ReleaseService) prependToChangelog(filename, newContent string) error {
	content, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		header := "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n"
		return os.WriteFile(filename, []byte(header+newContent), 0644)
	}
	if err != nil {
		return err
	}

	current := string(content)
	var sb strings.Builder

	idx := strings.Index(current, "\n## ")

	if idx != -1 {
		pre := current[:idx]
		post := current[idx:]

		sb.WriteString(strings.TrimSpace(pre))
		sb.WriteString("\n\n")
		sb.WriteString(strings.TrimSpace(newContent))
		sb.WriteString("\n")
		sb.WriteString(post)
	} else {
		if strings.HasPrefix(current, "# ") {
			sb.WriteString(strings.TrimSpace(current))
			sb.WriteString("\n\n")
			sb.WriteString(strings.TrimSpace(newContent))
			sb.WriteString("\n")
		} else {
			sb.WriteString(strings.TrimSpace(newContent))
			sb.WriteString("\n\n")
			sb.WriteString(strings.TrimSpace(current))
		}
	}

	return os.WriteFile(filename, []byte(sb.String()), 0644)
}

func (s *ReleaseService) analyzeDependencyChanges(ctx context.Context, release *models.Release) ([]models.DependencyChange, error) {
	if s.vcsClient == nil {
		return []models.DependencyChange{}, nil
	}
	return s.depAnalyzer.AnalyzeAll(ctx, s.vcsClient, release.PreviousVersion, release.Version)
}

// categorizeCommits categoriza los commits según conventional commits
func (s *ReleaseService) categorizeCommits(release *models.Release) {
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

func (s *ReleaseService) CommitChangelog(ctx context.Context, version string) error {
	mainGoFile := "cmd/main.go"
	if s.config != nil && s.config.VersionFile != "" {
		mainGoFile = s.config.VersionFile
	}

	if _, err := os.Stat(mainGoFile); err == nil {
		if err := s.git.AddFileToStaging(ctx, mainGoFile); err != nil {
			return fmt.Errorf("error haciendo git add de archivo de versión (%s): %w", mainGoFile, err)
		}
	}

	if !s.git.HasStagedChanges(ctx) {
		fmt.Println(s.trans.GetMessage("release.commit_no_staged", 0, nil))
		return nil
	}

	message := fmt.Sprintf("chore: update changelog and bump version to %s", version)
	if err := s.git.CreateCommit(ctx, message); err != nil {
		return fmt.Errorf("%s", s.trans.GetMessage("release.error_committing_changelog", 0, map[string]interface{}{"Error": err}))
	}
	return nil
}

func (s *ReleaseService) UpdateAppVersion(version string) error {
	mainGoFile := "cmd/main.go"
	versionPattern := `Version:\s*".*"`

	if s.config != nil {
		if s.config.VersionFile != "" {
			mainGoFile = s.config.VersionFile
		}
		if s.config.VersionPattern != "" {
			versionPattern = s.config.VersionPattern
		}
	}

	content, err := os.ReadFile(mainGoFile)
	if err != nil {
		return fmt.Errorf("error leyendo archivo de versión (%s): %w", mainGoFile, err)
	}

	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return fmt.Errorf("patrón de versión inválido (%s): %w", versionPattern, err)
	}

	currentContent := string(content)
	if !re.MatchString(currentContent) {
		return fmt.Errorf("no se encontró el patrón de versión en %s", mainGoFile)
	}

	match := re.FindString(currentContent)

	valueRe := regexp.MustCompile(`"(.*)"`)
	valMatch := valueRe.FindStringIndex(match)

	if valMatch == nil {
		return fmt.Errorf("no se encontró una cadena entre comillas en el patrón coincidente")
	}

	cleanVersion := strings.TrimPrefix(version, "v")
	newMatch := match[:valMatch[0]] + fmt.Sprintf(`"%s"`, cleanVersion) + match[valMatch[1]:]

	newContent := strings.Replace(currentContent, match, newMatch, 1)

	if err := os.WriteFile(mainGoFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("error escribiendo archivo de versión: %w", err)
	}

	return nil
}
