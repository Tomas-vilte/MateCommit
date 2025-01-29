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

	MockJiraService struct {
		mock.Mock
	}
)

func (m *MockJiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	args := m.Called(ticketID)
	return args.Get(0).(*models.TicketInfo), args.Error(1)
}

func (m *MockGitService) HasStagedChanges() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGitService) AddFileToStaging(file string) error {
	args := m.Called(file)
	return args.Error(0)
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

func (m *MockGitService) GetCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockAIProvider) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
	args := m.Called(ctx, info, count)
	return args.Get(0).([]models.CommitSuggestion), args.Error(1)
}

func TestCommitService_GenerateSuggestions(t *testing.T) {
	t.Run("successful generation with ticket info", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		mockGit.On("GetCurrentBranch").Return("feature/PROJ-1234-user-authentication", nil)

		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-1234",
			Title:       "Implement user authentication",
			Description: "As a user, I want to log in to the system so that I can access my account.",
			Criteria:    []string{"User can log in with valid credentials", "User cannot log in with invalid credentials"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-1234").Return(ticketInfo, nil)

		changes := []models.GitChange{{
			Path:   "file1.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "feat: implement user authentication",
			Files:       []string{"file1.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file1.go"},
			Diff:        "some diff",
			TicketTitle: "Implement user authentication",
			TicketDesc:  "As a user, I want to log in to the system so that I can access my account.",
			Criteria:    []string{"User can log in with valid credentials", "User cannot log in with invalid credentials"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})

	t.Run("no changes detected", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		mockGit.On("GetCurrentBranch").Return("feature/PROJ-1234-user-authentication", nil)
		mockJiraService.On("GetTicketInfo", "PROJ-1234").Return(&models.TicketInfo{}, nil)
		mockGit.On("GetChangedFiles").Return([]models.GitChange{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)

		mockGit.AssertExpectations(t)
	})

	t.Run("error getting diff", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		changes := []models.GitChange{
			{
				Path:   "file1.go",
				Status: "M",
			},
		}
		mockGit.On("GetCurrentBranch").Return("feature/PROJ-1234-user-authentication", nil)
		mockJiraService.On("GetTicketInfo", "PROJ-1234").Return(&models.TicketInfo{}, nil)
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("", errors.New("git error"))

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)

		mockGit.AssertExpectations(t)
	})

	t.Run("branch without ticket ID", func(t *testing.T) {
		// arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Simular una branch sin ID de ticket
		mockGit.On("GetCurrentBranch").Return("main", nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

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

func TestCommitService_GenerateSuggestions_DifferentBranchNames(t *testing.T) {
	t.Run("branch with feature prefix and ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("feature/PROJ-1234-user-authentication", nil)

		// Configurar el mock para manejar la llamada a GetTicketInfo
		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-1234",
			Title:       "Implement user authentication",
			Description: "As a user, I want to log in to the system so that I can access my account.",
			Criteria:    []string{"User can log in with valid credentials", "User cannot log in with invalid credentials"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-1234").Return(ticketInfo, nil)

		// Simular la obtención de los archivos modificados y el diff
		changes := []models.GitChange{{
			Path:   "file1.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		// Simular la generación de sugerencias de commit
		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "feat: implement user authentication",
			Files:       []string{"file1.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file1.go"},
			Diff:        "some diff",
			TicketTitle: "Implement user authentication",
			TicketDesc:  "As a user, I want to log in to the system so that I can access my account.",
			Criteria:    []string{"User can log in with valid credentials", "User cannot log in with invalid credentials"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})

	t.Run("branch with bugfix prefix and ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("bugfix/PROJ-5678-fix-login", nil)

		// Configurar el mock para manejar la llamada a GetTicketInfo
		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-5678",
			Title:       "Fix login issue",
			Description: "As a user, I want to log in without errors so that I can access my account.",
			Criteria:    []string{"User can log in without errors", "Error messages are clear"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-5678").Return(ticketInfo, nil)

		// Simular la obtención de los archivos modificados y el diff
		changes := []models.GitChange{{
			Path:   "file2.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		// Simular la generación de sugerencias de commit
		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "fix: resolve login issue",
			Files:       []string{"file2.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file2.go"},
			Diff:        "some diff",
			TicketTitle: "Fix login issue",
			TicketDesc:  "As a user, I want to log in without errors so that I can access my account.",
			Criteria:    []string{"User can log in without errors", "Error messages are clear"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})

	t.Run("branch with hotfix prefix and ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("hotfix/PROJ-9999-critical-bug", nil)

		// Configurar el mock para manejar la llamada a GetTicketInfo
		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-9999",
			Title:       "Fix critical bug",
			Description: "As a user, I want the system to be stable so that I can use it without issues.",
			Criteria:    []string{"System should not crash", "Critical functionality should work"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-9999").Return(ticketInfo, nil)

		// Simular la obtención de los archivos modificados y el diff
		changes := []models.GitChange{{
			Path:   "file3.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		// Simular la generación de sugerencias de commit
		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "fix: resolve critical bug",
			Files:       []string{"file3.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file3.go"},
			Diff:        "some diff",
			TicketTitle: "Fix critical bug",
			TicketDesc:  "As a user, I want the system to be stable so that I can use it without issues.",
			Criteria:    []string{"System should not crash", "Critical functionality should work"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})

	t.Run("branch with release prefix and ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("release/PROJ-1000-final-release", nil)

		// Configurar el mock para manejar la llamada a GetTicketInfo
		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-1000",
			Title:       "Final release",
			Description: "As a user, I want the final version of the system so that I can use all features.",
			Criteria:    []string{"All features should work", "No known bugs"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-1000").Return(ticketInfo, nil)

		// Simular la obtención de los archivos modificados y el diff
		changes := []models.GitChange{{
			Path:   "file4.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		// Simular la generación de sugerencias de commit
		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "chore: prepare for final release",
			Files:       []string{"file4.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file4.go"},
			Diff:        "some diff",
			TicketTitle: "Final release",
			TicketDesc:  "As a user, I want the final version of the system so that I can use all features.",
			Criteria:    []string{"All features should work", "No known bugs"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})

	t.Run("branch without ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("main", nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, suggestions)

		mockGit.AssertExpectations(t)
	})

	t.Run("branch with custom prefix and ticket ID", func(t *testing.T) {
		// Arrange
		mockGit := new(MockGitService)
		mockAI := new(MockAIProvider)
		mockJiraService := new(MockJiraService)

		// Configurar el mock para manejar la llamada a GetCurrentBranch
		mockGit.On("GetCurrentBranch").Return("custom/PROJ-2000-custom-feature", nil)

		// Configurar el mock para manejar la llamada a GetTicketInfo
		ticketInfo := &models.TicketInfo{
			ID:          "PROJ-2000",
			Title:       "Custom feature",
			Description: "As a user, I want a custom feature so that I can do something specific.",
			Criteria:    []string{"Custom feature should work", "No side effects"},
		}
		mockJiraService.On("GetTicketInfo", "PROJ-2000").Return(ticketInfo, nil)

		// Simular la obtención de los archivos modificados y el diff
		changes := []models.GitChange{{
			Path:   "file5.go",
			Status: "M",
		}}
		mockGit.On("GetChangedFiles").Return(changes, nil)
		mockGit.On("GetDiff").Return("some diff", nil)

		// Simular la generación de sugerencias de commit
		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "feat: add custom feature",
			Files:       []string{"file5.go"},
			Explanation: "some explanation",
		}}
		expectedInfo := models.CommitInfo{
			Files:       []string{"file5.go"},
			Diff:        "some diff",
			TicketTitle: "Custom feature",
			TicketDesc:  "As a user, I want a custom feature so that I can do something specific.",
			Criteria:    []string{"Custom feature should work", "No side effects"},
		}
		mockAI.On("GenerateSuggestions", mock.Anything, expectedInfo, 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJiraService)

		// Act
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)

		mockGit.AssertExpectations(t)
		mockAI.AssertExpectations(t)
		mockJiraService.AssertExpectations(t)
	})
}
