package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/logger"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func (s *ReleaseService) UpdateLocalChangelog(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) error {
	log := logger.FromContext(ctx)

	repoRoot, err := s.git.GetRepoRoot(ctx)
	if err != nil {
		log.Error("failed to get repository root for changelog", "error", err)
		return fmt.Errorf("failed to get repository root: %w", err)
	}
	changelogFile := filepath.Join(repoRoot, "CHANGELOG.md")

	log.Debug("updating local changelog",
		"version", release.Version,
		"file", changelogFile)

	if _, err := os.Stat(changelogFile); err == nil {
		content, readErr := os.ReadFile(changelogFile)
		if readErr == nil && strings.Contains(string(content), "## [Unreleased]") {
			if err := s.MoveUnreleasedToVersion(changelogFile, release, notes); err != nil {
				log.Warn("failed to move Unreleased section", "error", err)
			} else {
				log.Info("moved Unreleased section to new version", "version", release.Version)
				if s.hasUnreleasedContent(content) {
					return nil
				}
				log.Debug("unreleased section was empty, will add new version entry")
			}
		}
	}

	newContent := s.buildChangelogFromNotes(context.Background(), release, notes)

	if err := s.prependToChangelog(changelogFile, newContent); err != nil {
		log.Error("failed to update changelog",
			"error", err,
			"file", changelogFile)
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to update CHANGELOG.md",
			err,
		).WithSuggestion(
			"Make sure CHANGELOG.md is writable and not locked by another process.\n" +
				"Try: chmod +w CHANGELOG.md",
		)
	}

	if err := s.EnsureUnreleasedSection(changelogFile); err != nil {
		log.Warn("failed to ensure Unreleased section", "error", err)
	}

	if warnings, err := s.ValidateChangelog(changelogFile); err == nil && len(warnings) > 0 {
		log.Warn("CHANGELOG validation warnings detected", "count", len(warnings))
		for _, warning := range warnings {
			log.Warn("CHANGELOG warning", "type", warning.Type, "message", warning.Message)
		}
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
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to read CHANGELOG.md",
			err,
		).WithSuggestion(
			"Ensure CHANGELOG.md exists and is readable.\n" +
				"Try: ls -la CHANGELOG.md",
		)
	}

	current := string(content)

	versionRegex := regexp.MustCompile(`## \[([^]]+)]`)
	matches := versionRegex.FindStringSubmatch(newContent)
	if len(matches) < 2 {
		// If we can't extract version, fallback to old behavior
		return s.prependToChangelogLegacy(filename, current, newContent)
	}

	newVersion := matches[1]

	headerPattern := regexp.MustCompile(`(?m)^## \[` + regexp.QuoteMeta(newVersion) + `\].*$`)
	if headerPattern.MatchString(current) {
		log := logger.FromContext(context.Background())
		log.Debug("version already exists in changelog, replacing",
			"version", newVersion)

		loc := headerPattern.FindStringIndex(current)
		if loc != nil {
			nextVersionPattern := regexp.MustCompile(`(?m)^## \[[^]]+]`)
			nextLoc := nextVersionPattern.FindStringIndex(current[loc[1]:])

			end := len(current)
			if nextLoc != nil {
				end = loc[1] + nextLoc[0]
			}

			before := current[:loc[0]]
			after := current[end:]

			before = strings.TrimRight(before, "\n")

			var sb strings.Builder
			sb.WriteString(before)
			sb.WriteString("\n\n")
			sb.WriteString(strings.TrimSpace(newContent))
			sb.WriteString("\n\n")
			sb.WriteString(strings.TrimLeft(after, "\n"))

			result := sb.String()

			result = s.consolidateLinkDefinitions(result)

			return os.WriteFile(filename, []byte(result), 0644)
		}
	}

	return s.prependToChangelogLegacy(filename, current, newContent)
}

