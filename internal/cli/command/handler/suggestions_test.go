package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockGitService struct {
	mock.Mock
}

func (m *mockGitService) GetChangedFiles(ctx context.Context) ([]models.GitChange, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *mockGitService) GetDiff(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockGitService) HasStagedChanges(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *mockGitService) CreateCommit(ctx context.Context, message string) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *mockGitService) AddFileToStaging(ctx context.Context, file string) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *mockGitService) GetCurrentBranch(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockGitService) GetRepoInfo(ctx context.Context) (string, string, string, error) {
	args := m.Called(ctx)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *mockGitService) GetLastTag(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *mockGitService) GetCommitCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockGitService) GetCommitsSinceTag(ctx context.Context, tag string) ([]models.Commit, error) {
	args := m.Called(ctx, tag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Commit), args.Error(1)
}

func (m *mockGitService) GetCommitsBetweenTags(ctx context.Context, fromTag, toTag string) ([]models.Commit, error) {
	args := m.Called(ctx, fromTag, toTag)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Commit), args.Error(1)
}

func (m *mockGitService) GetTagDate(ctx context.Context, tag string) (string, error) {
	args := m.Called(ctx, tag)
	return args.String(0), args.Error(1)
}

func (m *mockGitService) CreateTag(ctx context.Context, version, message string) error {
	args := m.Called(ctx, version, message)
	return args.Error(0)
}

func (m *mockGitService) PushTag(ctx context.Context, version string) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

func (m *mockGitService) GetRecentCommitMessages(ctx context.Context, count int) (string, error) {
	args := m.Called(ctx, count)
	return args.String(0), args.Error(1)
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	if err := w.Close(); err != nil {
		panic("could not close pipe")
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		panic("could not capture output")
	}
	return buf.String()
}

func simulateInput(input string, f func()) {
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		if _, err := w.Write([]byte(input)); err != nil {
			panic("could not write to pipe")
		}
		if err := w.Close(); err != nil {
			panic("could not close pipe")
		}
	}()

	f()

	os.Stdin = old
}

func TestNewSuggestionHandler(t *testing.T) {
	t.Run("should create new suggestion handler", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)

		// Act
		handler := NewSuggestionHandler(mockGit, nil, translations)

		// Assert
		assert.NotNil(t, handler)
		assert.Equal(t, mockGit, handler.gitService)
		assert.Equal(t, translations, handler.t)
		assert.Nil(t, handler.vcsClient)
	})
}

func TestSuggestionHandler_DisplaySuggestions(t *testing.T) {
	t.Run("should display suggestions correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

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
}

func TestSuggestionHandler_HandleCommitSelection(t *testing.T) {
	t.Run("should handle valid selection", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		mockGit.On("AddFileToStaging", mock.Anything, "test.go").Return(nil)
		mockGit.On("CreateCommit", mock.Anything, "feat: test feature").Return(nil)

		// Act - simula: 1 (selección), n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("1\nn\nn\ny\n", func() {
			err = handler.handleCommitSelection(context.Background(), suggestions)
		})

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle operation canceled (selection 0)", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("0\n", func() {
			err = handler.handleCommitSelection(context.Background(), suggestions)
		})

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle invalid selection number", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("999\n", func() {
			err = handler.handleCommitSelection(context.Background(), suggestions)
		})

		// Assert
		assert.Error(t, err)
	})

	t.Run("should handle invalid input", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("invalid\n", func() {
			err = handler.handleCommitSelection(context.Background(), suggestions)
		})

		// Assert
		assert.Error(t, err)
	})
}

func TestSuggestionHandler_HandleSuggestions(t *testing.T) {
	t.Run("should process commit successfully", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go", "file2.go"},
				Explanation: "Added new functionality",
			},
		}

		mockGit.On("AddFileToStaging", mock.Anything, "file1.go").Return(nil)
		mockGit.On("AddFileToStaging", mock.Anything, "file2.go").Return(nil)
		mockGit.On("CreateCommit", mock.Anything, "feat: add new feature").Return(nil)

		// Act - simula: n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("n\nn\ny\n", func() {
			err = handler.processCommit(context.Background(), suggestions[0], mockGit)
		})

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle error when adding file to staging", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}

		expectedErr := errors.New("staging error")
		mockGit.On("AddFileToStaging", mock.Anything, "file1.go").Return(expectedErr)

		// act - simula: n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("n\nn\ny\n", func() {
			err = handler.processCommit(context.Background(), suggestions[0], mockGit)
		})

		// assert
		assert.Error(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle error when creating commit", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go"},
				Explanation: "Added new functionality",
			},
		}
		mockGit.On("AddFileToStaging", mock.Anything, "file1.go").Return(nil)
		expectedErr := errors.New("commit error")
		mockGit.On("CreateCommit", mock.Anything, "feat: add new feature").Return(expectedErr)

		// Act - simula: n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("n\nn\ny\n", func() {
			err = handler.processCommit(context.Background(), suggestions[0], mockGit)
		})

		// Assert
		assert.Error(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle cancel operation", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestions := []models.CommitSuggestion{
			{
				CommitTitle: "feat: test feature",
				Files:       []string{"test.go"},
				Explanation: "Test explanation",
			},
		}

		// Act
		simulateInput("0\n", func() {
			err = handler.HandleSuggestions(context.Background(), suggestions)
		})

		// Assert
		assert.NoError(t, err)
	})
}

func TestSuggestionHandler_ProcessCommit(t *testing.T) {
	t.Run("should trim commit title prefix correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestion := models.CommitSuggestion{
			CommitTitle: "Commit: feat: add new feature",
			Files:       []string{"file1.go"},
			Explanation: "Added new functionality",
		}

		mockGit.On("AddFileToStaging", mock.Anything, "file1.go").Return(nil)
		mockGit.On("CreateCommit", mock.Anything, "feat: add new feature").Return(nil)

		// Act - simula: n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("n\nn\ny\n", func() {
			err = handler.processCommit(context.Background(), suggestion, mockGit)
		})

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("should handle multiple files correctly", func(t *testing.T) {
		// Arrange
		mockGit := new(mockGitService)
		translations, err := i18n.NewTranslations("en", "../../../../internal/i18n/locales")
		assert.NoError(t, err)
		handler := NewSuggestionHandler(mockGit, nil, translations)

		suggestion := models.CommitSuggestion{
			CommitTitle: "feat: multi-file change",
			Files:       []string{"file1.go", "file2.go", "file3.go"},
			Explanation: "Multiple file changes",
		}

		for _, file := range suggestion.Files {
			mockGit.On("AddFileToStaging", mock.Anything, file).Return(nil)
		}
		mockGit.On("CreateCommit", mock.Anything, "feat: multi-file change").Return(nil)

		// Act - simula: n (no ver diff), n (no editar mensaje), y (confirmar commit)
		simulateInput("n\nn\ny\n", func() {
			err = handler.processCommit(context.Background(), suggestion, mockGit)
		})

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})
}
