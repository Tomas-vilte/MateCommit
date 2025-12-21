package models

type TicketInfo struct {
	TicketID    string   `json:"ticket_id"`
	TicketTitle string   `json:"ticket_title"`
	TitleDesc   string   `json:"title_desc"`
	Criteria    []string `json:"criteria"`
}
