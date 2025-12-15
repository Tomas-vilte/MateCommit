package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPRService_SummarizePR_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	cfg := &config.Config{}
	require.NoError(t, err)

	prData := models.PRData{
		ID:      123,
		Creator: "user1",
		Commits: []models.Commit{
			{Message: "fix: bug correction"},
			{Message: "feat: new feature"},
		},
		Diff: "diff --git a/file.txt b/file.txt",
	}

	expectedSummary := models.PRSummary{
		Title:  "Improved features",
		Body:   "Summary of changes",
		Labels: []string{"enhancement"},
	}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)
	mockAI.On("GeneratePRSummary", ctx, mock.AnythingOfType("string")).Return(expectedSummary, nil)
	mockVCS.On("UpdatePR", ctx, prNumber, expectedSummary).Return(nil)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	// Act
	result, err := service.SummarizePR(ctx, prNumber, func(s string) {})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedSummary, result)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestPRService_SummarizePR_GetPRError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	cfg := &config.Config{}
	require.NoError(t, err)

	expectedError := errors.New("API error")

	mockVCS.On("GetPR", ctx, prNumber).Return(models.PRData{}, expectedError)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	// act
	_, err = service.SummarizePR(ctx, prNumber, func(s string) {})

	// assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error al obtener el PR")
	mockVCS.AssertExpectations(t)
	mockAI.AssertNotCalled(t, "GeneratePRSummary")
}

func TestPRService_SummarizePR_GenerateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{}
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	require.NoError(t, err)

	prData := models.PRData{
		ID:      123,
		Creator: "user1",
		Commits: []models.Commit{{Message: "fix: something"}},
		Diff:    "diff content",
	}

	expectedError := errors.New("AI failure")

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)
	mockAI.On("GeneratePRSummary", ctx, mock.Anything).Return(models.PRSummary{}, expectedError)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	// Act
	_, err = service.SummarizePR(ctx, prNumber, func(s string) {})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error al crear el resumen del PR")
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockVCS.AssertNotCalled(t, "UpdatePR")
}

func TestPRService_SummarizePR_UpdateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	cfg := &config.Config{}
	require.NoError(t, err)

	prData := models.PRData{
		ID:      123,
		Creator: "user1",
		Commits: []models.Commit{{Message: "fix: something"}},
	}

	summary := models.PRSummary{
		Title:  "New title",
		Body:   "Summary body",
		Labels: []string{"bug"},
	}

	expectedError := errors.New("update failed")

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)
	mockAI.On("GeneratePRSummary", ctx, mock.Anything).Return(summary, nil)
	mockVCS.On("UpdatePR", ctx, prNumber, summary).Return(expectedError)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	_, err = service.SummarizePR(ctx, prNumber, func(s string) {})

	assert.ErrorContains(t, err, "Error al actualizar el PR: update failed")
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestPRService_SummarizePR_NilAIService(t *testing.T) {
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	require.NoError(t, err)
	cfg := &config.Config{}

	service := NewPRService(nil, nil, trans, cfg)

	summary, err := service.SummarizePR(context.Background(), 1, func(msg string) {})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "La IA no está configurada")
	assert.Empty(t, summary.Title)
}

