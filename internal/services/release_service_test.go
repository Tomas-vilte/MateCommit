package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func TestReleaseService_AnalyzeNextRelease(t *testing.T) {
	t.Run("Success with existing tag and feature commits", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockVCS := new(MockVCSClient)
		mockNotesGen := new(MockReleaseNotesGenerator)
		service := NewReleaseService(mockGit, WithReleaseVCSClient(mockVCS), WithReleaseNotesGenerator(mockNotesGen))

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return([]models.Commit{
			{Message: "feat: new feature"},
			{Message: "fix: bug fix"},
		}, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, release)
		assert.Equal(t, "v1.1.0", release.Version)
		assert.Equal(t, models.MinorBump, release.VersionBump)
		assert.Equal(t, "v1.0.0", release.PreviousVersion)
		assert.Len(t, release.Features, 1)
		assert.Len(t, release.BugFixes, 1)

		mockGit.AssertExpectations(t)
	})

	t.Run("Success with breaking change", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return([]models.Commit{
			{Message: "feat: new feature\n\nBREAKING CHANGE: something broke"},
		}, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "v2.0.0", release.Version)
		assert.Equal(t, models.MajorBump, release.VersionBump)

		mockGit.AssertExpectations(t)
	})

	t.Run("Success with no previous tags (initial release)", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("", nil)
		mockGit.On("GetCommitCount", mock.Anything).Return(1, nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v0.0.0").Return([]models.Commit{
			{Message: "feat: initial commit"},
		}, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "v0.1.0", release.Version) // v0.0.0 + minor bump
		assert.Equal(t, "v0.0.0", release.PreviousVersion)

		mockGit.AssertExpectations(t)
	})

	t.Run("Error getting last tag", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("", errors.New("git error"))

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "GIT: error getting last tag")

		mockGit.AssertExpectations(t)
	})

	t.Run("No changes since last tag", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return([]models.Commit{}, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "GIT: No staged changes detected")

		mockGit.AssertExpectations(t)
	})

	t.Run("Error getting commits", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return(nil, errors.New("git error"))

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "GIT: error getting commits")

		mockGit.AssertExpectations(t)
	})

	t.Run("Empty repository", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetLastTag", mock.Anything).Return("", nil)
		mockGit.On("GetCommitCount", mock.Anything).Return(0, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "GIT: no commits found in repository")

		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_GenerateReleaseNotes(t *testing.T) {
	t.Run("Use AI generator when available", func(t *testing.T) {
		mockNotesGen := new(MockReleaseNotesGenerator)
		service := NewReleaseService(nil, WithReleaseNotesGenerator(mockNotesGen))

		release := &models.Release{Version: "v1.1.0"}
		expectedNotes := &models.ReleaseNotes{
			Title:     "Release v1.1.0",
			Changelog: "AI Generated content",
		}

		mockNotesGen.On("GenerateNotes", mock.Anything, release).Return(expectedNotes, nil)

		notes, err := service.GenerateReleaseNotes(context.Background(), release)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotes, notes)
		mockNotesGen.AssertExpectations(t)
	})

	t.Run("Fallback to basic notes when generator is nil", func(t *testing.T) {
		service := NewReleaseService(nil) // No generator

		release := &models.Release{
			Version: "v1.1.0",
			Features: []models.ReleaseItem{
				{Description: "Feature 1", Scope: "core"},
			},
			BugFixes: []models.ReleaseItem{
				{Description: "Fix 1", PRNumber: "123"},
			},
		}

		notes, err := service.GenerateReleaseNotes(context.Background(), release)

		assert.NoError(t, err)
		assert.NotNil(t, notes)
		assert.Contains(t, notes.Title, "v1.1.0")
		assert.Contains(t, notes.Summary, "1 new features")
		assert.Contains(t, notes.Summary, "1 bug fixes")
		assert.Contains(t, notes.Changelog, "Feature 1")
		assert.Contains(t, notes.Changelog, "Fix 1")
	})

	t.Run("Error from generator", func(t *testing.T) {
		mockNotesGen := new(MockReleaseNotesGenerator)
		service := NewReleaseService(nil, WithReleaseNotesGenerator(mockNotesGen))

		release := &models.Release{Version: "v1.1.0"}

		mockNotesGen.On("GenerateNotes", mock.Anything, release).Return(nil, errors.New("ai error"))

		notes, err := service.GenerateReleaseNotes(context.Background(), release)

		assert.Error(t, err)
		assert.Nil(t, notes)
		assert.Equal(t, "ai error", err.Error())
		mockNotesGen.AssertExpectations(t)
	})
}

