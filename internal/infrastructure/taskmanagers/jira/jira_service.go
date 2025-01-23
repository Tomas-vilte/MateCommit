package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/httpclient"
	"net/http"
	"strings"
)

// Constantes para patrones de criterios de aceptación
const (
	AcceptanceCriteriaEN = "Acceptance Criteria"
	AcceptanceCriteriaES = "Criterio de Aceptación"
)

// JiraService representa el servicio para interactuar con la API de Jira.
type JiraService struct {
	BaseURL  string
	Token    string
	Username string
	Client   httpclient.HTTPClient
}

// NewJiraService crea una nueva instancia de JiraService.
func NewJiraService(baseURL, username, token string, client httpclient.HTTPClient) *JiraService {
	return &JiraService{
		BaseURL:  baseURL,
		Token:    token,
		Username: username,
		Client:   client,
	}
}

// JiraFields representa los campos de un ticket de Jira.
type (
	JiraFields struct {
		Summary     string            `json:"summary"`
		Description AtlassianDoc      `json:"description"`
		CustomField map[string]string `json:"customfield"`
	}

	AtlassianDoc struct {
		Type    string       `json:"type"`
		Version int          `json:"version"`
		Content []DocContent `json:"content"`
	}

	DocContent struct {
		Type    string       `json:"type"`
		Text    string       `json:"text,omitempty"`
		Content []DocContent `json:"content,omitempty"`
	}
)

// GetTicketInfo obtiene la información de un ticket de Jira.
func (s *JiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	customFields, err := s.GetCustomFields()
	if err != nil {
		return nil, fmt.Errorf("failed to get custom fields: %w", err)
	}

	criteriaFieldID := findCriteriaFieldID(customFields)
	ticketFields, err := s.fetchTicketFields(ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticket fields: %w", err)
	}

	description := parseAtlassianDoc(ticketFields.Description.Content)
	criteria, description := s.extractCriteria(ticketFields, criteriaFieldID, description)

	ticketInfo := &models.TicketInfo{
		ID:          ticketID,
		Title:       ticketFields.Summary,
		Description: description,
		Criteria:    criteria,
	}

	return ticketInfo, nil
}

// fetchTicketFields obtiene los campos de un ticket de Jira.
func (s *JiraService) fetchTicketFields(ticketID string) (*JiraFields, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", s.BaseURL, ticketID)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error closing response body:", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching ticket: %s", resp.Status)
	}

	var result struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	jiraFields := &JiraFields{
		CustomField: make(map[string]string),
	}

	if rawSummary, ok := result.Fields["summary"]; ok {
		var summary string
		if err := json.Unmarshal(rawSummary, &summary); err != nil {
			return nil, fmt.Errorf("error unmarshaling summary: %w", err)
		}
		jiraFields.Summary = summary
	}

	if rawDescription, ok := result.Fields["description"]; ok {
		var description AtlassianDoc
		if err := json.Unmarshal(rawDescription, &description); err != nil {
			return nil, fmt.Errorf("error unmarshaling description: %w", err)
		}
		jiraFields.Description = description
	}

	for key, value := range result.Fields {
		if strings.HasPrefix(key, "customfield_") {
			var strVal string
			if err := json.Unmarshal(value, &strVal); err != nil {
				continue // Ignorar campos no string
			}
			jiraFields.CustomField[key] = strVal
		}
	}

	return jiraFields, nil
}

// extractCriteria extrae los criterios de aceptación de los campos del ticket.
func (s *JiraService) extractCriteria(fields *JiraFields, criteriaFieldID, description string) ([]string, string) {
	var criteria []string
	if criteriaFieldID != "" {
		criteria, _ = extractCriteriaFromCustomField(fields.CustomField, criteriaFieldID)
	}

	if criteria != nil {
		description = removeCriteriaFromDescription(description)
	} else {
		criteria, description = extractAndRemoveCriteria(description)
	}

	return criteria, description
}

// makeRequest realiza una solicitud HTTP a la API de Jira.
func (s *JiraService) makeRequest(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", getBasicAuth(s.Username, s.Token))
	req.Header.Set("Accept", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return resp, nil
}

// getBasicAuth genera el encabezado de autenticación básica.
func getBasicAuth(username, token string) string {
	credentials := fmt.Sprintf("%s:%s", username, token)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials)))
}

// findCriteriaFieldID busca el ID del campo de criterios de aceptación.
func findCriteriaFieldID(customFields map[string]string) string {
	patterns := []string{AcceptanceCriteriaEN, AcceptanceCriteriaES}

	for id, name := range customFields {
		for _, pattern := range patterns {
			if strings.Contains(name, pattern) {
				return id
			}
		}
	}

	return ""
}

// extractCriteriaFromCustomField extrae los criterios de un campo personalizado.
func extractCriteriaFromCustomField(fields map[string]string, fieldID string) ([]string, string) {
	if fieldID == "" {
		return nil, ""
	}

	criteriaText, ok := fields[fieldID]
	if !ok || criteriaText == "" {
		return nil, ""
	}

	return extractAndRemoveCriteria(criteriaText)
}

// GetCustomFields obtiene los campos personalizados de Jira.
func (s *JiraService) GetCustomFields() (map[string]string, error) {
	url := fmt.Sprintf("%s/rest/api/3/field", s.BaseURL)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("error fetching custom fields: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error closing response body:", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching custom fields: %s", resp.Status)
	}

	var fields []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("error decoding custom fields: %w", err)
	}

	customFields := make(map[string]string)
	for _, field := range fields {
		if fieldType, ok := field["custom"].(bool); ok && fieldType {
			id := field["id"].(string)
			name := field["name"].(string)
			customFields[id] = name
		}
	}

	return customFields, nil
}

// parseAtlassianDoc convierte el contenido de un documento de Atlassian en una cadena de texto.
func parseAtlassianDoc(content []DocContent) string {
	var result strings.Builder
	for _, item := range content {
		switch item.Type {
		case "text":
			result.WriteString(item.Text)
			result.WriteString("\n")
		case "paragraph", "listItem", "bulletList":
			if item.Content != nil {
				result.WriteString(parseAtlassianDoc(item.Content))
			}
		}
	}
	return result.String()
}

// extractAndRemoveCriteria extrae y elimina los criterios de aceptación de un texto.
func extractAndRemoveCriteria(text string) ([]string, string) {
	for _, pattern := range criteriaPatterns {
		if strings.Contains(text, pattern) {
			criteriaText := strings.SplitN(text, pattern, 2)[1]
			criteriaText = strings.TrimSpace(criteriaText)

			criteriaList := strings.Split(criteriaText, "\n")
			var filteredCriteria []string
			for _, criterion := range criteriaList {
				criterion = strings.TrimSpace(criterion)
				if criterion != "" {
					filteredCriteria = append(filteredCriteria, criterion)
				}
			}

			description := strings.SplitN(text, pattern, 2)[0]
			description = strings.TrimSpace(description)

			return filteredCriteria, description
		}
	}

	return nil, text
}

// criteriaPatterns contiene los patrones para identificar los criterios de aceptación.
var criteriaPatterns = []string{
	"Acceptance criteria:",
	"Criterio de aceptacion:",
}

// removeCriteriaFromDescription elimina los criterios de aceptación de la descripción.
func removeCriteriaFromDescription(description string) string {
	for _, pattern := range criteriaPatterns {
		if strings.Contains(description, pattern) {
			description = strings.SplitN(description, pattern, 2)[0]
			description = strings.TrimSpace(description)
			break
		}
	}
	return description
}
