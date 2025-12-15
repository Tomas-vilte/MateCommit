package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// VCSProviderFactory define la interfaz para crear clientes VCS
type VCSProviderFactory interface {
	// CreateClient crea un cliente VCS con las credenciales proporcionadas
	CreateClient(ctx context.Context, owner, repo, token string, trans *i18n.Translations) (ports.VCSClient, error)

	// ValidateConfig valida la configuración para este proveedor
	ValidateConfig(cfg *config.VCSConfig) error

	// Name retorna el nombre del proveedor
	Name() string
}

// VCSProviderRegistry gestiona el registro de proveedores VCS
type VCSProviderRegistry struct {
	mu        sync.RWMutex
	factories map[string]VCSProviderFactory
}

// NewVCSProviderRegistry crea un nuevo registro de proveedores VCS
func NewVCSProviderRegistry() *VCSProviderRegistry {
	return &VCSProviderRegistry{
		factories: make(map[string]VCSProviderFactory),
	}
}

// Register registra un nuevo proveedor VCS
func (r *VCSProviderRegistry) Register(name string, factory VCSProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("proveedor VCS '%s' ya esta registrado", name)
	}

	r.factories[name] = factory
	return nil
}

// Get obtiene un factory por nombre
func (r *VCSProviderRegistry) Get(name string) (VCSProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, errors.NewVCSProviderNotSupportedError(name)
	}

	return factory, nil
}

// List retorna la lista de proveedores registrados
func (r *VCSProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.factories))
	for name := range r.factories {
		providers = append(providers, name)
	}
	return providers
}

// IsRegistered verifica si un proveedor está registrado
func (r *VCSProviderRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// CreateClientFromConfig crea un cliente VCS usando la configuración
// Este método centraliza la lógica duplicada de creación de VCS clients
func (r *VCSProviderRegistry) CreateClientFromConfig(
	ctx context.Context,
	gitService ports.GitService,
	cfg *config.Config,
	trans *i18n.Translations,
) (ports.VCSClient, error) {
	owner, repo, provider, err := gitService.GetRepoInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("error al obtener la info del repo: %w", err)
	}

	vcsConfig, exists := cfg.VCSConfigs[provider]
	if !exists {
		if cfg.ActiveVCSProvider != "" {
			vcsConfig, exists = cfg.VCSConfigs[cfg.ActiveVCSProvider]
			if !exists {
				return nil, errors.NewVCSConfigNotFoundError(cfg.ActiveVCSProvider)
			}
			provider = cfg.ActiveVCSProvider
		} else {
			return nil, errors.NewVCSProviderNotConfiguredError(provider)
		}
	}

	factory, err := r.Get(provider)
	if err != nil {
		return nil, err
	}

	if err := factory.ValidateConfig(&vcsConfig); err != nil {
		return nil, fmt.Errorf("configuracion VCS invalida para %s: %w", provider, err)
	}

	return factory.CreateClient(ctx, owner, repo, vcsConfig.Token, trans)
}
