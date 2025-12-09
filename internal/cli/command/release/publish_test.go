package release

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func runPublishTest(t *testing.T, args []string, mockService *MockReleaseService) error {
	trans, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
	if err != nil {
		t.Logf("Warning: using empty translations: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "publish",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "version"},
					&cli.BoolFlag{Name: "draft"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return publishReleaseAction(mockService, trans)(ctx, c)
				},
			},
		},
	}
	fullArgs := append([]string{"matecommit", "publish"}, args...)
	return app.Run(context.Background(), fullArgs)
}

func TestPublishCommand_Success(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	notes := &models.ReleaseNotes{
		Title:   "Version 1.0.0",
		Summary: "Major release",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("PublishRelease", mock.Anything, release, notes, false).Return(nil)

	err := runPublishTest(t, []string{}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPublishCommand_WithDraftFlag(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	notes := &models.ReleaseNotes{
		Title:   "Version 1.0.0",
		Summary: "Major release",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("PublishRelease", mock.Anything, release, notes, true).Return(nil)

	err := runPublishTest(t, []string{"--draft"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPublishCommand_WithVersionOverride(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	notes := &models.ReleaseNotes{
		Title:   "Version 2.0.0",
		Summary: "Major release",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
		return r.Version == "v2.0.0"
	})).Return(notes, nil)
	mockService.On("PublishRelease", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
		return r.Version == "v2.0.0"
	}), notes, false).Return(nil)

	err := runPublishTest(t, []string{"--version", "v2.0.0"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPublishCommand_AnalyzeError(t *testing.T) {
	mockService := new(MockReleaseService)

	mockService.On("AnalyzeNextRelease", mock.Anything).Return((*models.Release)(nil), errors.New("analyze failed"))

	err := runPublishTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	mockService.AssertExpectations(t)
}

func TestPublishCommand_GenerateNotesError(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return((*models.ReleaseNotes)(nil), errors.New("generate failed"))

	err := runPublishTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	mockService.AssertExpectations(t)
}

func TestPublishCommand_PublishError(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	notes := &models.ReleaseNotes{
		Title:   "Version 1.0.0",
		Summary: "Major release",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("PublishRelease", mock.Anything, release, notes, false).Return(errors.New("publish failed"))

	err := runPublishTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "publish failed")

	mockService.AssertExpectations(t)
}
