package providers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/jira"
	"github.com/thomas-vilte/matecommit/internal/ports"
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
