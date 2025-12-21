package suggests_commits

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/thomas-vilte/matecommit/internal/models"
)

type MockCommitService struct {
	mock.Mock
}

func (m *MockCommitService) GenerateSuggestions(ctx context.Context, count int, issueNumber int, progress func(models.ProgressEvent)) ([]models.CommitSuggestion, error) {
	args := m.Called(ctx, count, issueNumber, progress)
	return args.Get(0).([]models.CommitSuggestion), args.Error(1)
}

// Mock para CommitHandler
type MockCommitHandler struct {
	mock.Mock
}

func (m *MockCommitHandler) HandleSuggestions(ctx context.Context, suggestions []models.CommitSuggestion) error {
	args := m.Called(ctx, suggestions)
	return args.Error(0)
}

func setupTestEnv(t *testing.T) (*config.Config, *i18n.Translations, func()) {
	tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
	if err != nil {
		t.Fatal(err)
	}

	tmpConfigPath := filepath.Join(tmpDir, "config.json")
	cfg := &config.Config{
		PathFile:         tmpConfigPath,
		Language:         "es",
		UseEmoji:         true,
		SuggestionsCount: 3,
	}

	translations, err := i18n.NewTranslations("es", "../../i18n/locales")
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	return cfg, translations, cleanup
}

func TestSuggestCommand(t *testing.T) {
	t.Run("should successfully generate and handle suggestions", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupTestEnv(t)
		defer cleanup()

		mockService := new(MockCommitService)
		mockHandler := new(MockCommitHandler)
		ctx := context.Background()

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go", "file2.go"},
				Explanation: "Added new functionality",
			},
		}

		mockService.On("GenerateSuggestions", mock.Anything, cfg.SuggestionsCount, 0, mock.Anything).Return(suggestions, nil)
		mockHandler.On("HandleSuggestions", mock.Anything, suggestions).Return(nil)

		factory := NewSuggestCommandFactory(mockService, mockHandler)
		cmd := factory.CreateCommand(translations, cfg)

		// Act
		err := cmd.Run(ctx, []string{"suggest"})

		// Assert
		assert.NoError(t, err)
		mockService.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
	})

	t.Run("should fail with invalid count parameter", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupTestEnv(t)
		defer cleanup()

		mockService := new(MockCommitService)
		mockHandler := new(MockCommitHandler)

		factory := NewSuggestCommandFactory(mockService, mockHandler)
		cmd := factory.CreateCommand(translations, cfg)

		ctx := context.Background()

		// Act
		err := cmd.Run(ctx, []string{"suggest", "--count", "11"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("invalid_suggestions_count", 0, struct {
			Min int
			Max int
		}{1, 10}))
		mockService.AssertNotCalled(t, "GenerateSuggestions")
		mockHandler.AssertNotCalled(t, "HandleSuggestions")
	})

	t.Run("should respect custom language setting", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupTestEnv(t)
		defer cleanup()

		mockService := new(MockCommitService)
		mockHandler := new(MockCommitHandler)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}

		mockService.On("GenerateSuggestions", mock.Anything, cfg.SuggestionsCount, 0, mock.Anything).Return(suggestions, nil)
		mockHandler.On("HandleSuggestions", mock.Anything, suggestions).Return(nil)

		factory := NewSuggestCommandFactory(mockService, mockHandler)
		command := factory.CreateCommand(translations, cfg)

		ctx := context.Background()

		// Act
		err := command.Run(ctx, []string{"suggest", "--lang", "en", fmt.Sprintf("--count=%d", cfg.SuggestionsCount)})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "en", cfg.Language)
		mockService.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
	})

	t.Run("should respect emoji configuration", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupTestEnv(t)
		defer cleanup()

		mockService := new(MockCommitService)
		mockHandler := new(MockCommitHandler)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}

		mockService.On("GenerateSuggestions", mock.Anything, cfg.SuggestionsCount, 0, mock.Anything).Return(suggestions, nil)
		mockHandler.On("HandleSuggestions", mock.Anything, suggestions).Return(nil)

		factory := NewSuggestCommandFactory(mockService, mockHandler)
		command := factory.CreateCommand(translations, cfg)

		ctx := context.Background()

		// Act
		err := command.Run(ctx, []string{"suggest", "--no-emoji", fmt.Sprintf("--count=%d", cfg.SuggestionsCount)})

		// Assert
		assert.NoError(t, err)
		assert.False(t, cfg.UseEmoji)
		mockService.AssertExpectations(t)
		mockHandler.AssertExpectations(t)
	})

	t.Run("should handle service error", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupTestEnv(t)
		defer cleanup()

		mockService := new(MockCommitService)
		mockHandler := new(MockCommitHandler)

		expectedError := fmt.Errorf("service error")
		mockService.On("GenerateSuggestions", mock.Anything, cfg.SuggestionsCount, 0, mock.Anything).Return([]models.CommitSuggestion{}, expectedError)

		factory := NewSuggestCommandFactory(mockService, mockHandler)
		command := factory.CreateCommand(translations, cfg)

		ctx := context.Background()

		// Act
		err := command.Run(ctx, []string{"suggest", fmt.Sprintf("--count=%d", cfg.SuggestionsCount)})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("suggestion_generation_error", 0, struct{ Error error }{expectedError}))
		mockService.AssertExpectations(t)
		mockHandler.AssertNotCalled(t, "HandleSuggestions")
	})
}
