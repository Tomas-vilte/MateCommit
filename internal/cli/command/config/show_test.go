package config

import (
	"bytes"
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"io"
	"os"
	"testing"
)

func TestShowCommand(t *testing.T) {
	t.Run("should display configuration with API key set", func(t *testing.T) {
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.UseEmoji = true
		cfg.GeminiAPIKey = "test-api-key"
		assert.NoError(t, config.SaveConfig(cfg))

		cmd := NewConfigCommandFactory().newShowCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		defer cleanup()

		// Act
		err := app.Run(ctx, []string{"config", "show"})

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should display configuration without API key set", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.UseEmoji = true
		assert.NoError(t, config.SaveConfig(cfg))
		defer cleanup()

		cmd := NewConfigCommandFactory().newShowCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "show"})

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should display configuration with ticket service enabled for Jira", func(t *testing.T) {
		// arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.UseTicket = true
		cfg.ActiveTicketService = "jira"
		cfg.GeminiAPIKey = "test-api-key"
		cfg.JiraConfig = config.JiraConfig{
			BaseURL: "https://example.atlassian.net",
			Email:   "user@example.com",
			APIKey:  "test-api-key",
		}
		cfg.AIConfig = config.AIConfig{
			ActiveAI: config.AIGemini,
			Models: map[config.AI]config.Model{
				config.AIGemini: config.ModelGeminiV15Flash,
				config.AIOpenAI: config.ModelGPTV4o,
			},
		}
		assert.NoError(t, config.SaveConfig(cfg))
		defer cleanup()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := NewConfigCommandFactory().newShowCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// act
		err := app.Run(ctx, []string{"config", "show"})

		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		// assert
		assert.NoError(t, err)

		assert.Contains(t, output, "Servicio de tickets habilitado: jira")
		assert.Contains(t, output, "Configuración de Jira - BaseURL: https://example.atlassian.net, Email: user@example.com")

		// Check AI models
		assert.Contains(t, output, "gemini: gemini-1.5-flash")
		assert.Contains(t, output, "openai: gpt-4o")
	})
}
