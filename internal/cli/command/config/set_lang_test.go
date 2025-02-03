package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"os"
	"testing"
)

func TestSetLangCommand(t *testing.T) {
	t.Run("should successfully set valid language to English", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.Language = "es"
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetLangCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-lang", "-lang", "en"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)

		assert.Equal(t, "en", loadedCfg.Language)
	})

	t.Run("should fail with unsupported language", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetLangCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-lang", "--lang", "fr"})

		// Assert
		assert.Error(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "es", loadedCfg.Language)
	})

	t.Run("config save error", func(t *testing.T) {
		// arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		factory := NewConfigCommandFactory()
		cmd := factory.newSetLangCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		err = app.Run(ctx, []string{"config", "set-lang", "--lang", "en"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")
		assert.Equal(t, "en", cfg.Language)
	})
}
