package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"os"
	"testing"
)

func TestSetAPIKeyCommand(t *testing.T) {
	t.Run("should fail with invalid API key length", func(t *testing.T) {
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cmd := NewConfigCommandFactory().newSetAPIKeyCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-api-key", "--key", "short"})

		// Assert
		assert.Error(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Empty(t, loadedCfg.GeminiAPIKey)
	})

	t.Run("should successfully update existing API key", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		cfg.GeminiAPIKey = "old_api_key-12345"
		assert.NoError(t, config.SaveConfig(cfg))
		defer cleanup()

		cmd := NewConfigCommandFactory().newSetAPIKeyCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		newKey := "new-api-key-67890"

		// Act
		err := app.Run(ctx, []string{"config", "set-api-key", "--key", newKey})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, newKey, loadedCfg.GeminiAPIKey)
	})

	t.Run("save config error", func(t *testing.T) {
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		factory := NewConfigCommandFactory()
		cmd := factory.newSetAPIKeyCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-api-key", "--key", "test-api-key-123456"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")
		assert.Empty(t, cfg.GeminiAPIKey)
	})
}
