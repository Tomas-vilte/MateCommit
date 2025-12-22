package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/dependency"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/ports"
	"github.com/thomas-vilte/matecommit/internal/regex"
	"golang.org/x/mod/semver"
)

// releaseGitService defines only the methods needed by ReleaseService.
type releaseGitService interface {
	GetLastTag(ctx context.Context) (string, error)
	GetCommitCount(ctx context.Context) (int, error)
	GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error)
	CreateTag(ctx context.Context, version, message string) error
	PushTag(ctx context.Context, version string) error
	GetTagDate(ctx context.Context, version string) (string, error)
	AddFileToStaging(ctx context.Context, file string) error
	HasStagedChanges(ctx context.Context) bool
	CreateCommit(ctx context.Context, message string) error
	Push(ctx context.Context) error
	GetRepoInfo(ctx context.Context) (string, string, string, error)
	GetCurrentBranch(ctx context.Context) (string, error)
	FetchTags(ctx context.Context) error
	ValidateTagExists(ctx context.Context, tag string) error
}

type ReleaseService struct {
	git         releaseGitService
	vcsClient   ports.VCSClient
	notesGen    ports.ReleaseNotesGenerator
	depAnalyzer *dependency.AnalyzerRegistry
	config      *config.Config
}

type ReleaseOption func(*ReleaseService)

func WithReleaseVCSClient(vcs ports.VCSClient) ReleaseOption {
	return func(s *ReleaseService) {
		s.vcsClient = vcs
	}
}

func WithReleaseNotesGenerator(rng ports.ReleaseNotesGenerator) ReleaseOption {
	return func(s *ReleaseService) {
		s.notesGen = rng
	}
}

func WithReleaseConfig(cfg *config.Config) ReleaseOption {
	return func(s *ReleaseService) {
		s.config = cfg
	}
}

func NewReleaseService(
	gitSvc releaseGitService,
	opts ...ReleaseOption,
) *ReleaseService {
	s := &ReleaseService{
		git:         gitSvc,
		depAnalyzer: dependency.NewAnalyzerRegistry(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ReleaseService) AnalyzeNextRelease(ctx context.Context) (*models.Release, error) {
	log := logger.FromContext(ctx)

	if s.config != nil && s.config.AutoFetchTags {
		if err := s.git.FetchTags(ctx); err != nil {
			log.Warn("failed to fetch tags, continuing with local tags", "error", err)
		}
	}

	lastTag, err := s.git.GetLastTag(ctx)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting last tag", err)
	}

	if lastTag == "" {
		count, _ := s.git.GetCommitCount(ctx)
		if count == 0 {
			return nil, domainErrors.NewAppError(domainErrors.TypeGit, "no commits found in repository", nil)
		}
		lastTag = "v0.0.0"
		log.Info("no previous tag found, using v0.0.0 as baseline")
	} else {
		if !regex.SemVer.MatchString(lastTag) {
			log.Warn("last tag does not match semver format", "tag", lastTag)
			return nil, fmt.Errorf("%w: tag '%s'", domainErrors.ErrInvalidTagFormat, lastTag)
		}
	}

	commits, err := s.git.GetCommitsSinceTag(ctx, lastTag)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeGit, "error getting commits", err)
	}

	if len(commits) == 0 {
		return nil, domainErrors.ErrNoChanges
	}

	validCommits := s.filterValidCommits(commits)
	if len(validCommits) == 0 && len(commits) > 0 {
		log.Warn("no conventional commits found, but commits exist",
			"total_commits", len(commits))
	}

	release := &models.Release{
		PreviousVersion: lastTag,
		AllCommits:      commits,
	}

	s.categorizeCommits(release)

	newVersion, bump := s.calculateVersion(lastTag, release)
	release.Version = newVersion
	release.VersionBump = bump

	if err := s.validateVersionIncrement(lastTag, newVersion); err != nil {
		log.Warn("version increment validation failed", "error", err)
	}

	return release, nil
}

