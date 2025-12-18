package jira

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
)

// JiraProviderFactory implementa TicketProviderFactory para Jira
type JiraProviderFactory struct{}

// NewJiraProviderFactory crea una nueva factory para Jira
func NewJiraProviderFactory() *JiraProviderFactory {
	return &JiraProviderFactory{}
}

// CreateClient crea un cliente Jira
func (f *JiraProviderFactory) CreateClient(
	_ context.Context,
	cfg config.TicketProviderConfig,
	_ *i18n.Translations,
) (ports.TickerManager, error) {
	return NewJiraService(cfg.BaseURL, cfg.APIKey, cfg.Email, &http.Client{}), nil
}

// ValidateConfig valida la configuración de Jira
func (f *JiraProviderFactory) ValidateConfig(cfg config.TicketProviderConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("jira base URL es requerida")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("jira API key es requerida")
	}
	if cfg.Email == "" {
		return fmt.Errorf("jira email es requerido")
	}
	return nil
}

// Name retorna el nombre del proveedor
func (f *JiraProviderFactory) Name() string {
	return "jira"
}
