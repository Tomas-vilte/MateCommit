package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/ai/gemini"
	"github.com/thomas-vilte/matecommit/internal/config"
	domainErrors "github.com/thomas-vilte/matecommit/internal/errors"
	"github.com/thomas-vilte/matecommit/internal/models"
	"github.com/thomas-vilte/matecommit/internal/vcs/github"
)

func TestPRService_SummarizePR_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{}

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

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	// Act
	result, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

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
	cfg := &config.Config{}

	expectedError := errors.New("API error")

	mockVCS.On("GetPR", ctx, prNumber).Return(models.PRData{}, expectedError)

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	// act
	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	// assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VCS: error getting PR")
	mockVCS.AssertExpectations(t)
	mockAI.AssertNotCalled(t, "GeneratePRSummary")
}

func TestPRService_SummarizePR_GenerateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{}

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

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	// Act
	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI: error generating PR summary")
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockVCS.AssertNotCalled(t, "UpdatePR")
}

func TestPRService_SummarizePR_UpdateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{}

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

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	assert.ErrorContains(t, err, "VCS: error updating PR")
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestPRService_SummarizePR_NilAIService(t *testing.T) {
	cfg := &config.Config{}

	service := NewPRService(
		WithPRConfig(cfg),
	)

	summary, err := service.SummarizePR(context.Background(), 1, func(e models.ProgressEvent) {})

	assert.Error(t, err)
	assert.ErrorIs(t, err, domainErrors.ErrAPIKeyMissing)
	assert.Empty(t, summary.Title)
}

func TestPRService_SummarizePR_WithRelatedIssues(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{Language: "es"}

	prData := models.PRData{
		ID:          prNumber,
		Title:       "fix/123-bug",
		BranchName:  "fix/123-bug",
		Description: "Closes #456",
		Commits:     []models.Commit{{Message: "Fix #789"}},
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
	mockVCS.On("GetPRIssues", ctx, prData.BranchName, []string{"Fix #789"}, prData.Description).
		Return(relatedIssues, nil)

	mockAI.On("GeneratePRSummary", ctx, mock.MatchedBy(func(prompt string) bool {
		return contextContains(prompt, "Bug 1", "Bug 2", "Bug 3")
	})).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, mock.MatchedBy(func(s models.PRSummary) bool {
		return contextContains(s.Body, "Summary content", "Fixes #123", "Fixes #456", "Fixes #789", "## Test Plan")
	})).Return(nil)

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	assert.NoError(t, err)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
}

func TestPRService_SummarizePR_BreakingChanges(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	cfg := &config.Config{}

	prData := models.PRData{
		ID:      prNumber,
		Commits: []models.Commit{{Message: "feat!: breaking change here"}},
	}

	expectedSummary := models.PRSummary{Title: "Title", Body: "Body"}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)

	mockAI.On("GeneratePRSummary", ctx, mock.MatchedBy(func(prompt string) bool {
		return contextContains(prompt, "⚠️ Breaking Changes:", "feat!: breaking change here")
	})).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, mock.MatchedBy(func(s models.PRSummary) bool {
		return contextContains(s.Body, "Breaking Changes", "feat!: breaking change here")
	})).Return(nil)

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
	)

	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

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
	prompt := service.buildPRPrompt(prData, nil)

	// Assert
	expected := `PR #456 by dev123
Branch: 

Stats: 2 commits, 1 files, ~0 lines

Commits:
- feat: add new API
- docs: update readme

Main files modified:
- api.go

Changes (diff completo):
diff --git a/api.go b/api.go`

	assert.Equal(t, expected, prompt)

}

