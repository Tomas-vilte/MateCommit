package config

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/urfave/cli/v3"
)

func TestShowCommand(t *testing.T) {
	t.Run("should display configuration with API key set", func(t *testing.T) {
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.UseEmoji = true
		cfg.AIProviders = map[string]config.AIProviderConfig{
			"gemini": {
				APIKey: "test-api-key",
			},
		}
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
		cfg.AIProviders = map[string]config.AIProviderConfig{
			"gemini": {
				APIKey: "test-api-key",
			},
		}
		cfg.TicketProviders = map[string]config.TicketProviderConfig{
			"jira": {
				BaseURL: "https://example.atlassian.net",
				Email:   "user@example.com",
				APIKey:  "test-api-key",
			},
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

		assert.Contains(t, output, "Ticket service enabled: jira")
		assert.Contains(t, output, "Jira Settings - BaseURL: https://example.atlassian.net, Email: user@example.com")

		assert.Contains(t, output, "gemini: gemini-1.5-flash")
		assert.Contains(t, output, "openai: gpt-4o")
	})

	t.Run("labels the resolved config as local (not global) when run inside a repo, with no duplicate section", func(t *testing.T) {
		// Inside a git repo, LoadConfigWithHierarchy resolves entirely to
		// the local config (not a merge with global) — the display must
		// reflect that, and must not print the same data twice under both
		// a "Global" and a "Local" header.
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.GitFallback = config.GitConfig{UserName: "Fallback Name", UserEmail: "fallback@example.com"}
		assert.NoError(t, config.SaveConfig(cfg))
		defer cleanup()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := NewConfigCommandFactory().newShowCommand(translations, cfg)
		app := &cli.Command{Commands: []*cli.Command{cmd}}

		err := app.Run(context.Background(), []string{"config", "show"})

		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		assert.NoError(t, err)
		assert.Contains(t, output, "Local Configuration")
		assert.NotContains(t, output, "Global Configuration")
		assert.Equal(t, 1, strings.Count(output, "Fallback Name"),
			"the fallback identity should be printed exactly once, not duplicated across sections")
		assert.Equal(t, 1, strings.Count(output, "━━━━━━━━━━━━━━━━━━━━━━━"),
			"there should be exactly one configuration section header")
	})
}
