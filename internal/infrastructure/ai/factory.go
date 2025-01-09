package ai

import (
	"errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
)

type ProviderFactory struct {
	providers map[string]ports.AIProvider
}

func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		providers: make(map[string]ports.AIProvider),
	}
}

func (f *ProviderFactory) Register(provider ports.AIProvider) {
	f.providers[provider.Name()] = provider
}

func (f *ProviderFactory) Get(name string) (ports.AIProvider, error) {
	provider, exists := f.providers[name]
	if !exists {
		return nil, errors.New("provider no encontrado")
	}
	return provider, nil
}
