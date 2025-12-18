package di

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai/registry"
	ticketregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/registry"
	vcsregistry "github.com/Tomas-vilte/MateCommit/internal/infrastructure/vcs/registry"
	"github.com/Tomas-vilte/MateCommit/internal/services"
)

// Container gestiona las dependencias de la aplicación
type Container struct {
	config       *config.Config
	translations *i18n.Translations

	// Registries
	aiRegistry     *registry.AIProviderRegistry
	vcsRegistry    *vcsregistry.VCSProviderRegistry
	ticketRegistry *ticketregistry.TicketProviderRegistry

	// Services (lazy initialized)
	gitService    ports.GitService
	commitService ports.CommitService
	prService     ports.PRService
}

// NewContainer crea un nuevo contenedor de dependencias
func NewContainer(cfg *config.Config, trans *i18n.Translations) *Container {
	return &Container{
		config:         cfg,
		translations:   trans,
		aiRegistry:     registry.NewAIProviderRegistry(),
		vcsRegistry:    vcsregistry.NewVCSProviderRegistry(),
		ticketRegistry: ticketregistry.NewTicketProviderRegistry(),
	}
}

// RegisterAIProvider registra un proveedor de IA
func (c *Container) RegisterAIProvider(name string, factory registry.AIProviderFactory) error {
	return c.aiRegistry.Register(name, factory)
}

// RegisterVCSProvider registra un proveedor VCS
func (c *Container) RegisterVCSProvider(name string, factory vcsregistry.VCSProviderFactory) error {
	return c.vcsRegistry.Register(name, factory)
}

// RegisterTicketProvider registra un proveedor de tickets
func (c *Container) RegisterTicketProvider(name string, factory ticketregistry.TicketProviderFactory) error {
	return c.ticketRegistry.Register(name, factory)
}

// SetGitService establece el servicio Git
func (c *Container) SetGitService(gitService ports.GitService) {
	c.gitService = gitService
}

// GetGitService retorna el servicio Git
func (c *Container) GetGitService() ports.GitService {
	return c.gitService
}

// GetAIRegistry retorna el registro de proveedores AI
func (c *Container) GetAIRegistry() *registry.AIProviderRegistry {
	return c.aiRegistry
}

// GetVCSRegistry retorna el registro de proveedores VCS
func (c *Container) GetVCSRegistry() *vcsregistry.VCSProviderRegistry {
	return c.vcsRegistry
}

// GetTicketRegistry retorna el registro de proveedores de tickets
func (c *Container) GetTicketRegistry() *ticketregistry.TicketProviderRegistry {
	return c.ticketRegistry
}

// GetCommitService retorna el servicio de commits (lazy initialization)
func (c *Container) GetCommitService(ctx context.Context) (ports.CommitService, error) {
	if c.commitService != nil {
		return c.commitService, nil
	}

	if c.gitService == nil {
		return nil, fmt.Errorf("servicio git no creado")
	}

	var aiProvider ports.CommitSummarizer
	if c.config.AIConfig.ActiveAI != "" {
		aiFactory, err := c.aiRegistry.Get(string(c.config.AIConfig.ActiveAI))
		if err == nil {
			aiProvider, err = aiFactory.CreateCommitSummarizer(ctx, c.config, c.translations)
			if err != nil {
				// Log warning pero continuar sin AI
				aiProvider = nil
			}
		}
	}

	ticketManager, err := c.ticketRegistry.CreateClientFromConfig(ctx, c.config, c.translations)
	if err != nil {
		ticketManager = nil
	}

	c.commitService = services.NewCommitService(
		c.gitService,
		aiProvider,
		ticketManager,
		nil,
		c.config,
		c.translations,
	)

	return c.commitService, nil
}

// GetPRSummarizer retorna el servicio de IA para resumir PRs (lazy initialization)
func (c *Container) GetPRSummarizer(ctx context.Context) (ports.PRSummarizer, error) {
	if c.config.AIConfig.ActiveAI == "" {
		return nil, fmt.Errorf("no hay IA activa configurada")
	}

	aiFactory, err := c.aiRegistry.Get(string(c.config.AIConfig.ActiveAI))
	if err != nil {
		return nil, fmt.Errorf("proveedor de IA '%s' no encontrado: %w", c.config.AIConfig.ActiveAI, err)
	}

	aiSummarizer, err := aiFactory.CreatePRSummarizer(ctx, c.config, c.translations)
	if err != nil {
		return nil, fmt.Errorf("error al crear el servicio de IA para PRs: %w", err)
	}

	return aiSummarizer, nil
}

// GetPRService retorna el servicio de PRs (lazy initialization)
func (c *Container) GetPRService(ctx context.Context) (ports.PRService, error) {
	if c.prService != nil {
		return c.prService, nil
	}

	if c.gitService == nil {
		return nil, fmt.Errorf("servicio git no creado")
	}

	var aiSummarizer ports.PRSummarizer
	aiSummarizer, err := c.GetPRSummarizer(ctx)
	if err != nil {
		aiSummarizer = nil
	}

	vcsClient, err := c.vcsRegistry.CreateClientFromConfig(ctx, c.gitService, c.config, c.translations)
	if err != nil {
		return nil, fmt.Errorf("error al crear cliente VCS: %w", err)
	}

	c.prService = services.NewPRService(vcsClient, aiSummarizer, c.translations, c.config)

	return c.prService, nil
}

// GetConfig retorna la configuración
func (c *Container) GetConfig() *config.Config {
	return c.config
}

// GetTranslations retorna las traducciones
func (c *Container) GetTranslations() *i18n.Translations {
	return c.translations
}