func (s *ReleaseService) GenerateReleaseNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	log := logger.FromContext(ctx)

	log.Info("generating release notes",
		"version", release.Version,
		"previous_version", release.PreviousVersion,
	)

	log.Debug("categorizing commits",
		"total_commits", len(release.AllCommits),
	)

	log.Debug("commits categorized",
		"featues", len(release.Features),
		"fixes", len(release.BugFixes),
		"breaking", len(release.Breaking),
		"other", len(release.Other),
	)
	if s.notesGen == nil {
		return s.generateBasicNotes(release), nil
	}

	return s.notesGen.GenerateNotes(ctx, release)
}

func (s *ReleaseService) PublishRelease(ctx context.Context, release *models.Release, notes *models.ReleaseNotes, draft bool, buildBinaries bool) error {
	log := logger.FromContext(ctx)

	log.Info("publishing release",
		"version", release.Version,
		"draft", draft,
		"build_binaries", buildBinaries)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for release publishing")
		return domainErrors.ErrConfigMissing
	}

	if err := s.vcsClient.CreateRelease(ctx, release, notes, draft, buildBinaries); err != nil {
		log.Error("failed to publish release",
			"error", err,
			"version", release.Version)
		return err
	}

	log.Info("release published successfully",
		"version", release.Version,
		"draft", draft)

	return nil
}

func (s *ReleaseService) CreateTag(ctx context.Context, version, message string) error {
	log := logger.FromContext(ctx)

	log.Info("creating git tag",
		"version", version)

	if err := s.git.CreateTag(ctx, version, message); err != nil {
		log.Error("failed to create git tag",
			"error", err,
			"version", version)
		return err
	}

	log.Info("git tag created successfully",
		"version", version)

	return nil
}

func (s *ReleaseService) PushTag(ctx context.Context, version string) error {
	log := logger.FromContext(ctx)

	log.Info("pushing git tag",
		"version", version)

	if err := s.git.PushTag(ctx, version); err != nil {
		log.Error("failed to push git tag",
			"error", err,
			"version", version)
		return err
	}

	log.Info("git tag pushed successfully",
		"version", version)

	return nil
}

func (s *ReleaseService) GetRelease(ctx context.Context, version string) (*models.VCSRelease, error) {
	if s.vcsClient == nil {
		return nil, domainErrors.ErrVCSNotSupported
	}
	return s.vcsClient.GetRelease(ctx, version)
}

func (s *ReleaseService) UpdateRelease(ctx context.Context, version, body string) error {
	log := logger.FromContext(ctx)

	log.Info("updating release",
		"version", version)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for updating release")
		return domainErrors.ErrConfigMissing
	}

	if err := s.vcsClient.UpdateRelease(ctx, version, body); err != nil {
		log.Error("failed to update release",
			"error", err,
			"version", version)
		return err
	}

	log.Info("release updated successfully",
		"version", version)

	return nil
}

func (s *ReleaseService) EnrichReleaseContext(ctx context.Context, release *models.Release) error {
	log := logger.FromContext(ctx)

	log.Info("enriching release context",
		"version", release.Version,
		"previous_version", release.PreviousVersion)

	if s.vcsClient == nil {
		log.Error("VCS client not configured for enriching release context")
		return domainErrors.ErrConfigMissing
	}

	if issues, err := s.vcsClient.GetClosedIssuesBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.ClosedIssues = issues
		log.Debug("closed issues fetched",
			"count", len(issues))
	}

	if prs, err := s.vcsClient.GetMergedPRsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.MergedPRs = prs
		log.Debug("merged PRs fetched",
			"count", len(prs))
	}

	if contributors, err := s.vcsClient.GetContributorsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.Contributors = contributors
		release.NewContributors = contributors
		log.Debug("contributors fetched",
			"count", len(contributors))
	}

	if stats, err := s.vcsClient.GetFileStatsBetweenTags(ctx, release.PreviousVersion, release.Version); err == nil {
		release.FileStats = *stats
		log.Debug("file stats fetched",
			"files_changed", stats.FilesChanged,
			"insertions", stats.Insertions,
			"deletions", stats.Deletions)
	}

	if deps, err := s.analyzeDependencyChanges(ctx, release); err == nil {
		release.Dependencies = deps
		log.Debug("dependencies analyzed")
	}

	log.Info("release context enriched successfully")

	return nil
}