// prependToChangelogLegacy is the original prepend logic
func (s *ReleaseService) prependToChangelogLegacy(filename, current, newContent string) error {
	var sb strings.Builder

	versionPattern := regexp.MustCompile(`\n## \[[^]]+]`)
	loc := versionPattern.FindStringIndex(current)

	if loc != nil {
		pre := current[:loc[0]]
		post := current[loc[0]:]

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

	result := sb.String()

	// Remove empty Unreleased sections that might be between versions
	result = s.removeEmptyUnreleasedSections(result)

	result = s.consolidateLinkDefinitions(result)

	return os.WriteFile(filename, []byte(result), 0644)
}

// removeEmptyUnreleasedSections removes empty ## [Unreleased] sections that appear between versions
func (s *ReleaseService) removeEmptyUnreleasedSections(content string) string {
	emptyUnreleasedPattern := regexp.MustCompile(`(?s)## \[Unreleased\]\s*\n(## \[)`)
	return emptyUnreleasedPattern.ReplaceAllString(content, "$1")
}

// consolidateLinkDefinitions removes duplicate link reference definitions
func (s *ReleaseService) consolidateLinkDefinitions(content string) string {
	linkDefPattern := regexp.MustCompile(`(?m)^\[([^]]+)]:\s*(.+)$`)

	lines := strings.Split(content, "\n")
	seen := make(map[string]string)
	var result []string

	for _, line := range lines {
		matches := linkDefPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			version := matches[1]
			url := matches[2]

			if existingURL, exists := seen[version]; exists {
				if existingURL != url {
					log := logger.FromContext(context.Background())
					log.Warn("duplicate link definition with different URL",
						"version", version,
						"existing_url", existingURL,
						"new_url", url)
				}
				continue
			}
			seen[version] = url
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// EnsureUnreleasedSection ensures an Unreleased section exists in the CHANGELOG
func (s *ReleaseService) EnsureUnreleasedSection(filename string) error {
	content, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		header := `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

`
		return os.WriteFile(filename, []byte(header), 0644)
	}
	if err != nil {
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to read CHANGELOG.md for Unreleased section",
			err,
		).WithSuggestion(
			"Check if CHANGELOG.md is readable and not corrupted.\n" +
				"Try: cat CHANGELOG.md",
		)
	}

	current := string(content)

	if strings.Contains(current, "## [Unreleased]") {
		return nil
	}

	versionPattern := regexp.MustCompile(`\n## \[[^]]+]`)
	loc := versionPattern.FindStringIndex(current)

	if loc == nil {
		current = strings.TrimSpace(current) + "\n\n## [Unreleased]\n\n"
	} else {
		pre := current[:loc[0]]
		post := current[loc[0]:]
		current = strings.TrimSpace(pre) + "\n\n## [Unreleased]\n\n" + post
	}

	return os.WriteFile(filename, []byte(current), 0644)
}

// parseUnreleasedSection extracts the Unreleased section content
func (s *ReleaseService) parseUnreleasedSection(content string) string {
	unreleasedPattern := regexp.MustCompile(`(?s)## \[Unreleased]\s*\n(.*?)(?:## \[|$)`)
	matches := unreleasedPattern.FindStringSubmatch(content)

	if len(matches) < 2 {
		return ""
	}

	return strings.TrimSpace(matches[1])
}

// hasUnreleasedContent checks if the Unreleased section has actual content
func (s *ReleaseService) hasUnreleasedContent(content []byte) bool {
	return s.parseUnreleasedSection(string(content)) != ""
}

// MoveUnreleasedToVersion moves Unreleased section content to a new version
func (s *ReleaseService) MoveUnreleasedToVersion(filename string, release *models.Release, notes *models.ReleaseNotes) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to read CHANGELOG.md for Unreleased migration",
			err,
		).WithSuggestion(
			"Ensure CHANGELOG.md exists and is readable.",
		)
	}

	current := string(content)

	unreleasedContent := s.parseUnreleasedSection(current)

	log := logger.FromContext(context.Background())
	log.Debug("parsed unreleased section",
		"unreleased_content", unreleasedContent,
		"unreleased_content_length", len(unreleasedContent),
		"is_empty", unreleasedContent == "")

	if unreleasedContent == "" {
		log.Info("unreleased section is empty, skipping migration")
		return nil
	}

	log.Info("moving Unreleased section to new version",
		"version", release.Version,
		"unreleased_content_length", len(unreleasedContent))

	versionEntry := s.buildChangelogFromNotes(context.Background(), release, notes)

	versionEntry = strings.TrimSpace(versionEntry) + "\n\n" + unreleasedContent + "\n"

	unreleasedPattern := regexp.MustCompile(`(?s)## \[Unreleased]\s*\n.*?(\n## \[|$)`)
	current = unreleasedPattern.ReplaceAllString(current, "$1")

	log.Debug("writing modified changelog without unreleased section",
		"filename", filename,
		"content_length", len(current))

	if err := os.WriteFile(filename, []byte(current), 0644); err != nil {
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to write CHANGELOG.md during Unreleased migration",
			err,
		).WithSuggestion(
			"Check file permissions and disk space.\n" +
				"Try: df -h . && chmod +w CHANGELOG.md",
		)
	}

	log.Debug("changelog written successfully, prepending new version")

	if err := s.prependToChangelog(filename, versionEntry); err != nil {
		return domainErrors.NewAppError(
			domainErrors.TypeInternal,
			"Failed to prepend new version during Unreleased migration",
			err,
		).WithSuggestion(
			"The Unreleased section was removed but the new version couldn't be added.\n" +
				"You may need to manually restore CHANGELOG.md from git.",
		)
	}

	return s.EnsureUnreleasedSection(filename)
}

