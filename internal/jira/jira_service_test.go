package jira

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thomas-vilte/matecommit/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient es un mock para httpclient.HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestGetTicketInfo_Success(t *testing.T) {
	mockClient := new(MockHTTPClient)

	service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

	customFieldsResponse := []map[string]interface{}{
		{
			"id":     "customfield_12345",
			"name":   "Acceptance Criteria",
			"custom": true,
		},
	}
	customFieldsJSON, _ := json.Marshal(customFieldsResponse)
	customFieldsResp := httptest.NewRecorder()
	_, err := customFieldsResp.Write(customFieldsJSON)
	if err != nil {
		return
	}
	customFieldsResp.Code = http.StatusOK

	ticketResponse := map[string]interface{}{
		"fields": map[string]interface{}{
			"summary": "Test Ticket Summary",
			"description": map[string]interface{}{
				"type":    "doc",
				"version": 1,
				"content": []map[string]interface{}{
					{
						"type": "paragraph",
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": "This is a test description.",
							},
						},
					},
				},
			},
			"customfield_12345": map[string]interface{}{
				"type": "doc",
				"content": []map[string]interface{}{
					{
						"type": "paragraph",
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": "Acceptance Criteria:\n- Criterion 1\n- Criterion 2",
							},
						},
					},
				},
			},
		},
	}
	ticketJSON, _ := json.Marshal(ticketResponse)
	ticketResp := httptest.NewRecorder()
	_, err = ticketResp.Write(ticketJSON)
	if err != nil {
		return
	}
	ticketResp.Code = http.StatusOK

	mockClient.On("Do", mock.Anything).Return(customFieldsResp.Result(), nil).Once()
	mockClient.On("Do", mock.Anything).Return(ticketResp.Result(), nil).Once()

	ticketInfo, err := service.GetTicketInfo("TEST-123")

	assert.NoError(t, err)

	expectedTicketInfo := &models.TicketInfo{
		TicketID:    "TEST-123",
		TicketTitle: "Test Ticket Summary",
		TitleDesc:   "This is a test description.",
		Criteria:    []string{"Criterion 1", "Criterion 2"},
	}
	assert.Equal(t, expectedTicketInfo, ticketInfo)

	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_CustomFieldsError(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)

	service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("")),
	}, errors.New("internal server error")).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, ticketInfo)
	assert.Contains(t, err.Error(), "failed to get custom fields from Jira")
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_TicketFieldsError(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)

	service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

	customFieldsResponse := `[{"id":"customfield_12345","name":"Acceptance Criteria","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
	}, errors.New("not found")).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, ticketInfo)
	assert.Contains(t, err.Error(), "failed to fetch ticket fields from Jira")
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_ExtractCriteriaFromCustomField(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)

	service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

	customFieldsResponse := `[{"id":"customfield_12345","name":"Acceptance Criteria","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	ticketFieldsResponse := `{"fields":{"summary":"Test Ticket","description":{"type":"doc","version":1,"content":[{"type":"text","text":"This is a test description."}]},"customfield_12345":{"type":"text","text":"Criterion 1\nCriterion 2"}}}`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(ticketFieldsResponse)),
	}, nil).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "TEST-123", ticketInfo.TicketID)
	assert.Equal(t, "Test Ticket", ticketInfo.TicketTitle)
	assert.Equal(t, "This is a test description.", strings.TrimSpace(ticketInfo.TitleDesc))
	assert.Equal(t, []string{"Criterion 1", "Criterion 2"}, ticketInfo.Criteria)
	mockClient.AssertExpectations(t)
}

func TestRemoveCriteriaFromDescription(t *testing.T) {
	// Caso 1: Descripción con criterios de aceptación en inglés
	description := "This is a test description.\nAcceptance criteria:\nCriterion 1\nCriterion 2"
	expectedDescription := "This is a test description."
	result := removeCriteriaFromDescription(description)
	assert.Equal(t, expectedDescription, result, "La descripción no se limpió correctamente (caso en inglés)")

	// Caso 2: Descripción con criterios de aceptación en español
	description = "Esta es una descripción de prueba. Criterio de aceptacion: Criterio 1 Criterio 2"
	expectedDescription = "Esta es una descripción de prueba."
	result = removeCriteriaFromDescription(description)
	assert.Equal(t, expectedDescription, result, "La descripción no se limpió correctamente (caso en español)")

	// Caso 3: Descripción sin criterios de aceptación
	description = "Esta es una descripción sin criterios."
	expectedDescription = "Esta es una descripción sin criterios."
	result = removeCriteriaFromDescription(description)
	assert.Equal(t, expectedDescription, result, "La descripción no debería modificarse si no hay criterios")

	// Caso 4: Descripción con múltiples patrones de criterios (solo el primero debe ser eliminado)
	description = "Descripción con múltiples patrones.\nAcceptance criteria:\nCriterion 1\nCriterio de aceptacion:\nCriterio 2"
	expectedDescription = "Descripción con múltiples patrones."
	result = removeCriteriaFromDescription(description)
	assert.Equal(t, expectedDescription, result, "Solo el primer patrón de criterios debería ser eliminado")
}