func (s *ReleaseService) UpdateLocalChangelog(release *models.Release, notes *models.ReleaseNotes) error {
	const changelogFile = "CHANGELOG.md"

	log := logger.FromContext(context.Background())

	log.Debug("updating local changelog",
		"version", release.Version,
		"file", changelogFile)

	newContent := s.buildChangelogFromNotes(context.Background(), release, notes)

	if err := s.prependToChangelog(changelogFile, newContent); err != nil {
		log.Error("failed to update changelog",
			"error", err,
			"file", changelogFile)
		return err
	}

	log.Info("changelog updated successfully",
		"file", changelogFile,
		"version", release.Version)

	return nil
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

// buildChangelogFromNotes formats the changelog using AI-generated highlights
func (s *ReleaseService) buildChangelogFromNotes(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) string {
	var sb strings.Builder

	tagDate, err := s.git.GetTagDate(ctx, release.Version)
	if err != nil {
		tagDate = ""
	}

	owner, repo, provider, _ := s.git.GetRepoInfo(ctx)

	versionHeader := fmt.Sprintf("## [%s]", release.Version)
	if tagDate != "" {
		versionHeader += fmt.Sprintf(" - %s", tagDate)
	}

	if provider == "github" && owner != "" && repo != "" {
		compareURL := ""
		if release.PreviousVersion != "" {
			compareURL = fmt.Sprintf("https://github.com/%s/%s/compare/%s...%s", owner, repo, release.PreviousVersion, release.Version)
		} else {
			compareURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, release.Version)
		}
		versionHeader += fmt.Sprintf("\n\n[%s]: %s", release.Version, compareURL)
	}

	sb.WriteString(versionHeader + "\n\n")

	if notes.Summary != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", notes.Summary))
	}

	if len(notes.Highlights) > 0 {
		sb.WriteString("### ✨ Highlights\n\n")
		for _, highlight := range notes.Highlights {
			sb.WriteString(fmt.Sprintf("- %s\n", highlight))
		}
		sb.WriteString("\n")
	}

	if len(notes.BreakingChanges) > 0 {
		sb.WriteString("### ⚠️ Breaking Changes\n\n")
		for _, bc := range notes.BreakingChanges {
			sb.WriteString(fmt.Sprintf("- %s\n", bc))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildChangelog formats the changelog from raw commits (fallback when AI is not available)
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
			return domainErrors.NewAppError(domainErrors.TypeGit, fmt.Sprintf("failed to add version file to staging: %s", mainGoFile), err)
		}
	}

	if !s.git.HasStagedChanges(ctx) {
		return domainErrors.ErrNoChanges
	}

	message := fmt.Sprintf("chore: update changelog and bump version to %s", version)
	if err := s.git.CreateCommit(ctx, message); err != nil {
		return domainErrors.NewAppError(domainErrors.TypeGit, "failed to commit changelog and version bump", err)
	}
	return nil
}

// PushChanges pushes committed changes to the remote repository
func (s *ReleaseService) PushChanges(ctx context.Context) error {
	return s.git.Push(ctx)
}

