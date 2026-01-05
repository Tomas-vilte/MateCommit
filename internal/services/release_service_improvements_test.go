package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/models"
)

// TestBuildChangelogFromNotes_WithDate tests that dates are always included
func TestBuildChangelogFromNotes_WithDate(t *testing.T) {
	mockGit := &mockGitService{
		owner:    "test",
		repo:     "repo",
		provider: "github",
	}

	service := &ReleaseService{
		git: mockGit,
	}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v1.8.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "Test release",
		Highlights: []string{
			"Feature 1",
			"Feature 2",
		},
	}

	result := service.buildChangelogFromNotes(ctx, release, notes)

	// Should always include a date in ISO 8601 format
	assert.Contains(t, result, "## [v1.8.0] - ")
	// Date should be in YYYY-MM-DD format
	assert.Regexp(t, `## \[v1\.8\.0\] - \d{4}-\d{2}-\d{2}`, result)
}

// TestBuildChangelogFromNotes_WithSemanticSections tests semantic sections rendering
func TestBuildChangelogFromNotes_WithSemanticSections(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v1.8.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "Test release with sections",
		Sections: []models.ReleaseNotesSection{
			{
				Title: "✨ AI & Generation Improvements",
				Items: []string{
					"Improved AI accuracy",
					"Added new models",
				},
			},
			{
				Title: "🛠️ Templates & Configuration",
				Items: []string{
					"New template system",
				},
			},
		},
	}

	result := service.buildChangelogFromNotes(ctx, release, notes)

	// Should render sections
	assert.Contains(t, result, "### ✨ AI & Generation Improvements")
	assert.Contains(t, result, "- Improved AI accuracy")
	assert.Contains(t, result, "- Added new models")
	assert.Contains(t, result, "### 🛠️ Templates & Configuration")
	assert.Contains(t, result, "- New template system")

	// Should NOT render Highlights section
	assert.NotContains(t, result, "### ✨ Highlights")
}

// TestBuildChangelogFromNotes_FallbackToHighlights tests backward compatibility
func TestBuildChangelogFromNotes_FallbackToHighlights(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v1.8.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "Test release",
		Highlights: []string{
			"Feature 1",
			"Feature 2",
		},
		// No sections - should fall back to Highlights
	}

	result := service.buildChangelogFromNotes(ctx, release, notes)

	// Should render Highlights when no sections
	assert.Contains(t, result, "### ✨ Highlights")
	assert.Contains(t, result, "- Feature 1")
	assert.Contains(t, result, "- Feature 2")
}

// TestBuildChangelogFromNotes_WithBreakingChanges tests breaking changes rendering
func TestBuildChangelogFromNotes_WithBreakingChanges(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v2.0.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "Major release",
		Highlights: []string{
			"New feature",
		},
		BreakingChanges: []string{
			"Removed deprecated API",
			"Changed configuration format",
		},
	}

	result := service.buildChangelogFromNotes(ctx, release, notes)

	// Should render breaking changes
	assert.Contains(t, result, "### ⚠️ Breaking Changes")
	assert.Contains(t, result, "- Removed deprecated API")
	assert.Contains(t, result, "- Changed configuration format")
}

// TestPrependToChangelog_NewFile tests creating a new CHANGELOG
func TestPrependToChangelog_NewFile(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	newContent := "## [v1.0.0] - 2026-01-05\n\nFirst release"

	err := service.prependToChangelog(changelogPath, newContent)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)
	assert.Contains(t, result, "# Changelog")
	assert.Contains(t, result, "All notable changes")
	assert.Contains(t, result, "## [v1.0.0] - 2026-01-05")
}

// TestPrependToChangelog_PreventsDuplicates tests duplicate prevention
func TestPrependToChangelog_PreventsDuplicates(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create initial CHANGELOG with v1.0.0
	initial := `# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

Initial release

### ✨ Highlights
- Feature 1
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	// Try to add v1.0.0 again (should replace, not duplicate)
	newContent := `## [v1.0.0] - 2026-01-05

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0

Updated release notes

### ✨ Highlights
- Feature 1 (updated)
- Feature 2
`

	err = service.prependToChangelog(changelogPath, newContent)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)

	// Should only have ONE occurrence of v1.0.0
	count := strings.Count(result, "## [v1.0.0]")
	assert.Equal(t, 1, count, "Should have exactly one v1.0.0 entry")

	// Should have the updated content
	assert.Contains(t, result, "Feature 2")
	assert.Contains(t, result, "Updated release notes")
}

// TestPrependToChangelog_AddsNewVersion tests adding a new version
func TestPrependToChangelog_AddsNewVersion(t *testing.T) {
	service := &ReleaseService{}

	tmpDir := t.TempDir()
	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")

	// Create initial CHANGELOG with v1.0.0
	initial := `# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-01-04

Initial release
`

	err := os.WriteFile(changelogPath, []byte(initial), 0644)
	require.NoError(t, err)

	// Add v1.1.0
	newContent := `## [v1.1.0] - 2026-01-05

New features

### ✨ Highlights
- Feature A
`

	err = service.prependToChangelog(changelogPath, newContent)
	require.NoError(t, err)

	content, err := os.ReadFile(changelogPath)
	require.NoError(t, err)

	result := string(content)

	// Should have both versions
	assert.Contains(t, result, "## [v1.1.0]")
	assert.Contains(t, result, "## [v1.0.0]")

	// v1.1.0 should come before v1.0.0
	v110Pos := strings.Index(result, "## [v1.1.0]")
	v100Pos := strings.Index(result, "## [v1.0.0]")
	assert.Less(t, v110Pos, v100Pos, "v1.1.0 should appear before v1.0.0")
}

// TestConsolidateLinkDefinitions tests link deduplication
func TestConsolidateLinkDefinitions(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

## [v1.1.0] - 2026-01-05

[v1.1.0]: https://github.com/test/repo/compare/v1.0.0...v1.1.0

New features

## [v1.0.0] - 2026-01-04

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0
[v1.1.0]: https://github.com/test/repo/compare/v1.0.0...v1.1.0

Initial release
`

	result := service.consolidateLinkDefinitions(content)

	// Should only have ONE link definition for v1.1.0
	count := strings.Count(result, "[v1.1.0]:")
	assert.Equal(t, 1, count, "Should have exactly one v1.1.0 link definition")

	// Should still have the link
	assert.Contains(t, result, "[v1.1.0]: https://github.com/test/repo/compare/v1.0.0...v1.1.0")
}

