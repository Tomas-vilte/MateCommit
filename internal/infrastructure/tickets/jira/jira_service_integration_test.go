package jira

import (
	"net/http"
	"os"
	"testing"
)

func TestJiraService_GetTicketInfo_Integration(t *testing.T) {
	//t.Skip("skipping integration test")
	// Arrange
	client := &http.Client{}
	service := &JiraService{
		baseURL:   os.Getenv("JIRA_BASE_URL"),
		jiraEmail: os.Getenv("JIRA_EMAIL"),
		apiKey:    os.Getenv("JIRA_API_KEY"),
		client:    client,
	}

	// Act
	ticketInfo, err := service.GetTicketInfo("MAT-13")

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if ticketInfo.TicketID != "MAT-13" {
		t.Errorf("Expected ticket ID MAT-13, got %s", ticketInfo.TicketID)
	}

	if ticketInfo.TicketTitle == "" {
		t.Error("Expected a non-empty ticket title")
	}

	if ticketInfo.Criteria[0] == "" {
		t.Error("Expected a non-empty acceptance criteria")
	}
}