func (s *ReleaseService) UpdateAppVersion(ctx context.Context, version string) error {
	log := logger.FromContext(ctx)

	versionFile, versionPattern, err := s.FindVersionFile(ctx)
	if err != nil {
		log.Warn("could not auto-detect version file, using defaults", "error", err)
		versionFile = "cmd/main.go"
		versionPattern = `Version:\s*".*"`

		if s.config != nil {
			if s.config.VersionFile != "" {
				versionFile = s.config.VersionFile
			}
			if s.config.VersionPattern != "" {
				versionPattern = s.config.VersionPattern
			}
		}
	}

	log.Debug("updating version",
		"file", versionFile,
		"pattern", versionPattern,
		"new_version", version)

	content, err := os.ReadFile(versionFile)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("failed to read version file: %s", versionFile), err)
	}

	re, err := regexp.Compile(versionPattern)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("invalid version pattern: %s", versionPattern), err)
	}

	currentContent := string(content)
	if !re.MatchString(currentContent) {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("version pattern not found in %s with pattern %s", versionFile, versionPattern), nil)
	}

	match := re.FindString(currentContent)
	if match == "" {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			"could not find version match", nil)
	}

	cleanVersion := strings.TrimPrefix(version, "v")

	var newMatch string
	ext := filepath.Ext(versionFile)

	switch ext {
	case ".json":
		versionRe := regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"version": "%s"`, cleanVersion))
	case ".toml":
		versionRe := regexp.MustCompile(`version\s*=\s*"([^"]+)"`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`version = "%s"`, cleanVersion))
	case ".xml":
		versionRe := regexp.MustCompile(`<version>([^<]+)</version>`)
		newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`<version>%s</version>`, cleanVersion))
	case ".py":
		if strings.Contains(match, `"`) {
			versionRe := regexp.MustCompile(`"([^"]+)"`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
		} else if strings.Contains(match, `'`) {
			versionRe := regexp.MustCompile(`'([^']+)'`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`'%s'`, cleanVersion))
		}
	case ".go", ".js", ".ts", ".rs", ".php", ".rb":
		if strings.Contains(match, `"`) {
			valMatch := regex.QuotedString.FindStringIndex(match)
			if valMatch != nil {
				newMatch = match[:valMatch[0]] + fmt.Sprintf(`"%s"`, cleanVersion) + match[valMatch[1]:]
			} else {
				versionRe := regexp.MustCompile(`"([^"]+)"`)
				newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
			}
		} else if strings.Contains(match, `'`) {
			versionRe := regexp.MustCompile(`'([^']+)'`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`'%s'`, cleanVersion))
		} else {
			versionRe := regexp.MustCompile(`[\d.]+`)
			newMatch = versionRe.ReplaceAllString(match, cleanVersion)
		}
	default:
		if strings.Contains(match, `"`) {
			versionRe := regexp.MustCompile(`"([^"]+)"`)
			newMatch = versionRe.ReplaceAllString(match, fmt.Sprintf(`"%s"`, cleanVersion))
		} else {
			versionRe := regexp.MustCompile(`[\d.]+`)
			newMatch = versionRe.ReplaceAllString(match, cleanVersion)
		}
	}

	if newMatch == "" {
		newMatch = match
	}

	newContent := strings.Replace(currentContent, match, newMatch, 1)

	if err := os.WriteFile(versionFile, []byte(newContent), 0644); err != nil {
		return domainErrors.NewAppError(domainErrors.TypeInternal,
			fmt.Sprintf("failed to write version file: %s", versionFile), err)
	}

	log.Info("version updated successfully",
		"file", versionFile,
		"version", version)

	return nil
}

var commonVersionPatterns = []struct {
	name    string
	pattern string
}{
	{"const Version", `const\s+Version\s*=\s*"([^"]+)"`},
	{"var Version", `var\s+Version\s*=\s*"([^"]+)"`},
	{"const Version", `const\s+Version\s*=\s*([\d.]+)`},
	{"var Version", `var\s+Version\s*=\s*([\d.]+)`},
	{"Version:", `Version:\s*"([^"]+)"`},
	{"Version =", `Version\s*=\s*"([^"]+)"`},
	{"CurrentVersion", `CurrentVersion\s*=\s*"([^"]+)"`},
	{"AppVersion", `AppVersion\s*=\s*"([^"]+)"`},
	{"VERSION", `VERSION\s*=\s*"([^"]+)"`},
}

