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

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func runInitCommandTest(t *testing.T, userInput string) (output string, finalCfg *config.Config) {
	tempDir := t.TempDir()
	fakeConfigPath := filepath.Join(tempDir, "config.yaml")
	cfg := &config.Config{
		PathFile: fakeConfigPath,
		Language: "es",
	}

	translations, err := i18n.NewTranslations("es", "../../../i18n/locales")
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

	actionErr := cmd.Action(context.Background(), &cli.Command{})

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
		output, finalCfg := runInitCommandTest(t, userInput)
		assert.Contains(t, output, "Introduce tu API Key de Gemini")
		assert.Contains(t, output, "Introduce tu token de acceso de GitHub")
		assert.Contains(t, output, "URL base de tu instancia de Jira")
		assert.Contains(t, output, "Resumen de configuración")
		assert.Equal(t, "my-gemini-api-key", finalCfg.GeminiAPIKey)
		assert.Equal(t, "en", finalCfg.Language)
		assert.True(t, finalCfg.UseTicket)
	})

	t.Run("should skip optional VCS and Tickets sections", func(t *testing.T) {
		userInput := strings.Join([]string{
			"test-api-key", "", "", "no", "n", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput)

		assert.Contains(t, output, "Servicio de tickets deshabilitado")
		assert.NotContains(t, output, "Introduce tu token de acceso de GitHub")

		assert.Equal(t, "test-api-key", finalCfg.GeminiAPIKey)
		assert.Equal(t, config.Model("gemini-2.5-pro"), finalCfg.AIConfig.Models[config.AIGemini])
		assert.Equal(t, "", finalCfg.ActiveVCSProvider)
		assert.False(t, finalCfg.UseTicket)
	})

	t.Run("should handle invalid language and keep original", func(t *testing.T) {
		userInput := strings.Join([]string{
			"", "", "fr", "no", "no", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput)
		assert.Contains(t, output, "Idioma inválido. Por favor ingresa 'en' o 'es'.")
		assert.Equal(t, "es", finalCfg.Language)
	})

	t.Run("should run configuration again if user enters yes", func(t *testing.T) {
		userInput := strings.Join([]string{
			"first-run-key", "", "es", "no", "no", "yes",
			"second-run-key", "gemini-pro", "en", "no", "no", "no",
		}, "\n") + "\n"
		output, finalCfg := runInitCommandTest(t, userInput)

		assert.Equal(t, 2, strings.Count(output, "Introduce tu API Key de Gemini"))
		assert.Equal(t, 2, strings.Count(output, "Resumen de configuración"))
		assert.Equal(t, "second-run-key", finalCfg.GeminiAPIKey)
		assert.Equal(t, "en", finalCfg.Language)
		assert.Equal(t, config.Model("gemini-pro"), finalCfg.AIConfig.Models[config.AIGemini])
	})
}
