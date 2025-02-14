package pr

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

type MockPRService struct {
	mock.Mock
}

func (m *MockPRService) SummarizePR(ctx context.Context, prNumber int) (models.PRSummary, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

func setupSummarizeTest(t *testing.T) (*MockPRService, *i18n.Translations, *config.Config) {
	mockPRService := new(MockPRService)

	cfg := &config.Config{
		ActiveVCSProvider: "github",
		VCSConfigs: map[string]config.VCSConfig{
			"github": {
				Owner: "testowner",
				Repo:  "testrepo",
			},
		},
	}

	translations, err := i18n.NewTranslations("es", "../../../i18n/locales")
	require.NoError(t, err)

	return mockPRService, translations, cfg
}

func TestSummarizeCommand(t *testing.T) {
	t.Run("should successfully summarize PR", func(t *testing.T) {
		// Arrange
		mockPRService, translations, cfg := setupSummarizeTest(t)

		prNumber := 123
		summary := models.PRSummary{
			Title: "Test PR",
		}

		mockPRService.On("SummarizePR", mock.Anything, prNumber).Return(summary, nil)

		cmd := NewSummarizeCommand(mockPRService).CreateCommand(translations, cfg)
		app := cmd

		ctx := context.Background()

		err := app.Run(ctx, []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.NoError(t, err)
		mockPRService.AssertExpectations(t)
	})

	t.Run("should fail when repo is not configured", func(t *testing.T) {
		// Arrange
		mockPRService, translations, cfg := setupSummarizeTest(t)
		cfg.ActiveVCSProvider = ""
		cfg.VCSConfigs = nil

		cmd := NewSummarizeCommand(mockPRService).CreateCommand(translations, cfg)
		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.no_repo_configured", 0, nil))
	})

	t.Run("should fail with invalid repo format", func(t *testing.T) {
		// Arrange
		mockPRService, translations, cfg := setupSummarizeTest(t)

		cmd := NewSummarizeCommand(mockPRService).CreateCommand(translations, cfg)
		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"summarize-pr", "--pr-number", "123", "--repo", "invalid-format"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.invalid_repo_format", 0, nil))
	})

	t.Run("should fail when PR service returns error", func(t *testing.T) {
		// Arrange
		mockPRService, translations, cfg := setupSummarizeTest(t)

		prNumber := 123
		mockError := fmt.Errorf("service error")

		mockPRService.On("SummarizePR", mock.Anything, prNumber).Return(models.PRSummary{}, mockError)

		cmd := NewSummarizeCommand(mockPRService).CreateCommand(translations, cfg)
		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.pr_summary_error", 0, nil))
		mockPRService.AssertExpectations(t)
	})
}
