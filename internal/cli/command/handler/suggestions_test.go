package handler

import (
	"bytes"
	"errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"os"
	"testing"
)

type mockGitService struct {
	mock.Mock
}

func (m *mockGitService) GetChangedFiles() ([]models.GitChange, error) {
	args := m.Called()
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *mockGitService) GetDiff() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *mockGitService) HasStagedChanges() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockGitService) CreateCommit(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *mockGitService) AddFileToStaging(file string) error {
	args := m.Called(file)
	return args.Error(0)
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func simulateInput(input string, f func()) {
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		w.Write([]byte(input))
		w.Close()
	}()

	f()

	os.Stdin = old
}

func TestNewSuggestionHandler(t *testing.T) {
	t.Run("should create new suggestion handler", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)

		// Act
		handler := NewSuggestionHandler(mockGit, translations)

		// Assert
		assert.NotNil(t, handler)
		assert.Equal(t, mockGit, handler.gitService)
		assert.Equal(t, translations, handler.t)
	})
}

func TestSuggestionHandler_DisplaySuggestions(t *testing.T) {
	t.Run("should display suggestions correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: first feature",
				Files:       []string{"file1.go", "file2.go"},
				Explanation: "First explanation",
			},
			{
				CommitTitle: "fix: bug fix",
				Files:       []string{"file3.go"},
				Explanation: "Second explanation",
			},
		}

		// Act
		output := captureOutput(func() {
			handler.displaySuggestions(suggestions)
		})

		// Assert
		assert.Contains(t, output, "feat: first feature")
		assert.Contains(t, output, "file1.go")
		assert.Contains(t, output, "file2.go")
		assert.Contains(t, output, "First explanation")
		assert.Contains(t, output, "fix: bug fix")
		assert.Contains(t, output, "file3.go")
		assert.Contains(t, output, "Second explanation")
	})

	t.Run("should display empty suggestions list", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		var suggestions []models.CommitSuggestion

		// Act
		output := captureOutput(func() {
			handler.displaySuggestions(suggestions)
		})

		// Assert
		assert.Contains(t, output, "━━━━━━━━━━━━━━━━━━━━━━━")
	})
}

func TestSuggestionHandler_HandleCommitSelection(t *testing.T) {
	t.Run("should handle valid selection", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		mockGit.On("AddFileToStaging", "test.go").Return(nil)
		mockGit.On("CreateCommit", "feat: test feature").Return(nil)

		// Act
		simulateInput("1\n", func() {
			err = handler.handleCommitSelection(suggestions)
		})

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle operation canceled (selection 0)", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("0\n", func() {
			err = handler.handleCommitSelection(suggestions)
		})

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle invalid selection number", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("999\n", func() {
			err = handler.handleCommitSelection(suggestions)
		})

		// Assert
		assert.Error(t, err)
	})

	t.Run("should handle invalid input", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("invalid\n", func() {
			err = handler.handleCommitSelection(suggestions)
		})

		// Assert
		assert.Error(t, err)
	})
}

func TestSuggestionHandler_HandleSuggestions(t *testing.T) {
	t.Run("should process commit successfully", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go", "file2.go"},
				Explanation: "Added new functionality",
			},
		}

		mockGit.On("AddFileToStaging", "file1.go").Return(nil)
		mockGit.On("AddFileToStaging", "file2.go").Return(nil)
		mockGit.On("CreateCommit", "feat: add new feature").Return(nil)

		// Act
		err = handler.processCommit(suggestions[0], mockGit)

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle error when adding file to staging", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}

		expectedErr := errors.New("staging error")
		mockGit.On("AddFileToStaging", "file1.go").Return(expectedErr)

		// act
		err = handler.processCommit(suggestions[0], mockGit)

		// assert
		assert.Error(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle error when creating commit", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}
		mockGit.On("AddFileToStaging", "file1.go").Return(nil)
		expectedErr := errors.New("commit error")
		mockGit.On("CreateCommit", "feat: add new feature").Return(expectedErr)

		// Act
		err = handler.processCommit(suggestions[0], mockGit)

		// Assert
		assert.Error(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle cancel operation", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("0\n", func() {
			err = handler.HandleSuggestions(suggestions)
		})

		// Assert
		assert.NoError(t, err)
	})
}

func TestSuggestionHandler_ProcessCommit(t *testing.T) {
	t.Run("should trim commit title prefix correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestion := models.CommitSuggestion{
			CommitTitle: "Commit: feat: add new feature",
			Files:       []string{"file1.go"},
			Explanation: "Added new functionality",
		}

		mockGit.On("AddFileToStaging", "file1.go").Return(nil)
		mockGit.On("CreateCommit", "feat: add new feature").Return(nil)

		// Act
		err = handler.processCommit(suggestion, mockGit)

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle multiple files correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, translations)

		suggestion := models.CommitSuggestion{
			CommitTitle: "feat: multi-file change",
			Files:       []string{"file1.go", "file2.go", "file3.go"},
			Explanation: "Multiple file changes",
		}

		for _, file := range suggestion.Files {
			mockGit.On("AddFileToStaging", file).Return(nil)
		}
		mockGit.On("CreateCommit", "feat: multi-file change").Return(nil)

		// Act
		err = handler.processCommit(suggestion, mockGit)

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})
}
