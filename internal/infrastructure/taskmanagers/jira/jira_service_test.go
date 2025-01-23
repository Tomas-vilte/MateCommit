package jira

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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
	// Arrange
	mockClient := new(MockHTTPClient)
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock response for custom fields
	customFieldsResponse := `[{"id":"customfield_12345","name":"Acceptance Criteria","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	// Mock response for ticket fields
	ticketFieldsResponse := `{"fields":{"summary":"Test Ticket","description":{"type":"doc","version":1,"content":[{"type":"text","text":"This is a test description."}]}}}`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(ticketFieldsResponse)),
	}, nil).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "TEST-123", ticketInfo.ID)
	assert.Equal(t, "Test Ticket", ticketInfo.Title)
	assert.Equal(t, "This is a test description.", strings.TrimSpace(ticketInfo.Description))
	assert.Empty(t, ticketInfo.Criteria)
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_CustomFieldsError(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock error for custom fields request
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("")),
	}, errors.New("internal server error")).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, ticketInfo)
	assert.Contains(t, err.Error(), "failed to get custom fields")
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_TicketFieldsError(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock response for custom fields
	customFieldsResponse := `[{"id":"customfield_12345","name":"Acceptance Criteria","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	// Mock error for ticket fields request
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("")),
	}, errors.New("not found")).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, ticketInfo)
	assert.Contains(t, err.Error(), "failed to fetch ticket fields")
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_ExtractCriteriaFromCustomField(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock response for custom fields
	customFieldsResponse := `[{"id":"customfield_12345","name":"Acceptance Criteria","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	// Mock response for ticket fields with custom criteria
	ticketFieldsResponse := `{"fields":{"summary":"Test Ticket","description":{"type":"doc","version":1,"content":[{"type":"text","text":"This is a test description."}]},"customfield_12345":"Acceptance criteria:\nCriterion 1\nCriterion 2"}}`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(ticketFieldsResponse)),
	}, nil).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "TEST-123", ticketInfo.ID)
	assert.Equal(t, "Test Ticket", ticketInfo.Title)
	assert.Equal(t, "This is a test description.", strings.TrimSpace(ticketInfo.Description))
	assert.Equal(t, []string{"Criterion 1", "Criterion 2"}, ticketInfo.Criteria)
	mockClient.AssertExpectations(t)
}

func TestGetTicketInfo_ExtractCriteriaFromDescription(t *testing.T) {
	// Arrange
	mockClient := new(MockHTTPClient)
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock response for custom fields
	customFieldsResponse := `[{"id":"customfield_12345","name":"Other Field","custom":true}]`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(customFieldsResponse)),
	}, nil).Once()

	// Mock response for ticket fields with criteria in description
	ticketFieldsResponse := `{"fields":{"summary":"Test Ticket","description":{"type":"doc","version":1,"content":[{"type":"text","text":"This is a test description.\nAcceptance criteria:\nCriterion 1\nCriterion 2"}]}}}`
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(ticketFieldsResponse)),
	}, nil).Once()

	// Act
	ticketInfo, err := service.GetTicketInfo("TEST-123")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "TEST-123", ticketInfo.ID)
	assert.Equal(t, "Test Ticket", ticketInfo.Title)
	assert.Equal(t, "This is a test description.", ticketInfo.Description)
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
	description = "Esta es una descripción de prueba.\nCriterio de aceptacion:\nCriterio 1\nCriterio 2"
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
	expected := "This is a simple paragraph.\n"
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
	expected = "Item 1\nItem 2\n"
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
	expected = "Nested paragraph in list item.\n"
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
	service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

	// Mock response with a non-OK status code
	mockClient.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: http.StatusInternalServerError, // 500 Internal Server Error
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil).Once()

	// Act
	customFields, err := service.GetCustomFields()

	// Assert
	assert.Error(t, err, "Se esperaba un error debido al código de estado no OK")
	assert.Nil(t, customFields, "No debería devolverse ningún campo personalizado")
	assert.Contains(t, err.Error(), "error fetching custom fields: 500 Internal Server Error", "El mensaje de error no coincide")
	mockClient.AssertExpectations(t)
}

func TestExtractCriteriaFromCustomField_EmptyFieldID(t *testing.T) {
	// Arrange
	fields := map[string]string{
		"customfield_12345": "Acceptance criteria:\nCriterion 1\nCriterion 2",
	}
	fieldID := "" // Campo vacío

	// Act
	criteria, description := extractCriteriaFromCustomField(fields, fieldID)

	// Assert
	assert.Nil(t, criteria, "No debería devolverse ningún criterio cuando el fieldID está vacío")
	assert.Empty(t, description, "La descripción debería estar vacía cuando el fieldID está vacío")
}

func TestMakeRequest(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Arrange
		mockClient := new(MockHTTPClient)
		service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

		// Mock response for a successful request
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
		service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

		// Simulamos un error en http.NewRequest pasando una URL inválida.
		invalidURL := "://invalid-url" // URL inválida

		// Act
		resp, err := service.makeRequest("GET", invalidURL)

		// Assert
		assert.Error(t, err, "Se esperaba un error al crear la solicitud HTTP")
		assert.Nil(t, resp, "No debería devolverse ninguna respuesta")
		assert.Contains(t, err.Error(), "error creating request", "El mensaje de error no coincide")
	})

	t.Run("ErrorMakingRequest", func(t *testing.T) {
		// Arrange
		mockClient := new(MockHTTPClient)
		service := NewJiraService("https://jira.example.com", "user", "token", mockClient)

		// Mock error for request
		mockClient.On("Do", mock.Anything).Return(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}, errors.New("request failed")).Once()

		// Act
		_, err := service.makeRequest("GET", "https://jira.example.com/rest/api/3/issue/TEST-123")

		// Assert
		assert.Error(t, err, "Se esperaba un error al realizar la solicitud HTTP")
		assert.Contains(t, err.Error(), "error making request", "El mensaje de error no coincide")
		mockClient.AssertExpectations(t)
	})
}
