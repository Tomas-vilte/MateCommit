package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/stretchr/testify/assert"
)

func setupConfigTest(t *testing.T) (*config.Config, *i18n.Translations, string, func()) {
	tmpDir, err := os.MkdirTemp("", "matecommit-config-test-*")
	assert.NoError(t, err)

	tmpConfigPath := filepath.Join(tmpDir, "config.json")

	cfg := &config.Config{
		PathFile: tmpConfigPath,
		Language: "en",
	}

	translations, err := i18n.NewTranslations("en", "../../i18n/locales")
	assert.NoError(t, err)

	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Error cleaning temporary directory: %v", err)
		}
	}

	return cfg, translations, tmpConfigPath, cleanup
}
