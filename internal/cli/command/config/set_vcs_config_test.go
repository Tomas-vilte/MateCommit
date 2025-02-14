package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestSetVCSConfigCommand(t *testing.T) {
	t.Run("should successfully set new VCS config", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetVCSConfigCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"set-vcs", "--provider", "github", "--token", "abc123", "--owner", "testuser", "--repo", "testrepo"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)

		vcsConfig, exists := loadedCfg.VCSConfigs["github"]
		assert.True(t, exists)
		assert.Equal(t, "github", vcsConfig.Provider)
		assert.Equal(t, "abc123", vcsConfig.Token)
		assert.Equal(t, "testuser", vcsConfig.Owner)
		assert.Equal(t, "testrepo", vcsConfig.Repo)
	})

	t.Run("should update existing VCS config", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.VCSConfigs = map[string]config.VCSConfig{
			"github": {
				Provider: "github",
				Token:    "old-token",
				Owner:    "old-owner",
				Repo:     "old-repo",
			},
		}
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetVCSConfigCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"set-vcs", "--provider", "github", "--owner", "new-owner"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)

		vcsConfig, exists := loadedCfg.VCSConfigs["github"]
		assert.True(t, exists)
		assert.Equal(t, "github", vcsConfig.Provider)
		assert.Equal(t, "old-token", vcsConfig.Token)
		assert.Equal(t, "new-owner", vcsConfig.Owner)
		assert.Equal(t, "old-repo", vcsConfig.Repo)
	})

	t.Run("should create VCSConfigs map if nil", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cfg.VCSConfigs = nil
		assert.NoError(t, config.SaveConfig(cfg))

		factory := NewConfigCommandFactory()
		cmd := factory.newSetVCSConfigCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err := app.Run(ctx, []string{"set-vcs", "--provider", "github", "--token", "abc123"})

		// Assert
		assert.NoError(t, err)
		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)

		vcsConfig, exists := loadedCfg.VCSConfigs["github"]
		assert.True(t, exists)
		assert.Equal(t, "github", vcsConfig.Provider)
		assert.Equal(t, "abc123", vcsConfig.Token)
	})

	t.Run("config save error", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		err := os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		factory := NewConfigCommandFactory()
		cmd := factory.newSetVCSConfigCommand(translations, cfg)

		app := cmd
		ctx := context.Background()

		// Act
		err = app.Run(ctx, []string{"set-vcs", "--provider", "github", "--token", "abc123"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")

		if cfg.VCSConfigs != nil {
			vcsConfig, exists := cfg.VCSConfigs["github"]
			assert.True(t, exists)
			assert.Equal(t, "github", vcsConfig.Provider)
			assert.Equal(t, "abc123", vcsConfig.Token)
		}
	})
}
