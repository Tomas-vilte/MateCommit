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

func runPreviewTest(t *testing.T, args []string, mockService *MockReleaseService) error {
	trans, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
	if err != nil {
		t.Logf("Warning: using empty translations: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "preview",
				Action: func(ctx context.Context, c *cli.Command) error {
					return previewReleaseAction(mockService, trans)(ctx, c)
				},
			},
		},
	}
	fullArgs := append([]string{"matecommit", "preview"}, args...)
	return app.Run(context.Background(), fullArgs)
}

func TestPreviewCommand_Success(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		Breaking:        []models.ReleaseItem{{Description: "break"}},
		Features:        []models.ReleaseItem{{Description: "feat"}},
	}
	notes := &models.ReleaseNotes{
		Title:      "Release v1.0.0",
		Summary:    "Summary",
		Changelog:  "- change 1",
		Highlights: []string{"highlight"},
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)

	err := runPreviewTest(t, []string{}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPreviewCommand_AnalyzeError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockService.On("AnalyzeNextRelease", mock.Anything).Return((*models.Release)(nil), errors.New("git error"))
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)

	err := runPreviewTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "git error")
}

func TestPreviewCommand_GenerateError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return((*models.ReleaseNotes)(nil), errors.New("ai error"))

	err := runPreviewTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "ai error")
}
