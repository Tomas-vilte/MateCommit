package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReleaseNotesGenerator struct {
	mock.Mock
}

func (m *MockReleaseNotesGenerator) GenerateNotes(ctx context.Context, release *models.Release) (*models.ReleaseNotes, error) {
	args := m.Called(ctx, release)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ReleaseNotes), args.Error(1)
}

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
