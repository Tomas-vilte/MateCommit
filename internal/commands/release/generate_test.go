package release

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func runGenerateTest(t *testing.T, args []string, mockService *MockReleaseService) error {
	trans, err := i18n.NewTranslations("en", "../../i18n/locales")
	if err != nil {
		t.Logf("Warning: using empty translations: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "generate",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Value: "RELEASE_NOTES.md"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return generateReleaseAction(mockService, trans)(ctx, c)
				},
			},
		},
	}
	fullArgs := append([]string{"matecommit", "generate"}, args...)
	return app.Run(context.Background(), fullArgs)
}

func TestGenerateCommand_Success(t *testing.T) {
	mockService := new(MockReleaseService)
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "NOTES.md")

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
	}
	notes := &models.ReleaseNotes{
		Title:      "Release v1.0.0",
		Summary:    "Summary",
		Changelog:  "- change 1",
		Highlights: []string{"Highlight 1"},
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)

	err := runGenerateTest(t, []string{"--output", outputFile}, mockService)
	assert.NoError(t, err)

	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	sContent := string(content)

	assert.Contains(t, sContent, "# Release v1.0.0")
	assert.Contains(t, sContent, "Highlight 1")
	assert.Contains(t, sContent, "- change 1")

	mockService.AssertExpectations(t)
}

func TestGenerateCommand_AnalyzeError(t *testing.T) {
	mockService := new(MockReleaseService)
	mockService.On("AnalyzeNextRelease", mock.Anything).Return((*models.Release)(nil), errors.New("git error"))
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)

	err := runGenerateTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "git error")
}

func TestGenerateCommand_GenerateError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return((*models.ReleaseNotes)(nil), errors.New("ai error"))

	err := runGenerateTest(t, []string{}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "ai error")
}

func TestGenerateCommand_WriteError(t *testing.T) {
	mockService := new(MockReleaseService)
	release := &models.Release{Version: "v1.0.0"}
	notes := &models.ReleaseNotes{Title: "Title"}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("EnrichReleaseContext", mock.Anything, mock.Anything).Return(nil)
	mockService.On("GenerateReleaseNotes", mock.Anything, release).Return(notes, nil)

	invalidPath := "/path/to/non/existent/dir/file.md"

	err := runGenerateTest(t, []string{"--output", invalidPath}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "Error writing file")
}
