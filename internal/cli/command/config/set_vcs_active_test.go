package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestSetActiveVCSCommand(t *testing.T) {
	t.Run("should successfully set active VCS provider", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.VCSConfigs = map[string]config.VCSConfig{
			"github": {Provider: "github"},
			"gitlab": {Provider: "gitlab"},
		}
		cfg.ActiveVCSProvider = "gitlab"
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetActiveVCSCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"set-active-vcs", "--provider", "github"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "github", loadedCfg.ActiveVCSProvider)
	})

	t.Run("should fail with non-existent provider", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.VCSConfigs = map[string]config.VCSConfig{
			"github": {Provider: "github"},
		}
		cfg.ActiveVCSProvider = "github"
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetActiveVCSCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"set-active-vcs", "--provider", "bitbucket"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), translations.GetMessage("error.vcs_provider_not_configured", 0, map[string]interface{}{
			"Provider": "bitbucket",
		}))
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.Equal(t, "github", loadedCfg.ActiveVCSProvider)
	})

	t.Run("config save error", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		cfg.VCSConfigs = map[string]config.VCSConfig{
			"github": {Provider: "github"},
		}

		factory := NewConfigCommandFactory()
		cmd := factory.newSetActiveVCSCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"set-active-vcs", "--provider", "github"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error al guardar la configuración")
		assert.Equal(t, "github", cfg.ActiveVCSProvider)
	})
}
