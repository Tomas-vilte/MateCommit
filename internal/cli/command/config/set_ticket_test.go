package config

import (
	"context"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"os"
	"path/filepath"
	"testing"
)

func TestSetTicketCommand(t *testing.T) {
	t.Run("should successfully enable ticket", func(t *testing.T) {
		// Arrange
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		defer cleanup()

		cmd := NewConfigCommandFactory().newSetTicketCommand(translations, cfg)

		// Act
		err := cmd.Commands[1].Action(context.Background(), &cli.Command{})

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
		cfg, translations, tmpConfigPath, cleanup := setupConfigTest(t)
		cfg.UseTicket = true
		defer cleanup()

		cmd := NewConfigCommandFactory().newSetTicketCommand(translations, cfg)

		// act
		err := cmd.Commands[0].Action(context.Background(), &cli.Command{})

		// assert
		assert.NoError(t, err)
		assert.False(t, cfg.UseTicket)

		loadedCfg, err := config.LoadConfig(tmpConfigPath)
		assert.NoError(t, err)
		assert.False(t, loadedCfg.UseTicket)
	})

	t.Run("save error config with enable", func(t *testing.T) {
		// arrange
		tempDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tempDir, "config.json")
		err = os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		// config inicial
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.PathFile = tmpConfigPath
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetTicketCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// act
		err = app.Run(ctx, []string{"config", "ticket", "disable"})

		// assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")

	})

	t.Run("save error config with disable", func(t *testing.T) {
		// arrange
		tempDir, err := os.MkdirTemp("", "matecommit-test-*")
		assert.NoError(t, err)
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				return
			}
		}()

		tmpConfigPath := filepath.Join(tempDir, "config.json")
		err = os.Mkdir(tmpConfigPath, 0755)
		assert.NoError(t, err)

		// config inicial
		cfg, translations, _, cleanup := setupConfigTest(t)
		cfg.PathFile = tmpConfigPath
		defer cleanup()

		factory := NewConfigCommandFactory()
		cmd := factory.newSetTicketCommand(translations, cfg)

		app := &cli.Command{Commands: []*cli.Command{cmd}}
		ctx := context.Background()

		// act
		err = app.Run(ctx, []string{"config", "ticket", "enable"})

		// assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Error al guardar la configuración")

	})
}
