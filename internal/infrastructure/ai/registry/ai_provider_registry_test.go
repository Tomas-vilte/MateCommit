package registry

import (
	"context"
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAIProviderFactory es un mock para testing
type MockAIProviderFactory struct {
	name string
}

func (m *MockAIProviderFactory) CreateCommitSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.CommitSummarizer, error) {
	return nil, nil
}

func (m *MockAIProviderFactory) CreatePRSummarizer(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.PRSummarizer, error) {
	return nil, nil
}

func (m *MockAIProviderFactory) CreateIssueContentGenerator(_ context.Context, _ *config.Config, _ *i18n.Translations) (ports.IssueContentGenerator, error) {
	return nil, nil
}

func (m *MockAIProviderFactory) ValidateConfig(_ *config.Config) error {
	return nil
}

func (m *MockAIProviderFactory) Name() string {
	return m.name
}

func TestNewAIProviderRegistry(t *testing.T) {
	registry := NewAIProviderRegistry()
	assert.NotNil(t, registry)
	assert.Empty(t, registry.List())
}

func TestRegister(t *testing.T) {
	registry := NewAIProviderRegistry()
	mockFactory := &MockAIProviderFactory{name: "test-provider"}

	err := registry.Register("test", mockFactory)
	assert.NoError(t, err)
	assert.True(t, registry.IsRegistered("test"))

	err = registry.Register("test", mockFactory)
	assert.Error(t, err)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ya esta registrado")
}

func TestGet(t *testing.T) {
	registry := NewAIProviderRegistry()
	mockFactory := &MockAIProviderFactory{name: "test-provider"}

	_, err := registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no encontrado en el registro")

	err = registry.Register("test", mockFactory)
	require.NoError(t, err)

	factory, err := registry.Get("test")
	assert.NoError(t, err)
	assert.NotNil(t, factory)
	assert.Equal(t, "test-provider", factory.Name())
}

func TestList(t *testing.T) {
	registry := NewAIProviderRegistry()

	assert.Empty(t, registry.List())

	_ = registry.Register("gemini", &MockAIProviderFactory{name: "gemini"})
	_ = registry.Register("openai", &MockAIProviderFactory{name: "openai"})
	_ = registry.Register("claude", &MockAIProviderFactory{name: "claude"})

	providers := registry.List()
	assert.Len(t, providers, 3)
	assert.Contains(t, providers, "gemini")
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "claude")
}

func TestIsRegistered(t *testing.T) {
	registry := NewAIProviderRegistry()
	mockFactory := &MockAIProviderFactory{name: "test"}

	assert.False(t, registry.IsRegistered("test"))

	_ = registry.Register("test", mockFactory)
	assert.True(t, registry.IsRegistered("test"))
	assert.False(t, registry.IsRegistered("other"))
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewAIProviderRegistry()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			mockFactory := &MockAIProviderFactory{name: "provider"}
			_ = registry.Register(string(rune('a'+id)), mockFactory)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	providers := registry.List()
	assert.Len(t, providers, 10)
}
