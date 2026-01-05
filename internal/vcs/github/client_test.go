package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v80/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/models"
)

func newTestClient(pr *MockPRService, issues *MockIssuesService, release *MockReleaseService, userService *MockUserService) *GitHubClient {
	repo := &MockRepoService{}
	client := NewGitHubClientWithServices(
		pr,
		issues,
		repo,
		release,
		userService,
		"test-owner",
		"test-repo",
	)
	client.mainPath = "../../../../../cmd/main.go"
	return client
}

func TestGitHubClient_UpdatePR(t *testing.T) {
	t.Run("should update PR successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

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
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

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
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prNumber := 123
		labels := []string{"feature", "fix"}

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{
				{Name: github.Ptr("fix")},
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
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prNumber := 123
		expectedUser := "test-user"
		expectedCommitMsg := "commit message"
		expectedDiff := "diff content"

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", prNumber).
			Return(&github.PullRequest{User: &github.User{Login: github.Ptr(expectedUser)}}, &github.Response{}, nil)

		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return([]*github.RepositoryCommit{
				{Commit: &github.Commit{Message: github.Ptr(expectedCommitMsg)}},
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
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

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
			client := newTestClient(nil, nil, nil, nil)
			result := client.labelExists(tt.existingLabels, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitHubClient_UpdatePR_ErrorCases(t *testing.T) {
	t.Run("should return error when Edit fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prNumber := 123
		summary := models.PRSummary{Title: "Title", Body: "Body"}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, assert.AnError)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to update PR #%d", prNumber))
		mockPR.AssertExpectations(t)
	})

	t.Run("should return error when AddLabelsToPR fails after successful Edit", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prNumber := 123
		summary := models.PRSummary{Labels: []string{"fix"}}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, &github.Response{}, nil)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("failed to add labels to PR #%d", prNumber))
		mockIssues.AssertExpectations(t)
	})

	t.Run("should return helpful error message for 403 insufficient permissions", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prNumber := 123
		summary := models.PRSummary{Title: "Title", Body: "Body"}

		// Simular un error 403
		resp403 := &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
			},
		}

		mockPR.On("Edit", mock.Anything, "test-owner", "test-repo", prNumber, mock.Anything).
			Return(&github.PullRequest{}, resp403, assert.AnError)

		err := client.UpdatePR(context.Background(), prNumber, summary)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
		mockPR.AssertExpectations(t)
	})
}

func TestGitHubClient_AddLabelsToPR_ErrorCases(t *testing.T) {
	t.Run("should return error if GetRepoLabels fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"fix"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repository labels")
	})

	t.Run("should return error if CreateLabel fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{}, &github.Response{}, nil)

		mockIssues.On("CreateLabel", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"feature"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create label 'feature'")
	})

	t.Run("should return error if AddLabelsToIssue fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockIssues.On("ListLabels", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Label{{Name: github.Ptr("fix")}}, &github.Response{}, nil)

		mockIssues.On("AddLabelsToIssue", mock.Anything, "test-owner", "test-repo", 123, []string{"fix"}).
			Return([]*github.Label{}, &github.Response{}, assert.AnError)

		err := client.AddLabelsToPR(context.Background(), 123, []string{"fix"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add labels to PR #123")
	})
}

func TestGitHubClient_GetPR_ErrorCases(t *testing.T) {
	t.Run("should return error if Get fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{}, &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get PR #123")
	})

	t.Run("should return error if ListCommits fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{}, &github.Response{}, nil)
		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return([]*github.RepositoryCommit{}, &github.Response{}, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.Contains(t, err.Error(), "failed to get commits for PR #123")
	})

	t.Run("should return error if GetRaw fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockPR.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.PullRequest{User: &github.User{Login: github.Ptr("test-user")}}, &github.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)
		mockPR.On("ListCommits", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return([]*github.RepositoryCommit{}, &github.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)
		mockPR.On("GetRaw", mock.Anything, "test-owner", "test-repo", 123, mock.Anything).
			Return("", nil, assert.AnError)

		_, err := client.GetPR(context.Background(), 123)
		assert.Contains(t, err.Error(), "failed to get diff for PR #123")
	})
}

