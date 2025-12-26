package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func setupTest(t *testing.T) (*MockGitService, *MockAIProvider, *MockJiraService, *MockVCSClient, *config.Config) {
	mockGit := new(MockGitService)
	mockAI := new(MockAIProvider)
	mockJiraService := new(MockJiraService)
	mockVCS := new(MockVCSClient)
	cfgApp := &config.Config{UseTicket: true}
	return mockGit, mockAI, mockJiraService, mockVCS, cfgApp
}

func TestCommitService_GenerateSuggestions(t *testing.T) {
	t.Run("successful generation with ticket info", func(t *testing.T) {
		mockGit, mockAI, mockJira, mockVCS, cfg := setupTest(t)

		mockGit.On("GetCurrentBranch", mock.Anything).Return("feature/PROJ-1234-user-authentication", nil)

		ticketInfo := &models.TicketInfo{
			TicketID:    "PROJ-1234",
			TicketTitle: "Implement user authentication",
			TitleDesc:   "As a user, I want to log in...",
			Criteria:    []string{"User can log in"},
		}
		mockJira.On("GetTicketInfo", "PROJ-1234").Return(ticketInfo, nil)

		changes := []string{"file1.go"}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("some diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
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

		service := NewCommitService(mockGit, mockAI,
			WithTicketManager(mockJira),
			WithVCSClient(mockVCS),
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse, suggestions)
		mockGit.AssertExpectations(t)
		mockJira.AssertExpectations(t)
		mockAI.AssertExpectations(t)
	})

	t.Run("no changes detected", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{}, nil)

		service := NewCommitService(mockGit, mockAI,
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.ErrorIs(t, err, domainErrors.ErrNoChanges)
	})

	t.Run("error getting diff", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)

		changes := []string{"file1.go"}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("", errors.New("git error"))

		service := NewCommitService(mockGit, mockAI,
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "GIT: error getting git diff")
	})

	t.Run("no differences string (empty diff)", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)

		changes := []string{"file1.go"}
		mockGit.On("GetChangedFiles", mock.Anything).Return(changes, nil)
		mockGit.On("GetDiff", mock.Anything).Return("", nil)

		service := NewCommitService(mockGit, mockAI,
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.ErrorIs(t, err, domainErrors.ErrNoChanges)
	})

	t.Run("error getting branch name", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("", errors.New("branch error"))

		service := NewCommitService(mockGit, mockAI,
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "GIT: error getting branch name")
	})

	t.Run("branch without ticket ID", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)

		service := NewCommitService(mockGit, mockAI,
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "GIT: ticket ID not found in branch")
	})

	t.Run("error getting ticket info", func(t *testing.T) {
		mockGit, mockAI, mockJira, _, cfg := setupTest(t)

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("feat/PROJ-123", nil)
		mockJira.On("GetTicketInfo", "PROJ-123").Return(&models.TicketInfo{}, errors.New("jira error"))

		service := NewCommitService(mockGit, mockAI,
			WithTicketManager(mockJira),
			WithConfig(cfg),
		)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})

		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.Contains(t, err.Error(), "INTERNAL: error getting ticket info")
	})

	t.Run("AI service nil", func(t *testing.T) {
		mockGit, _, _, _, _ := setupTest(t)
		service := NewCommitService(mockGit, nil)
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.Error(t, err)
		assert.Nil(t, suggestions)
		assert.ErrorIs(t, err, domainErrors.ErrAPIKeyMissing)
	})

	t.Run("Detect Issue from Commits - Error", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 5).Return([]string{}, errors.New("git log error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
	})

	t.Run("Detect Issue from Commits - Simple Pattern", func(t *testing.T) {
		mockGit, mockAI, _, mockVCS, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("main", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 5).Return([]string{"Just a commit fixes #999"}, nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "github", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{"github": {Token: "token"}}

		mockVCS.On("GetIssue", mock.Anything, 999).Return(&models.Issue{Number: 999}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 999
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithVCSClient(mockVCS), WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
		mockVCS.AssertCalled(t, "GetIssue", mock.Anything, 999)
	})

	t.Run("GetOrCreateVCSClient - Error getting repo info", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("", "", "", errors.New("repo error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
	})

	t.Run("GetOrCreateVCSClient - Provider config not found", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "gitlab", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{}

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
	})

	t.Run("GetOrCreateVCSClient - Unsupported provider", func(t *testing.T) {
		mockGit, mockAI, _, _, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123", nil)

		mockGit.On("GetRepoInfo", mock.Anything).Return("owner", "repo", "bitbucket", nil)
		cfg.VCSConfigs = map[string]config.VCSConfig{"bitbucket": {Token: "token"}}

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
	})

}

func TestCommitService_GenerateSuggestionsWithIssue(t *testing.T) {
	t.Run("explicit issue number", func(t *testing.T) {
		mockGit, mockAI, _, mockVCS, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)

		mockVCS.On("GetIssue", mock.Anything, 100).Return(&models.Issue{Number: 100, Title: "Explicit Issue"}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 100
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithVCSClient(mockVCS), WithConfig(cfg))
		suggestions, err := service.GenerateSuggestions(context.Background(), 3, 100, func(e models.ProgressEvent) {})

		assert.NoError(t, err)
		assert.NotNil(t, suggestions)
	})

	t.Run("issue fetch error", func(t *testing.T) {
		mockGit, mockAI, _, mockVCS, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f.go"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("diff", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)

		mockVCS.On("GetIssue", mock.Anything, 100).Return(&models.Issue{}, errors.New("fetch error"))

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo == nil
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithVCSClient(mockVCS), WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 100, func(e models.ProgressEvent) {})

		assert.NoError(t, err)
	})
}

func TestCommitService_IssueDetection(t *testing.T) {
	t.Run("detect from branch name", func(t *testing.T) {
		mockGit, mockAI, _, mockVCS, cfg := setupTest(t)
		cfg.UseTicket = false

		mockGit.On("GetCurrentBranch", mock.Anything).Return("issue/123-fix", nil)
		mockGit.On("GetChangedFiles", mock.Anything).Return([]string{"f"}, nil)
		mockGit.On("GetDiff", mock.Anything).Return("d", nil)
		mockGit.On("GetRecentCommitMessages", mock.Anything, 10).Return([]string{"history"}, nil)
		mockVCS.On("GetIssue", mock.Anything, 123).Return(&models.Issue{Number: 123}, nil)

		mockAI.On("GenerateSuggestions", mock.Anything, mock.MatchedBy(func(info models.CommitInfo) bool {
			return info.IssueInfo != nil && info.IssueInfo.Number == 123
		}), 3).Return([]models.CommitSuggestion{}, nil)

		service := NewCommitService(mockGit, mockAI, WithVCSClient(mockVCS), WithConfig(cfg))
		_, err := service.GenerateSuggestions(context.Background(), 3, 0, func(e models.ProgressEvent) {})
		assert.NoError(t, err)
		mockVCS.AssertCalled(t, "GetIssue", mock.Anything, 123)
	})
}
