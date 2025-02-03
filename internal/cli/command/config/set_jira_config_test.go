package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"os"
	"testing"
)

func TestSetJiraConfigCommand(t *testing.T) {
	t.Run("should successfully set Jira configuration", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetJiraConfigCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		baseURL := "https://example.atlassian.net"
		apiKey := "test-api-key"
		email := "user@example.com"

		// Act
		err := app.Run(ctx, []string{"config", "jira", "--base-url", baseURL, "--api-key", apiKey, "--email", email})

		// Assert
		assert.NoError(t, err)

		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, baseURL, loadedCfg.JiraConfig.BaseURL)
		assert.Equal(t, apiKey, loadedCfg.JiraConfig.APIKey)
		assert.Equal(t, email, loadedCfg.JiraConfig.Email)
	})

	t.Run("should fail when missing base URL", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetJiraConfigCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		apiKey := "test-api-key"
		email := "user@example.com"

		// Act
		err := app.Run(ctx, []string{"config", "jira", "--api-key", apiKey, "--email", email})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Todos los campos son requeridos")
	})

	t.Run("should fail when missing API key", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetJiraConfigCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		baseURL := "https://example.atlassian.net"
		email := "user@example.com"

		// Act
		err := app.Run(ctx, []string{"config", "jira", "--base-url", baseURL, "--email", email})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Todos los campos son requeridos")
	})

	t.Run("should fail when missing email", func(t *testing.T) {
		// Arrange
		cfg, translations, _, cleanup := setupConfigTest(t)
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetJiraConfigCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		baseURL := "https://example.atlassian.net"
		apiKey := "test-api-key"

		// Act
		err := app.Run(ctx, []string{"config", "jira", "--base-url", baseURL, "--api-key", apiKey})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Todos los campos son requeridos")
	})

	t.Run("should handle error when saving configuration fails", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		// Hacer que la ruta de configuración sea un directorio para forzar un error de guardado
		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		factory := NewConfigCommandFactory()
		cmd := factory.newSetJiraConfigCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		baseURL := "https://example.atlassian.net"
		apiKey := "test-api-key"
		email := "user@example.com"

		// Act
		err = app.Run(ctx, []string{"config", "jira", "--base-url", baseURL, "--api-key", apiKey, "--email", email})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")
	})
}