func (s *ReleaseService) FindVersionFile(ctx context.Context) (string, string, error) {
	log := logger.FromContext(ctx)

	if s.config != nil && s.config.VersionFile != "" {
		pattern := s.config.VersionPattern
		if pattern == "" {
			detectedPattern, err := s.detectPatternInFile(s.config.VersionFile)
			if err == nil && detectedPattern != "" {
				pattern = detectedPattern
			} else {
				pattern = `Version:\s*".*"`
			}
		}
		log.Debug("using configured version file",
			"file", s.config.VersionFile,
			"pattern", pattern)
		return s.config.VersionFile, pattern, nil
	}

	projectType := s.detectProjectType()
	log.Debug("detected project type", "type", projectType)

	if files, ok := versionFilesByLanguage[projectType]; ok {
		for _, filePath := range files {
			if strings.Contains(filePath, "*") {
				matches, err := filepath.Glob(filePath)
				if err == nil && len(matches) > 0 {
					filePath = matches[0]
				} else {
					continue
				}
			}

			if _, err := os.Stat(filePath); err == nil {
				pattern, err := s.detectPatternInFileForLanguage(filePath, projectType)
				if err == nil && pattern != "" {
					log.Debug("version file found automatically",
						"file", filePath,
						"pattern", pattern,
						"language", projectType)
					return filePath, pattern, nil
				}
			}
		}
	}

	foundFile, foundPattern, err := s.searchVersionFileRecursive(projectType)
	if err == nil && foundFile != "" {
		log.Debug("version file found recursively",
			"file", foundFile,
			"pattern", foundPattern)
		return foundFile, foundPattern, nil
	}

	return "", "", fmt.Errorf("could not find version file for project type: %s", projectType)
}

func (s *ReleaseService) ValidateMainBranch(ctx context.Context) error {
	log := logger.FromContext(ctx)

	branch, err := s.git.GetCurrentBranch(ctx)
	if err != nil {
		return domainErrors.NewAppError(domainErrors.TypeGit, "error getting current branch", err)
	}

	if branch != "main" && branch != "master" {
		return fmt.Errorf("%w: currently on '%s'", domainErrors.ErrInvalidBranch, branch)
	}

	log.Debug("branch validation passed",
		"branch", branch,
	)
	return nil
}

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

func (s *ReleaseService) detectPatternInFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	fileContent := string(content)

	lang := detectLanguageFromFile(filePath)

	for _, patternInfo := range commonVersionPatterns {
		re, err := regexp.Compile(patternInfo.pattern)
		if err != nil {
			continue
		}
		if re.MatchString(fileContent) {
			adjustedPattern := s.adjustPatternForReplacement(patternInfo.pattern, fileContent, lang)
			return adjustedPattern, nil
		}
	}
	return "", fmt.Errorf("no version pattern found in file")
}

func (s *ReleaseService) adjustPatternForReplacement(basePattern, content, lang string) string {
	patterns := map[string][]string{
		"go": {
			`const\s+Version\s*=\s*"[^"]*"`,
			`var\s+Version\s*=\s*"[^"]*"`,
			`Version:\s*"[^"]*"`,
		},
		"python": {
			`__version__\s*=\s*"[^"]*"`,
			`__version__\s*=\s*'[^']*'`,
			`version\s*=\s*"[^"]*"`,
		},
		"js": {
			`"version"\s*:\s*"[^"]*"`,
			`'version'\s*:\s*'[^']*'`,
		},
		"rust": {
			`version\s*=\s*"[^"]*"`,
		},
		"java": {
			`<version>[^<]+</version>`,
		},
		"csharp": {
			`AssemblyVersion\s*\(\s*"[^"]*"`,
			`<Version>[^<]+</Version>`,
		},
		"php": {
			`"version"\s*:\s*"[^"]*"`,
		},
		"ruby": {
			`VERSION\s*=\s*['"][^'"]*['"]`,
		},
	}

	if langPatterns, ok := patterns[lang]; ok {
		for _, pattern := range langPatterns {
			re := regexp.MustCompile(pattern)
			if re.MatchString(content) {
				return pattern
			}
		}
	}

	return basePattern
}

