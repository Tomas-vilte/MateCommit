package gemini

import (
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestParseResponse_MultilineCodeExamples(t *testing.T) {
	generator := &ReleaseNotesGenerator{}
	release := &models.Release{Version: "v1.0.0", VersionBump: "minor"}

	t.Run("parses multiline code examples with backticks", func(t *testing.T) {
		// Arrange - Simula el formato que genera la IA
		content := `TITLE: Release v1.0.0
SUMMARY: Test release with examples
HIGHLIGHTS:
- Feature 1

EXAMPLES:
EXAMPLE_1:
TITLE: Basic usage
DESCRIPTION: How to use the command
LANGUAGE: bash
CODE:
` + "```bash\nmatecommit\n```" + `

EXAMPLE_2:
TITLE: With flags
DESCRIPTION: Using flags
LANGUAGE: bash
CODE:
` + "```bash\nmatecommit --help\n```" + `

BREAKING_CHANGES:
- None`

		// Act
		notes, err := generator.parseResponse(content, release)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, notes.Examples, 2)

		// Verificar primer ejemplo
		assert.Equal(t, "Basic usage", notes.Examples[0].Title)
		assert.Equal(t, "How to use the command", notes.Examples[0].Description)
		assert.Equal(t, "bash", notes.Examples[0].Language)
		assert.Contains(t, notes.Examples[0].Code, "matecommit")
		assert.NotContains(t, notes.Examples[0].Code, "```", "Code should not contain backticks")

		// Verificar segundo ejemplo
		assert.Equal(t, "With flags", notes.Examples[1].Title)
		assert.Equal(t, "Using flags", notes.Examples[1].Description)
		assert.Equal(t, "bash", notes.Examples[1].Language)
		assert.Contains(t, notes.Examples[1].Code, "matecommit --help")
		assert.NotContains(t, notes.Examples[1].Code, "```", "Code should not contain backticks")
	})
}
