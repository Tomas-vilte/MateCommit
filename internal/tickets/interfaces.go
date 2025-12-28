package tickets

import "github.com/thomas-vilte/matecommit/internal/models"

type TicketManager interface {
	GetTicketInfo(ticketID string) (*models.TicketInfo, error)
}
