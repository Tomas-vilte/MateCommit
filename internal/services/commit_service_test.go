package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*MockGitService, *MockAIProvider, *MockJiraService, *MockVCSClient, *config.Config, *i18n.Translations) {
	mockGit := new(MockGitService)
	mockAI := new(MockAIProvider)
	mockJiraService := new(MockJiraService)
	mockVCS := new(MockVCSClient)
	cfgApp := &config.Config{UseTicket: true}
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	require.NoError(t, err)
	return mockGit, mockAI, mockJiraService, mockVCS, cfgApp, trans
}

func TestCommitService_GenerateSuggestions(t *testing.T) {
	t.Run("successful generation with ticket info", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("feature/PROJ-1234-user-authentication", nil)

		ticketInfo := &models.TicketInfo{
			TicketID:    "PROJ-1234",
			TicketTitle: "Implement user authentication",
			TitleDesc:   "As a user, I want to log in...",
			Criteria:    []string{"User can log in"},
		}
		mockJira.On("GetTicketInfo", "PROJ-1234").Return(ticketInfo, nil)

		changes := []models.GitChange{{Path: "file1.go", Status: "M"}}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("some diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockVCS.On("GetIssue", mock.Anything, 1234).Return(&models.Issue{Number: 1234, Title: "Issue Title"}, nil)

		cfg.VCSConfigs = map[string]config.VCSConfig{
			"github": {Token: "token"},
		}

		expectedResponse := []models.CommitSuggestion{{
			CommitTitle: "feat: implement user authentication",
			Files:       []string{"file1.go"},
			Explanation: "some explanation",
		}}

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.TicketInfo.TicketID == "PROJ-1234" && info.Diff == "some diff"
		}), 3).Return(expectedResponse, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)
		mockGit.AssertExpectations(t)
		mockJira.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("no changes detected", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "No hay cambios detectados")
	})

	t.Run("error getting diff", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		changes := []models.GitChange{{Path: "file1.go", Status: "M"}}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("", errors.New("git error"))

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "Error al obtener los cambios")
	})

	t.Run("no differences string (empty diff)", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		changes := []models.GitChange{{Path: "file1.go", Status: "M"}}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("", nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "No se detectaron diferencias en los archivos")
	})

	t.Run("error getting branch name", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("", errors.New("branch error"))

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "Error al obtener el nombre de la branch")
	})

	t.Run("branch without ticket ID", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "No se encontro un ID de ticket en el nombre de la branch")
	})

	t.Run("error getting ticket info", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("feat/PROJ-123", nil)
		mockJira.On("GetTicketInfo", "PROJ-123").Return(&models.TicketInfo{}, errors.New("jira error"))

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "Error al obtener informacion del ticket")
	})

	t.Run("AI service nil", func(t *testing.T) {
		mockGit, _, mockJira, mockVCS, cfg, trans := setupTest(t)
		service := NewCommitService(mockGit, nil, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "La IA no está configurada")
	})

	t.Run("Detect Issue from Commits - Error", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 5).Return("", errors.New("git log error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
	})

	t.Run("Detect Issue from Commits - Simple Pattern", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 5).Return("Just a commit #999", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "github", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{"github": {Token: "token"}}

		mockVCS.On("GetIssue", mock.Anything, 999).Return(&models.Issue{Number: 999}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 999
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
		mockVCS.AssertCalled(t, "GetIssue", mock.Anything, 999)
	})

	t.Run("GetOrCreateVCSClient - Error getting repo info", func(t *testing.T) {
		mockGit, mockAI, mockJira, _, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("", "", "", errors.New("repo error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, nil, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
	})

	t.Run("GetOrCreateVCSClient - Provider config not found", func(t *testing.T) {
		mockGit, mockAI, mockJira, _, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "gitlab", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{}

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, nil, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
	})

	t.Run("GetOrCreateVCSClient - Unsupported provider", func(t *testing.T) {
		mockGit, mockAI, mockJira, _, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "bitbucket", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{"bitbucket": {Token: "token"}}

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, nil, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
	})

}

func TestCommitService_GenerateSuggestionsWithIssue(t *testing.T) {
	t.Run("explicit issue number", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)

		mockVCS.On("GetIssue", mock.Anything, 100).Return(&models.Issue{Number: 100, Title: "Explicit Issue"}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 100
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		suggestions, err := service.GenerateSuggestionsWithIssue(context.Background(), 3, 100)

		assert.NoError(t, err)
		assert.NotNil(t, suggestions)
	})

	t.Run("issue fetch error", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f.go"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)

		mockVCS.On("GetIssue", mock.Anything, 100).Return(&models.Issue{}, errors.New("fetch error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		_, err := service.GenerateSuggestionsWithIssue(context.Background(), 3, 100)

		assert.NoError(t, err)
	})
}

func TestCommitService_IssueDetection(t *testing.T) {
	t.Run("detect from branch name", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg, trans := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123-fix", nil)
		mockGit.On("GetChangedFiles", mock.Anything).Return([]models.GitChange{{Path: "f"}}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("d", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return("history", nil)
		mockVCS.On("GetIssue", mock.Anything, 123).Return(&models.Issue{Number: 123}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 123
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, mockJira, mockVCS, cfg, trans)
		_, err := service.GenerateSuggestions(context.Background(), 3)
		assert.NoError(t, err)
		mockVCS.AssertCalled(t, "GetIssue", mock.Anything, 123)
	})
}
