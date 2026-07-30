package config

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
)

// setupDoctorTestRepo creates a fresh git repo in a temp dir, with global
// git identity explicitly cleared for the duration of the test via
// GIT_CONFIG_GLOBAL/GIT_CONFIG_SYSTEM, so the checks can reliably exercise
// the "git config user.name/email is unset" path regardless of the host's
// real git configuration.
func setupDoctorTestRepo(t *testing.T) func() {
	tmpDir, err := os.MkdirTemp("", "matecommit-doctor-test-*")
	assert.NoError(t, err)

	originalDir, err := os.Getwd()
	assert.NoError(t, err)

	assert.NoError(t, os.Chdir(tmpDir))
	assert.NoError(t, exec.Command("git", "init").Run())

	emptyConfig := tmpDir + "/empty-gitconfig"
	assert.NoError(t, os.WriteFile(emptyConfig, []byte{}, 0644))
	t.Setenv("GIT_CONFIG_GLOBAL", emptyConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", emptyConfig)

	return func() {
		_ = os.Chdir(originalDir)
		_ = os.RemoveAll(tmpDir)
	}
}

func newDoctorTestTranslations(t *testing.T) *i18n.Translations {
	translations, err := i18n.NewTranslations("en", "../../i18n/locales")
	assert.NoError(t, err)
	return translations
}

func TestDoctorCommand_checkGitUserName(t *testing.T) {
	t.Run("errors when unset and no fallback configured", func(t *testing.T) {
		trans := newDoctorTestTranslations(t)
		cleanup := setupDoctorTestRepo(t)
		defer cleanup()

		d := NewDoctorCommand()
		result := d.checkGitUserName(context.Background(), trans, &config.Config{})

		assert.Equal(t, checkStatusError, result.status)
		assert.NotEmpty(t, result.suggestion)
	})

	t.Run("warns instead of erroring when a matecommit fallback identity is configured", func(t *testing.T) {
		trans := newDoctorTestTranslations(t)
		cleanup := setupDoctorTestRepo(t)
		defer cleanup()

		d := NewDoctorCommand()
		cfg := &config.Config{GitFallback: config.GitConfig{UserName: "Fallback Name"}}
		result := d.checkGitUserName(context.Background(), trans, cfg)

		assert.Equal(t, checkStatusWarning, result.status)
		assert.Contains(t, result.message, "Fallback Name")
	})

	t.Run("ok when git user.name is actually configured", func(t *testing.T) {
		trans := newDoctorTestTranslations(t)
		cleanup := setupDoctorTestRepo(t)
		defer cleanup()
		assert.NoError(t, exec.Command("git", "config", "user.name", "Real Name").Run())

		d := NewDoctorCommand()
		result := d.checkGitUserName(context.Background(), trans, &config.Config{})

		assert.Equal(t, checkStatusOK, result.status)
		assert.Contains(t, result.message, "Real Name")
	})
}

func TestDoctorCommand_checkGitUserEmail(t *testing.T) {
	t.Run("warns instead of erroring when a matecommit fallback identity is configured", func(t *testing.T) {
		trans := newDoctorTestTranslations(t)
		cleanup := setupDoctorTestRepo(t)
		defer cleanup()

		d := NewDoctorCommand()
		cfg := &config.Config{GitFallback: config.GitConfig{UserEmail: "fallback@example.com"}}
		result := d.checkGitUserEmail(context.Background(), trans, cfg)

		assert.Equal(t, checkStatusWarning, result.status)
		assert.Contains(t, result.message, "fallback@example.com")
	})
}
