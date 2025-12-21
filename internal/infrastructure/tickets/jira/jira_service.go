package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	domainErrors "github.com/Tomas-vilte/MateCommit/internal/domain/errors"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// Constants for acceptance criteria patterns
const (
	AcceptanceCriteriaEN = "Acceptance Criteria"
	AcceptanceCriteriaES = "Criterio de Aceptación"
)

// httpClient is a minimal interface for testing purposes
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// JiraService represents the service to interact with the Jira API.
type JiraService struct {
	baseURL   string
	apiKey    string
	jiraEmail string
	client    httpClient
}

// NewJiraService creates a new instance of JiraService.
func NewJiraService(baseURL, apiKey, email string, client httpClient) *JiraService {
	return &JiraService{
		baseURL:   baseURL,
		apiKey:    apiKey,
		jiraEmail: email,
		client:    client,
	}
}

// JiraFields represents the fields of a Jira ticket.
type (
	JiraFields struct {
		Summary     string                 `json:"summary"`
		Description AtlassianDoc           `json:"description"`
		CustomField map[string]CustomField `json:"customfield"`
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

	CustomField struct {
		Type    string       `json:"type"`
		Text    string       `json:"text"`
		Content []DocContent `json:"content,omitempty"`
	}
)

// GetTicketInfo gets the information of a Jira ticket.
func (s *JiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	customFields, err := s.GetCustomFields()
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to get custom fields from Jira", err)
	}

	criteriaFieldID := findCriteriaFieldID(customFields)
	ticketFields, err := s.fetchTicketFields(ticketID)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to fetch ticket fields from Jira", err)
	}

	description := parseAtlassianDoc(ticketFields.Description.Content)
	criteria, description := s.extractCriteria(ticketFields, criteriaFieldID, description)

	ticketInfo := &models.TicketInfo{
		TicketID:    ticketID,
		TicketTitle: ticketFields.Summary,
		TitleDesc:   description,
		Criteria:    criteria,
	}

	return ticketInfo, nil
}

// fetchTicketFields gets the fields of a Jira ticket.
func (s *JiraService) fetchTicketFields(ticketID string) (*JiraFields, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", s.baseURL, ticketID)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to make request to Jira API", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, fmt.Sprintf("ticket %s not found in Jira", ticketID), nil)
	case http.StatusUnauthorized:
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "unauthorized: check Jira credentials", nil)
	case http.StatusInternalServerError:
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "jira internal server error", nil)
	default:
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, fmt.Sprintf("unexpected error fetching ticket: %s", resp.Status), nil)
	}

	var result struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to decode jira response", err)
	}

	jiraFields := &JiraFields{
		CustomField: make(map[string]CustomField),
	}

	if rawSummary, ok := result.Fields["summary"]; ok {
		var summary string
		if err := json.Unmarshal(rawSummary, &summary); err != nil {
			return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to unmarshal ticket summary", err)
		}
		jiraFields.Summary = summary
	}

	if rawDescription, ok := result.Fields["description"]; ok {
		var description AtlassianDoc
		if err := json.Unmarshal(rawDescription, &description); err != nil {
			return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to unmarshal ticket description", err)
		}
		jiraFields.Description = description
	}

	for key, value := range result.Fields {
		if strings.HasPrefix(key, "customfield_") {
			var customField CustomField
			if err := json.Unmarshal(value, &customField); err != nil {
				continue
			}
			jiraFields.CustomField[key] = customField
		}
	}

	return jiraFields, nil
}

// extractCriteria extracts acceptance criteria from the ticket fields.
func (s *JiraService) extractCriteria(fields *JiraFields, criteriaFieldID, description string) ([]string, string) {
	var criteria []string

	if criteriaFieldID != "" {
		criteria, _ = extractCriteriaFromCustomField(fields.CustomField, criteriaFieldID)
	}

	if len(criteria) == 0 {
		criteria, description = extractAndRemoveCriteria(description)
	} else {
		description = removeCriteriaFromDescription(description)
	}

	return criteria, description
}

// makeRequest performs an HTTP request to the Jira API.
func (s *JiraService) makeRequest(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to create jira request", err)
	}

	req.Header.Set("Authorization", getBasicAuth(s.jiraEmail, s.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to perform jira request", err)
	}

	return resp, nil
}

// getBasicAuth generates the basic authentication header.
func getBasicAuth(username, token string) string {
	credentials := fmt.Sprintf("%s:%s", username, token)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials)))
}

