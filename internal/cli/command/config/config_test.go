package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"os"
	"path/filepath"
	"testing"
)

func setupTestApp(t *i18n.Translations, cfg *config.Config) (*cli.Command, func()) {
	tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
	if err != nil {
		panic(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			newSetLangCommand(t, cfg),
			newShowCommand(t, cfg),
			newSetAPIKeyCommand(t, cfg),
		},
	}

	return app, cleanup
}

func TestNewSetLangCommand(t *testing.T) {
	t.Run("should successfully set valid language to English", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:    tmpConfigPath,
			MaxLength:   72,
			DefaultLang: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		args := []string{"app", "set-lang", "--lang", "en"}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "en", loadedCfg.DefaultLang)
	})

	t.Run("should fail with unsupported language", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:    tmpConfigPath,
			MaxLength:   72,
			DefaultLang: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		args := []string{"app", "set-lang", "--lang", "fr"}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.Error(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "es", loadedCfg.DefaultLang)
	})
}

func TestNewShowCommand(t *testing.T) {
	t.Run("should display configuration with API key set", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:     tmpConfigPath,
			MaxLength:    72,
			DefaultLang:  "es",
			UseEmoji:     true,
			GeminiAPIKey: "test-api-key",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		args := []string{"app", "show"}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should display configuration without API key set", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:    tmpConfigPath,
			MaxLength:   72,
			DefaultLang: "es",
			UseEmoji:    true,
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		args := []string{"app", "show"}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.NoError(t, err)
	})
}

func TestNewSetAPIKeyCommand(t *testing.T) {
	t.Run("should fail with invalid API key length", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:    tmpConfigPath,
			MaxLength:   72,
			DefaultLang: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		args := []string{"app", "set-api-key", "--key", "short"}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.Error(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Empty(t, loadedCfg.GeminiAPIKey)
	})

	t.Run("should successfully update existing API key", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:     tmpConfigPath,
			MaxLength:    72,
			DefaultLang:  "es",
			GeminiAPIKey: "old-api-key-12345",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		newKey := "new-api-key-67890"
		args := []string{"app", "set-api-key", "--key", newKey}

		// Act
		err = app.Run(ctx, args)

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, newKey, loadedCfg.GeminiAPIKey)
	})
}
