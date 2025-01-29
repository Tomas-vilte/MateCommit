package ports

import "github.com/Tomas-vilte/MateCommit/internal/domain/models"

type TickerManager interface {
	GetTicketInfo(ticketID string) (*models.TicketInfo, error)
}
