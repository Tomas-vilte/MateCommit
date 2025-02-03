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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
		}

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
		cfg := &config.Config{
			PathFile: tmpConfigPath,
			Language: "es",
		}

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
