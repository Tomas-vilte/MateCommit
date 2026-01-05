package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/models"
)

// TestEnsureUnreleasedSection_NewFile tests creating CHANGELOG with Unreleased
func TestEnsureUnreleasedSection_NewFile(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	err := service.EnsureUnreleasedSection(changelogPath)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)
	assert.Contains(t, result, "# Changelog")
	assert.Contains(t, result, "## [Unreleased]")
	assert.Contains(t, result, "Keep a Changelog")
	assert.Contains(t, result, "Semantic Versioning")
}

// TestEnsureUnreleasedSection_ExistingFile tests adding Unreleased to existing CHANGELOG
func TestEnsureUnreleasedSection_ExistingFile(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create existing CHANGELOG without Unreleased
	initial := `# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-01-05

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	err = service.EnsureUnreleasedSection(changelogPath)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)
	assert.Contains(t, result, "## [Unreleased]")

	// Unreleased should come before v1.0.0
	unreleasedPos := strings.Index(result, "## [Unreleased]")
	v100Pos := strings.Index(result, "## [v1.0.0]")
	assert.Less(t, unreleasedPos, v100Pos, "Unreleased should appear before v1.0.0")
}

// TestEnsureUnreleasedSection_AlreadyExists tests idempotency
func TestEnsureUnreleasedSection_AlreadyExists(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create CHANGELOG with Unreleased
	initial := `# Changelog

## [Unreleased]

### Added
- Pending feature

## [v1.0.0] - 2026-01-05

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	err = service.EnsureUnreleasedSection(changelogPath)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)

	// Should only have ONE Unreleased section
	count := strings.Count(result, "## [Unreleased]")
	assert.Equal(t, 1, count, "Should have exactly one Unreleased section")

	// Content should be unchanged
	assert.Contains(t, result, "Pending feature")
}

// TestParseUnreleasedSection tests extracting Unreleased content
func TestParseUnreleasedSection(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [Unreleased]

### Added
- New authentication system
- User profiles

### Fixed
- Login bug

## [v1.0.0] - 2026-01-05

Initial release
`

	result := service.parseUnreleasedSection(content)

	assert.Contains(t, result, "### Added")
	assert.Contains(t, result, "New authentication system")
	assert.Contains(t, result, "User profiles")
	assert.Contains(t, result, "### Fixed")
	assert.Contains(t, result, "Login bug")

	// Should NOT contain the version section
	assert.NotContains(t, result, "## [v1.0.0]")
	assert.NotContains(t, result, "Initial release")
}

// TestParseUnreleasedSection_Empty tests with no Unreleased content
func TestParseUnreleasedSection_Empty(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.0.0] - 2026-01-05

Initial release
`

	result := service.parseUnreleasedSection(content)
	assert.Empty(t, result, "Should return empty string when no Unreleased section")
}

// TestMoveUnreleasedToVersion tests moving Unreleased to new version
func TestMoveUnreleasedToVersion(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create CHANGELOG with Unreleased content
	initial := `# Changelog

## [Unreleased]

### Added
- New feature from Unreleased
- Another feature

## [v1.0.0] - 2026-01-04

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	release := &models.Release{
		Version:         "v1.1.0",
		PreviousVersion: "v1.0.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "New release",
		Highlights: []string{
			"Feature from AI",
		},
	}

	err = service.MoveUnreleasedToVersion(changelogPath, release, notes)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)

	// Should have new version
	assert.Contains(t, result, "## [v1.1.0]")

	// New version should contain Unreleased content
	assert.Contains(t, result, "New feature from Unreleased")
	assert.Contains(t, result, "Another feature")

	// Should also contain AI-generated content
	assert.Contains(t, result, "Feature from AI")

	// Should have a NEW empty Unreleased section
	assert.Contains(t, result, "## [Unreleased]")

	// Unreleased should come before v1.1.0
	unreleasedPos := strings.Index(result, "## [Unreleased]")
	v110Pos := strings.Index(result, "## [v1.1.0]")
	assert.Less(t, unreleasedPos, v110Pos, "Unreleased should appear before v1.1.0")

	// v1.1.0 should come before v1.0.0
	v100Pos := strings.Index(result, "## [v1.0.0]")
	assert.Less(t, v110Pos, v100Pos, "v1.1.0 should appear before v1.0.0")
}

// TestMoveUnreleasedToVersion_NoUnreleased tests when no Unreleased content exists
func TestMoveUnreleasedToVersion_NoUnreleased(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create CHANGELOG without Unreleased content
	initial := `# Changelog

## [v1.0.0] - 2026-01-04

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	release := &models.Release{
		Version:         "v1.1.0",
		PreviousVersion: "v1.0.0",
	}

	notes := &models.ReleaseNotes{
		Summary:    "New release",
		Highlights: []string{"Feature"},
	}

	err = service.MoveUnreleasedToVersion(changelogPath, release, notes)
	require.NoError(t, err)

	// Should succeed without error (no-op when no Unreleased content)
}

// TestUpdateLocalChangelog_WithUnreleased tests the integration
func TestUpdateLocalChangelog_WithUnreleased(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create CHANGELOG with Unreleased
	initial := `# Changelog

## [Unreleased]

### Added
- Feature from Unreleased

## [v1.0.0] - 2026-01-04

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	// Change to temp dir so UpdateLocalChangelog finds the file
	oldWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	}()
	_ = os.Chdir(tmpDir)

	release := &models.Release{
		Version:         "v1.1.0",
		PreviousVersion: "v1.0.0",
	}

	notes := &models.ReleaseNotes{
		Summary:    "New release",
		Highlights: []string{"AI feature"},
	}

	err = service.UpdateLocalChangelog(release, notes)
	require.NoError(t, err)

	content, err := os.ReadFile("CHANGELOG.md")
	require.NoError(t, err)

	result := string(content)

	// Should have moved Unreleased to v1.1.0
	assert.Contains(t, result, "## [v1.1.0]")
	assert.Contains(t, result, "Feature from Unreleased")

	// Should have new empty Unreleased section
	assert.Contains(t, result, "## [Unreleased]")
}