func TestReleaseService_GetRelease(t *testing.T) {
	t.Run("should get release successfully", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		expectedRelease := &models.VCSRelease{
			TagName: "v1.2.0",
			Name:    "Release v1.2.0",
			Body:    "Release notes content",
		}

		mockVCS.On("GetRelease", mock.Anything, "v1.2.0").Return(expectedRelease, nil)

		release, err := service.GetRelease(context.Background(), "v1.2.0")

		assert.NoError(t, err)
		assert.NotNil(t, release)
		assert.Equal(t, "v1.2.0", release.TagName)
		assert.Equal(t, "Release v1.2.0", release.Name)
		assert.Equal(t, "Release notes content", release.Body)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		service := NewReleaseService(nil)

		release, err := service.GetRelease(context.Background(), "v1.2.0")

		assert.Error(t, err)
		assert.Nil(t, release)
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		mockVCS.On("GetRelease", mock.Anything, "v1.2.0").Return((*models.VCSRelease)(nil), errors.New("release not found"))

		release, err := service.GetRelease(context.Background(), "v1.2.0")

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "release not found")
		mockVCS.AssertExpectations(t)
	})
}

func TestReleaseService_UpdateRelease(t *testing.T) {
	t.Run("should update release successfully", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		mockVCS.On("UpdateRelease", mock.Anything, "v1.2.0", "Updated release notes").Return(nil)

		err := service.UpdateRelease(context.Background(), "v1.2.0", "Updated release notes")

		assert.NoError(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		service := NewReleaseService(nil)

		err := service.UpdateRelease(context.Background(), "v1.2.0", "Updated notes")

		assert.Error(t, err)
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		mockVCS.On("UpdateRelease", mock.Anything, "v1.2.0", "Updated notes").Return(errors.New("update failed"))

		err := service.UpdateRelease(context.Background(), "v1.2.0", "Updated notes")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
		mockVCS.AssertExpectations(t)
	})
}

