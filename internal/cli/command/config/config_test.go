package config

import (
	"bytes"
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"io"
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
		if err := os.RemoveAll(tmpDir); err != nil {
			panic(err)
		}
	}

	factory := NewConfigCommandFactory()
	app := factory.CreateCommand(t, cfg)

	return app, cleanup
}

func TestSetLangCommand(t *testing.T) {
	t.Run("should successfully set valid language to English", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-lang", "--lang", "en"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "en", loadedCfg.Language)
	})

	t.Run("should fail with unsupported language", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-lang", "--lang", "fr"})

		// Assert
		assert.Error(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "es", loadedCfg.Language)
	})
}

func TestShowCommand(t *testing.T) {
	t.Run("should display configuration with API key set", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:     tmpConfigPath,
			Language:     "es",
			UseEmoji:     true,
			GeminiAPIKey: "test-api-key",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "show"})

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should display configuration without API key set", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			UseEmoji: true,
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "show"})

		// Assert
		assert.NoError(t, err)
	})
}

func TestSetAPIKeyCommand(t *testing.T) {
	t.Run("should fail with invalid API key length", func(t *testing.T) {
		// Arrange
		translations, err := i18n.NewTranslations("es", "../../../../locales")
		assert.NoError(t, err)

		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-api-key", "--key", "short"})

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
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:     tmpConfigPath,
			Language:     "es",
			GeminiAPIKey: "old-api-key-12345",
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()
		newKey := "new-api-key-67890"

		// Act
		err = app.Run(ctx, []string{"config", "set-api-key", "--key", newKey})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, newKey, loadedCfg.GeminiAPIKey)
	})
}

func TestSetTicketCommand(t *testing.T) {
	translations, err := i18n.NewTranslations("es", "../../../../locales")
	assert.NoError(t, err)

	t.Run("should successfully enable ticket", func(t *testing.T) {
		// Arrange
		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:  tmpConfigPath,
			UseTicket: false,
			Language:  "es",
		}

		cmd := NewConfigCommandFactory().newSetTicketCommand(translations, cfg)

		// Act
		err = cmd.Commands[1].Action(context.Background(), &cli.Command{})

		// Assert
		assert.NoError(t, err)
		assert.True(t, cfg.UseTicket)

		// Verificar que la configuración se guardó correctamente
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.True(t, loadedCfg.UseTicket)
	})

	t.Run("should successfully disable ticket", func(t *testing.T) {
		// arrange
		tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tmpDir, "config.json")
		cfg := &config.Config{
			PathFile:  tmpConfigPath,
			UseTicket: true,
			Language:  "es",
		}

		cmd := NewConfigCommandFactory().newSetTicketCommand(translations, cfg)

		// act
		err = cmd.Commands[0].Action(context.Background(), &cli.Command{})

		// assert
		assert.NoError(t, err)
		assert.False(t, cfg.UseTicket)

		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.False(t, loadedCfg.UseTicket)
	})
}

func TestSetAIActiveCommand(t *testing.T) {
	translations, err := i18n.NewTranslations("es", "../../../../locales")
	assert.NoError(t, err)
	tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			return
		}
	}()

	tmpConfigPath := filepath.Join(tmpDir, "config.json")

	t.Run("should successfully set active AI to gemini", func(t *testing.T) {
		// arrange
		cfg := &config.Config{
			Language: "es",
			PathFile: tmpConfigPath,
			AIConfig: config.AIConfig{
				ActiveAI: config.AIOpenAI,
			},
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
		cfg := &config.Config{
			AIConfig: config.AIConfig{
				ActiveAI: config.AIGemini,
			},
		}
		cmd := NewConfigCommandFactory().newSetAIActiveCommand(translations, cfg)

		// Simular argumentos de la línea de comandos
		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"config", "set-ai-active", "invalid-ai"})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, config.AIGemini, cfg.AIConfig.ActiveAI)
	})

	t.Run("should show available AIs when no AI is provided", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{
			Language: "es",
			PathFile: tmpConfigPath,
			AIConfig: config.AIConfig{
				ActiveAI: config.AIOpenAI,
			},
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

}

func TestSetAIModelCommand(t *testing.T) {
	translations, err := i18n.NewTranslations("es", "../../../../locales")
	assert.NoError(t, err)

	tmpDir, err := os.MkdirTemp("", "matecommit-test-*")
	assert.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			return
		}
	}()

	tmpConfigPath := filepath.Join(tmpDir, "config.json")

	t.Run("should successfully set model for Gemini", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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

		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			// Sin inicializar AIConfig
		}
		assert.NoError(t, config.SaveConfig(cfg))

		app, cleanup := setupTestApp(translations, cfg)
		defer cleanup()

		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"config", "set-ai-model", "gemini", "gemini-1.5-flash"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, config.ModelGeminiV15Flash, loadedCfg.AIConfig.Models[config.AIGemini])
	})

	t.Run("should show available models for OpenAI when no model provided", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: map[config.AI]config.Model{
					config.AIOpenAI: config.ModelGPTV4o,
				},
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: map[config.AI]config.Model{
					config.AIOpenAI: config.ModelGPTV4o,
				},
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
			AIConfig: config.AIConfig{
				Models: make(map[config.AI]config.Model),
			},
		}
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
