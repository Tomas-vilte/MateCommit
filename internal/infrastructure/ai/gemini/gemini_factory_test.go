package gemini

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProviderFactory(t *testing.T) {
	factory := NewGeminiProviderFactory()
	trans, err := i18n.NewTranslations("en", "../../../i18n/locales/")
	require.NoError(t, err)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "gemini", factory.Name())
	})

	t.Run("ValidateConfig - Valid", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{
				"gemini": {APIKey: "test-key"},
			},
		}
		err := factory.ValidateConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("ValidateConfig - Missing Provider", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}
		err := factory.ValidateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuracion de gemini no encontrada")
	})

	t.Run("ValidateConfig - Missing API Key", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{
				"gemini": {APIKey: ""},
			},
		}
		err := factory.ValidateConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gemini API key es requerida")
	})

	t.Run("CreateServices - Missing API Key Errors", func(t *testing.T) {
		cfg := &config.Config{
			AIProviders: map[string]config.AIProviderConfig{},
		}
		ctx := context.Background()

		_, err := factory.CreateCommitSummarizer(ctx, cfg, trans)
		assert.Error(t, err)

		_, err = factory.CreatePRSummarizer(ctx, cfg, trans)
		assert.Error(t, err)

		_, err = factory.CreateIssueContentGenerator(ctx, cfg, trans)
		assert.Error(t, err)
	})
}
