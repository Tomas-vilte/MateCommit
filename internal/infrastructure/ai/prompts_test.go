package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

		// Act
		result := GetCommitPromptTemplate(lang, true)

		// Assert
		assert.Equal(t, promptTemplateWithTicketEN, result)
	})

	t.Run("returns English template without ticket", func(t *testing.T) {
		// Arrange
		lang := "en"

		// Act
		result := GetCommitPromptTemplate(lang, false)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketEN, result)
	})

	t.Run("returns Spanish template with ticket", func(t *testing.T) {
		// Arrange
		lang := "es"

		// Act
		result := GetCommitPromptTemplate(lang, true)

		// Assert
		assert.Equal(t, promptTemplateWithTicketES, result)
	})

	t.Run("returns Spanish template without ticket", func(t *testing.T) {
		// Arrange
		lang := "es"

		// Act
		result := GetCommitPromptTemplate(lang, false)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketES, result)
	})

	t.Run("defaults to English with ticket for unknown language", func(t *testing.T) {
		// Arrange
		lang := "fr"

		// Act
		result := GetCommitPromptTemplate(lang, true)

		// Assert
		assert.Equal(t, promptTemplateWithTicketEN, result)
	})

	t.Run("defaults to English without ticket for unknown language", func(t *testing.T) {
		// Arrange
		lang := "fr"

		// Act
		result := GetCommitPromptTemplate(lang, false)

		// Assert
		assert.Equal(t, promptTemplateWithoutTicketEN, result)
	})
}

func TestGetReleasePromptTemplate(t *testing.T) {
	t.Run("returns English template for 'en'", func(t *testing.T) {
		lang := "en"
		result := GetReleasePromptTemplate(lang)
		assert.Equal(t, releasePromptTemplateEN, result)
	})

	t.Run("returns Spanish template for 'es'", func(t *testing.T) {
		lang := "es"
		result := GetReleasePromptTemplate(lang)
		assert.Equal(t, releasePromptTemplateES, result)
	})

	t.Run("defaults to English for unknown language", func(t *testing.T) {
		lang := "fr"
		result := GetReleasePromptTemplate(lang)
		assert.Equal(t, releasePromptTemplateEN, result)
	})

	t.Run("defaults to English for empty language", func(t *testing.T) {
		lang := ""
		result := GetReleasePromptTemplate(lang)
		assert.Equal(t, releasePromptTemplateEN, result)
	})
}

func TestGetIssuePromptTemplate(t *testing.T) {
	t.Run("returns English template for 'en'", func(t *testing.T) {
		lang := "en"
		result := GetIssuePromptTemplate(lang)
		assert.Equal(t, issuePromptTemplateEN, result)
	})

	t.Run("returns Spanish template for 'es'", func(t *testing.T) {
		lang := "es"
		result := GetIssuePromptTemplate(lang)
		assert.Equal(t, issuePromptTemplateES, result)
	})

	t.Run("defaults to English for unknown language", func(t *testing.T) {
		lang := "fr"
		result := GetIssuePromptTemplate(lang)
		assert.Equal(t, issuePromptTemplateEN, result)
	})

	t.Run("defaults to English for empty language", func(t *testing.T) {
		lang := ""
		result := GetIssuePromptTemplate(lang)
		assert.Equal(t, issuePromptTemplateEN, result)
	})
}
