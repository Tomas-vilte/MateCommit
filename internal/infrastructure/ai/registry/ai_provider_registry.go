package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// AIProviderFactory define la interfaz para crear servicios de IA
type AIProviderFactory interface {
	// CreateCommitSummarizer crea un servicio para generar sugerencias de commits
	CreateCommitSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (ports.CommitSummarizer, error)

	// CreatePRSummarizer crea un servicio para resumir Pull Requests
	CreatePRSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (ports.PRSummarizer, error)

	// ValidateConfig valida la configuración para este proveedor
	ValidateConfig(cfg *config.Config) error

	// Name retorna el nombre del proveedor
	Name() string
}

// AIProviderRegistry gestiona el registro de proveedores de IA
type AIProviderRegistry struct {
	mu        sync.RWMutex
	factories map[string]AIProviderFactory
}

// NewAIProviderRegistry crea un nuevo registro de proveedores de IA
func NewAIProviderRegistry() *AIProviderRegistry {
	return &AIProviderRegistry{
		factories: make(map[string]AIProviderFactory),
	}
}

// Register registra un nuevo proveedor de IA
func (r *AIProviderRegistry) Register(name string, factory AIProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("proveedor IA '%s' ya esta registrado", name)
	}

	r.factories[name] = factory
	return nil
}

// Get obtiene un factory por nombre
func (r *AIProviderRegistry) Get(name string) (AIProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("proveedor IA '%s' no encontrado en el registro", name)
	}

	return factory, nil
}

// List retorna la lista de proveedores registrados
func (r *AIProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.factories))
	for name := range r.factories {
		providers = append(providers, name)
	}
	return providers
}

// IsRegistered verifica si un proveedor está registrado
func (r *AIProviderRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}
