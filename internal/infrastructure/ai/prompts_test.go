package ai

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPRPromptTemplate(t *testing.T) {
	t.Run("returns English template for 'en'", func(t *testing.T) {
		// Arrange
		lang := "en"

		// Act
		result := GetPRPromptTemplate(lang)

		// Assert
		assert.Equal(t, prPromptTemplateEN, result)
	})

	t.Run("returns Spanish template for 'es'", func(t *testing.T) {
		// Arrange
		lang := "es"

		// Act
		result := GetPRPromptTemplate(lang)

		// Assert
		assert.Equal(t, prPromptTemplateES, result)
	})

	t.Run("defaults to English for unknown language", func(t *testing.T) {
		// Arrange
		lang := "fr"

		// Act
		result := GetPRPromptTemplate(lang)

		// Assert
		assert.Equal(t, prPromptTemplateEN, result)
	})

	t.Run("defaults to English for empty language", func(t *testing.T) {
		// Arrange
		lang := ""

		// Act
		result := GetPRPromptTemplate(lang)

		// Assert
		assert.Equal(t, prPromptTemplateEN, result)
	})
}

func TestGetCommitPromptTemplate(t *testing.T) {
	t.Run("returns English template with ticket", func(t *testing.T) {
		// Arrange
		lang := "en"
		hasTicket := true

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithTicketEN, result)
	})

	t.Run("returns English template without ticket", func(t *testing.T) {
		// Arrange
		lang := "en"
		hasTicket := false

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketEN, result)
	})

	t.Run("returns Spanish template with ticket", func(t *testing.T) {
		// Arrange
		lang := "es"
		hasTicket := true

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithTicketES, result)
	})

	t.Run("returns Spanish template without ticket", func(t *testing.T) {
		// Arrange
		lang := "es"
		hasTicket := false

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketES, result)
	})

	t.Run("defaults to English with ticket for unknown language", func(t *testing.T) {
		// Arrange
		lang := "fr"
		hasTicket := true

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithTicketEN, result)
	})

	t.Run("defaults to English without ticket for unknown language", func(t *testing.T) {
		// Arrange
		lang := "fr"
		hasTicket := false

		// Act
		result := GetCommitPromptTemplate(lang, hasTicket)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketEN, result)
	})
}