// ChangelogWarning represents a validation warning
type ChangelogWarning struct {
	Type    string
	Message string
}

// validateChangelogEntry validates a CHANGELOG entry and returns warnings
func (s *ReleaseService) validateChangelogEntry(content string, version string) []ChangelogWarning {
	var warnings []ChangelogWarning

	datePattern := regexp.MustCompile(`## \[` + regexp.QuoteMeta(version) + `\] - \d{4}-\d{2}-\d{2}`)
	if !datePattern.MatchString(content) {
		warnings = append(warnings, ChangelogWarning{
			Type:    "missing_date",
			Message: fmt.Sprintf("Version %s is missing a date or date is not in ISO 8601 format (YYYY-MM-DD)", version),
		})
	}

	linkPattern := regexp.MustCompile(`\[` + regexp.QuoteMeta(version) + `\]:\s*https?://`)
	if !linkPattern.MatchString(content) {
		warnings = append(warnings, ChangelogWarning{
			Type:    "missing_link",
			Message: fmt.Sprintf("Version %s is missing a comparison link", version),
		})
	}

	if !strings.Contains(content, "###") {
		warnings = append(warnings, ChangelogWarning{
			Type:    "no_sections",
			Message: fmt.Sprintf("Version %s has no sections (###). Consider organizing changes into sections.", version),
		})
	}

	versionHeaderPattern := regexp.MustCompile(`(?s)## \[` + regexp.QuoteMeta(version) + `\].*?\n\n(.*?)(?:## \[|$)`)
	matches := versionHeaderPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		actualContent := strings.TrimSpace(matches[1])
		actualContent = regexp.MustCompile(`(?m)^\[.*?]:.*$`).ReplaceAllString(actualContent, "")
		actualContent = strings.TrimSpace(actualContent)

		if len(actualContent) < 50 {
			warnings = append(warnings, ChangelogWarning{
				Type:    "short_content",
				Message: fmt.Sprintf("Version %s has very little content. Consider adding more details.", version),
			})
		}
	}

	return warnings
}

