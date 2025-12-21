package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/tickets/jira"
)

// NewTicketManager creates a TicketManager based on the configured provider
func NewTicketManager(ctx context.Context, cfg *config.Config) (ports.TickerManager, error) {
	if cfg.ActiveTicketService == "" || !cfg.UseTicket {
		return nil, nil // Tickets disabled
	}

	ticketCfg, exists := cfg.TicketProviders[cfg.ActiveTicketService]
	if !exists {
		return nil, fmt.Errorf("ticket provider '%s' not configured", cfg.ActiveTicketService)
	}

	switch cfg.ActiveTicketService {
	case "jira":
		return jira.NewJiraService(ticketCfg.BaseURL, ticketCfg.APIKey, ticketCfg.Email, &http.Client{}), nil
	default:
		return nil, fmt.Errorf("ticket provider '%s' not supported", cfg.ActiveTicketService)
	}
}