func (s *ReleaseService) searchVersionFileRecursive(lang string) (string, string, error) {
	searchDirs := map[string][]string{
		"go":     {"internal", "pkg", "version", "cmd"},
		"python": {"src", "lib", "."},
		"js":     {"src", "lib", "."},
		"rust":   {"."},
		"java":   {"src", "."},
		"csharp": {"Properties", "."},
		"php":    {"src", "."},
		"ruby":   {"lib", "."},
	}

	dirs, ok := searchDirs[lang]
	if !ok {
		dirs = []string{"."}
	}

	extensions := map[string][]string{
		"go":     {".go"},
		"python": {".py"},
		"js":     {".js", ".ts"},
		"rust":   {".rs", ".toml"},
		"java":   {".xml", ".gradle", ".properties"},
		"csharp": {".cs", ".csproj", ".props"},
		"php":    {".php", ".json"},
		"ruby":   {".rb", ".gemspec"},
	}

	exts, ok := extensions[lang]
	if !ok {
		exts = []string{""}
	}

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			hasValidExt := false
			for _, ext := range exts {
				if ext == "" || strings.HasSuffix(path, ext) {
					hasValidExt = true
					break
				}
			}
			if !hasValidExt {
				return nil
			}

			if strings.Contains(strings.ToLower(path), "version") ||
				strings.Contains(strings.ToLower(info.Name()), "version") ||
				path == "package.json" || path == "Cargo.toml" || path == "setup.py" {
				pattern, err := s.detectPatternInFileForLanguage(path, lang)
				if err == nil && pattern != "" {
					return fmt.Errorf("found: %s", path)
				}
			}
			return nil
		})

		if err != nil && strings.HasPrefix(err.Error(), "found: ") {
			foundPath := strings.TrimPrefix(err.Error(), "found: ")
			pattern, _ := s.detectPatternInFileForLanguage(foundPath, lang)
			return foundPath, pattern, nil
		}
	}

	return "", "", fmt.Errorf("version file not found")
}

func (s *ReleaseService) detectProjectType() string {
	indicators := map[string][]string{
		"go":     {"go.mod", "Gopkg.toml", "glide.yaml"},
		"python": {"setup.py", "pyproject.toml", "requirements.txt", "Pipfile"},
		"js":     {"package.json", "yarn.lock", "package-lock.json"},
		"rust":   {"Cargo.toml"},
		"java":   {"pom.xml", "build.gradle", "build.gradle.kts"},
		"csharp": {".csproj", ".sln", "project.json"},
		"php":    {"composer.json"},
		"ruby":   {"Gemfile", "Rakefile"},
	}

	for lang, files := range indicators {
		for _, file := range files {
			if _, err := os.Stat(file); err == nil {
				return lang
			}
		}
	}

	if hasFilesWithExtension(".go") {
		return "go"
	}
	if hasFilesWithExtension(".py") {
		return "python"
	}
	if hasFilesWithExtension(".js") || hasFilesWithExtension(".ts") {
		return "js"
	}
	if hasFilesWithExtension(".rs") {
		return "rust"
	}

	return "unknown"
}

func hasFilesWithExtension(ext string) bool {
	entries, err := os.ReadDir(".")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ext) {
			return true
		}
	}
	return false
}

var versionFilesByLanguage = map[string][]string{
	"go": {
		"internal/version/version.go",
		"pkg/version/version.go",
		"version/version.go",
		"cmd/main.go",
		"main.go",
		"internal/version.go",
		"pkg/version.go",
	},
	"python": {
		"__version__.py",
		"version.py",
		"src/*/__version__.py",
		"setup.py",
		"pyproject.toml",
	},
	"js": {
		"package.json",
		"src/version.js",
		"src/version.ts",
		"lib/version.js",
	},
	"rust": {
		"Cargo.toml",
	},
	"java": {
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
		"src/main/resources/version.properties",
	},
	"csharp": {
		"Properties/AssemblyInfo.cs",
		"Directory.Build.props",
		"*.csproj",
	},
	"php": {
		"composer.json",
		"src/Version.php",
	},
	"ruby": {
		"lib/*/version.rb",
		"version.rb",
		"*.gemspec",
	},
}

