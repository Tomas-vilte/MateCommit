package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateChangelogEntry_Valid tests validation with valid entry
func TestValidateChangelogEntry_Valid(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

Summary text

### ✨ Highlights
- Feature 1
- Feature 2
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")
	assert.Empty(t, warnings, "Valid entry should have no warnings")
}

// TestValidateChangelogEntry_MissingDate tests missing date detection
func TestValidateChangelogEntry_MissingDate(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0]

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

### ✨ Highlights
- Feature 1
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")
	assert.NotEmpty(t, warnings)

	// Check for missing_date warning
	var hasMissingDate bool
	for _, w := range warnings {
		if w.Type == "missing_date" {
			hasMissingDate = true
			assert.Contains(t, w.Message, "missing a date")
			break
		}
	}
	assert.True(t, hasMissingDate, "Should have missing_date warning")
}

// TestValidateChangelogEntry_InvalidDateFormat tests invalid date format
func TestValidateChangelogEntry_InvalidDateFormat(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 01/05/2026

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

### ✨ Highlights
- Feature 1
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")
	assert.NotEmpty(t, warnings)

	// Check for missing_date warning with ISO 8601 message
	var hasMissingDate bool
	for _, w := range warnings {
		if w.Type == "missing_date" {
			hasMissingDate = true
			assert.Contains(t, w.Message, "ISO 8601")
			break
		}
	}
	assert.True(t, hasMissingDate, "Should have missing_date warning")
}

// TestValidateChangelogEntry_MissingLink tests missing comparison link
func TestValidateChangelogEntry_MissingLink(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 2026-01-05

### ✨ Highlights
- Feature 1
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")
	assert.NotEmpty(t, warnings)

	// Check for missing_link warning
	var hasMissingLink bool
	for _, w := range warnings {
		if w.Type == "missing_link" {
			hasMissingLink = true
			assert.Contains(t, w.Message, "missing a comparison link")
			break
		}
	}
	assert.True(t, hasMissingLink, "Should have missing_link warning")
}

// TestValidateChangelogEntry_NoSections tests missing sections
func TestValidateChangelogEntry_NoSections(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

Just some text without sections
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")
	assert.NotEmpty(t, warnings)

	// Check for no_sections warning
	var hasNoSections bool
	for _, w := range warnings {
		if w.Type == "no_sections" {
			hasNoSections = true
			assert.Contains(t, w.Message, "no sections")
			break
		}
	}
	assert.True(t, hasNoSections, "Should have no_sections warning")
}

// TestValidateChangelogEntry_ShortContent tests short content detection
func TestValidateChangelogEntry_ShortContent(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

### ✨ Highlights
- X
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")

	// Should have warning about short content
	var hasShortContent bool
	for _, w := range warnings {
		if w.Type == "short_content" {
			hasShortContent = true
			break
		}
	}
	assert.True(t, hasShortContent, "Should warn about short content")
}

// TestValidateChangelogEntry_MultipleIssues tests multiple warnings
func TestValidateChangelogEntry_MultipleIssues(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0]

Short
`

	warnings := service.validateChangelogEntry(content, "v1.0.0")

	// Should have multiple warnings
	assert.GreaterOrEqual(t, len(warnings), 3, "Should have at least 3 warnings")

	// Check for specific warning types
	types := make(map[string]bool)
	for _, w := range warnings {
		types[w.Type] = true
	}

	assert.True(t, types["missing_date"], "Should warn about missing date")
	assert.True(t, types["missing_link"], "Should warn about missing link")
	assert.True(t, types["no_sections"], "Should warn about no sections")
}

// TestValidateChangelog tests validating entire CHANGELOG
func TestValidateChangelog(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	content := `# Changelog

## [Unreleased]

## [v1.1.0] - 2026-01-05

[v1.1.0]: https://github.com/test/repo/compare/v1.0.0...v1.1.0

### ✨ Features
- New feature with detailed description
- Another feature with more details
- Third feature to ensure content is long enough

### 🐛 Bug Fixes
- Fixed important bug
- Fixed another bug

## [v1.0.0]

Initial release
`

	err := os.WriteFile(changelogPath, []byte(content), 0644)
	require.NoError(t, err)

	warnings, err := service.ValidateChangelog(changelogPath)
	require.NoError(t, err)

	// v1.1.0 should be valid, v1.0.0 should have warnings
	assert.NotEmpty(t, warnings, "Should have warnings for v1.0.0")

	// Check that warnings are for v1.0.0, not v1.1.0 or Unreleased
	for _, w := range warnings {
		assert.Contains(t, w.Message, "v1.0.0")
		assert.NotContains(t, w.Message, "v1.1.0")
		assert.NotContains(t, w.Message, "Unreleased")
	}
}

// TestValidateChangelog_SkipsUnreleased tests that Unreleased is skipped
func TestValidateChangelog_SkipsUnreleased(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	content := `# Changelog

## [Unreleased]

Some pending changes

## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

### ✨ Features
- Feature
`

	err := os.WriteFile(changelogPath, []byte(content), 0644)
	require.NoError(t, err)

	warnings, err := service.ValidateChangelog(changelogPath)
	require.NoError(t, err)

	// Should not have warnings about Unreleased
	for _, w := range warnings {
		assert.NotContains(t, w.Message, "Unreleased")
	}
}
