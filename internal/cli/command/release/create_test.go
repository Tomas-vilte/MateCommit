package release

import (
	"bufio"
	"bytes"
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

func runCreateTest(t *testing.T, userInput string, args []string, mockService *MockReleaseService) error {
	trans, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
	if err != nil {
		t.Logf("Advertencia: usando traducciones vacías por error: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"c"},
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "auto", Aliases: []string{"y"}},
					&cli.StringFlag{Name: "version", Aliases: []string{"v"}},
					&cli.BoolFlag{Name: "publish"},
					&cli.BoolFlag{Name: "draft"},
					&cli.BoolFlag{Name: "changelog"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					reader := bufio.NewReader(bytes.NewBufferString(userInput))
					return createReleaseAction(mockService, trans, reader, nil)(ctx, c)
				},
			},
		},
	}

	fullArgs := append([]string{"matecommit", "create"}, args...)
	return app.Run(context.Background(), fullArgs)
}

func TestCreateCommand_Success(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.1.0",
		PreviousVersion: "v1.0.0",
		VersionBump:     "minor",
		Features:        []models.ReleaseItem{{Description: "feat: new thing"}},
	}

	notes := &models.ReleaseNotes{
		Title:   "Release v1.1.0",
		Summary: "Summary of release",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.1.0", "Release v1.1.0\n\nSummary of release").Return(nil)

	err := runCreateTest(t, "yes\n", []string{}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestCreateCommand_WithVersionOverride(t *testing.T) {
	mockService := new(MockReleaseService)

	release := &models.Release{
		Version:         "v1.1.0",
		PreviousVersion: "v1.0.0",
	}

	notes := &models.ReleaseNotes{
		Title: "Release v2.0.0",
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, mock.MatchedBy(func(r *models.Release) bool {
		return r.Version == "v2.0.0"
	})).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v2.0.0", mock.Anything).Return(nil)

	err := runCreateTest(t, "y\n", []string{"--version", "v2.0.0"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestCreateCommand_AutoConfirm(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.1"}
	notes := &models.ReleaseNotes{Title: "Fix"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.0.1", mock.Anything).Return(nil)

	err := runCreateTest(t, "", []string{"--auto"}, mockService)
	assert.NoError(t, err)
}

func TestCreateCommand_Cancelled(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.1"}
	notes := &models.ReleaseNotes{Title: "Fix"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	err := runCreateTest(t, "no\n", []string{}, mockService)
	assert.NoError(t, err)

	mockService.AssertNotCalled(t, "CreateTag")
}

func TestCreateCommand_AnalyzeError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockService.On("AnalyzeNextRelease", mock.Anything).Return((*models.Release)(nil), errors.New("git error"))
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)

	err := runCreateTest(t, "", []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git error")
}

func TestCreateCommand_GenerateNotesError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return((*models.ReleaseNotes)(nil), errors.New("ai error"))

	err := runCreateTest(t, "", []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "ai error")
}

func TestCreateCommand_CreateTagError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.0.0", mock.Anything).Return(errors.New("tag error"))

	err := runCreateTest(t, "y\n", []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "tag error")
}

func TestCreateCommand_WithPublish(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.0.0", mock.Anything).Return(nil)

	// Expect PublishRelease
	mockService.On("PublishRelease", mock.Anything, release, notes, false).Return(nil)

	err := runCreateTest(t, "y\n", []string{"--publish"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestCreateCommand_WithPublishDraft(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.0.0", mock.Anything).Return(nil)

	// Expect PublishRelease with draft=true
	mockService.On("PublishRelease", mock.Anything, release, notes, true).Return(nil)

	err := runCreateTest(t, "y\n", []string{"--publish", "--draft"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestCreateCommand_PublishError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)
	mockService.On("CreateTag", mock.Anything, "v1.0.0", mock.Anything).Return(nil)

	// Expect PublishRelease error
	mockService.On("PublishRelease", mock.Anything, release, notes, false).Return(errors.New("publish error"))

	err := runCreateTest(t, "y\n", []string{"--publish"}, mockService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish error")
}

func TestCreateCommand_WithChangelog(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)

	mockService.On("UpdateLocalChangelog", release, notes).Return(nil)
	mockService.On("UpdateAppVersion", "v1.0.0").Return(nil)
	mockService.On("CommitChangelog", mock.Anything, "v1.0.0").Return(nil)

	mockService.On("CreateTag", mock.Anything, "v1.0.0", mock.Anything).Return(nil)

	err := runCreateTest(t, "y\n", []string{"--changelog"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}