func TestBuildPRPrompt_WithTemplate(t *testing.T) {
	// Arrange
	prData := models.PRData{
		ID:      789,
		Creator: "dev",
		Diff:    "diff content",
	}

	template := &models.IssueTemplate{
		Name:        "MyTemplate",
		BodyContent: "## TODO\n- [ ] Check this",
	}

	service := PRService{config: &config.Config{}}

	// Act
	prompt := service.buildPRPrompt(prData, template)

	// Assert
	assert.Contains(t, prompt, "## TODO")
	assert.Contains(t, prompt, "- [ ] Check this")
	assert.Contains(t, prompt, "PR #789")
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

	githubClient := github.NewGitHubClient(
		testConfig.GithubOwner,
		testConfig.GithubRepo,
		testConfig.GithubToken,
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
				config.AIGemini: config.ModelGeminiV15Flash,
			},
		},
	}

	ctx := context.Background()
	geminiSummarizer, err := gemini.NewGeminiPRSummarizer(ctx, cfg, nil)
	if err != nil {
		return nil, err
	}

	prService := NewPRService(
		WithPRVCSClient(githubClient),
		WithPRAIProvider(geminiSummarizer),
		WithPRConfig(cfg),
	)

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
		summary, err := prService.SummarizePR(ctx, testConfig.PRNumber, func(e models.ProgressEvent) {
			t.Logf("Progress: %v", e)
		})

		require.NoError(t, err)
		require.NotEmpty(t, summary)

		t.Logf("Resumen generado: %+v", summary)
	})
}

type MockPRTemplateService struct {
	mock.Mock
}

func (m *MockPRTemplateService) GetPRTemplate(ctx context.Context, name string) (*models.IssueTemplate, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IssueTemplate), args.Error(1)
}

func (m *MockPRTemplateService) ListPRTemplates(ctx context.Context) ([]models.TemplateMetadata, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TemplateMetadata), args.Error(1)
}

func TestPRService_SummarizePR_WithTemplate(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	mockTemplate := new(MockPRTemplateService)
	cfg := &config.Config{}

	prData := models.PRData{
		ID:      prNumber,
		Creator: "user1",
		Commits: []models.Commit{{Message: "feat: something"}},
		Diff:    "diff content",
	}

	expectedSummary := models.PRSummary{
		Title: "Title",
		Body:  "Body",
	}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)

	// Template setup
	mockTemplate.On("ListPRTemplates", ctx).Return([]models.TemplateMetadata{
		{Name: "Default", FilePath: "PULL_REQUEST_TEMPLATE.md"},
	}, nil)

	templateContent := &models.IssueTemplate{
		Name:        "Default",
		BodyContent: "## Checklist\n- [ ] Done",
	}
	mockTemplate.On("GetPRTemplate", ctx, "PULL_REQUEST_TEMPLATE.md").Return(templateContent, nil)

	mockAI.On("GeneratePRSummary", ctx, mock.MatchedBy(func(prompt string) bool {
		return strings.Contains(prompt, "## Checklist") && strings.Contains(prompt, "- [ ] Done")
	})).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, expectedSummary).Return(nil)

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
		WithPRTemplateService(mockTemplate),
	)

	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	assert.NoError(t, err)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockTemplate.AssertExpectations(t)
}

func TestPRService_SummarizePR_WithTemplateError(t *testing.T) {
	ctx := context.Background()
	prNumber := 123

	mockVCS := new(MockVCSClient)
	mockAI := new(MockPRSummarizer)
	mockTemplate := new(MockPRTemplateService)
	cfg := &config.Config{}

	prData := models.PRData{
		ID:      prNumber,
		Creator: "user1",
		Commits: []models.Commit{{Message: "feat: something"}},
	}

	expectedSummary := models.PRSummary{Title: "Title", Body: "Body"}

	mockVCS.On("GetPR", ctx, prNumber).Return(prData, nil)
	mockVCS.On("GetPRIssues", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]models.Issue(nil), nil)

	mockTemplate.On("ListPRTemplates", ctx).Return([]models.TemplateMetadata(nil), errors.New("io error"))

	mockAI.On("GeneratePRSummary", ctx, mock.Anything).Return(expectedSummary, nil)

	mockVCS.On("UpdatePR", ctx, prNumber, expectedSummary).Return(nil)

	service := NewPRService(
		WithPRVCSClient(mockVCS),
		WithPRAIProvider(mockAI),
		WithPRConfig(cfg),
		WithPRTemplateService(mockTemplate),
	)

	_, err := service.SummarizePR(ctx, prNumber, func(e models.ProgressEvent) {})

	assert.NoError(t, err)
	mockVCS.AssertExpectations(t)
	mockAI.AssertExpectations(t)
	mockTemplate.AssertExpectations(t)
}