func TestGitHubClient_CreateRelease(t *testing.T) {
	t.Run("should create release successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		var client *GitHubClient

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{Title: "Release v1.0.0", Changelog: "Changes"}

		repo := &MockRepoService{}
		mockFactory := &MockBinaryBuilderFactory{}
		mockPackager := &MockBinaryPackager{}

		client = NewGitHubClientWithServices(
			mockPR,
			mockIssues,
			repo,
			mockRelease,
			mockUserService,
			"test-owner",
			"test-repo",
		)
		client.binaryBuilderFactory = mockFactory

		mockFactory.On("NewBuilder", mock.Anything, mock.Anything, mock.Anything).
			Return(mockPackager)

		tmpFile, err := os.CreateTemp("", "dummy-archive-*.tar.gz")
		require.NoError(t, err)
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				t.Logf("error eliminado archivo temporal: %v", err)
			}
		}()

		mockPackager.On("BuildAndPackageAll", mock.Anything, mock.Anything).
			Return([]string{tmpFile.Name()}, nil)

		mockRelease.On("UploadReleaseAsset", mock.Anything, "test-owner", "test-repo", int64(123), mock.Anything, mock.Anything).
			Return(&github.ReleaseAsset{}, &github.Response{}, nil)

		mockRelease.On("CreateRelease", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(r *github.RepositoryRelease) bool {
			return *r.TagName == "v1.0.0" && *r.Name == "Release v1.0.0" && *r.Body == "Changes" && *r.Draft == false
		})).Return(&github.RepositoryRelease{ID: github.Ptr(int64(123))}, &github.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil)

		repo.On("GetCommit", mock.Anything, "test-owner", "test-repo", "HEAD", mock.Anything).
			Return(&github.RepositoryCommit{SHA: github.Ptr("sha123"), Commit: &github.Commit{Committer: &github.CommitAuthor{Date: &github.Timestamp{Time: time.Now()}}}}, &github.Response{}, nil)

		err = client.CreateRelease(context.Background(), release, notes, false, true, nil)
		assert.NoError(t, err)
		mockRelease.AssertExpectations(t)
	})

	t.Run("should return error if release already exists", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		resp := &github.Response{Response: &http.Response{StatusCode: http.StatusUnprocessableEntity}}
		mockRelease.On("CreateRelease", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.RepositoryRelease{}, resp, assert.AnError)

		err := client.CreateRelease(context.Background(), release, notes, false, true, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create release")
	})

	t.Run("should return error if repo or tag not found", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		resp := &github.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}
		mockRelease.On("CreateRelease", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.RepositoryRelease{}, resp, assert.AnError)

		err := client.CreateRelease(context.Background(), release, notes, false, true, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not found")
	})

	t.Run("should return generic error for other failures", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		release := &models.Release{Version: "v1.0.0"}
		notes := &models.ReleaseNotes{}

		mockRelease.On("CreateRelease", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return(&github.RepositoryRelease{}, &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, assert.AnError)

		err := client.CreateRelease(context.Background(), release, notes, false, true, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create release")
	})
}

func TestGitHubClient_GetRelease(t *testing.T) {
	t.Run("should get release successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		expectedRelease := &github.RepositoryRelease{
			TagName: github.Ptr("v1.0.0"),
			Name:    github.Ptr("Release v1.0.0"),
			Body:    github.Ptr("Release body"),
			Draft:   github.Ptr(false),
			HTMLURL: github.Ptr("https://github.com/test/repo/releases/v1.0.0"),
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return(expectedRelease, &github.Response{}, nil)

		release, err := client.GetRelease(context.Background(), "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, release)
		assert.Equal(t, "v1.0.0", release.TagName)
		assert.Equal(t, "Release v1.0.0", release.Name)
		assert.Equal(t, "Release body", release.Body)
		assert.False(t, release.Draft)
		assert.Equal(t, "https://github.com/test/repo/releases/v1.0.0", release.URL)
		mockRelease.AssertExpectations(t)
	})

	t.Run("should return error if release not found (404)", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		resp := &github.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}
		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return((*github.RepositoryRelease)(nil), resp, assert.AnError)

		release, err := client.GetRelease(context.Background(), "v1.0.0")

		assert.Error(t, err)
		assert.Nil(t, release)
		assert.Contains(t, err.Error(), "repository not found")
		mockRelease.AssertExpectations(t)
	})

	t.Run("should return generic error for other failures", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		resp := &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}
		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return((*github.RepositoryRelease)(nil), resp, assert.AnError)

		release, err := client.GetRelease(context.Background(), "v1.0.0")

		assert.Error(t, err)
		assert.Nil(t, release)
		mockRelease.AssertExpectations(t)
	})
}

