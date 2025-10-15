package github

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/google/go-github/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestClient(pr *MockPRService, issues *MockIssuesService) *GitHubClient {
	trans, _ := i18n.NewTranslations("es", "../../../i18n/locales/")
	return NewGitHubClientWithServices(
		pr,
		issues,
		"test-owner",
		"test-repo",
		trans,
	)
}

func TestGitHubClient_UpdatePR(t *testing.T) {
	t.Run("should update PR successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		prNumber := 123
		summary := models.PRSummary{
			Title:  "New Title",
			Body:   "New Body",
			Labels: []string{"fix"},
		}

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, nil).Once()

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(label *github.Label) bool {
			return *label.Name == "fix"
		})).Return(&github.Label{}, &github.Response{}, nil).Once()

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{}, nil)

		mockIssues.On("AddLabelsToIssue", mock.Anything, "test-owner", "test-repo", prNumber, summary.Labels).
			Return([]*github.Label{}, &github.Response{}, nil)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.NoError(t, err)
		mockPR.AssertExpectations(t)
		mockIssues.AssertExpectations(t)
	})

	t.Run("should handle full label workflow", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		labelsToAdd := []string{"fix"}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{}, nil)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, nil)

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.Label{}, &github.Response{}, nil)

		mockIssues.On("AddLabelsToIssue", mock.Anything, "test-owner", "test-repo", 123, labelsToAdd).
			Return([]*github.Label{}, &github.Response{}, nil)

		err := client.UpdatePR(context.Background(), 123, models.PRSummary{
			Labels: labelsToAdd,
		})

		assert.NoError(t, err)
		mockIssues.AssertNumberOfCalls(t, "CreateLabel", 1)
		mockIssues.AssertNumberOfCalls(t, "AddLabelsToIssue", 1)
	})
}

func TestGitHubClient_AddLabelsToPR(t *testing.T) {
	t.Run("should create missing labels and add all", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		prNumber := 123
		labels := []string{"feature", "fix"}

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{
				{Name: github.String("fix")},
			}, &github.Response{}, nil)

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(label *github.Label) bool {
			return *label.Name == "feature"
		})).Return(&github.Label{}, &github.Response{}, nil)

		mockIssues.On("AddLabelsToIssue", mock.Anything, "test-owner", "test-repo", prNumber, labels).
			Return([]*github.Label{}, &github.Response{}, nil)

		err := client.AddLabelsToPR(context.Background(), prNumber, labels)

		assert.NoError(t, err)
		mockIssues.AssertExpectations(t)
	})
}

func TestGitHubClient_GetPR(t *testing.T) {
	t.Run("should return PR data correctly", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		prNumber := 123
		expectedUser := "test-user"
		expectedCommitMsg := "commit message"
		expectedDiff := "diff content"

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", prNumber).
			Return(&github.PullRequest{User: &github.User{Login: github.String(expectedUser)}}, &github.Response{}, nil)

		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return([]*github.RepositoryCommit{
				{Commit: &github.Commit{Message: github.String(expectedCommitMsg)}},
			}, &github.Response{}, nil)

		mockPR.On("GetRaw", mock.Anything, "test-owner", "test-repo", prNumber, github.RawOptions{Type: github.Diff}).
			Return(expectedDiff, &github.Response{}, nil)

		result, err := client.GetPR(context.Background(), prNumber)

		require.NoError(t, err)
		assert.Equal(t, prNumber, result.ID)
		assert.Equal(t, expectedUser, result.Creator)
		assert.Len(t, result.Commits, 1)
		assert.Equal(t, expectedCommitMsg, result.Commits[0].Message)
		assert.Equal(t, expectedDiff, result.Diff)
	})
}

func TestGitHubClient_CreateLabel(t *testing.T) {
	t.Run("should create label with correct parameters", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		labelName := "new-label"
		color := "FFFFFF"
		description := "test description"

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", &github.Label{
			Name:        &labelName,
			Color:       &color,
			Description: &description,
		}).Return(&github.Label{}, &github.Response{}, nil)

		err := client.CreateLabel(context.Background(), labelName, color, description)

		assert.NoError(t, err)
		mockIssues.AssertCalled(t, "CreateLabel", mock.Anything, "test-owner", "test-repo", mock.Anything)
	})
}

func TestLabelExists(t *testing.T) {
	tests := []struct {
		name           string
		existingLabels []string
		target         string
		expected       bool
	}{
		{"Exact match", []string{"bug", "feature"}, "bug", true},
		{"Case insensitive", []string{"Bug", "Feature"}, "bug", true},
		{"No match", []string{"feature"}, "bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(nil, nil)
			result := client.labelExists(tt.existingLabels, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubClient_UpdatePR_ErrorCases(t *testing.T) {
	t.Run("should return error when Edit fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		prNumber := 123
		summary := models.PRSummary{Title: "Title", Body: "Body"}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{}, assert.AnError)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.ErrorContains(t, err, client.trans.GetMessage("error.update_pr", 0, map[string]interface{}{"pr_number": prNumber}))
		mockPR.AssertExpectations(t)
	})

	t.Run("should return error when AddLabelsToPR fails after successful Edit", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		prNumber := 123
		summary := models.PRSummary{Labels: []string{"fix"}}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{}, nil)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.Error(t, err)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.add_labels", 0, map[string]interface{}{"pr_number": prNumber}))
		mockIssues.AssertExpectations(t)
	})
}

func TestGitHubClient_AddLabelsToPR_ErrorCases(t *testing.T) {
	t.Run("should return error if GetRepoLabels fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"fix"})
		assert.Error(t, err)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.get_labels", 0, nil))
	})

	t.Run("should return error if CreateLabel fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, nil)

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"feature"})
		assert.Error(t, err)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.create_label", 0, map[string]interface{}{"label": "feature"}))
	})

	t.Run("should return error if AddLabelsToIssue fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{{Name: github.String("fix")}}, &github.Response{}, nil)

		mockIssues.On("AddLabelsToIssue", mock.Anything, "test-owner", "test-repo", 123, []string{"fix"}).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"fix"})
		assert.Error(t, err)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.add_labels", 0, map[string]interface{}{"pr_number": 123}))
	})
}

func TestGitHubClient_GetPR_ErrorCases(t *testing.T) {
	t.Run("should return error if Get fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{}, &github.Response{}, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.get_pr", 0, map[string]interface{}{"pr_number": 123}))
	})

	t.Run("should return error if ListCommits fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{}, &github.Response{}, nil)
		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return([]*github.RepositoryCommit{}, &github.Response{}, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.get_commits", 0, map[string]interface{}{"pr_number": 123}))
	})

	t.Run("should return error if GetRaw fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		client := newTestClient(mockPR, mockIssues)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{}, &github.Response{}, nil)
		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return([]*github.RepositoryCommit{}, &github.Response{}, nil)
		mockPR.On("GetRaw", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return("", &github.Response{}, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.ErrorContains(t, err, client.trans.GetMessage("error.get_diff", 0, map[string]interface{}{"pr_number": 123}))
	})
}
