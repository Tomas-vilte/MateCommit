package services

import (
	"context"
	"errors"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type MockVCSClient struct {
	mock.Mock
}

func (m *MockVCSClient) UpdatePR(ctx context.Context, prNumber int, summary models.PRSummary) error {
	args := m.Called(ctx, prNumber, summary)
	return args.Error(0)
}

func (m *MockVCSClient) GetPR(ctx context.Context, prNumber int) (models.PRData, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).(models.PRData), args.Error(1)
}

func (m *MockVCSClient) GetRepoLabels(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVCSClient) CreateLabel(ctx context.Context, name, color, description string) error {
	args := m.Called(ctx, name, color, description)
	return args.Error(0)
}

func (m *MockVCSClient) AddLabelsToPR(ctx context.Context, prNumber int, labels []string) error {
	args := m.Called(ctx, prNumber, labels)
	return args.Error(0)
}

type MockPRSummarizer struct {
	mock.Mock
}

func (m *MockPRSummarizer) GeneratePRSummary(ctx context.Context, prompt string) (models.PRSummary, error) {
	args := m.Called(ctx, prompt)
	return args.Get(0).(models.PRSummary), args.Error(1)
}

func TestPRService_SummarizePR_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)

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
	mockAI.On("GeneratePRSummary", ctx, mock.AnythingOfType("string")).Return(expectedSummary, nil)
	mockVCS.On("UpdatePR", ctx, prNumber, expectedSummary).Return(nil)

	service := NewPRService(mockVCS, mockAI)

	// Act
	result, err := service.SummarizePR(ctx, prNumber)

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

	expectedError := errors.New("API error")

	mockVCS.On("GetPR", ctx, prNumber).Return(models.PRData{}, expectedError)

	service := NewPRService(mockVCS, mockAI)

	// act
	_, err := service.SummarizePR(ctx, prNumber)

	// assert
	assert.ErrorContains(t, err, "hubo un error al obtener el pr")
	assert.ErrorIs(t, err, expectedError)
	mockVCS.AssertExpectations(t)
	mockAI.AssertNotCalled(t, "GeneratePRSummary")
}

func TestPRService_SummarizePR_GenerateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)

	prData := models.PRData{
		ID:      123,
		Creator: "user1",
		Commits: []models.Commit{{Message: "fix: something"}},
		Diff:    "diff content",
	}

	expectedError := errors.New("AI failure")

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockAI.On("GeneratePRSummary", ctx, mock.Anything).Return(models.PRSummary{}, expectedError)

	service := NewPRService(mockVCS, mockAI)

	// Act
	_, err := service.SummarizePR(ctx, prNumber)

	// Assert
	assert.ErrorContains(t, err, "hubo un error al crear el resumen del pull requests")
	assert.ErrorIs(t, err, expectedError)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockVCS.AssertNotCalled(t, "UpdatePR")
}

func TestPRService_SummarizePR_UpdateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)

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
	mockAI.On("GeneratePRSummary", ctx, mock.Anything).Return(summary, nil)
	mockVCS.On("UpdatePR", ctx, prNumber, summary).Return(expectedError)

	service := NewPRService(mockVCS, mockAI)

	_, err := service.SummarizePR(ctx, prNumber)

	assert.ErrorContains(t, err, "hubo un error al actualizar el pull requests")
	assert.ErrorIs(t, err, expectedError)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
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

Commits:
- feat: add new API
- docs: update readme

Changes:
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
		GithubRepo:   "MateCommit",
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
		GeminiAPIKey: testConfig.GeminiAPIKey,
		Language:     "es",
		AIConfig: config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV15Flash,
			},
		},
	}

	ctx := context.Background()
	geminiSummarizer, err := gemini.NewGeminiPRSummarizer(ctx, cfg, trans)
	if err != nil {
		return nil, err
	}

	prService := NewPRService(githubClient, geminiSummarizer)

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
		summary, err := prService.SummarizePR(ctx, testConfig.PRNumber)

		require.NoError(t, err)
		require.NotEmpty(t, summary)

		t.Logf("Resumen generado: %s", summary)
	})
}
