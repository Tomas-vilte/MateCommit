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

func TestSetAIModelCommand(t *testing.T) {
	t.Run("should successfully set model for Gemini", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "gemini", "gemini-1.5-flash"})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, config.ModelGeminiV15Flash, cfg.AIConfig.Models[config.AIGemini])
	})

	t.Run("should fail with invalid AI", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		// Simular argumentos de la línea de comandos
		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "invalid-ai", "gpt-v4o"})

		// Assert
		assert.Error(t, err)
	})

	t.Run("should handle empty AI config", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		assert.NoError(t, config.SaveConfig(cfg))
		defer cleanup()

		ctx := context.Background()

		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)
		app := &cli.Command{Commands: []*cli.Command{cmd}}

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "gemini", "gemini-1.5-flash"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, config.ModelGeminiV15Flash, loadedCfg.AIConfig.Models[config.AIGemini])
	})

	t.Run("should show available models for OpenAI when no model provided", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		// Capturar stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "openai"})

		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, output, string(config.ModelGPTV4o))
		assert.Contains(t, output, string(config.ModelGPTV4oMini))
	})

	t.Run("should show available AIs list when no arguments provided", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		// Capturar stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model"})

		// Restaurar stdout y capturar salida
		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, output, "- gemini")
		assert.Contains(t, output, "- openai")

		expectedMsg := translations.GetMessage("config_models.error_missing_ai", 0, nil)
		assert.Contains(t, err.Error(), expectedMsg)
	})

	t.Run("should show current model when AI has model configured", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIOpenAI: config.ModelGPTV4o,
			},
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "openai"})

		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		// Assert
		assert.Error(t, err)
		expectedNoModelMsg := translations.GetMessage("config_models.config_no_model_selected_for_ai", 0, map[string]interface{}{
			"AI": "openai",
		})
		assert.Contains(t, output, expectedNoModelMsg)
		assert.Contains(t, output, "- gpt-4o")
		assert.Contains(t, output, "- gpt-4o-mini")
	})

	t.Run("should show current model and available models when AI has model set", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: map[config.AI]config.Model{
				config.AIOpenAI: config.ModelGPTV4o,
			},
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "openai"})

		if err := w.Close(); err != nil {
			assert.NoError(t, err)
		}
		os.Stdout = oldStdout
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			assert.NoError(t, err)
		}
		output := buf.String()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, output, string(config.ModelGPTV4o))
		assert.Contains(t, output, string(config.ModelGPTV4oMini))
	})

	t.Run("should fail with invalid model", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "openai", "invalid-model"})

		// Assert
		assert.Error(t, err)
		assert.Empty(t, cfg.AIConfig.Models[config.AIOpenAI])
	})

	t.Run("should successfully set OpenAI model", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.AIConfig = config.AIConfig{
			Models: make(map[config.AI]config.Model),
		}
		defer cleanup()
		cmd := NewConfigCommandFactory().newSetAIModelCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-model", "openai", "gpt-4o"})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, config.ModelGPTV4o, cfg.AIConfig.Models[config.AIOpenAI])
	})
}