// TestConsolidateLinkDefinitions_DifferentURLs tests warning for conflicting URLs
func TestConsolidateLinkDefinitions_DifferentURLs(t *testing.T) {
	service := &ReleaseService{}

	content := `# Changelog

[v1.0.0]: https://github.com/test/repo/releases/tag/v1.0.0
[v1.0.0]: https://github.com/different/url

Content
`

	result := service.consolidateLinkDefinitions(content)

	// Should only keep the first one
	count := strings.Count(result, "[v1.0.0]:")
	assert.Equal(t, 1, count, "Should have exactly one v1.0.0 link definition")
}

// TestBuildChangelogPreview tests the public preview method
func TestBuildChangelogPreview(t *testing.T) {
	mockGit := &mockGitService{owner: "test", repo: "repo", provider: "github"}
	service := &ReleaseService{git: mockGit}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v1.8.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary: "Preview test",
		Sections: []models.ReleaseNotesSection{
			{
				Title: "✨ New Features",
				Items: []string{"Feature X"},
			},
		},
	}

	result := service.BuildChangelogPreview(ctx, release, notes)

	// Should be the same as buildChangelogFromNotes
	assert.Contains(t, result, "## [v1.8.0]")
	assert.Contains(t, result, "Preview test")
	assert.Contains(t, result, "### ✨ New Features")
}

// TestBuildChangelogFromNotes_WithGitHubLinks tests GitHub comparison links
func TestBuildChangelogFromNotes_WithGitHubLinks(t *testing.T) {
	// Create a mock git service
	mockGit := &mockGitService{
		owner:    "thomas-vilte",
		repo:     "matecommit",
		provider: "github",
	}

	service := &ReleaseService{
		git: mockGit,
	}
	ctx := context.Background()

	release := &models.Release{
		Version:         "v1.8.0",
		PreviousVersion: "v1.7.0",
	}

	notes := &models.ReleaseNotes{
		Summary:    "Test",
		Highlights: []string{"Feature"},
	}

	result := service.buildChangelogFromNotes(ctx, release, notes)

	// Should include GitHub comparison link
	assert.Contains(t, result, "[v1.8.0]: https://github.com/thomas-vilte/matecommit/compare/v1.7.0...v1.8.0")
}

// mockGitService is a simple mock for testing
type mockGitService struct {
	owner    string
	repo     string
	provider string
	tagDate  string
}

func (m *mockGitService) GetTagDate(ctx context.Context, version string) (string, error) {
	if m.tagDate != "" {
		return m.tagDate, nil
	}
	return "", nil // Simulate tag not found
}

func (m *mockGitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	return m.owner, m.repo, m.provider, nil
}

// Implement other required methods as no-ops
func (m *mockGitService) GetLastTag(ctx context.Context) (string, error)  { return "", nil }
func (m *mockGitService) GetCommitCount(ctx context.Context) (int, error) { return 0, nil }
func (m *mockGitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	return nil, nil
}
func (m *mockGitService) CreateTag(ctx context.Context, version, message string) error { return nil }
func (m *mockGitService) PushTag(ctx context.Context, version string) error            { return nil }
func (m *mockGitService) AddFileToStaging(ctx context.Context, file string) error      { return nil }
func (m *mockGitService) HasStagedChanges(ctx context.Context) bool                    { return false }
func (m *mockGitService) CreateCommit(ctx context.Context, message string) error       { return nil }
func (m *mockGitService) Push(ctx context.Context) error                               { return nil }
func (m *mockGitService) GetCurrentBranch(ctx context.Context) (string, error)         { return "main", nil }
func (m *mockGitService) FetchTags(ctx context.Context) error                          { return nil }
func (m *mockGitService) ValidateTagExists(ctx context.Context, tag string) error      { return nil }
