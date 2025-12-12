package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReleaseService_AnalyzeNextRelease(t *testing.T) {
	t.Run("Success with existing tag and feature commits", func(t *testing.T) {
		mockGit := new(MockGitService)
		mockVCS := new(MockVCSClient)
		mockNotesGen := new(MockReleaseNotesGenerator)
		service := NewReleaseService(mockGit, mockVCS, mockNotesGen, nil)

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
		service := NewReleaseService(mockGit, nil, nil, nil)

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
		service := NewReleaseService(mockGit, nil, nil, nil)

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
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("GetLastTag", mock.Anything).Return("", errors.New("git error"))

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "error al obtener ultimo tag")

		mockGit.AssertExpectations(t)
	})

	t.Run("No changes since last tag", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return([]models.Commit{}, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "no hay commits nuevos")

		mockGit.AssertExpectations(t)
	})

	t.Run("Error getting commits", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("GetLastTag", mock.Anything).Return("v1.0.0", nil)
		mockGit.On("GetCommitsSinceTag", mock.Anything, "v1.0.0").Return(nil, errors.New("git error"))

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "error al obtener commits")

		mockGit.AssertExpectations(t)
	})

	t.Run("Empty repository", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("GetLastTag", mock.Anything).Return("", nil)
		mockGit.On("GetCommitCount", mock.Anything).Return(0, nil)

		release, err := service.AnalyzeNextRelease(context.Background())

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "no hay commits en el repositorio")

		mockGit.AssertExpectations(t)
	})
}

func TestReleaseService_GenerateReleaseNotes(t *testing.T) {
	t.Run("Use AI generator when available", func(t *testing.T) {
		mockNotesGen := new(MockReleaseNotesGenerator)
		service := NewReleaseService(nil, nil, mockNotesGen, nil)

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
		service := NewReleaseService(nil, nil, nil, nil) // No generator

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
		service := NewReleaseService(nil, nil, mockNotesGen, nil)

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
		service := NewReleaseService(nil, mockVCS, nil, nil)

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
		trans, err := i18n.NewTranslations("en", "../i18n/locales")
		if err != nil {
			trans, _ = i18n.NewTranslations("en", "../../i18n/locales")
		}
		service := NewReleaseService(nil, nil, nil, trans)

		release, err := service.GetRelease(context.Background(), "v1.2.0")

		assert.Error(t, err)
		assert.Nil(t, release)
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, mockVCS, nil, nil)

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
		service := NewReleaseService(nil, mockVCS, nil, nil)

		mockVCS.On("UpdateRelease", mock.Anything, "v1.2.0", "Updated release notes").Return(nil)

		err := service.UpdateRelease(context.Background(), "v1.2.0", "Updated release notes")

		assert.NoError(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		trans, err := i18n.NewTranslations("en", "../i18n/locales")
		if err != nil {
			trans, _ = i18n.NewTranslations("en", "../../i18n/locales")
		}
		service := NewReleaseService(nil, nil, nil, trans)

		err = service.UpdateRelease(context.Background(), "v1.2.0", "Updated notes")

		assert.Error(t, err)
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, mockVCS, nil, nil)

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
		service := NewReleaseService(nil, mockVCS, nil, nil)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{Title: "Release v1.0.0"}

		mockVCS.On("CreateRelease", mock.Anything, release, notes, false).Return(nil)

		err := service.PublishRelease(context.Background(), release, notes, false)

		assert.NoError(t, err)
		mockVCS.AssertExpectations(t)
	})

	t.Run("should return error if VCS client not configured", func(t *testing.T) {
		service := NewReleaseService(nil, nil, nil, nil)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		err := service.PublishRelease(context.Background(), release, notes, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cliente VCS no configurado")
	})

	t.Run("should propagate VCS client error", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, mockVCS, nil, nil)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		mockVCS.On("CreateRelease", mock.Anything, release, notes, true).Return(errors.New("publish failed"))

		err := service.PublishRelease(context.Background(), release, notes, true)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publish failed")
		mockVCS.AssertExpectations(t)
	})
}

func TestReleaseService_CreateTag(t *testing.T) {
	t.Run("should create tag successfully", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("CreateTag", mock.Anything, "v1.0.0", "Release v1.0.0").Return(nil)

		err := service.CreateTag(context.Background(), "v1.0.0", "Release v1.0.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should propagate git error", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

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
		service := NewReleaseService(mockGit, nil, nil, nil)

		mockGit.On("PushTag", mock.Anything, "v1.0.0").Return(nil)

		err := service.PushTag(context.Background(), "v1.0.0")

		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should propagate git error", func(t *testing.T) {
		mockGit := new(MockGitService)
		service := NewReleaseService(mockGit, nil, nil, nil)

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
		service := NewReleaseService(nil, mockVCS, nil, nil)

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
		trans, err := i18n.NewTranslations("en", "../i18n/locales")
		if err != nil {
			trans, _ = i18n.NewTranslations("en", "../../i18n/locales")
		}
		service := NewReleaseService(nil, nil, nil, trans)

		release := &models.Release{}

		err = service.EnrichReleaseContext(context.Background(), release)

		assert.Error(t, err)
	})

	t.Run("should continue even if some enrichments fail", func(t *testing.T) {
		mockVCS := new(MockVCSClient)
		service := NewReleaseService(nil, mockVCS, nil, nil)

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