func TestGitHubClient_UpdateRelease(t *testing.T) {
	t.Run("should update release successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		existingRelease := &github.RepositoryRelease{
			ID: github.Ptr(int64(123)),
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return(existingRelease, &github.Response{}, nil)

		mockRelease.On("EditRelease", mock.Anything, "test-owner", "test-repo", int64(123), mock.MatchedBy(func(r *github.RepositoryRelease) bool {
			return *r.Body == "Updated body"
		})).Return(&github.RepositoryRelease{}, &github.Response{}, nil)

		err := client.UpdateRelease(context.Background(), "v1.0.0", "Updated body")

		assert.NoError(t, err)
		mockRelease.AssertExpectations(t)
	})

	t.Run("should return error if release not found (404)", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		resp := &github.Response{Response: &http.Response{StatusCode: http.StatusNotFound}}
		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return((*github.RepositoryRelease)(nil), resp, assert.AnError)

		err := client.UpdateRelease(context.Background(), "v1.0.0", "Updated body")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not found")
		mockRelease.AssertExpectations(t)
	})

	t.Run("should return error if EditRelease fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		existingRelease := &github.RepositoryRelease{
			ID: github.Ptr(int64(123)),
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return(existingRelease, &github.Response{}, nil)

		mockRelease.On("EditRelease", mock.Anything, "test-owner", "test-repo", int64(123), mock.Anything).
			Return((*github.RepositoryRelease)(nil), &github.Response{}, assert.AnError)

		err := client.UpdateRelease(context.Background(), "v1.0.0", "Updated body")

		assert.Error(t, err)
		mockRelease.AssertExpectations(t)
	})
}

func TestGitHubClient_GetIssue(t *testing.T) {
	t.Run("should get issue successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		issueNumber := 123
		expectedIssue := &github.Issue{
			ID:      github.Ptr(int64(456)),
			Number:  github.Ptr(issueNumber),
			Title:   github.Ptr("Issue Title"),
			Body:    github.Ptr("Issue Body"),
			State:   github.Ptr("open"),
			User:    &github.User{Login: github.Ptr("author-name")},
			HTMLURL: github.Ptr("https://github.com/owner/repo/issues/123"),
			Labels: []*github.Label{
				{Name: github.Ptr("bug")},
				{Name: github.Ptr("high-priority")},
			},
		}

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", issueNumber).
			Return(expectedIssue, &github.Response{}, nil)

		result, err := client.GetIssue(context.Background(), issueNumber)

		assert.NoError(t, err)
		assert.Equal(t, 456, result.ID)
		assert.Equal(t, issueNumber, result.Number)
		assert.Equal(t, "Issue Title", result.Title)
		assert.Equal(t, "Issue Body", result.Description)
		assert.Equal(t, "open", result.State)
		assert.Equal(t, "author-name", result.Author)
		assert.Equal(t, "https://github.com/owner/repo/issues/123", result.URL)
		assert.Contains(t, result.Labels, "bug")
		assert.Contains(t, result.Labels, "high-priority")
		mockIssues.AssertExpectations(t)
	})

	t.Run("should handle nil fields in issue", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		issueNumber := 123
		expectedIssue := &github.Issue{
			ID:     github.Ptr(int64(456)),
			Number: github.Ptr(issueNumber),
			Title:  github.Ptr("Issue Title"),
		}

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", issueNumber).
			Return(expectedIssue, &github.Response{}, nil)

		result, err := client.GetIssue(context.Background(), issueNumber)

		assert.NoError(t, err)
		assert.Equal(t, "Issue Title", result.Title)
		assert.Empty(t, result.Description)
		assert.Empty(t, result.State)
		assert.Empty(t, result.Author)
		assert.Empty(t, result.URL)
		assert.Empty(t, result.Labels)
	})

	t.Run("should return error when Get fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		issueNumber := 123

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", issueNumber).
			Return((*github.Issue)(nil), &github.Response{}, assert.AnError)

		_, err := client.GetIssue(context.Background(), issueNumber)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("error getting issue #%d", issueNumber))
		mockIssues.AssertExpectations(t)
	})
}

