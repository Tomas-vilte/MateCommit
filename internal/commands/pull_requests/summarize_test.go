package pull_requests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
)

type MockPRService struct {
	mock.Mock
}

func (m *MockPRService) SummarizePR(ctx context.Context, prNumber int, hint string, progress func(models.ProgressEvent)) (models.PRSummary, error) {
	args := m.Called(ctx, prNumber, hint, progress)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

func setupSummarizeTest(t *testing.T) (*MockPRService, *i18n.Translations, *config.Config) {
	mockPRService := new(MockPRService)

	cfg := &config.Config{}

	translations, err := i18n.NewTranslations("en", "../../i18n/locales")
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

		mockPRService.On("SummarizePR", mock.Anything, prNumber, mock.Anything, mock.Anything).Return(summary, nil)

		prProvider := func(ctx context.Context) (PRService, error) {
			return mockPRService, nil
		}
		prCommand := NewSummarizeCommand(prProvider)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.NoError(t, err)

		mockPRService.AssertExpectations(t)
	})

	t.Run("should fail when factory returns error", func(t *testing.T) {
		// Arrange
		_, translations, cfg := setupSummarizeTest(t)

		mockError := fmt.Errorf("factory error")
		prProvider := func(ctx context.Context) (PRService, error) {
			return nil, mockError
		}
		prCommand := NewSummarizeCommand(prProvider)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.pr_service_creation_error", 0, nil))

	})

	t.Run("should fail when PR service returns error", func(t *testing.T) {
		// Arrange
		mockPRService, translations, cfg := setupSummarizeTest(t)

		prNumber := 123
		mockError := fmt.Errorf("service error")

		mockPRService.On("SummarizePR", mock.Anything, prNumber, mock.Anything, mock.Anything).Return(models.PRSummary{}, mockError)

		prProvider := func(ctx context.Context) (PRService, error) {
			return mockPRService, nil
		}
		prCommand := NewSummarizeCommand(prProvider)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.pr_summary_error", 0, nil))

		mockPRService.AssertExpectations(t)
	})
}
