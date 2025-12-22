package issues

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/urfave/cli/v3"
)

func setupIssuesTest(t *testing.T) (*MockIssueGeneratorService, *MockIssueTemplateService, IssueServiceProvider, *i18n.Translations, *config.Config) {
	mockGen := &MockIssueGeneratorService{}
	mockTemp := &MockIssueTemplateService{}

	provider := func(ctx context.Context) (IssueGeneratorService, error) {
		return mockGen, nil
	}

	trans, err := i18n.NewTranslations("en", "../../i18n/locales")
	require.NoError(t, err)
	cfg := &config.Config{Language: "en"}
	return mockGen, mockTemp, provider, trans, cfg
}

func withStdin(input string, f func()) {
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	go func() {
		defer func() {
			if err := w.Close(); err != nil {
				panic(err)
			}
		}()
		_, _ = w.Write([]byte(input))
	}()
	f()
	os.Stdin = origStdin
}

func TestIssueGenerateAction(t *testing.T) {
	t.Run("should fail if no input provided", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "generate"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "You must specify either --from-diff or --description")
	})

	t.Run("should fail if multiple sources provided", func(t *testing.T) {
		_, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "generate", "--from-diff", "--from-pr", "123"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "You can only specify one of: --from-diff, --from-pr, or --description")
	})

	t.Run("should generate from diff successfully", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Test Issue",
			Description: "Test Description",
			Labels:      []string{"bug"},
		}

		mockGen.On("GenerateFromDiff", mock.Anything, "hint", false).Return(expectedResult, nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string(nil)).Return(&models.Issue{Number: 1, URL: "http://test.com"}, nil)

		withStdin("y\n", func() {
			app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
			err := app.Run(context.Background(), []string{"test", "issue", "generate", "--from-diff", "--hint", "hint"})
			assert.NoError(t, err)
		})

		mockGen.AssertExpectations(t)
	})

	t.Run("should handle dry-run correctly", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		expectedResult := &models.IssueGenerationResult{
			Title:       "Dry Run Issue",
			Description: "Dry Run Desc",
		}

		mockGen.On("GenerateFromDescription", mock.Anything, "desc", false).Return(expectedResult, nil)

		app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
		err := app.Run(context.Background(), []string{"test", "issue", "generate", "--description", "desc", "--dry-run"})

		assert.NoError(t, err)
		mockGen.AssertExpectations(t)
		mockGen.AssertNotCalled(t, "CreateIssue", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should handle assign-me flag", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		expectedResult := &models.IssueGenerationResult{
			Title: "Assigned Issue",
		}

		mockGen.On("GenerateFromDiff", mock.Anything, "", false).Return(expectedResult, nil)
		mockGen.On("GetAuthenticatedUser", mock.Anything).Return("test-user", nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string{"test-user"}).Return(&models.Issue{Number: 1}, nil)

		withStdin("y\n", func() {
			app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
			err := app.Run(context.Background(), []string{"test", "issue", "generate", "--from-diff", "--assign-me"})
			assert.NoError(t, err)
		})

		mockGen.AssertExpectations(t)
	})

	t.Run("should cancel if user chooses no", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		mockGen.On("GenerateFromDiff", mock.Anything, "", false).Return(&models.IssueGenerationResult{Title: "T"}, nil)

		withStdin("n\n", func() {
			app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
			err := app.Run(context.Background(), []string{"test", "issue", "generate", "--from-diff"})
			assert.NoError(t, err)
		})

		mockGen.AssertNotCalled(t, "CreateIssue", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("should use template if specified", func(t *testing.T) {
		mockGen, mockTemp, provider, trans, cfg := setupIssuesTest(t)
		factory := NewIssuesCommandFactory(provider, mockTemp)
		cmd := factory.CreateCommand(trans, cfg)

		expectedResult := &models.IssueGenerationResult{Title: "Template Issue"}

		mockGen.On("GenerateWithTemplate", mock.Anything, "bug", "", true, "", false).Return(expectedResult, nil)
		mockGen.On("CreateIssue", mock.Anything, expectedResult, []string(nil)).Return(&models.Issue{Number: 1}, nil)

		withStdin("y\n", func() {
			app := &cli.Command{Name: "test", Commands: []*cli.Command{cmd}}
			err := app.Run(context.Background(), []string{"test", "issue", "generate", "--from-diff", "--template", "bug"})
			assert.NoError(t, err)
		})

		mockGen.AssertExpectations(t)
	})
}