func TestReleaseService_PublishRelease(t *testing.T) {
	t.Run("should publish release successfully", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{Title: "Release v1.0.0"}

		mockVCS.On("CreateRelease", mock.Anything, release, notes, false, true, mock.Anything).Return(nil)

		err := service.PublishRelease(context.Background(), release, notes, false, true, nil)

		assert.NoError(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		service := NewReleaseService(nil)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		err := service.PublishRelease(context.Background(), release, notes, false, true, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "CONFIGURATION: Configuration is missing")
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		mockVCS.On("CreateRelease", mock.Anything, release, notes, true, true, mock.Anything).Return(errors.New("publish failed"))

		err := service.PublishRelease(context.Background(), release, notes, true, true, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publish failed")
		mockVCS.AssertExpectations(t)
	})
}

func TestReleaseService_CreateTag(t *testing.T) {
	t.Run("should create tag successfully", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("CreateTag", mock.Anything, "v1.0.0", "Release v1.0.0").Return(nil)

		err := service.CreateTag(context.Background(), "v1.0.0", "Release v1.0.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should propagate git error", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("CreateTag", mock.Anything, "v1.0.0", "Release v1.0.0").Return(errors.New("tag already exists"))

		err := service.CreateTag(context.Background(), "v1.0.0", "Release v1.0.0")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tag already exists")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_PushTag(t *testing.T) {
	t.Run("should push tag successfully", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("PushTag", mock.Anything, "v1.0.0").Return(nil)

		err := service.PushTag(context.Background(), "v1.0.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should propagate git error", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("PushTag", mock.Anything, "v1.0.0").Return(errors.New("push failed"))

		err := service.PushTag(context.Background(), "v1.0.0")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "push failed")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_EnrichReleaseContext(t *testing.T) {
	t.Run("should enrich release context successfully", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		release := &models.Release{
			PreviousVersion: "v1.0.0",
			Version:         "v1.1.0",
		}

		mockVCS.On("GetClosedIssuesBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]models.Issue{{Number: 1, Title: "Issue 1"}}, nil)
		mockVCS.On("GetMergedPRsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]models.PullRequest{{Number: 2, Title: "PR 1"}}, nil)
		mockVCS.On("GetContributorsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]string{"user1", "user2"}, nil)
		mockVCS.On("GetFileStatsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return(&models.FileStatistics{FilesChanged: 5}, nil)
		mockVCS.On("GetFileAtTag", mock.Anything, mock.Anything, mock.Anything).
			Return("", errors.New("not found"))

		err := service.EnrichReleaseContext(context.Background(), release)

		assert.NoError(t, err)
		assert.Len(t, release.ClosedIssues, 1)
		assert.Len(t, release.MergedPRs, 1)
		assert.Len(t, release.Contributors, 2)
		assert.Equal(t, 5, release.FileStats.FilesChanged)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		service := NewReleaseService(nil)

		release := &models.Release{}

		err := service.EnrichReleaseContext(context.Background(), release)

		assert.Error(t, err)
	})

	t.Run("should continue even if some enrichments fail", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, WithReleaseVCSClient(mockVCS))

		release := &models.Release{
			PreviousVersion: "v1.0.0",
			Version:         "v1.1.0",
		}

		mockVCS.On("GetClosedIssuesBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]models.Issue{}, errors.New("api error"))
		mockVCS.On("GetMergedPRsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]models.PullRequest{{Number: 1}}, nil)
		mockVCS.On("GetContributorsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return([]string{}, errors.New("api error"))
		mockVCS.On("GetFileStatsBetweenTags", mock.Anything, "v1.0.0", "v1.1.0").
			Return(&models.FileStatistics{FilesChanged: 3}, nil)
		mockVCS.On("GetFileAtTag", mock.Anything, mock.Anything, mock.Anything).
			Return("", errors.New("not found"))

		err := service.EnrichReleaseContext(context.Background(), release)

		assert.NoError(t, err)
		assert.Len(t, release.ClosedIssues, 0)
		assert.Len(t, release.MergedPRs, 1)
		assert.Len(t, release.Contributors, 0)
		assert.Equal(t, 3, release.FileStats.FilesChanged)
		mockVCS.AssertExpectations(t)
	})
}

func TestReleaseService_UpdateAppVersion(t *testing.T) {
	t.Run("should update version in default file", func(t *testing.T) {
		dir := t.TempDir()
		cmdDir := filepath.Join(dir, "cmd")
		err := os.MkdirAll(cmdDir, 0755)
		require.NoError(t, err)
		ctx := context.Background()

		mainGoPath := filepath.Join(cmdDir, "main.go")
		initialContent := `package main
var (
	Version: "1.0.0"
)`

		err = os.WriteFile(mainGoPath, []byte(initialContent), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			VersionFile: mainGoPath,
		}

		service := NewReleaseService(nil, WithReleaseConfig(cfg))

		err = service.UpdateAppVersion(ctx, "v1.1.0")
		assert.NoError(t, err)

		newContent, err := os.ReadFile(mainGoPath)
		require.NoError(t, err)

		assert.Contains(t, string(newContent), `"1.1.0"`)
	})

	t.Run("should update version with custom pattern", func(t *testing.T) {
		dir := t.TempDir()
		versionFile := filepath.Join(dir, "version.go")
		ctx := context.Background()

		initialContent := `package version
const CurrentVersion = "0.0.1"
`
		err := os.WriteFile(versionFile, []byte(initialContent), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			VersionFile:    versionFile,
			VersionPattern: `CurrentVersion\s*=\s*".*"`,
		}

		service := NewReleaseService(nil, WithReleaseConfig(cfg))

		err = service.UpdateAppVersion(ctx, "v0.0.2")
		assert.NoError(t, err)

		newContent, err := os.ReadFile(versionFile)
		require.NoError(t, err)

		assert.Contains(t, string(newContent), `CurrentVersion = "0.0.2"`)
	})

	t.Run("should fail if pattern not found", func(t *testing.T) {
		dir := t.TempDir()
		ctx := context.Background()
		versionFile := filepath.Join(dir, "version.go")
		err := os.WriteFile(versionFile, []byte("package version\n"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{VersionFile: versionFile}
		service := NewReleaseService(nil, WithReleaseConfig(cfg))

		err = service.UpdateAppVersion(ctx, "v1.0.0")
		assert.Error(t, err)
	})
}

func TestReleaseService_CategorizeCommits_AllTypes(t *testing.T) {
	service := &ReleaseService{}
	release := &models.Release{
		AllCommits: []models.Commit{
			{Message: "feat: new feature"},
			{Message: "fix: bug fix"},
			{Message: "docs: update readme"},
			{Message: "style: linting"},
			{Message: "refactor: clean code"},
			{Message: "perf: optimize"},
			{Message: "test: add tests"},
			{Message: "chore: update deps"},
			{Message: "build: update build script"},
			{Message: "ci: fix pipeline"},
			{Message: "unknown: something"},
		},
	}

	service.categorizeCommits(release)

	assert.Len(t, release.Features, 1)
	assert.Len(t, release.BugFixes, 1)
	assert.Len(t, release.Documentation, 1)
	assert.Len(t, release.Improvements, 2)
	assert.Len(t, release.Other, 6)
}

func TestReleaseService_CalculateVersion_Exhaustive(t *testing.T) {
	service := &ReleaseService{}

	tests := []struct {
		name       string
		currentTag string
		release    *models.Release
		expVersion string
		expBump    models.VersionBump
	}{
		{
			name:       "Major bump due to breaking change",
			currentTag: "v1.2.3",
			release: &models.Release{
				Breaking: []models.ReleaseItem{{Type: "feat", Breaking: true}},
			},
			expVersion: "v2.0.0",
			expBump:    models.MajorBump,
		},
		{
			name:       "Minor bump due to feature",
			currentTag: "v1.2.3",
			release: &models.Release{
				Features: []models.ReleaseItem{{Type: "feat"}},
			},
			expVersion: "v1.3.0",
			expBump:    models.MinorBump,
		},
		{
			name:       "Patch bump due to fix",
			currentTag: "v1.2.3",
			release: &models.Release{
				BugFixes: []models.ReleaseItem{{Type: "fix"}},
			},
			expVersion: "v1.2.4",
			expBump:    models.PatchBump,
		},
		{
			name:       "Patch bump due to improvement",
			currentTag: "v1.2.3",
			release: &models.Release{
				Improvements: []models.ReleaseItem{{Type: "perf"}},
			},
			expVersion: "v1.2.4",
			expBump:    models.PatchBump,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, bump := service.calculateVersion(tt.currentTag, tt.release)
			assert.Equal(t, tt.expVersion, version)
			assert.Equal(t, tt.expBump, bump)
		})
	}
}

func TestReleaseService_PrependToChangelog(t *testing.T) {
	service := &ReleaseService{}
	dir := t.TempDir()
	changelogPath := filepath.Join(dir, "CHANGELOG.md")

	t.Run("Create new changelog with content", func(t *testing.T) {
		err := service.prependToChangelog(changelogPath, "## [1.0.0]\nNew version")
		assert.NoError(t, err)

		content, _ := os.ReadFile(changelogPath)
		assert.Contains(t, string(content), "## [1.0.0]")
	})

	t.Run("Prepend to existing changelog", func(t *testing.T) {
		err := service.prependToChangelog(changelogPath, "## [1.1.0]\nNewer version\n")
		assert.NoError(t, err)

		content, _ := os.ReadFile(changelogPath)
		assert.Contains(t, string(content), "## [1.1.0]")
		assert.Contains(t, string(content), "## [1.0.0]")
		assert.True(t, strings.HasPrefix(string(content), "# Changelog"))
	})
}

func TestReleaseService_FindVersionFile_RealScenarios(t *testing.T) {
	t.Run("Go project with standard layout", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()

		goModPath := filepath.Join(dir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0644)
		require.NoError(t, err)

		versionDir := filepath.Join(dir, "internal", "version")
		err = os.MkdirAll(versionDir, 0755)
		require.NoError(t, err)

		versionFile := filepath.Join(versionDir, "version.go")
		versionContent := `package version

const Version = "1.0.0"
`
		err = os.WriteFile(versionFile, []byte(versionContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "internal/version/version.go", foundFile)
		assert.Contains(t, pattern, "Version")
	})

	t.Run("Python project with setup.py", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		setupPy := filepath.Join(dir, "setup.py")
		setupContent := `from setuptools import setup

setup(
    name="test",
    version="0.1.0",
)
`
		err := os.WriteFile(setupPy, []byte(setupContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "setup.py", foundFile)
		assert.Contains(t, pattern, "version")
	})

	t.Run("JavaScript project with package.json", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		packageJSON := filepath.Join(dir, "package.json")
		packageContent := `{
  "name": "test-package",
  "version": "2.3.4",
  "description": "Test package"
}
`
		err := os.WriteFile(packageJSON, []byte(packageContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "package.json", foundFile)
		assert.Contains(t, pattern, "version")
	})

	t.Run("Rust project with Cargo.toml", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		cargoToml := filepath.Join(dir, "Cargo.toml")
		cargoContent := `[package]
name = "test"
version = "0.5.0"
edition = "2021"
`
		err := os.WriteFile(cargoToml, []byte(cargoContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "Cargo.toml", foundFile)
		assert.Contains(t, pattern, "version")
	})

	t.Run("Config-specified version file takes precedence", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		customVersion := filepath.Join(dir, "custom_version.go")
		content := `package main

var AppVersion = "3.0.0"
`
		err := os.WriteFile(customVersion, []byte(content), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		cfg := &config.Config{
			VersionFile:    customVersion,
			VersionPattern: `AppVersion\s*=\s*".*"`,
		}
		service := &ReleaseService{config: cfg}

		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, customVersion, foundFile)
		assert.Equal(t, `AppVersion\s*=\s*".*"`, pattern)
	})

	t.Run("Recursive search finds version file in nested directory", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		goModPath := filepath.Join(dir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n"), 0644)
		require.NoError(t, err)

		pkgVersionDir := filepath.Join(dir, "pkg", "version")
		err = os.MkdirAll(pkgVersionDir, 0755)
		require.NoError(t, err)

		versionFile := filepath.Join(pkgVersionDir, "version.go")
		content := `package version

const Version = "1.2.3"
`
		err = os.WriteFile(versionFile, []byte(content), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		foundFile, pattern, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, "pkg/version/version.go", foundFile)
		assert.Contains(t, pattern, "Version")
	})

	t.Run("Error when no version file found", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		err := os.Chdir(dir)
		require.NoError(t, err)

		service := &ReleaseService{}
		_, _, err = service.FindVersionFile(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Version file not found")
	})
}

func TestReleaseService_DetectProjectType_RealScenarios(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string]string
		expectedType string
	}{
		{
			name: "Go project with go.mod",
			files: map[string]string{
				"go.mod": "module test",
			},
			expectedType: "go",
		},
		{
			name: "Python project with requirements.txt",
			files: map[string]string{
				"requirements.txt": "flask==2.0.0",
			},
			expectedType: "python",
		},
		{
			name: "JavaScript project with package.json",
			files: map[string]string{
				"package.json": `{"name": "test"}`,
			},
			expectedType: "js",
		},
		{
			name: "Rust project with Cargo.toml",
			files: map[string]string{
				"Cargo.toml": "[package]",
			},
			expectedType: "rust",
		},
		{
			name: "PHP project with composer.json",
			files: map[string]string{
				"composer.json": `{"name": "test/package"}`,
			},
			expectedType: "php",
		},
		{
			name: "Ruby project with Gemfile",
			files: map[string]string{
				"Gemfile": "source 'https://rubygems.org'",
			},
			expectedType: "ruby",
		},
		{
			name: "Go project detected by .go files",
			files: map[string]string{
				"main.go": "package main",
				"util.go": "package util",
			},
			expectedType: "go",
		},
		{
			name: "Python project detected by .py files",
			files: map[string]string{
				"app.py":  "import flask",
				"test.py": "import unittest",
			},
			expectedType: "python",
		},
		{
			name:         "Unknown project type",
			files:        map[string]string{},
			expectedType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			origDir, _ := os.Getwd()
			defer func() {
				if err := os.Chdir(origDir); err != nil {
					t.Fatal(err)
				}
			}()
			for filename, content := range tt.files {
				filePath := filepath.Join(dir, filename)
				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			err := os.Chdir(dir)
			require.NoError(t, err)

			service := &ReleaseService{}
			projectType := service.detectProjectType()

			assert.Equal(t, tt.expectedType, projectType)
		})
	}
}

func TestReleaseService_ValidateMainBranch_RealScenarios(t *testing.T) {
	t.Run("Valid main branch", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)

		err := service.ValidateMainBranch(context.Background())

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Valid master branch", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("master", nil)

		err := service.ValidateMainBranch(context.Background())

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Invalid feature branch", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("feature/new-feature", nil)

		err := service.ValidateMainBranch(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currently on 'feature/new-feature'")
		mockGit.AssertExpectations(t)
	})

	t.Run("Invalid develop branch", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("develop", nil)

		err := service.ValidateMainBranch(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currently on 'develop'")
		mockGit.AssertExpectations(t)
	})

	t.Run("Git error getting branch", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("", errors.New("not a git repository"))

		err := service.ValidateMainBranch(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error getting current branch")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_CommitChangelog_RealScenarios(t *testing.T) {
	t.Run("Successfully commit changelog with version file", func(t *testing.T) {
		dir := t.TempDir()
		versionFile := filepath.Join(dir, "cmd", "main.go")
		err := os.MkdirAll(filepath.Dir(versionFile), 0755)
		require.NoError(t, err)

		content := `package main
const Version = "1.0.0"
`
		err = os.WriteFile(versionFile, []byte(content), 0644)
		require.NoError(t, err)

		mockGit := new(MockGitService)
		cfg := &config.Config{VersionFile: versionFile}
		service := NewReleaseService(mockGit, WithReleaseConfig(cfg))

		mockGit.On("AddFileToStaging", mock.Anything, "CHANGELOG.md").Return(nil)
		mockGit.On("AddFileToStaging", mock.Anything, versionFile).Return(nil)
		mockGit.On("HasStagedChanges", mock.Anything).Return(true)
		mockGit.On("CreateCommit", mock.Anything, "chore: update changelog and bump version to v1.1.0").Return(nil)

		err = service.CommitChangelog(context.Background(), "v1.1.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Error when no staged changes", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("AddFileToStaging", mock.Anything, "CHANGELOG.md").Return(nil)
		mockGit.On("HasStagedChanges", mock.Anything).Return(false)

		err := service.CommitChangelog(context.Background(), "v1.1.0")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No staged changes detected")
		mockGit.AssertExpectations(t)
	})

	t.Run("Skips missing version file and commits if other files staged", func(t *testing.T) {
		mockGit := new(MockGitService)
		cfg := &config.Config{VersionFile: "/non/existent/file.go"}
		service := NewReleaseService(mockGit, WithReleaseConfig(cfg))

		mockGit.On("AddFileToStaging", mock.Anything, "CHANGELOG.md").Return(nil)
		mockGit.On("HasStagedChanges", mock.Anything).Return(true)
		mockGit.On("CreateCommit", mock.Anything, "chore: update changelog and bump version to v2.0.0").Return(nil)

		err := service.CommitChangelog(context.Background(), "v2.0.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Error adding file to staging", func(t *testing.T) {
		dir := t.TempDir()
		versionFile := filepath.Join(dir, "version.txt")
		err := os.WriteFile(versionFile, []byte("1.0.0"), 0644)
		require.NoError(t, err)

		mockGit := new(MockGitService)
		cfg := &config.Config{VersionFile: versionFile}
		service := NewReleaseService(mockGit, WithReleaseConfig(cfg))

		mockGit.On("AddFileToStaging", mock.Anything, "CHANGELOG.md").Return(nil)
		mockGit.On("AddFileToStaging", mock.Anything, versionFile).Return(errors.New("permission denied"))

		err = service.CommitChangelog(context.Background(), "v1.1.0")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add version file to staging")
		mockGit.AssertExpectations(t)
	})

	t.Run("Error creating commit", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("AddFileToStaging", mock.Anything, "CHANGELOG.md").Return(nil)
		mockGit.On("HasStagedChanges", mock.Anything).Return(true)
		mockGit.On("CreateCommit", mock.Anything, mock.Anything).Return(errors.New("commit failed"))

		err := service.CommitChangelog(context.Background(), "v1.0.0")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit changelog and version bump")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_UpdateAppVersion_MultiLanguageRealScenarios(t *testing.T) {
	tests := []struct {
		name            string
		fileExt         string
		initialContent  string
		versionPattern  string
		expectedContent string
		version         string
	}{
		{
			name:    "Go file with const Version",
			fileExt: ".go",
			initialContent: `package version

const Version = "1.0.0"
`,
			versionPattern:  `const\s+Version\s*=\s*"[^"]*"`,
			expectedContent: `const Version = "2.0.0"`,
			version:         "v2.0.0",
		},
		{
			name:    "Python file with __version__",
			fileExt: ".py",
			initialContent: `"""Version module"""

__version__ = "0.1.0"
`,
			versionPattern:  `__version__\s*=\s*"[^"]*"`,
			expectedContent: `__version__ = "0.2.0"`,
			version:         "v0.2.0",
		},
		{
			name:    "JavaScript package.json",
			fileExt: ".json",
			initialContent: `{
  "name": "test",
  "version": "1.2.3",
  "description": "Test"
}`,
			versionPattern:  `"version"\s*:\s*"[^"]*"`,
			expectedContent: `"version": "1.3.0"`,
			version:         "v1.3.0",
		},
		{
			name:    "Rust Cargo.toml",
			fileExt: ".toml",
			initialContent: `[package]
name = "test"
version = "0.1.0"
edition = "2021"`,
			versionPattern:  `version\s*=\s*"[^"]*"`,
			expectedContent: `version = "0.2.0"`,
			version:         "v0.2.0",
		},
		{
			name:    "Ruby version.rb",
			fileExt: ".rb",
			initialContent: `module MyGem
  VERSION = "1.0.0"
end`,
			versionPattern:  `VERSION\s*=\s*"[^"]*"`,
			expectedContent: `VERSION = "1.1.0"`,
			version:         "v1.1.0",
		},
		{
			name:    "PHP composer.json",
			fileExt: ".json",
			initialContent: `{
  "name": "vendor/package",
  "version": "2.0.0"
}`,
			versionPattern:  `"version"\s*:\s*"[^"]*"`,
			expectedContent: `"version": "3.0.0"`,
			version:         "v3.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			versionFile := filepath.Join(dir, "version"+tt.fileExt)
			err := os.WriteFile(versionFile, []byte(tt.initialContent), 0644)
			require.NoError(t, err)

			cfg := &config.Config{
				VersionFile:    versionFile,
				VersionPattern: tt.versionPattern,
			}
			service := &ReleaseService{config: cfg}

			err = service.UpdateAppVersion(context.Background(), tt.version)
			assert.NoError(t, err)

			updatedContent, err := os.ReadFile(versionFile)
			require.NoError(t, err)

			assert.Contains(t, string(updatedContent), tt.expectedContent)
		})
	}
}

func TestReleaseService_FilterValidCommits_RealScenarios(t *testing.T) {
	service := &ReleaseService{}

	tests := []struct {
		name     string
		commits  []models.Commit
		expected int
	}{
		{
			name: "All conventional commits",
			commits: []models.Commit{
				{Message: "feat: add feature"},
				{Message: "fix: fix bug"},
				{Message: "docs: update docs"},
			},
			expected: 3,
		},
		{
			name: "Mixed conventional and non-conventional",
			commits: []models.Commit{
				{Message: "feat: add feature"},
				{Message: "WIP: work in progress"},
				{Message: "fix: fix bug"},
				{Message: "random commit message"},
			},
			expected: 2,
		},
		{
			name: "No conventional commits",
			commits: []models.Commit{
				{Message: "WIP"},
				{Message: "update stuff"},
				{Message: "changes"},
			},
			expected: 0,
		},
		{
			name: "Conventional commits with scopes",
			commits: []models.Commit{
				{Message: "feat(api): add endpoint"},
				{Message: "fix(ui): fix button"},
				{Message: "refactor(core): improve logic"},
			},
			expected: 3,
		},
		{
			name: "Breaking changes",
			commits: []models.Commit{
				{Message: "feat!: breaking change"},
				{Message: "fix: normal fix"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := service.filterValidCommits(tt.commits)
			assert.Len(t, valid, tt.expected)
		})
	}
}

func TestReleaseService_PushChanges_RealScenarios(t *testing.T) {
	t.Run("Successfully push changes", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("Push", mock.Anything).Return(nil)

		err := service.PushChanges(context.Background())

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("Error pushing to remote", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		mockGit.On("Push", mock.Anything).Return(errors.New("failed to push: remote rejected"))

		err := service.PushChanges(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "remote rejected")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_UpdateLocalChangelog_RealScenarios(t *testing.T) {
	t.Run("Update changelog with basic notes", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		err := os.Chdir(dir)
		require.NoError(t, err)

		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		release := &models.Release{
			Version:         "v1.1.0",
			PreviousVersion: "v1.0.0",
		}

		notes := &models.ReleaseNotes{
			Title:   "Version 1.1.0",
			Summary: "This release includes 2 new features",
			Highlights: []string{
				"Improved performance",
				"Added new API endpoint",
			},
		}

		mockGit.On("GetTagDate", mock.Anything, "v1.1.0").Return("2025-01-15", nil)
		mockGit.On("GetRepoInfo", mock.Anything).Return("user", "repo", "github", nil)

		err = service.UpdateLocalChangelog(release, notes)

		assert.NoError(t, err)

		changelogPath := filepath.Join(dir, "CHANGELOG.md")
		content, err := os.ReadFile(changelogPath)
		require.NoError(t, err)

		assert.Contains(t, string(content), "## [v1.1.0]")
		assert.Contains(t, string(content), "2025-01-15")
		assert.Contains(t, string(content), "This release includes 2 new features")
		assert.Contains(t, string(content), "Improved performance")
		mockGit.AssertExpectations(t)
	})

	t.Run("Prepend to existing changelog", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(origDir); err != nil {
				t.Fatal(err)
			}
		}()
		changelogPath := filepath.Join(dir, "CHANGELOG.md")
		existingContent := `# Changelog

## [v1.0.0] - 2025-01-01

Initial release
`
		err := os.WriteFile(changelogPath, []byte(existingContent), 0644)
		require.NoError(t, err)

		err = os.Chdir(dir)
		require.NoError(t, err)

		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit)

		release := &models.Release{
			Version:         "v1.1.0",
			PreviousVersion: "v1.0.0",
		}

		notes := &models.ReleaseNotes{
			Title:   "Version 1.1.0",
			Summary: "Bug fixes and improvements",
		}

		mockGit.On("GetTagDate", mock.Anything, "v1.1.0").Return("2025-01-15", nil)
		mockGit.On("GetRepoInfo", mock.Anything).Return("", "", "", errors.New("not a repo"))

		err = service.UpdateLocalChangelog(release, notes)

		assert.NoError(t, err)

		content, err := os.ReadFile(changelogPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "## [v1.1.0]")
		assert.Contains(t, contentStr, "## [v1.0.0]")
		v110Pos := strings.Index(contentStr, "## [v1.1.0]")
		v100Pos := strings.Index(contentStr, "## [v1.0.0]")
		assert.Less(t, v110Pos, v100Pos, "New version should come before old version")
		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_FindVersionFile_OptimizationAndValidation(t *testing.T) {
	t.Run("Should ignore invalid semver versions", func(t *testing.T) {
		dir := t.TempDir()
		versionFile := filepath.Join(dir, "version.go")
		content := `package main
const Version = "dev"
`
		err := os.WriteFile(versionFile, []byte(content), 0644)
		require.NoError(t, err)

		service := &ReleaseService{}

		cwd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				t.Fatal(err)
			}
		}()
		_ = os.Chdir(dir)

		_ = os.WriteFile("go.mod", []byte("module test"), 0644)

		foundFile, pattern, err := service.FindVersionFile(context.Background())
		assert.Error(t, err)
		assert.Equal(t, "", foundFile)
		assert.Equal(t, "", pattern)
	})

	t.Run("Should ignore node_modules", func(t *testing.T) {
		dir := t.TempDir()
		cwd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				t.Fatal(err)
			}
		}()
		_ = os.Chdir(dir)

		_ = os.WriteFile("package.json", []byte(`{"name":"test"}`), 0644)

		nodeModules := filepath.Join(dir, "node_modules")
		_ = os.Mkdir(nodeModules, 0755)

		ignoredFile := filepath.Join(nodeModules, "package.json")
		_ = os.WriteFile(ignoredFile, []byte(`{"version": "1.0.0"}`), 0644)

		service := &ReleaseService{}
		foundFile, _, err := service.FindVersionFile(context.Background())

		assert.Error(t, err)
		assert.NotContains(t, foundFile, "node_modules")
	})

	t.Run("Should respect max recursion depth", func(t *testing.T) {
		dir := t.TempDir()
		cwd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				t.Fatal(err)
			}
		}()
		_ = os.Chdir(dir)
		_ = os.WriteFile("go.mod", []byte("module test"), 0644)

		deepDir := filepath.Join(dir, "1", "2", "3", "4", "5", "6")
		_ = os.MkdirAll(deepDir, 0755)

		versionFile := filepath.Join(deepDir, "version.go")
		_ = os.WriteFile(versionFile, []byte(`package main
const Version = "1.0.0"`), 0644)

		service := &ReleaseService{}
		foundFile, _, err := service.FindVersionFile(context.Background())

		assert.Error(t, err)
		assert.Equal(t, "", foundFile)
	})

	t.Run("Should find file within recursion depth", func(t *testing.T) {
		dir := t.TempDir()
		cwd, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				t.Fatal(err)
			}
		}()
		_ = os.Chdir(dir)
		_ = os.WriteFile("go.mod", []byte("module test"), 0644)

		shallowDir := filepath.Join(dir, "internal")
		_ = os.MkdirAll(shallowDir, 0755)

		versionFile := filepath.Join(shallowDir, "version.go")
		_ = os.WriteFile(versionFile, []byte(`package main
const Version = "1.0.0"`), 0644)

		service := &ReleaseService{}
		foundFile, _, err := service.FindVersionFile(context.Background())

		assert.NoError(t, err)
		assert.Contains(t, foundFile, "version.go")
	})
}