// ValidateChangelog validates the entire CHANGELOG file
func (s *ReleaseService) ValidateChangelog(filename string) ([]ChangelogWarning, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var allWarnings []ChangelogWarning
	current := string(content)

	versionPattern := regexp.MustCompile(`## \[([^]]+)]`)
	matches := versionPattern.FindAllStringSubmatch(current, -1)

	for _, match := range matches {
		if len(match) > 1 {
			version := match[1]
			if version == "Unreleased" {
				continue
			}

			warnings := s.validateChangelogEntry(current, version)
			allWarnings = append(allWarnings, warnings...)
		}
	}

	return allWarnings, nil
}
func (s *ReleaseService) buildChangelogFromNotes(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) string {
	var sb strings.Builder

	tagDate, err := s.git.GetTagDate(ctx, release.Version)
	if err != nil || tagDate == "" {
		tagDate = time.Now().Format("2006-01-02")
	}

	owner, repo, provider, _ := s.git.GetRepoInfo(ctx)

	versionHeader := fmt.Sprintf("## [%s] - %s", release.Version, tagDate)

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

	usedReferences := make(map[string]bool)

	if len(notes.Sections) > 0 {
		for _, section := range notes.Sections {
			if section.Title != "" && len(section.Items) > 0 {
				sb.WriteString(fmt.Sprintf("### %s\n\n", section.Title))
				for _, item := range section.Items {
					sb.WriteString(fmt.Sprintf("- %s\n", s.formatNoteBulletWithReference(release, item, owner, repo, provider, usedReferences)))
				}
				sb.WriteString("\n")
			}
		}
	} else if len(notes.Highlights) > 0 {
		sb.WriteString("### Highlights\n\n")
		for _, highlight := range notes.Highlights {
			sb.WriteString(fmt.Sprintf("- %s\n", s.formatNoteBulletWithReference(release, highlight, owner, repo, provider, usedReferences)))
		}
		sb.WriteString("\n")
	}

	if len(notes.BreakingChanges) > 0 {
		sb.WriteString("### Breaking Changes\n\n")
		for _, bc := range notes.BreakingChanges {
			sb.WriteString(fmt.Sprintf("- %s\n", s.formatNoteBulletWithReference(release, bc, owner, repo, provider, usedReferences)))
		}
		sb.WriteString("\n")
	}

	references := s.collectReleaseReferences(release, owner, repo, provider, usedReferences)
	if len(references) > 0 {
		sb.WriteString("### References\n\n")
		for _, reference := range references {
			sb.WriteString(fmt.Sprintf("- %s\n", reference))
		}
		sb.WriteString("\n")
	}

	if len(release.MergedPRs) > 0 {
		sb.WriteString("### Pull Requests\n\n")
		for _, pr := range release.MergedPRs {
			if pr.URL != "" {
				sb.WriteString(fmt.Sprintf("- [#%d](%s) %s (by @%s)\n", pr.Number, pr.URL, pr.Title, pr.Author))
			} else {
				sb.WriteString(fmt.Sprintf("- #%d %s (by @%s)\n", pr.Number, pr.Title, pr.Author))
			}
		}
		sb.WriteString("\n")
	}

	if len(release.Contributors) > 0 {
		sb.WriteString("### Contributors\n\n")
		sb.WriteString("Thanks to ")
		for i, contributor := range release.Contributors {
			sb.WriteString(fmt.Sprintf("@%s", contributor))
			if i < len(release.Contributors)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (s *ReleaseService) formatNoteBulletWithReference(release *models.Release, itemText, owner, repo, provider string, usedReferences map[string]bool) string {
	reference := s.matchReferenceForNoteItem(release, itemText, owner, repo, provider)
	if reference == "" || usedReferences[reference] {
		return itemText
	}

	usedReferences[reference] = true
	return fmt.Sprintf("%s (%s)", itemText, reference)
}

func (s *ReleaseService) collectReleaseReferences(release *models.Release, owner, repo, provider string, exclude map[string]bool) []string {
	seen := make(map[string]bool)
	refs := make([]string, 0)

	appendItem := func(item models.ReleaseItem) {
		ref := formatReleaseReference(item, owner, repo, provider)
		if ref == "" || seen[ref] || exclude[ref] {
			return
		}
		seen[ref] = true
		refs = append(refs, ref)
	}

	for _, item := range release.Breaking {
		appendItem(item)
	}
	for _, item := range release.Features {
		appendItem(item)
	}
	for _, item := range release.BugFixes {
		appendItem(item)
	}
	for _, item := range release.Improvements {
		appendItem(item)
	}
	for _, item := range release.Documentation {
		appendItem(item)
	}
	for _, item := range release.Other {
		appendItem(item)
	}

	return refs
}

func (s *ReleaseService) matchReferenceForNoteItem(release *models.Release, itemText, owner, repo, provider string) string {
	normalizedItem := normalizeChangelogMatchText(itemText)
	if len(normalizedItem) < 12 {
		return ""
	}

	for _, releaseItem := range s.allReleaseItems(release) {
		normalizedDescription := normalizeChangelogMatchText(releaseItem.Description)
		if len(normalizedDescription) < 12 {
			continue
		}

		if normalizedItem == normalizedDescription || strings.Contains(normalizedItem, normalizedDescription) || strings.Contains(normalizedDescription, normalizedItem) {
			return formatReleaseReference(releaseItem, owner, repo, provider)
		}
	}

	return ""
}

func (s *ReleaseService) allReleaseItems(release *models.Release) []models.ReleaseItem {
	items := make([]models.ReleaseItem, 0, len(release.Breaking)+len(release.Features)+len(release.BugFixes)+len(release.Improvements)+len(release.Documentation)+len(release.Other))
	items = append(items, release.Breaking...)
	items = append(items, release.Features...)
	items = append(items, release.BugFixes...)
	items = append(items, release.Improvements...)
	items = append(items, release.Documentation...)
	items = append(items, release.Other...)
	return items
}

func normalizeChangelogMatchText(text string) string {
	text = strings.ToLower(text)
	replacer := strings.NewReplacer(
		"`", " ",
		"*", " ",
		"_", " ",
		"#", " ",
		"-", " ",
		"/", " ",
		":", " ",
		";", " ",
		",", " ",
		".", " ",
		"(", " ",
		")", " ",
	)
	text = replacer.Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

func formatReleaseReference(item models.ReleaseItem, owner, repo, provider string) string {
	if item.PRNumber != "" {
		if provider == "github" && owner != "" && repo != "" {
			return fmt.Sprintf("[#%s](https://github.com/%s/%s/pull/%s)", item.PRNumber, owner, repo, item.PRNumber)
		}
		return fmt.Sprintf("#%s", item.PRNumber)
	}

	if item.CommitHash == "" {
		return ""
	}

	shortHash := item.CommitHash
	if len(shortHash) > 7 {
		shortHash = shortHash[:7]
	}

	if provider == "github" && owner != "" && repo != "" {
		return fmt.Sprintf("[`%s`](https://github.com/%s/%s/commit/%s)", shortHash, owner, repo, item.CommitHash)
	}

	return fmt.Sprintf("`%s`", shortHash)
}

// BuildChangelogPreview generates a preview of how the CHANGELOG entry will look
func (s *ReleaseService) BuildChangelogPreview(ctx context.Context, release *models.Release, notes *models.ReleaseNotes) string {
	return s.buildChangelogFromNotes(ctx, release, notes)
}

// buildChangelog formats the changelog from raw commits (fallback when AI is not available)
func (s *ReleaseService) buildChangelog(release *models.Release) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s\n\n", release.Version))

	if len(release.Breaking) > 0 {
		sb.WriteString("### BREAKING CHANGES\n\n")
		for _, item := range release.Breaking {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Features) > 0 {
		sb.WriteString("### New Features\n\n")
		for _, item := range release.Features {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.BugFixes) > 0 {
		sb.WriteString("### Bug Fixes\n\n")
		for _, item := range release.BugFixes {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Improvements) > 0 {
		sb.WriteString("### Improvements\n\n")
		for _, item := range release.Improvements {
			sb.WriteString(s.formatReleaseItem(item))
		}
		sb.WriteString("\n")
	}

	if len(release.Documentation) > 0 {
		sb.WriteString("### Documentation\n\n")
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

	if ref := formatReleaseReference(item, "", "", ""); ref != "" {
		line += fmt.Sprintf(" (%s)", ref)
	}

	line += "\n"
	return line
}

func (s *ReleaseService) CommitChangelog(ctx context.Context, version string) error {
	log := logger.FromContext(ctx)
	log.Info("starting changelog commit process", "version", version)

	repoRoot, err := s.git.GetRepoRoot(ctx)
	if err != nil {
		log.Error("failed to get repository root for committing changelog", "error", err)
		return domainErrors.NewAppError(domainErrors.TypeGit, "failed to get repository root", err)
	}
	changelogFile := filepath.Join(repoRoot, "CHANGELOG.md")

	if err := s.git.AddFileToStaging(ctx, changelogFile); err != nil {
		log.Error("failed to add CHANGELOG.md to staging", "error", err)
		return domainErrors.NewAppError(domainErrors.TypeGit, "failed to add CHANGELOG.md to staging", err)
	}
	log.Debug("CHANGELOG.md added to staging")

	versionFile, _, err := s.FindVersionFile(ctx)
	if err != nil {
		if s.config != nil && s.config.VersionFile != "" {
			versionFile = s.config.VersionFile
		} else {
			versionFile = ""
		}
	}

	if versionFile != "" {
		if _, err := os.Stat(versionFile); err == nil {
			log.Debug("adding version file to staging", "file", versionFile)
			if err := s.git.AddFileToStaging(ctx, versionFile); err != nil {
				log.Error("failed to add version file to staging", "file", versionFile, "error", err)
				return domainErrors.NewAppError(domainErrors.TypeGit, fmt.Sprintf("failed to add version file to staging: %s", versionFile), err)
			}
			log.Debug("version file added to staging", "file", versionFile)
		} else {
			log.Debug("version file not found, skipping", "file", versionFile)
		}
	}

	log.Debug("checking for staged changes")
	if !s.git.HasStagedChanges(ctx) {
		log.Error("no staged changes detected after adding files")
		return domainErrors.ErrNoChanges
	}
	log.Debug("staged changes detected, proceeding with commit")

	message := fmt.Sprintf("chore: update changelog and bump version to %s", version)
	if err := s.git.CreateCommit(ctx, message); err != nil {
		log.Error("failed to create commit", "error", err)
		return domainErrors.NewAppError(domainErrors.TypeGit, "failed to commit changelog and version bump", err)
	}

	log.Info("changelog commit process completed successfully")
	return nil

}

// PushChanges pushes committed changes to the remote repository
func (s *ReleaseService) PushChanges(ctx context.Context) error {
	return s.git.Push(ctx)
}
