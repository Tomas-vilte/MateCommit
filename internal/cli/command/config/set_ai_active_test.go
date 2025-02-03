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

func TestSetAIActiveCommand(t *testing.T) {
	t.Run("should successfully set active AI to gemini", func(t *testing.T) {
		// arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()
		cfg.AIConfig = config.AIConfig{
			ActiveAI: config.AIOpenAI,
		}
		cmd := NewConfigCommandFactory().newSetAIActiveCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		err := app.Run(ctx, []string{"config", "set-ai-active", "gemini"})

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, config.AIGemini, cfg.AIConfig.ActiveAI)
	})

	t.Run("should fail with invalid AI", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()
		originalAI := config.AIOpenAI
		cfg.AIConfig = config.AIConfig{
			ActiveAI: originalAI,
		}
		cmd := NewConfigCommandFactory().newSetAIActiveCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-active", "invalid-ai"})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, originalAI, cfg.AIConfig.ActiveAI)
	})

	t.Run("should show available AIs when no AI is provided", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()
		cfg.AIConfig = config.AIConfig{
			ActiveAI: config.AIOpenAI,
		}
		cmd := NewConfigCommandFactory().newSetAIActiveCommand(translations, cfg)

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-active"})

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
		assert.Contains(t, output, "gemini")
		assert.Contains(t, output, "openai")
		// Verificar que el AI activo no cambió
		assert.Equal(t, config.AIOpenAI, cfg.AIConfig.ActiveAI)
	})

	t.Run("save config error", func(t *testing.T) {
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.AIConfig.ActiveAI = config.AIOpenAI

		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		factory := NewConfigCommandFactory()
		cmd := factory.newSetAIActiveCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-ai-active", "gemini"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")
		assert.Equal(t, config.AIOpenAI, cfg.AIConfig.ActiveAI)
	})
}
