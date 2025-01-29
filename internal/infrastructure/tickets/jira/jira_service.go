package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/httpclient"
	"net/http"
	"regexp"
	"strings"
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
func NewJiraService(cfg *config.Config, client httpclient.HTTPClient) *JiraService {
	return &JiraService{
		baseURL:   cfg.JiraConfig.BaseURL,
		apiKey:    cfg.JiraConfig.APIKey,
		jiraEmail: cfg.JiraConfig.Email,
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
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", s.baseURL, ticketID)
	resp, err := s.makeRequest("GET", url)
	if err != nil {
		return nil, fmt.Errorf("error making request to jira API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Println("error closing response body:", err)
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
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	jiraFields := &JiraFields{
		CustomField: make(map[string]CustomField),
	}

	// Procesar el campo "summary"
	if rawSummary, ok := result.Fields["summary"]; ok {
		var summary string
		if err := json.Unmarshal(rawSummary, &summary); err != nil {
			return nil, fmt.Errorf("error unmarshaling summary: %w", err)
		}
		jiraFields.Summary = summary
	}

	// Procesar el campo "description"
	if rawDescription, ok := result.Fields["description"]; ok {
		var description AtlassianDoc
		if err := json.Unmarshal(rawDescription, &description); err != nil {
			return nil, fmt.Errorf("error unmarshaling description: %w", err)
		}
		jiraFields.Description = description
	}

	// Procesar campos personalizados
	for key, value := range result.Fields {
		if strings.HasPrefix(key, "customfield_") {
			var customField CustomField
			if err := json.Unmarshal(value, &customField); err != nil {
				//fmt.Printf("Error al deserializar el campo %s: %v\n", key, err)
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

	// Extraer criterios del campo personalizado si existe
	if criteriaFieldID != "" {
		criteria, _ = extractCriteriaFromCustomField(fields.CustomField, criteriaFieldID)
	}

	// Si no se encontraron criterios en el campo personalizado, intentar extraerlos de la descripción
	if len(criteria) == 0 {
		criteria, description = extractAndRemoveCriteria(description)
	} else {
		// Si se encontraron criterios en el campo personalizado, limpiar la descripción
		description = removeCriteriaFromDescription(description)
	}

	return criteria, description
}

// makeRequest realiza una solicitud HTTP a la API de Jira.
func (s *JiraService) makeRequest(method, url string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", getBasicAuth(s.jiraEmail, s.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
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
func extractCriteriaFromCustomField(fields map[string]CustomField, fieldID string) ([]string, string) {
	if fieldID == "" {
		return nil, ""
	}

	fieldValue, ok := fields[fieldID]
	if !ok {
		return nil, "" // Campo no existe en el mapa
	}

	var criteriaText string

	// Si el campo es un objeto AtlassianDoc, extraer el texto
	if fieldValue.Type == "doc" {
		criteriaText = parseAtlassianDoc(fieldValue.Content)
	} else if fieldValue.Text != "" {
		// Si el campo es un string, usar el texto directamente
		criteriaText = fieldValue.Text
	} else {
		return nil, "" // No hay contenido válido
	}

	// Eliminar "Acceptance Criteria:" del texto
	criteriaText = strings.ReplaceAll(criteriaText, "Acceptance Criteria:", "")
	criteriaText = strings.ReplaceAll(criteriaText, "Criterio de aceptacion:", "")

	// Dividir el texto en líneas y formatear cada línea como un criterio
	criteriaList := strings.Split(criteriaText, "\n")
	var filteredCriteria []string
	for _, criterion := range criteriaList {
		criterion = strings.TrimSpace(criterion)
		if criterion != "" {
			// Eliminar viñetas o números al inicio de la línea
			if strings.HasPrefix(criterion, "- ") || strings.HasPrefix(criterion, "* ") {
				criterion = strings.TrimPrefix(criterion, "- ")
				criterion = strings.TrimPrefix(criterion, "* ")
			} else if matches := regexp.MustCompile(`^\d+\.\s*`).FindStringSubmatch(criterion); len(matches) > 0 {
				// Eliminar números al inicio de la línea (por ejemplo, "1. ", "2. ")
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
	parseAtlassianDocRecursive(content, &result)
	return strings.TrimSpace(result.String()) // Eliminar espacios y saltos de línea al final
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
					result.WriteString("\n") // Añade un salto de línea solo si hay contenido
				}
			}
		case "listItem":
			if item.Content != nil {
				parseAtlassianDocRecursive(item.Content, result)
			}
			result.WriteString("\n") // Añade un salto de línea al final del elemento de lista
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
	// Dividir el texto en líneas
	lines := strings.Split(text, "\n")
	var criteria []string
	var descriptionLines []string

	// Bandera para indicar si estamos en la sección de criterios
	inCriteriaSection := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Si encontramos "Acceptance Criteria:", activamos la bandera
		if strings.Contains(trimmedLine, "Acceptance Criteria:") || strings.Contains(trimmedLine, "Criterio de aceptacion:") {
			inCriteriaSection = true
			continue // Saltamos la línea que contiene "Acceptance Criteria:"
		}

		// Si estamos en la sección de criterios y la línea comienza con "- " o "* ", la agregamos a los criterios
		if inCriteriaSection && (strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ")) {
			criterion := strings.TrimPrefix(trimmedLine, "- ")
			criterion = strings.TrimPrefix(criterion, "* ")
			criteria = append(criteria, criterion)
		} else {
			// Si no es un criterio, agregamos la línea a la descripción
			descriptionLines = append(descriptionLines, line)
		}
	}

	// Unimos las líneas de la descripción en un solo string
	description := strings.Join(descriptionLines, "\n")

	return criteria, description
}

// removeCriteriaFromDescription elimina los criterios de aceptación de la descripción.
func removeCriteriaFromDescription(description string) string {
	// Patrones para identificar los criterios de aceptación en inglés y español
	patterns := []string{
		"Acceptance criteria:.*(\n.*)*",    // Elimina todo lo que sigue después de "Acceptance criteria:"
		"Criterio de aceptacion:.*(\n.*)*", // Elimina todo lo que sigue después de "Criterio de aceptacion:"
	}

	// Recorremos los patrones y eliminamos las coincidencias
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		description = re.ReplaceAllString(description, "")
	}

	// Eliminamos espacios y saltos de línea adicionales al final
	return strings.TrimSpace(description)
}
