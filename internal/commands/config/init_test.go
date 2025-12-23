package config

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func runInitCommandTest(t *testing.T, userInput string, fullMode bool) (output string, finalCfg *config.Config) {
	tempDir := t.TempDir()
	fakeConfigPath := filepath.Join(tempDir, "config.yaml")
	cfg := &config.Config{
		PathFile: fakeConfigPath,
		Language: "en",
	}

	translations, err := i18n.NewTranslations("en", "../../i18n/locales")
	require.NoError(t, err)

	originalStdin := os.Stdin
	originalStdout := os.Stdout
	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
	}()

	rIn, wIn, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = rIn

	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = wOut

	go func() {
		defer func() {
			_ = wIn.Close()
		}()
		_, err := wIn.Write([]byte(userInput))
		require.NoError(t, err)
	}()

	var outputBuffer bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&outputBuffer, rOut)
	}()

	factory := &ConfigCommandFactory{}
	cmd := factory.newInitCommand(translations, cfg)

	app := &cli.Command{
		Name:     "test",
		Commands: []*cli.Command{cmd},
	}

	// Build args based on mode
	args := []string{"test", "init"}
	if fullMode {
		args = append(args, "--full")
	}

	actionErr := app.Run(context.Background(), args)

	_ = wOut.Close()
	wg.Wait()

	if actionErr != nil && !strings.Contains(actionErr.Error(), "pipe") {
		require.NoError(t, actionErr)
	}

	return outputBuffer.String(), cfg
}

func TestInitCommand(t *testing.T) {
	t.Run("should configure all options successfully", func(t *testing.T) {
		userInput := strings.Join([]string{
			"my-gemini-api-key", "gemini-1.5-flash", "en", "yes", "my-github-token", "yes",
			"https://myjira.atlassian.net", "user@example.com", "my-jira-token", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)
		assert.Contains(t, output, "Enter your Gemini API Key")
		assert.Contains(t, output, "Enter your GitHub Personal Access Token")
		assert.Contains(t, output, "Base URL of your Jira instance")
		assert.Contains(t, output, "Configuration summary")
		if finalCfg.AIProviders != nil && finalCfg.AIProviders["gemini"].APIKey != "" {
			assert.Equal(t, "my-gemini-api-key", finalCfg.AIProviders["gemini"].APIKey)
		}
		assert.Equal(t, "en", finalCfg.Language)
		assert.True(t, finalCfg.UseTicket)
	})

	t.Run("should skip optional VCS and Tickets sections", func(t *testing.T) {
		userInput := strings.Join([]string{
			"test-api-key", "", "", "no", "n", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)

		assert.Contains(t, output, "Ticket service disabled")
		assert.NotContains(t, output, "Enter your GitHub Personal Access Token")

		if finalCfg.AIProviders != nil && finalCfg.AIProviders["gemini"].APIKey != "" {
			assert.Equal(t, "test-api-key", finalCfg.AIProviders["gemini"].APIKey)
		}
		assert.Equal(t, config.ModelGeminiV15Flash, finalCfg.AIConfig.Models[config.AIGemini])
		assert.Equal(t, "", finalCfg.ActiveVCSProvider)
		assert.False(t, finalCfg.UseTicket)
	})

	t.Run("should handle invalid language and keep original", func(t *testing.T) {
		userInput := strings.Join([]string{
			"", "", "fr", "no", "no", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)
		assert.Contains(t, output, "Invalid language. Please enter 'en' or 'es'.")
		assert.Equal(t, "en", finalCfg.Language)
	})

	t.Run("should run configuration again if user enters yes", func(t *testing.T) {
		userInput := strings.Join([]string{
			"first-run-key", "", "es", "no", "no", "yes",
			"second-run-key", "gemini-pro", "en", "no", "no", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)

		assert.Equal(t, 2, strings.Count(output, "Enter your Gemini API Key"))
		assert.Equal(t, 2, strings.Count(output, "Configuration summary"))
		if finalCfg.AIProviders != nil && finalCfg.AIProviders["gemini"].APIKey != "" {
			assert.Equal(t, "second-run-key", finalCfg.AIProviders["gemini"].APIKey)
		}
		assert.Equal(t, "en", finalCfg.Language)
		assert.Equal(t, config.Model("gemini-pro"), finalCfg.AIConfig.Models[config.AIGemini])
	})
}
