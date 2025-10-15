package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func setupConfigTest(t *testing.T) (*config.Config, *i18n.Translations, string, func()) {
	tmpDir, err := os.MkdirTemp("", "matecommit-config-test-*")
	assert.NoError(t, err)

	tmpConfigPath := filepath.Join(tmpDir, "config.json")

	cfg := &config.Config{
		PathFile: tmpConfigPath,
		Language: "es",
	}

	translations, err := i18n.NewTranslations("es", "../../../../internal/i18n/locales")
	assert.NoError(t, err)

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Error al limpiar directorio temporal: %v", err)
		}
	}

	return cfg, translations, tmpConfigPath, cleanup
}
