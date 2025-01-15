package services

import (
	"context"
	"errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type (
	MockGitService struct {
		mock.Mock
	}
	MockAIProvider struct {
		mock.Mock
	}
)

func (m *MockGitService) HasStagedChanges() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGitService) GetChangedFiles() ([]models.GitChange, error) {
	args := m.Called()
	return args.Get(0).([]models.GitChange), args.Error(1)
}

func (m *MockGitService) GetDiff() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockGitService) StageAllChanges() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockGitService) CreateCommit(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockAIProvider) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	args := m.Called(ctx, info, count)
	return args.Get(0).([]models.CommitSuggestion), args.Error(1)
}

func TestCommitService_GenerateSuggestions(t *testing.T) {
	t.Run("successful generation", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)

		changes := []models.GitChange{{
			Path:   "file1.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "feat: add new feature",
			Files:       []string{"file1.go", "file2.py"},
			Explanation: "some explanation",
		}}

		expectedInfo := models.CommitInfo{
			Files:  []string{"file1.go"},
			Diff:   "some diff",
			Format: "conventional",
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).
			Return([]models.CommitSuggestion{{
				CommitTitle: "feat: add new feature",
				Files:       []string{"file1.go", "file2.py"},
				Explanation: "some explanation",
			}}, nil)

		service := NewCommitService(mockGit, mockAI)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, "conventional")

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("no changes detected", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)

		mockGit.On("GetChangedFiles").Return([]models.GitChange{}, nil)

		service := NewCommitService(mockGit, mockAI)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, "conventional")

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)

		mockGit.AssertExpectations(t)
	})

	t.Run("error getting diff", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)

		changes := []models.GitChange{
			{
				Path:   "file1.go",
				Status: "M",
			},
		}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("", errors.New("git error"))

		service := NewCommitService(mockGit, mockAI)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, "conventional")

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)

		mockGit.AssertExpectations(t)
	})
}

func TestGitService_HasStagedChanges(t *testing.T) {
	t.Run("has staged changes", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockGit.On("HasStagedChanges").Return(true)

		// act
		result := mockGit.HasStagedChanges()

		// assert
		assert.True(t, result)
	})

	t.Run("no staged changes", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockGit.On("HasStagedChanges").Return(false)

		// act
		result := mockGit.HasStagedChanges()

		// assert
		assert.False(t, result)
	})
}
