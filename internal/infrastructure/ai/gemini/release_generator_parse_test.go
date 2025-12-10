package gemini

import (
	"testing"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestParseResponse_MultiLineSummary(t *testing.T) {
	response := `TÍTULO: v1.3.0: Gestión de Releases, Gemini 2.5 y Configuración Intuitiva

RESUMEN: En esta versión v1.3.0, me enfoqué en llevar la gestión de releases a un nuevo nivel, implementando comandos que automatizan gran parte del proceso. Además, potencié nuestras capacidades de IA actualizando a los modelos Gemini 2.5 y mejoré significativamente la configuración, haciéndola más accesible y robusta para todos ustedes.

HIGHLIGHTS:
- ¡Implementé un nuevo set de comandos para la **gestión automatizada de releases**!
- Actualicé nuestros modelos de IA a la potente versión **Gemini 2.5**
- Introduje el asistente de configuración config init

QUICK_START:
Para instalar o actualizar a la v1.3.0:
` + "```bash\ngo install example.com/cli-tool@latest\n```" + `

BREAKING_CHANGES:
- Ninguno

LINKS:
N/A`

	release := &models.Release{Version: "v1.3.0"}
	gen := &ReleaseNotesGenerator{}

	notes, err := gen.parseResponse(response, release)

	assert.NoError(t, err)
	assert.NotNil(t, notes)

	assert.Equal(t, "v1.3.0: Gestión de Releases, Gemini 2.5 y Configuración Intuitiva", notes.Title)

	assert.Contains(t, notes.Summary, "En esta versión v1.3.0")
	assert.Contains(t, notes.Summary, "implementando comandos")
	assert.Contains(t, notes.Summary, "Gemini 2.5")
	assert.NotEmpty(t, notes.Summary)

	assert.Len(t, notes.Highlights, 3)
	assert.Contains(t, notes.Highlights[0], "gestión automatizada de releases")

	assert.NotEmpty(t, notes.QuickStart)
	assert.Contains(t, notes.QuickStart, "go install")

	assert.Empty(t, notes.BreakingChanges)
}