func TestGitHubClient_GetClosedIssuesBetweenTags(t *testing.T) {
	t.Run("should get closed issues between tags", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		currTag := "v1.1.0"
		prevReleaseDate := github.Timestamp{Time: time.Now().Add(-24 * time.Hour)}

		prevRelease := &github.RepositoryRelease{
			CreatedAt: &prevReleaseDate,
		}

		expectedIssues := []*github.Issue{
			{Number: github.Ptr(1), Title: github.Ptr("Issue 1"), PullRequestLinks: nil, Labels: []*github.Label{{Name: github.Ptr("bug")}}},
			{Number: github.Ptr(2), Title: github.Ptr("Issue 2"), PullRequestLinks: nil},
			{Number: github.Ptr(3), Title: github.Ptr("PR 3"), PullRequestLinks: &github.PullRequestLinks{}},
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", prevTag).
			Return(prevRelease, &github.Response{}, nil)

		mockIssues.On("ListByRepo", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.IssueListByRepoOptions) bool {
			return opts.State == "closed" && opts.Since.Equal(prevReleaseDate.Time) && opts.Sort == "updated" && opts.Direction == "desc"
		})).Return(expectedIssues, &github.Response{}, nil)

		issues, err := client.GetClosedIssuesBetweenTags(context.Background(), prevTag, currTag)

		assert.NoError(t, err)
		assert.Len(t, issues, 2)
		assert.Equal(t, 1, issues[0].Number)
		assert.Equal(t, "Issue 1", issues[0].Title)
		assert.Equal(t, 2, issues[1].Number)
		assert.Equal(t, "Issue 2", issues[1].Title)
		mockRelease.AssertExpectations(t)
		mockIssues.AssertExpectations(t)
	})

	t.Run("should handle pagination", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		prevReleaseDate := github.Timestamp{Time: time.Now().Add(-24 * time.Hour)}

		prevRelease := &github.RepositoryRelease{
			CreatedAt: &prevReleaseDate,
		}

		page1Issues := []*github.Issue{
			{Number: github.Ptr(1), PullRequestLinks: nil},
		}
		page2Issues := []*github.Issue{
			{Number: github.Ptr(2), PullRequestLinks: nil},
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", prevTag).
			Return(prevRelease, &github.Response{}, nil)

		mockIssues.On("ListByRepo", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.IssueListByRepoOptions) bool {
			return opts.ListOptions.Page == 0
		})).Return(page1Issues, &github.Response{NextPage: 2}, nil)

		mockIssues.On("ListByRepo", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.IssueListByRepoOptions) bool {
			return opts.ListOptions.Page == 2
		})).Return(page2Issues, &github.Response{NextPage: 0}, nil)

		issues, err := client.GetClosedIssuesBetweenTags(context.Background(), prevTag, "v1.1.0")

		assert.NoError(t, err)
		assert.Len(t, issues, 2)
		assert.Equal(t, 1, issues[0].Number)
		assert.Equal(t, 2, issues[1].Number)
	})

	t.Run("should return error if GetReleaseByTag fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return((*github.RepositoryRelease)(nil), &github.Response{}, assert.AnError)

		_, err := client.GetClosedIssuesBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})

	t.Run("should return error if ListByRepo fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevRelease := &github.RepositoryRelease{
			CreatedAt: &github.Timestamp{},
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return(prevRelease, &github.Response{}, nil)

		mockIssues.On("ListByRepo", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.Issue{}, &github.Response{}, assert.AnError)

		_, err := client.GetClosedIssuesBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetMergedPRsBetweenTags(t *testing.T) {
	t.Run("should get merged PRs between tags", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		currTag := "v1.1.0"

		prevTime := time.Now().Add(-48 * time.Hour)
		mergedTime1 := time.Now().Add(-24 * time.Hour)
		mergedTime2 := time.Now().Add(-72 * time.Hour)

		prevReleaseDate := github.Timestamp{Time: prevTime}

		prevRelease := &github.RepositoryRelease{
			CreatedAt: &prevReleaseDate,
		}

		pr1 := &github.PullRequest{
			Number:   github.Ptr(1),
			Title:    github.Ptr("PR 1"),
			Body:     github.Ptr("Description 1"),
			User:     &github.User{Login: github.Ptr("user1")},
			Merged:   github.Ptr(true),
			MergedAt: &github.Timestamp{Time: mergedTime1},
			HTMLURL:  github.Ptr("url1"),
			Labels:   []*github.Label{{Name: github.Ptr("bug")}},
		}

		pr2 := &github.PullRequest{
			Number: github.Ptr(2),
			Merged: github.Ptr(false),
		}

		pr3 := &github.PullRequest{
			Number:   github.Ptr(3),
			Merged:   github.Ptr(true),
			MergedAt: &github.Timestamp{Time: mergedTime2},
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", prevTag).
			Return(prevRelease, &github.Response{}, nil)

		mockPR.On("List", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
			return opts.State == "closed" && opts.Sort == "updated" && opts.Direction == "desc"
		})).Return([]*github.PullRequest{pr1, pr2, pr3}, &github.Response{}, nil)

		prs, err := client.GetMergedPRsBetweenTags(context.Background(), prevTag, currTag)

		assert.NoError(t, err)
		assert.Len(t, prs, 1)
		assert.Equal(t, 1, prs[0].Number)
		assert.Equal(t, "PR 1", prs[0].Title)
		assert.Equal(t, "Description 1", prs[0].Description)
		assert.Equal(t, "user1", prs[0].Author)
		assert.Contains(t, prs[0].Labels, "bug")
		mockRelease.AssertExpectations(t)
		mockPR.AssertExpectations(t)
	})

	t.Run("should handle pagination", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		prevReleaseDate := github.Timestamp{Time: time.Now().Add(-24 * time.Hour)}

		prevRelease := &github.RepositoryRelease{
			CreatedAt: &prevReleaseDate,
		}

		mergedTime := time.Now()
		pr1 := &github.PullRequest{
			Number:   github.Ptr(1),
			Merged:   github.Ptr(true),
			MergedAt: &github.Timestamp{Time: mergedTime},
		}

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", prevTag).
			Return(prevRelease, &github.Response{}, nil)

		mockPR.On("List", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
			return opts.ListOptions.Page == 0
		})).Return([]*github.PullRequest{pr1}, &github.Response{NextPage: 2}, nil)

		mockPR.On("List", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(opts *github.PullRequestListOptions) bool {
			return opts.ListOptions.Page == 2
		})).Return([]*github.PullRequest{}, &github.Response{NextPage: 0}, nil)

		prs, err := client.GetMergedPRsBetweenTags(context.Background(), prevTag, "v1.1.0")

		assert.NoError(t, err)
		assert.Len(t, prs, 1)
	})

	t.Run("should return error if GetReleaseByTag fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return((*github.RepositoryRelease)(nil), &github.Response{}, assert.AnError)

		_, err := client.GetMergedPRsBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})

	t.Run("should return error if List fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockRelease.On("GetReleaseByTag", mock.Anything, "test-owner", "test-repo", "v1.0.0").
			Return(&github.RepositoryRelease{CreatedAt: &github.Timestamp{}}, &github.Response{}, nil)

		mockPR.On("List", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return([]*github.PullRequest{}, &github.Response{}, assert.AnError)

		_, err := client.GetMergedPRsBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetContributorsBetweenTags(t *testing.T) {
	t.Run("should get distinct contributors between tags", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		currTag := "v1.1.0"

		comparison := &github.CommitsComparison{
			Commits: []*github.RepositoryCommit{
				{Author: &github.User{Login: github.Ptr("user1")}},
				{Author: &github.User{Login: github.Ptr("user2")}},
				{Author: &github.User{Login: github.Ptr("user1")}},
			},
		}

		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		mockRepo.On("CompareCommits", mock.Anything, "test-owner", "test-repo", prevTag, currTag, mock.Anything).
			Return(comparison, &github.Response{}, nil)

		contributors, err := client.GetContributorsBetweenTags(context.Background(), prevTag, currTag)

		assert.NoError(t, err)
		assert.Len(t, contributors, 2)
		assert.Contains(t, contributors, "user1")
		assert.Contains(t, contributors, "user2")
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error if CompareCommits fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		mockRepo.On("CompareCommits", mock.Anything, "test-owner", "test-repo", "v1.0.0", "v1.1.0", mock.Anything).
			Return((*github.CommitsComparison)(nil), &github.Response{}, assert.AnError)

		_, err := client.GetContributorsBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetFileStatsBetweenTags(t *testing.T) {
	t.Run("should get file stats successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		prevTag := "v1.0.0"
		currTag := "v1.1.0"

		comparison := &github.CommitsComparison{
			Files: []*github.CommitFile{
				{Filename: github.Ptr("file1.go"), Additions: github.Ptr(10), Deletions: github.Ptr(5)},
				{Filename: github.Ptr("file2.go"), Additions: github.Ptr(2), Deletions: github.Ptr(3)},
				{Filename: github.Ptr("file3.go"), Additions: github.Ptr(100), Deletions: github.Ptr(0)}, // Top file
			},
		}

		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		mockRepo.On("CompareCommits", mock.Anything, "test-owner", "test-repo", prevTag, currTag, mock.Anything).
			Return(comparison, &github.Response{}, nil)

		stats, err := client.GetFileStatsBetweenTags(context.Background(), prevTag, currTag)

		assert.NoError(t, err)
		assert.Equal(t, 3, stats.FilesChanged)
		assert.Equal(t, 112, stats.Insertions)
		assert.Equal(t, 8, stats.Deletions)

		assert.Len(t, stats.TopFiles, 3)
		assert.Equal(t, "file3.go", stats.TopFiles[0].Path)
		assert.Equal(t, "file1.go", stats.TopFiles[1].Path)
		assert.Equal(t, "file2.go", stats.TopFiles[2].Path)

		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error if CompareCommits fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		mockRepo.On("CompareCommits", mock.Anything, "test-owner", "test-repo", "v1.0.0", "v1.1.0", mock.Anything).
			Return((*github.CommitsComparison)(nil), &github.Response{}, assert.AnError)

		_, err := client.GetFileStatsBetweenTags(context.Background(), "v1.0.0", "v1.1.0")

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetFileAtTag(t *testing.T) {
	t.Run("should get file content successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)
		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		tag := "v1.0.0"
		filepath := "test.txt"
		expectedContent := "test content"
		encodedContent := "dGVzdCBjb250ZW50"

		mockRepo.On("GetContents", mock.Anything, "test-owner", "test-repo", tag, &github.RepositoryContentGetOptions{Ref: tag}).
			Return(&github.RepositoryContent{
				Content:  github.Ptr(encodedContent),
				Encoding: github.Ptr("base64"),
			}, []*github.RepositoryContent{}, &github.Response{}, nil)

		content, err := client.GetFileAtTag(context.Background(), tag, filepath)

		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content)
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error if file not found", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)
		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		tag := "v1.0.0"
		filepath := "test.txt"

		mockRepo.On("GetContents", mock.Anything, "test-owner", "test-repo", tag, &github.RepositoryContentGetOptions{Ref: tag}).
			Return((*github.RepositoryContent)(nil), []*github.RepositoryContent{}, &github.Response{}, nil)

		_, err := client.GetFileAtTag(context.Background(), tag, filepath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("should return error if GetContents fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)
		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		tag := "v1.0.0"

		mockRepo.On("GetContents", mock.Anything, "test-owner", "test-repo", tag, mock.Anything).
			Return((*github.RepositoryContent)(nil), []*github.RepositoryContent{}, &github.Response{}, assert.AnError)

		_, err := client.GetFileAtTag(context.Background(), tag, "test.txt")

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetDiffFromCommits(t *testing.T) {
	t.Run("should get diff from commits successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)
		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		sha1 := "sha1123456789"
		sha2 := "sha2123456789"
		commits := []*github.RepositoryCommit{
			{SHA: github.Ptr(sha1), Commit: &github.Commit{Message: github.Ptr("commit message 1")}},
			{SHA: github.Ptr(sha2), Commit: &github.Commit{Message: github.Ptr("commit message 2")}},
		}

		mockRepo.On("GetCommit", mock.Anything, "test-owner", "test-repo", sha1, (*github.ListOptions)(nil)).
			Return(&github.RepositoryCommit{
				Stats:  &github.CommitStats{Total: github.Ptr(1)},
				Commit: &github.Commit{Message: github.Ptr("commit message 1")},
				Files: []*github.CommitFile{
					{Filename: github.Ptr("file1.go"), Patch: github.Ptr("patch1")},
				},
			}, &github.Response{}, nil)

		mockRepo.On("GetCommit", mock.Anything, "test-owner", "test-repo", sha2, (*github.ListOptions)(nil)).
			Return(&github.RepositoryCommit{
				Stats:  &github.CommitStats{Total: github.Ptr(1)},
				Commit: &github.Commit{Message: github.Ptr("commit message 2")},
				Files: []*github.CommitFile{
					{Filename: github.Ptr("file2.go"), Patch: github.Ptr("patch2")},
				},
			}, &github.Response{}, nil)

		diff, err := client.getDiffFromCommits(context.Background(), commits)

		assert.NoError(t, err)
		assert.Contains(t, diff, "# Commit: sha1")
		assert.Contains(t, diff, "# Message: commit message 1")
		assert.Contains(t, diff, "diff --git a/file1.go b/file1.go")
		assert.Contains(t, diff, "patch1")
		assert.Contains(t, diff, "# Commit: sha2")
		assert.Contains(t, diff, "# Message: commit message 2")
		assert.Contains(t, diff, "diff --git a/file2.go b/file2.go")
		assert.Contains(t, diff, "patch2")
		mockRepo.AssertExpectations(t)
	})

	t.Run("should return error if GetCommit fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)
		mockRepo := &MockRepoService{}
		client.repoService = mockRepo

		sha1 := "sha1123456789"
		commits := []*github.RepositoryCommit{
			{SHA: github.Ptr(sha1), Commit: &github.Commit{Message: github.Ptr("commit message 1")}},
		}

		mockRepo.On("GetCommit", mock.Anything, "test-owner", "test-repo", sha1, (*github.ListOptions)(nil)).
			Return((*github.RepositoryCommit)(nil), &github.Response{}, assert.AnError)

		_, err := client.getDiffFromCommits(context.Background(), commits)

		assert.Error(t, err)
	})
}

func TestGitHubClient_GetPRIssues(t *testing.T) {
	t.Run("should extract issues from branch name, commits and description", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		branchName := "feature/123-new-feature"
		prDescription := "Closes #456"
		commits := []string{"Fix bug #789", "Update readme"}

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.Issue{
				Number:  github.Ptr(123),
				Title:   github.Ptr("Issue 123"),
				State:   github.Ptr("open"),
				User:    &github.User{Login: github.Ptr("user1")},
				HTMLURL: github.Ptr("http://github.com/owner/repo/issues/123"),
			}, &github.Response{}, nil)

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 456).
			Return(&github.Issue{
				Number:  github.Ptr(456),
				Title:   github.Ptr("Issue 456"),
				State:   github.Ptr("closed"),
				User:    &github.User{Login: github.Ptr("user2")},
				HTMLURL: github.Ptr("http://github.com/owner/repo/issues/456"),
			}, &github.Response{}, nil)

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 789).
			Return(&github.Issue{
				Number:  github.Ptr(789),
				Title:   github.Ptr("Issue 789"),
				State:   github.Ptr("open"),
				User:    &github.User{Login: github.Ptr("user3")},
				HTMLURL: github.Ptr("http://github.com/owner/repo/issues/789"),
			}, &github.Response{}, nil)

		issues, err := client.GetPRIssues(context.Background(), branchName, commits, prDescription)

		assert.NoError(t, err)
		assert.Len(t, issues, 3)

		issueMap := make(map[int]models.Issue)
		for _, issue := range issues {
			issueMap[issue.Number] = issue
		}

		assert.Contains(t, issueMap, 123)
		assert.Contains(t, issueMap, 456)
		assert.Contains(t, issueMap, 789)

		mockIssues.AssertExpectations(t)
	})

	t.Run("should deduplicate issues", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		branchName := "feature/123-fix"
		prDescription := "Fixes #123"
		commits := []string{"Fix #123"}

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 123).
			Return(&github.Issue{
				Number: github.Ptr(123),
				Title:  github.Ptr("Issue 123"),
			}, &github.Response{}, nil).Once()

		issues, err := client.GetPRIssues(context.Background(), branchName, commits, prDescription)

		assert.NoError(t, err)
		assert.Len(t, issues, 1)
		assert.Equal(t, 123, issues[0].Number)
		mockIssues.AssertExpectations(t)
	})

	t.Run("should ignore issues that fail to be retrieved", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		branchName := ""
		prDescription := "Closes #999"
		commits := []string{}

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 999).
			Return((*github.Issue)(nil), &github.Response{}, assert.AnError)

		issues, err := client.GetPRIssues(context.Background(), branchName, commits, prDescription)

		assert.NoError(t, err)
		assert.Empty(t, issues)
		mockIssues.AssertExpectations(t)
	})

	t.Run("should support various patterns", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		branchName := "issue/111"
		commits := []string{"(#222)"}
		prDescription := "resolve #333"

		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 111).Return(&github.Issue{Number: github.Ptr(111)}, &github.Response{}, nil)
		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 222).Return(&github.Issue{Number: github.Ptr(222)}, &github.Response{}, nil)
		mockIssues.On("Get", mock.Anything, "test-owner", "test-repo", 333).Return(&github.Issue{Number: github.Ptr(333)}, &github.Response{}, nil)

		issues, err := client.GetPRIssues(context.Background(), branchName, commits, prDescription)
		assert.NoError(t, err)
		assert.Len(t, issues, 3)
	})
}

