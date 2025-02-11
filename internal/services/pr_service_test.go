package services

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/github"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

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
		GithubOwner:  os.Getenv("GITHUB_OWNER"),
		GithubRepo:   os.Getenv("GITHUB_REPO"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
		PRNumber:     272,
	}

	require.NotEmpty(t, conf.GithubToken, "GITHUB_TOKEN environment variable is required")
	require.NotEmpty(t, conf.GithubOwner, "GITHUB_OWNER environment variable is required")
	require.NotEmpty(t, conf.GithubRepo, "GITHUB_REPO environment variable is required")
	require.NotEmpty(t, conf.GeminiAPIKey, "GEMINI_API_KEY environment variable is required")

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
		GeminiAPIKey: testConfig.GeminiAPIKey,
		Language:     "es",
		AIConfig: config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV15Flash,
			},
		},
	}

	trans, _ := i18n.NewTranslations("es", "../i18n/locales/")

	ctx := context.Background()
	geminiSummarizer, err := gemini.NewGeminiPRSummarizer(ctx, cfg, trans)
	if err != nil {
		return nil, err
	}

	prService := NewPRService(githubClient, geminiSummarizer, trans)

	return prService, nil
}

func TestPRService_SummarizePR_Integration(t *testing.T) {
	t.Skip("omitir test")
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
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
