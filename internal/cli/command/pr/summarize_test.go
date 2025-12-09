package pr

import (
	"context"
	"fmt"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockPRService struct {
	mock.Mock
}

func (m *MockPRService) SummarizePR(ctx context.Context, prNumber int) (models.PRSummary, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

type MockPRServiceFactory struct {
	mock.Mock
}

func (m *MockPRServiceFactory) CreatePRService(ctx context.Context) (ports.PRService, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.PRService), args.Error(1)
}

func setupSummarizeTest(t *testing.T) (*MockPRService, *MockPRServiceFactory, *i18n.Translations, *config.Config) {
	mockPRService := new(MockPRService)
	mockFactory := new(MockPRServiceFactory)

	cfg := &config.Config{}

	translations, err := i18n.NewTranslations("es", "../../../i18n/locales")
	require.NoError(t, err)

	return mockPRService, mockFactory, translations, cfg
}

func TestSummarizeCommand(t *testing.T) {
	t.Run("should successfully summarize PR", func(t *testing.T) {
		// Arrange
		mockPRService, mockFactory, translations, cfg := setupSummarizeTest(t)

		prNumber := 123
		summary := models.PRSummary{
			Title: "Test PR",
		}

		mockFactory.On("CreatePRService", mock.Anything).Return(mockPRService, nil)
		mockPRService.On("SummarizePR", mock.Anything, prNumber).Return(summary, nil)

		prCommand := NewSummarizeCommand(mockFactory)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.NoError(t, err)
		mockFactory.AssertExpectations(t)
		mockPRService.AssertExpectations(t)
	})

	t.Run("should fail when factory returns error", func(t *testing.T) {
		// Arrange
		_, mockFactory, translations, cfg := setupSummarizeTest(t)

		mockError := fmt.Errorf("factory error")
		mockFactory.On("CreatePRService", mock.Anything).Return(nil, mockError)

		prCommand := NewSummarizeCommand(mockFactory)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.pr_service_creation_error", 0, nil))
		mockFactory.AssertExpectations(t)
	})

	t.Run("should fail when PR service returns error", func(t *testing.T) {
		// Arrange
		mockPRService, mockFactory, translations, cfg := setupSummarizeTest(t)

		prNumber := 123
		mockError := fmt.Errorf("service error")

		mockFactory.On("CreatePRService", mock.Anything).Return(mockPRService, nil)
		mockPRService.On("SummarizePR", mock.Anything, prNumber).Return(models.PRSummary{}, mockError)

		prCommand := NewSummarizeCommand(mockFactory)
		cmd := prCommand.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"summarize-pr", "--pr-number", "123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.pr_summary_error", 0, nil))
		mockFactory.AssertExpectations(t)
		mockPRService.AssertExpectations(t)
	})
}