// findCriteriaFieldID finds the ID of the acceptance criteria field.
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

// extractCriteriaFromCustomField extracts criteria from a custom field.
func extractCriteriaFromCustomField(fields map[string]CustomField, fieldID string) ([]string, string) {
	if fieldID == "" {
		return nil, ""
	}

	fieldValue, ok := fields[fieldID]
	if !ok {
		return nil, "" // Field does not exist in the map
	}

	var criteriaText string

	if fieldValue.Type == "doc" {
		criteriaText = parseAtlassianDoc(fieldValue.Content)
	} else if fieldValue.Text != "" {
		criteriaText = fieldValue.Text
	} else {
		return nil, ""
	}

	criteriaText = strings.ReplaceAll(criteriaText, "Acceptance Criteria:", "")
	criteriaText = strings.ReplaceAll(criteriaText, "Criterio de aceptacion:", "")

	criteriaList := strings.Split(criteriaText, "\n")
	var filteredCriteria []string
	for _, criterion := range criteriaList {
		criterion = strings.TrimSpace(criterion)
		if criterion != "" {
			if strings.HasPrefix(criterion, "- ") || strings.HasPrefix(criterion, "* ") {
				criterion = strings.TrimPrefix(criterion, "- ")
				criterion = strings.TrimPrefix(criterion, "* ")
			} else if matches := regexp.MustCompile(`^\d+\.\s*`).FindStringSubmatch(criterion); len(matches) > 0 {
				criterion = strings.TrimPrefix(criterion, matches[0])
			}
			filteredCriteria = append(filteredCriteria, criterion)
		}
	}

	return filteredCriteria, criteriaText
}

// GetCustomFields gets the custom fields from Jira.
func (s *JiraService) GetCustomFields() (map[string]string, error) {
	url := fmt.Sprintf("%s/rest/api/3/field", s.baseURL)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to fetch custom fields", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, fmt.Sprintf("failed to get custom fields: %s", resp.Status), nil)
	}

	var fields []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, domainErrors.NewAppError(domainErrors.TypeInternal, "failed to decode custom fields", err)
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

// parseAtlassianDoc converts an Atlassian document content into a string.
func parseAtlassianDoc(content []DocContent) string {
	var result strings.Builder
	parseAtlassianDocRecursive(content, &result)
	return strings.TrimSpace(result.String())
}

func parseAtlassianDocRecursive(content []DocContent, result *strings.Builder) {
	for _, item := range content {
		switch item.Type {
		case "text":
			result.WriteString(item.Text)
		case "paragraph":
			if item.Content != nil {
				parseAtlassianDocRecursive(item.Content, result)
				if len(item.Content) > 0 {
					result.WriteString("\n")
				}
			}
		case "listItem":
			if item.Content != nil {
				parseAtlassianDocRecursive(item.Content, result)
			}
			result.WriteString("\n")
		case "bulletList", "orderedList":
			if item.Content != nil {
				parseAtlassianDocRecursive(item.Content, result)
			}
		default:
			// Ignore other types or add handling as necessary
		}
	}
}

// extractAndRemoveCriteria extracts and removes acceptance criteria from text.
func extractAndRemoveCriteria(text string) ([]string, string) {
	lines := strings.Split(text, "\n")
	var criteria []string
	var descriptionLines []string

	inCriteriaSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.Contains(trimmedLine, "Acceptance Criteria:") || strings.Contains(trimmedLine, "Criterio de aceptacion:") {
			inCriteriaSection = true
			continue
		}

		if inCriteriaSection && (strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ")) {
			criterion := strings.TrimPrefix(trimmedLine, "- ")
			criterion = strings.TrimPrefix(criterion, "* ")
			criteria = append(criteria, criterion)
		} else {
			descriptionLines = append(descriptionLines, line)
		}
	}

	description := strings.Join(descriptionLines, "\n")

	return criteria, description
}

// removeCriteriaFromDescription removes acceptance criteria from the description.
func removeCriteriaFromDescription(description string) string {
	patterns := []string{
		"Acceptance criteria:.*(\n.*)*",    // Removes everything following "Acceptance criteria:"
		"Criterio de aceptacion:.*(\n.*)*", // Removes everything following "Criterio de aceptacion:"
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		description = re.ReplaceAllString(description, "")
	}

	return strings.TrimSpace(description)
}