var versionPatternsByLanguage = map[string][]struct {
	name    string
	pattern string
}{
	"go": {
		{"const Version", `const\s+Version\s*=\s*"([^"]+)"`},
		{"var Version", `var\s+Version\s*=\s*"([^"]+)"`},
		{"const Version", `const\s+Version\s*=\s*([\d.]+)`},
		{"var Version", `var\s+Version\s*=\s*([\d.]+)`},
		{"Version:", `Version:\s*"([^"]+)"`},
		{"Version =", `Version\s*=\s*"([^"]+)"`},
	},
	"python": {
		{"__version__", `__version__\s*=\s*"([^"]+)"`},
		{"__version__", `__version__\s*=\s*['"]([^'"]+)['"]`},
		{"version", `version\s*=\s*"([^"]+)"`},
		{"version", `version\s*=\s*['"]([^'"]+)['"]`},
		{"VERSION", `VERSION\s*=\s*"([^"]+)"`},
	},
	"js": {
		{"version", `"version"\s*:\s*"([^"]+)"`},
		{"version", `'version'\s*:\s*'([^']+)'`},
		{"export const version", `export\s+const\s+version\s*=\s*"([^"]+)"`},
		{"export const version", `export\s+const\s+version\s*=\s*'([^']+)'`},
	},
	"rust": {
		{"version", `version\s*=\s*"([^"]+)"`},
	},
	"java": {
		{"version", `<version>([^<]+)</version>`},
		{"version", `version\s*=\s*['"]([^'"]+)['"]`},
	},
	"csharp": {
		{"AssemblyVersion", `AssemblyVersion\s*\(\s*"([^"]+)"`},
		{"Version", `<Version>([^<]+)</Version>`},
		{"version", `"version"\s*:\s*"([^"]+)"`},
	},
	"php": {
		{"version", `"version"\s*:\s*"([^"]+)"`},
		{"const VERSION", `const\s+VERSION\s*=\s*['"]([^'"]+)['"]`},
	},
	"ruby": {
		{"VERSION", `VERSION\s*=\s*['"]([^'"]+)['"]`},
		{"version", `\.version\s*=\s*['"]([^'"]+)['"]`},
	},
}

func (s *ReleaseService) detectPatternInFileForLanguage(filePath, lang string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	fileContent := string(content)
	patterns, ok := versionPatternsByLanguage[lang]
	if !ok {
		return s.detectPatternInFile(filePath)
	}

	for _, patternInfo := range patterns {
		re, err := regexp.Compile(patternInfo.pattern)
		if err != nil {
			continue
		}
		if re.MatchString(fileContent) {
			adjustedPattern := s.adjustPatternForReplacement(patternInfo.pattern, fileContent, lang)
			return adjustedPattern, nil
		}
	}

	return "", fmt.Errorf("no version pattern found in file")
}

func detectLanguageFromFile(filePath string) string {
	ext := filepath.Ext(filePath)
	extToLang := map[string]string{
		".go":      "go",
		".py":      "python",
		".js":      "js",
		".ts":      "js",
		".rs":      "rust",
		".toml":    "rust",
		".xml":     "java",
		".cs":      "csharp",
		".csproj":  "csharp",
		".props":   "csharp",
		".php":     "php",
		".rb":      "ruby",
		".gemspec": "ruby",
		".json":    "js",
	}

	filename := strings.ToLower(filepath.Base(filePath))
	if filename == "package.json" || filename == "package-lock.json" {
		return "js"
	}
	if filename == "composer.json" {
		return "php"
	}
	if filename == "cargo.toml" {
		return "rust"
	}
	if filename == "pom.xml" {
		return "java"
	}
	if filename == "setup.py" || filename == "pyproject.toml" {
		return "python"
	}

	if lang, ok := extToLang[ext]; ok {
		return lang
	}

	return "unknown"
}
