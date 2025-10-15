package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEditTest(t *testing.T) (*config.Config, *i18n.Translations, func()) {
	tempDir := t.TempDir()
	fakeConfigPath := filepath.Join(tempDir, "config.yaml")

	err := os.WriteFile(fakeConfigPath, []byte("language: es"), 0600)
	require.NoError(t, err)

	cfg := &config.Config{
		PathFile: fakeConfigPath,
	}

	translations, err := i18n.NewTranslations("es", "../../../i18n/locales")
	require.NoError(t, err)

	cleanup := func() {
	}

	return cfg, translations, cleanup
}

func createFakeExecutable(t *testing.T, name string, exitCode int) (dir string, logFile string) {
	dir = t.TempDir()
	logFile = filepath.Join(dir, "log.txt")

	scriptContent := fmt.Sprintf(`#!/bin/sh
	echo "$@" > '%s'
	exit %d
	`, logFile, exitCode)

	execPath := filepath.Join(dir, name)
	err := os.WriteFile(execPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	return dir, logFile
}

func TestEditCommand(t *testing.T) {
	factory := &ConfigCommandFactory{}

	t.Run("should create command with correct name and usage", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		// Act
		cmd := factory.newEditCommand(translations, cfg)

		// Assert
		assert.Equal(t, "edit", cmd.Name)
		assert.Equal(t, translations.GetMessage("config_edit_usage", 0, nil), cmd.Usage)
		assert.NotNil(t, cmd.Action)
	})

	t.Run("should use editor from $EDITOR environment variable", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		fakeEditorDir, logFile := createFakeExecutable(t, "my-test-editor", 0)
		originalPath := os.Getenv("PATH")
		t.Setenv("PATH", fakeEditorDir+string(filepath.ListSeparator)+originalPath)
		t.Setenv("EDITOR", "my-test-editor")

		cmd := factory.newEditCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"edit"})

		// Assert
		assert.NoError(t, err)

		logBytes, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Equal(t, cfg.PathFile, strings.TrimSpace(string(logBytes)))
	})

	t.Run("should fallback to nano if $EDITOR is not set", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		fakeNanoDir, logFile := createFakeExecutable(t, "nano", 0)
		t.Setenv("PATH", fakeNanoDir)
		t.Setenv("EDITOR", "")

		cmd := factory.newEditCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"edit"})

		// Assert
		assert.NoError(t, err)
		logBytes, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Equal(t, cfg.PathFile, strings.TrimSpace(string(logBytes)))
	})

	t.Run("should fallback to vim if $EDITOR and nano are not available", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		fakeVimDir, logFile := createFakeExecutable(t, "vim", 0)
		t.Setenv("PATH", fakeVimDir)
		t.Setenv("EDITOR", "")

		cmd := factory.newEditCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"edit"})

		// Assert
		assert.NoError(t, err)
		logBytes, err := os.ReadFile(logFile)
		require.NoError(t, err)
		assert.Equal(t, cfg.PathFile, strings.TrimSpace(string(logBytes)))
	})

	t.Run("should return error if no editor is found", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		t.Setenv("PATH", t.TempDir())
		t.Setenv("EDITOR", "")

		cmd := factory.newEditCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"edit"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ningun editor de texto definido")
	})

	t.Run("should return error if editor fails to run", func(t *testing.T) {
		// Arrange
		cfg, translations, cleanup := setupEditTest(t)
		defer cleanup()

		fakeEditorDir, _ := createFakeExecutable(t, "failing-editor", 1)
		t.Setenv("PATH", fakeEditorDir)
		t.Setenv("EDITOR", "failing-editor")

		cmd := factory.newEditCommand(translations, cfg)

		// Act
		err := cmd.Run(context.Background(), []string{"edit"})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error al abrir el editor")
	})
}
