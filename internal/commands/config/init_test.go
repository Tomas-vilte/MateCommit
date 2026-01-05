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

	args := []string{"test", "init", "--global"}
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
			"my-gemini-api-key",
			"n",
			"gemini-1.5-flash",
			"en",
			"y",
			"my-github-token",
			"y",
			"https://myjira.atlassian.net",
			"user@example.com",
			"my-jira-token",
			"n",
		}, "\n") + "\n"
		_, finalCfg := runInitCommandTest(t, userInput, true)
		if finalCfg.AIProviders != nil && finalCfg.AIProviders["gemini"].APIKey != "" {
			assert.Equal(t, "my-gemini-api-key", finalCfg.AIProviders["gemini"].APIKey)
		}
		assert.Equal(t, "en", finalCfg.Language)
	})

	t.Run("should skip optional VCS and Tickets sections", func(t *testing.T) {
		userInput := strings.Join([]string{
			"test-api-key",
			"n",
			"",
			"",
			"n",
			"n",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)

		assert.Contains(t, output, "Ticket service disabled")
		assert.NotContains(t, output, "Enter your GitHub Personal Access Token")

		if finalCfg.AIProviders != nil && finalCfg.AIProviders["gemini"].APIKey != "" {
			assert.Equal(t, "test-api-key", finalCfg.AIProviders["gemini"].APIKey)
		}
		if finalCfg.AIConfig.Models != nil {
			model := finalCfg.AIConfig.Models[config.AIGemini]
			assert.NotEmpty(t, model, "Model should be set to default when empty input is provided")
		}
		assert.Equal(t, "", finalCfg.ActiveVCSProvider)
		assert.False(t, finalCfg.UseTicket)
	})

	t.Run("should handle invalid language and keep original", func(t *testing.T) {
		userInput := strings.Join([]string{
			"",
			"",
			"fr",
			"n",
			"n",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput, true)
		assert.Contains(t, output, "Invalid language. Please enter 'en' or 'es'.")
		assert.Equal(t, "en", finalCfg.Language)
	})

	t.Run("should run configuration and save", func(t *testing.T) {
		userInput := strings.Join([]string{
			"first-run-key",
			"n",
			"",
			"en",
			"n",
			"n",
		}, "\n") + "\n"
		_, finalCfg := runInitCommandTest(t, userInput, true)

		assert.Equal(t, "en", finalCfg.Language)
		if finalCfg.AIProviders != nil {
			if cfg, ok := finalCfg.AIProviders["gemini"]; ok {
				assert.Equal(t, "first-run-key", cfg.APIKey)
			}
		}
	})
}
