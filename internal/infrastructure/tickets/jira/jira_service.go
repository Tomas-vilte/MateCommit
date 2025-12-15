package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/httpclient"
)

// Constantes para patrones de criterios de aceptación
const (
	AcceptanceCriteriaEN = "Acceptance Criteria"
	AcceptanceCriteriaES = "Criterio de Aceptación"
)

// JiraService representa el servicio para interactuar con la API de Jira.
type JiraService struct {
	baseURL   string
	apiKey    string
	jiraEmail string
	client    httpclient.HTTPClient
}

// NewJiraService crea una nueva instancia de JiraService.
func NewJiraService(baseURL, apiKey, email string, client httpclient.HTTPClient) *JiraService {
	return &JiraService{
		baseURL:   baseURL,
		apiKey:    apiKey,
		jiraEmail: email,
		client:    client,
	}
}

// JiraFields representa los campos de un ticket de Jira.
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

// GetTicketInfo obtiene la información de un ticket de Jira.
func (s *JiraService) GetTicketInfo(ticketID string) (*models.TicketInfo, error) {
	customFields, err := s.GetCustomFields()
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los campos personalizados: %w", err)
	}

	criteriaFieldID := findCriteriaFieldID(customFields)
	ticketFields, err := s.fetchTicketFields(ticketID)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron obtener los campos de tickets: %w", err)
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

// fetchTicketFields obtiene los campos de un ticket de Jira.
func (s *JiraService) fetchTicketFields(ticketID string) (*JiraFields, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", s.baseURL, ticketID)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("error al realizar una solicitud a la API de Jira: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error al cerrar el cuerpo de la respuesta:", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		// Todo está bien, continuamos con el procesamiento
	case http.StatusNotFound:
		return nil, fmt.Errorf("ticket con ID %s no existe en Jira", ticketID)
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("no autorizado: verifica tus credenciales de Jira")
	case http.StatusInternalServerError:
		return nil, fmt.Errorf("error interno del servidor de Jira")
	default:
		return nil, fmt.Errorf("error inesperado al obtener el ticket: %s", resp.Status)
	}

	var result struct {
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decodificando respuesta: %w", err)
	}

	jiraFields := &JiraFields{
		CustomField: make(map[string]CustomField),
	}

	if rawSummary, ok := result.Fields["summary"]; ok {
		var summary string
		if err := json.Unmarshal(rawSummary, &summary); err != nil {
			return nil, fmt.Errorf("error decodificando resumen: %w", err)
		}
		jiraFields.Summary = summary
	}

	if rawDescription, ok := result.Fields["description"]; ok {
		var description AtlassianDoc
		if err := json.Unmarshal(rawDescription, &description); err != nil {
			return nil, fmt.Errorf("error decodificando descripcion: %w", err)
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

// extractCriteria extrae los criterios de aceptación de los campos del ticket.
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

// makeRequest realiza una solicitud HTTP a la API de Jira.
func (s *JiraService) makeRequest(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando requests: %w", err)
	}

	req.Header.Set("Authorization", getBasicAuth(s.jiraEmail, s.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al realizar la solicitud: %w", err)
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
func extractCriteriaFromCustomField(fields map[string]CustomField, fieldID string) ([]string, string) {
	if fieldID == "" {
		return nil, ""
	}

	fieldValue, ok := fields[fieldID]
	if !ok {
		return nil, "" // Campo no existe en el mapa
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

// GetCustomFields obtiene los campos personalizados de Jira.
func (s *JiraService) GetCustomFields() (map[string]string, error) {
	url := fmt.Sprintf("%s/rest/api/3/field", s.baseURL)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("error al obtener campos personalizados: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error al cerrar el cuerpo de la respuesta:", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error al obtener campos personalizados: %s", resp.Status)
	}

	var fields []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return nil, fmt.Errorf("error al decodificar campos personalizados: %w", err)
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
			// Ignora otros tipos o agrega manejo según sea necesario
		}
	}
}

// extractAndRemoveCriteria extrae y elimina los criterios de aceptación de un texto.
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

// removeCriteriaFromDescription elimina los criterios de aceptación de la descripción.
func removeCriteriaFromDescription(description string) string {
	patterns := []string{
		"Acceptance criteria:.*(\n.*)*",    // Elimina todo lo que sigue después de "Acceptance criteria:"
		"Criterio de aceptacion:.*(\n.*)*", // Elimina todo lo que sigue después de "Criterio de aceptacion:"
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		description = re.ReplaceAllString(description, "")
	}

	return strings.TrimSpace(description)
}
