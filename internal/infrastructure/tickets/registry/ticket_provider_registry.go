package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// TicketProviderFactory define la interfaz para crear clientes de tickets
type TicketProviderFactory interface {
	// CreateClient crea un cliente de tickets con la configuración proporcionada
	CreateClient(ctx context.Context, cfg config.TicketProviderConfig, trans *i18n.Translations) (ports.TickerManager, error)

	// ValidateConfig valida la configuración para este proveedor
	ValidateConfig(cfg config.TicketProviderConfig) error

	// Name retorna el nombre del proveedor
	Name() string
}

// TicketProviderRegistry gestiona el registro de proveedores de tickets
type TicketProviderRegistry struct {
	mu        sync.RWMutex
	factories map[string]TicketProviderFactory
}

// NewTicketProviderRegistry crea un nuevo registro de proveedores de tickets
func NewTicketProviderRegistry() *TicketProviderRegistry {
	return &TicketProviderRegistry{
		factories: make(map[string]TicketProviderFactory),
	}
}

// Register registra un nuevo proveedor de tickets
func (r *TicketProviderRegistry) Register(name string, factory TicketProviderFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("proveedor de tickets '%s' ya esta registrado", name)
	}

	r.factories[name] = factory
	return nil
}

// Get obtiene un factory por nombre
func (r *TicketProviderRegistry) Get(name string) (TicketProviderFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("proveedor de tickets '%s' no encontrado en el registro", name)
	}

	return factory, nil
}

// List retorna la lista de proveedores registrados
func (r *TicketProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.factories))
	for name := range r.factories {
		providers = append(providers, name)
	}
	return providers
}

// IsRegistered verifica si un proveedor está registrado
func (r *TicketProviderRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// CreateClientFromConfig crea un cliente de tickets usando la configuración
func (r *TicketProviderRegistry) CreateClientFromConfig(
	ctx context.Context,
	cfg *config.Config,
	trans *i18n.Translations,
) (ports.TickerManager, error) {
	if cfg.ActiveTicketService == "" || !cfg.UseTicket {
		// Return nil if no ticket service is active
		return nil, nil
	}

	ticketCfg, exists := cfg.TicketProviders[cfg.ActiveTicketService]
	if !exists {
		return nil, fmt.Errorf("configuración no encontrada para proveedor de tickets: %s", cfg.ActiveTicketService)
	}

	factory, err := r.Get(cfg.ActiveTicketService)
	if err != nil {
		return nil, err
	}

	if err := factory.ValidateConfig(ticketCfg); err != nil {
		return nil, fmt.Errorf("configuracion de tickets invalida para %s: %w", cfg.ActiveTicketService, err)
	}

	return factory.CreateClient(ctx, ticketCfg, trans)
}