func TestParseAtlassianDoc(t *testing.T) {
	// Caso 1: Documento con un párrafo simple
	content := []DocContent{
		{
			Type: "paragraph",
			Content: []DocContent{
				{
					Type: "text",
					Text: "This is a simple paragraph.",
				},
			},
		},
	}
	expected := "This is a simple paragraph."
	result := parseAtlassianDoc(content)
	assert.Equal(t, expected, result, "El párrafo simple no se parseó correctamente")

	// Caso 2: Documento con una lista de ítems
	content = []DocContent{
		{
			Type: "bulletList",
			Content: []DocContent{
				{
					Type: "listItem",
					Content: []DocContent{
						{
							Type: "text",
							Text: "Item 1",
						},
					},
				},
				{
					Type: "listItem",
					Content: []DocContent{
						{
							Type: "text",
							Text: "Item 2",
						},
					},
				},
			},
		},
	}
	expected = "Item 1\nItem 2"
	result = parseAtlassianDoc(content)
	assert.Equal(t, expected, result, "La lista de ítems no se parseó correctamente")

	// Caso 3: Documento con contenido anidado (párrafo dentro de un ítem de lista)
	content = []DocContent{
		{
			Type: "bulletList",
			Content: []DocContent{
				{
					Type: "listItem",
					Content: []DocContent{
						{
							Type: "paragraph",
							Content: []DocContent{
								{
									Type: "text",
									Text: "Nested paragraph in list item.",
								},
							},
						},
					},
				},
			},
		},
	}
	expected = "Nested paragraph in list item."
	result = parseAtlassianDoc(content)
	assert.Equal(t, expected, result, "El contenido anidado no se parseó correctamente")

	// Caso 4: Documento vacío
	content = []DocContent{}
	expected = ""
	result = parseAtlassianDoc(content)
	assert.Equal(t, expected, result, "El documento vacío no se manejó correctamente")
}

func TestGetCustomFields_ErrorStatusCode(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)

	service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Once()

	// Act
	customFields, err := service.GetCustomFields()

	// Assert
	assert.Error(t, err, "Se esperaba un error debido al código de estado no OK")
	assert.Nil(t, customFields, "No debería devolverse ningún campo personalizado")
	assert.Contains(t, err.Error(), "failed to get custom fields: 500 Internal Server Error", "El mensaje de error no coincide")
	mockClient.AssertExpectations(t)
}

func TestMakeRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockClient := new(MockHTTPClient)
		service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

		mockClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"key":"value"}`)),
		}, nil).Once()

		// Act
		resp, err := service.makeRequest("GET", "https://jira.example.com/rest/api/3/issue/TEST-123")

		// Assert
		assert.NoError(t, err, "No se esperaba un error en una solicitud exitosa")
		assert.NotNil(t, resp, "Debería devolverse una respuesta válida")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "El código de estado debería ser 200 OK")
		mockClient.AssertExpectations(t)
	})

	t.Run("ErrorCreatingRequest", func(t *testing.T) {
		// Arrange
		mockClient := new(MockHTTPClient)
		service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

		invalidURL := "://invalid-url"

		// Act
		resp, err := service.makeRequest("GET", invalidURL)

		// Assert
		assert.Error(t, err, "Se esperaba un error al crear la solicitud HTTP")
		assert.Nil(t, resp, "No debería devolverse ninguna respuesta")
		assert.Contains(t, err.Error(), "failed to create jira request", "El mensaje de error no coincide")
	})

	t.Run("ErrorMakingRequest", func(t *testing.T) {
		// Arrange
		mockClient := new(MockHTTPClient)
		service := NewJiraService("https://example.com", "token", "mail.example.com", mockClient)

		mockClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}, errors.New("request failed")).Once()

		// Act
		_, err := service.makeRequest("GET", "https://jira.example.com/rest/api/3/issue/TEST-123")

		// Assert
		assert.Error(t, err, "Se esperaba un error al realizar la solicitud HTTP")
		assert.Contains(t, err.Error(), "failed to perform jira request", "El mensaje de error no coincide")
		mockClient.AssertExpectations(t)
	})
}

func TestExtractAndRemoveCriteria(t *testing.T) {
	// Arrange: Preparamos el input y los valores esperados
	input := `This is a description.
Acceptance Criteria:
- Criterion 1
- Criterion 2
Some additional text.`

	expectedCriteria := []string{"Criterion 1", "Criterion 2"}
	expectedDescription := `This is a description.
Some additional text.`

	// Act: Llamamos a la función que queremos probar
	criteria, description := extractAndRemoveCriteria(input)

	// Assert: Verificamos que los resultados sean los esperados
	assert.Equal(t, expectedCriteria, criteria, "Los criterios extraídos no coinciden con los esperados")
	assert.Equal(t, expectedDescription, description, "La descripción limpia no coincide con la esperada")
}