func TestGitHubClient_CreateIssue(t *testing.T) {
	t.Run("should create issue successfully", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		title := "Bug Title"
		body := "Bug body description"
		labels := []string{"bug", "critical"}
		assignees := []string{"user1"}

		expectedGHIssue := &github.Issue{
			ID:      github.Ptr(int64(789)),
			Number:  github.Ptr(42),
			Title:   github.Ptr(title),
			Body:    github.Ptr(body),
			State:   github.Ptr("open"),
			HTMLURL: github.Ptr("https://github.com/owner/repo/issues/42"),
			User:    &github.User{Login: github.Ptr("test-user")},
			Labels: []*github.Label{
				{Name: github.Ptr("bug")},
				{Name: github.Ptr("critical")},
			},
		}

		mockIssues.On("Create", mock.Anything, "test-owner", "test-repo", mock.MatchedBy(func(req *github.IssueRequest) bool {
			return *req.Title == title && *req.Body == body && len(*req.Labels) == 2 && len(*req.Assignees) == 1
		})).Return(expectedGHIssue, &github.Response{}, nil)

		result, err := client.CreateIssue(context.Background(), title, body, labels, assignees)

		assert.NoError(t, err)
		assert.Equal(t, 42, result.Number)
		assert.Equal(t, title, result.Title)
		assert.Equal(t, body, result.Description)
		assert.Equal(t, "test-user", result.Author)
		assert.ElementsMatch(t, labels, result.Labels)
		mockIssues.AssertExpectations(t)
	})

	t.Run("should return error when Create fails", func(t *testing.T) {
		mockPR := &MockPRService{}
		mockIssues := &MockIssuesService{}
		mockRelease := &MockReleaseService{}
		mockUserService := &MockUserService{}
		client := newTestClient(mockPR, mockIssues, mockRelease, mockUserService)

		mockIssues.On("Create", mock.Anything, "test-owner", "test-repo", mock.Anything).
			Return((*github.Issue)(nil), &github.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}}, assert.AnError)

		_, err := client.CreateIssue(context.Background(), "Title", "Body", nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error creating issue")
		mockIssues.AssertExpectations(t)
	})
}