func TestPRService_SummarizePR_WithRelatedIssues(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	require.NoError(t, err)
	cfg := &config.Config{Language: "es"}

	prData := models.PRData{
		ID:            prNumber,
		BranchName:    "fix/123-bug",
		PRDescription: "Closes #456",
		Commits:       []models.Commit{{Message: "Fix #789"}},
	}

	relatedIssues := []models.Issue{
		{Number: 123, Title: "Bug 1"},
		{Number: 456, Title: "Bug 2"},
		{Number: 789, Title: "Bug 3"},
	}

	expectedSummary := models.PRSummary{
		Title: "Fix bugs",
		Body:  "Summary content",
	}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, prData.BranchName, []string{"Fix #789"}, prData.PRDescription).
		Return(relatedIssues, nil)

	mockAI.On("GeneratePRSummary", ctx, mock.MatchedBy(func(prompt string) bool {
		return contextContains(prompt, "Bug 1", "Bug 2", "Bug 3")
	})).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, mock.MatchedBy(func(s models.PRSummary) bool {
		return contextContains(s.Body, "Summary content", "Fixes #123", "Fixes #456", "Fixes #789", "## Test Plan")
	})).Return(nil)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	_, err = service.SummarizePR(ctx, prNumber, func(s string) {})

	assert.NoError(t, err)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestPRService_SummarizePR_BreakingChanges(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	trans, err := i18n.NewTranslations("es", "../i18n/locales")
	require.NoError(t, err)
	cfg := &config.Config{}

	prData := models.PRData{
		ID:      prNumber,
		Commits: []models.Commit{{Message: "feat!: breaking change here"}},
	}

	expectedSummary := models.PRSummary{Title: "Title", Body: "Body"}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)

	mockAI.On("GeneratePRSummary", ctx, mock.MatchedBy(func(prompt string) bool {
		return contextContains(prompt, "Breaking Changes detectados", "feat!: breaking change here")
	})).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, mock.MatchedBy(func(s models.PRSummary) bool {
		return contextContains(s.Body, "## ⚠️ Breaking Changes", "- feat!: breaking change here")
	})).Return(nil)

	service := NewPRService(mockVCS, mockAI, trans, cfg)

	_, err = service.SummarizePR(ctx, prNumber, func(s string) {})

	assert.NoError(t, err)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func contextContains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestBuildPRPrompt(t *testing.T) {
	// Arrange
	prData := models.PRData{
		ID:      456,
		Creator: "dev123",
		Commits: []models.Commit{
			{Message: "feat: add new API"},
			{Message: "docs: update readme"},
		},
		Diff: "diff --git a/api.go b/api.go",
	}

	service := PRService{}

	// Act
	prompt := service.buildPRPrompt(prData)

	// Assert
	expected := `PR #456 by dev123
Branch: 

📊 **Métricas:**
- 2 commits
- 1 archivos cambiados
- ~0 líneas en diff

Commits:
- feat: add new API
- docs: update readme

Archivos principales modificados:
- api.go

Changes (diff completo):
diff --git a/api.go b/api.go`

	assert.Equal(t, expected, prompt)
}

type TestConfig struct {
	GithubToken  string
	GithubOwner  string
	GithubRepo   string
	GeminiAPIKey string
	PRNumber     int
}

func setupTestConfig(t *testing.T) TestConfig {
	t.Helper()

	conf := TestConfig{
		GithubToken:  os.Getenv("GITHUB_TOKEN"),
		GithubOwner:  "Tomas-vilte",
		GithubRepo:   "ButakeroMusicBotGo",
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		PRNumber:     272,
	}

	require.NotEmpty(t, conf.GithubToken, "GITHUB_TOKEN falta esto")
	require.NotEmpty(t, conf.GeminiAPIKey, "GEMINI_API_KEY falta esto")

	return conf
}

func setupServices(t *testing.T, testConfig TestConfig) (*PRService, error) {
	t.Helper()

	trans, err := i18n.NewTranslations("es", "../i18n/locales/")
	require.NoError(t, err)

	githubClient := github.NewGitHubClient(
		testConfig.GithubOwner,
		testConfig.GithubRepo,
		testConfig.GithubToken,
		trans,
	)

	cfg := &config.Config{
		Language: "es",
		AIProviders: map[string]config.AIProviderConfig{
			"gemini": {
				APIKey:      testConfig.GeminiAPIKey,
				Model:       "gemini-2.5-flash",
				Temperature: 0.3,
				MaxTokens:   10000,
			},
		},
		AIConfig: config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV25Flash,
			},
		},
	}

	ctx := context.Background()
	geminiSummarizer, err := gemini.NewGeminiPRSummarizer(ctx, cfg, trans)
	if err != nil {
		return nil, err
	}

	prService := NewPRService(githubClient, geminiSummarizer, trans, cfg)

	return prService, nil
}

func TestPRService_SummarizePR_Integration(t *testing.T) {
	t.Skip("omitir test")
	if testing.Short() {
		t.Skip("saltar a modo corto")
	}

	testConfig := setupTestConfig(t)
	prService, err := setupServices(t, testConfig)
	require.NoError(t, err)

	t.Run("should successfully summarize a real PR", func(t *testing.T) {
		ctx := context.Background()
		summary, err := prService.SummarizePR(ctx, testConfig.PRNumber, func(s string) {
			t.Log(s)
		})

		require.NoError(t, err)
		require.NotEmpty(t, summary)

		t.Logf("Resumen generado: %s", summary)
	})
}
