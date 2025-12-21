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

func runPushTest(t *testing.T, args []string, mockService *MockReleaseService, mockGit *MockGitService) error {
	trans, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
	if err != nil {
		t.Logf("Warning: using empty translations: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "push",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "version"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					return pushReleaseAction(mockService, trans)(ctx, c)
				},
			},
		},
	}
	fullArgs := append([]string{"matecommit", "push"}, args...)
	return app.Run(context.Background(), fullArgs)
}

func TestPushCommand_Success(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)

	mockService.On("PushTag", mock.Anything, "v1.0.0").Return(nil)

	err := runPushTest(t, []string{"--version", "v1.0.0"}, mockService, mockGit)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPushCommand_Error(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)

	mockService.On("PushTag", mock.Anything, "v1.0.0").Return(errors.New("push failed"))

	err := runPushTest(t, []string{"--version", "v1.0.0"}, mockService, mockGit)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "push failed")

	mockService.AssertExpectations(t)
}

func TestPushCommand_AutoDetectVersion(t *testing.T) {
	mockService := new(MockReleaseService)
	mockGit := new(MockGitService)

	release := &models.Release{
		Version:         "v1.0.0",
		PreviousVersion: "v0.9.0",
		VersionBump:     models.MinorBump,
	}

	mockService.On("AnalyzeNextRelease", mock.Anything).Return(release, nil)
	mockService.On("PushTag", mock.Anything, "v1.0.0").Return(nil)

	err := runPushTest(t, []string{}, mockService, mockGit)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}
