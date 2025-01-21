package registry

import (
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"testing"
)

type mockCommandFactory struct {
}

func (m *mockCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name: "mock-command",
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Run("should register new factory successfully", func(t *testing.T) {
		cfg := &config.Config{}
		translations, err := i18n.NewTranslations("en", "../../../locales")
		assert.NoError(t, err)
		registry := NewRegistry(cfg, translations)
		factory := &mockCommandFactory{}

		// act
		err = registry.Register("test-command", factory)

		// assert
		assert.NoError(t, err)
		assert.Len(t, registry.factories, 1)
		assert.Contains(t, registry.factories, "test-command")
	})

	t.Run("should return error when registering duplicate factory", func(t *testing.T) {
		// arrange
		cfg := &config.Config{}
		translations, err := i18n.NewTranslations("en", "../../../locales")
		assert.NoError(t, err)
		registry := NewRegistry(cfg, translations)
		factory := &mockCommandFactory{}

		// act
		_ = registry.Register("test-command", factory)
		err = registry.Register("test-command", factory)

		// assert
		assert.Error(t, err)
		assert.Len(t, registry.factories, 1)
	})
}

func TestRegistry_CreateCommands(t *testing.T) {
	t.Run("should create commands from registered factories", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{}
		translations, err := i18n.NewTranslations("en", "../../../locales")
		assert.NoError(t, err)
		registry := NewRegistry(cfg, translations)
		factory1 := &mockCommandFactory{}
		factory2 := &mockCommandFactory{}

		_ = registry.Register("command1", factory1)
		_ = registry.Register("command2", factory2)

		// Act
		commands := registry.CreateCommands()

		// Assert
		assert.Len(t, commands, 2)
		assert.Equal(t, "mock-command", commands[0].Name)
		assert.Equal(t, "mock-command", commands[1].Name)
	})

	t.Run("should return empty slice when no factories registered", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{}
		translations, err := i18n.NewTranslations("en", "../../../locales")
		assert.NoError(t, err)
		registry := NewRegistry(cfg, translations)

		// Act
		commands := registry.CreateCommands()

		// Assert
		assert.Empty(t, commands)
	})
}
func TestNewRegistry(t *testing.T) {
	t.Run("should create new registry with empty factories", func(t *testing.T) {
		// Arrange
		cfg := &config.Config{}
		translations, err := i18n.NewTranslations("en", "../../../locales")
		assert.NoError(t, err)

		// Act
		registry := NewRegistry(cfg, translations)

		// Assert
		assert.NotNil(t, registry)
		assert.Empty(t, registry.factories)
		assert.Equal(t, cfg, registry.config)
		assert.Equal(t, translations, registry.t)
	})
}
