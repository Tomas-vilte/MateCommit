package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigError(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		innerErr := errors.New("inner error")
		err := NewConfigError("api_key", "invalid key", innerErr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key")
		assert.Contains(t, err.Error(), "invalid key")
		assert.Contains(t, err.Error(), "inner error")

		assert.Equal(t, innerErr, errors.Unwrap(err))
	})

	t.Run("without wrapped error", func(t *testing.T) {
		err := NewConfigError("api_key", "missing key", nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "api_key")
		assert.Contains(t, err.Error(), "missing key")
		assert.Nil(t, errors.Unwrap(err))
	})
}

func TestAIProviderNotFoundError(t *testing.T) {
	err := NewAIProviderNotFoundError("openai")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "openai")
	assert.Contains(t, err.Error(), "no encontrado")

	var aiErr *AIProviderNotFoundError
	assert.True(t, errors.As(err, &aiErr))
	assert.Equal(t, "openai", aiErr.Provider)
}

func TestAIProviderNotConfiguredError(t *testing.T) {
	t.Run("with reason", func(t *testing.T) {
		err := NewAIProviderNotConfiguredError("gemini", "API key missing")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no configurado")
		assert.Contains(t, err.Error(), "API key missing")
	})

	t.Run("without reason", func(t *testing.T) {
		err := NewAIProviderNotConfiguredError("gemini", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gemini")
		assert.Contains(t, err.Error(), "no configurado")
	})
}

func TestVCSConfigNotFoundError(t *testing.T) {
	err := NewVCSConfigNotFoundError("github")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, err.Error(), "no encontrado")

	var vcsErr *VCSConfigNotFoundError
	assert.True(t, errors.As(err, &vcsErr))
	assert.Equal(t, "github", vcsErr.Provider)
}

func TestVCSProviderNotConfiguredError(t *testing.T) {
	err := NewVCSProviderNotConfiguredError("gitlab")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gitlab")
	assert.Contains(t, err.Error(), "detectado pero no configurado")
}

func TestVCSProviderNotSupportedError(t *testing.T) {
	err := NewVCSProviderNotSupportedError("bitbucket")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bitbucket")
	assert.Contains(t, err.Error(), "no es soportado")
}

func TestErrorTypeAssertions(t *testing.T) {
	aiNotFound := NewAIProviderNotFoundError("test")
	aiNotConfigured := NewAIProviderNotConfiguredError("test", "reason")
	vcsNotFound := NewVCSConfigNotFoundError("test")

	// AIProviderNotFoundError
	var aiNotFoundErr *AIProviderNotFoundError
	assert.True(t, errors.As(aiNotFound, &aiNotFoundErr))
	assert.False(t, errors.As(aiNotConfigured, &aiNotFoundErr))

	// AIProviderNotConfiguredError
	var aiNotConfiguredErr *AIProviderNotConfiguredError
	assert.True(t, errors.As(aiNotConfigured, &aiNotConfiguredErr))
	assert.False(t, errors.As(aiNotFound, &aiNotConfiguredErr))

	// VCSConfigNotFoundError
	var vcsNotFoundErr *VCSConfigNotFoundError
	assert.True(t, errors.As(vcsNotFound, &vcsNotFoundErr))
	assert.False(t, errors.As(aiNotFound, &vcsNotFoundErr))
}
