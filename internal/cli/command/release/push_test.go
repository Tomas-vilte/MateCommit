package release

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func runPushTest(t *testing.T, args []string, mockService *MockReleaseService) error {
	trans, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
	if err != nil {
		t.Logf("Advertencia: usando traducciones vacías: %v", err)
		trans = &i18n.Translations{}
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			{
				Name: "push",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "version", Required: true},
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

	mockService.On("PushTag", mock.Anything, "v1.0.0").Return(nil)

	err := runPushTest(t, []string{"--version", "v1.0.0"}, mockService)
	assert.NoError(t, err)

	mockService.AssertExpectations(t)
}

func TestPushCommand_Error(t *testing.T) {
	mockService := new(MockReleaseService)

	mockService.On("PushTag", mock.Anything, "v1.0.0").Return(errors.New("push failed"))

	err := runPushTest(t, []string{"--version", "v1.0.0"}, mockService)
	assert.Error(t, err)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "push failed")

	mockService.AssertExpectations(t)
}
