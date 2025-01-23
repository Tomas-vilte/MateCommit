package models

type TicketInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Criteria    []string `json:"criteria"`
}
