package ports

import "github.com/thomas-vilte/matecommit/internal/models"

type TickerManager interface {
	GetTicketInfo(ticketID string) (*models.TicketInfo, error)
}
