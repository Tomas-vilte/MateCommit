package gemini

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"strings"
)

type GeminiService struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiService(ctx context.Context, apiKey string) *GeminiService {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		panic(err)
	}

	model := client.GenerativeModel("gemini-1.5-flash")
	return &GeminiService{
		client: client,
		model:  model,
	}
}

func (s *GeminiService) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]string, error) {
	var formatInstructions string
	if info.Format == "conventional" {
		formatInstructions = "Usa el formato de Conventional Commits (feat/fix/docs/etc)"
	} else if info.Format == "gitmoji" {
		formatInstructions = `Usa el formato Gitmoji con estos emojis específicos:
        - ✨ (sparkles) para nuevas características (feat)
        - 🐛 (bug) para correcciones (fix)
        - 📚 (books) para documentación (docs)
        - 💄 (lipstick) para cambios de estilo (style)
        - ♻️ (recycle) para refactorizaciones (refactor)
        - ✅ (check mark) para tests
        - 🔧 (wrench) para tareas de mantenimiento (chore)
        - ⚡️ (zap) para mejoras de rendimiento (perf)
        - 👷 (construction worker) para CI
        - 📦 (package) para cambios en el build
        - ⏪️ (rewind) para reverts
        
        El formato debe ser: [emoji] tipo: descripción`
	}

	prompt := fmt.Sprintf(`Genera %d sugerencias diferentes de mensajes de commit basados en estos cambios:
			Archivos modificados:
			%s
			Diff:
			%s
			Instrucciones:
			1. %s
			2. Cada mensaje no debe exceder 72 caracteres
			3. Los mensajes deben ser claros y descriptivos
			4. No incluyas puntos finales
			5. Genera exactamente %d sugerencias diferentes
			6. Cada sugerencia debe estar en una línea nueva
			7. No números las sugerencias
			8. Si es formato gitmoji, asegurate de incluir el emoji al inicio`,
		count,
		formatChanges(info.Files),
		info.Diff,
		formatInstructions,
		count)

	resp, err := s.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, err
	}

	responseText := formatResponse(resp)
	suggestions := processResponse(responseText)

	if info.Format == "gitmoji" {
		suggestions = ensureGitmojiFormat(suggestions)
	}

	if len(suggestions) > count {
		suggestions = suggestions[:count]
	}

	return suggestions, nil

}

func formatChanges(files []string) string {
	var result string
	for _, file := range files {
		result += fmt.Sprintf("- %s\n", file)
	}
	return result
}

func formatResponse(resp *genai.GenerateContentResponse) string {
	var formattedContent strings.Builder
	if resp != nil && resp.Candidates != nil {
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					formattedContent.WriteString(fmt.Sprintf("%v", part)) // Convertir part a string
				}
			}
		}
	}
	return formattedContent.String()
}

// processResponse toma la respuesta generada y la divide en sugerencias y explicaciones.
func processResponse(response string) []string {
	// Dividir la respuesta en líneas
	lines := strings.Split(response, "\n")
	var suggestions []string
	var currentCommitTitle string
	var currentCommitExplanation strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Detectar el inicio de una nueva sugerencia (es decir, el título del commit)
		if strings.HasPrefix(trimmedLine, "**Sugerencia") {
			// Si ya había un commit anterior, agregamos la sugerencia acumulada
			if currentCommitTitle != "" {
				suggestions = append(suggestions, fmt.Sprintf("Commit: %s\nExplicación: %s", currentCommitTitle, currentCommitExplanation.String()))
				currentCommitTitle = ""
				currentCommitExplanation.Reset()
			}
		}

		// Agregar el título del commit (por ejemplo, "feat(cli): ...")
		if strings.HasPrefix(trimmedLine, "feat") || strings.HasPrefix(trimmedLine, "fix") || strings.HasPrefix(trimmedLine, "chore") || strings.HasPrefix(trimmedLine, "refactor") {
			currentCommitTitle = trimmedLine
		} else if currentCommitTitle != "" {
			// Agregar líneas adicionales a la explicación del commit
			currentCommitExplanation.WriteString(" " + trimmedLine)
		}
	}

	// Agregar la última sugerencia si quedó pendiente
	if currentCommitTitle != "" {
		suggestions = append(suggestions, fmt.Sprintf("Commit: %s\nExplicación: %s", currentCommitTitle, currentCommitExplanation.String()))
	}

	return suggestions
}

func ensureGitmojiFormat(suggestions []string) []string {
	var formatted []string
	for _, suggestion := range suggestions {
		parts := strings.SplitN(suggestion, ":", 2)
		if len(parts) < 2 {
			continue
		}

		commitType := strings.TrimSpace(parts[0])
		description := strings.TrimSpace(parts[1])

		// Extraer el tipo de commit sin el emoji si ya existe uno
		typeWithoutEmoji := commitType
		for _, emoji := range gitmojiMap {
			typeWithoutEmoji = strings.TrimSpace(strings.ReplaceAll(typeWithoutEmoji, emoji, ""))
		}

		// Buscar el emoji correspondiente
		emoji, exists := gitmojiMap[typeWithoutEmoji]
		if exists {
			formatted = append(formatted, fmt.Sprintf("%s %s: %s", emoji, typeWithoutEmoji, description))
		} else {
			// Si no encontramos un emoji específico, usamos ✨ como default
			formatted = append(formatted, fmt.Sprintf("✨ %s: %s", typeWithoutEmoji, description))
		}
	}
	return formatted
}

var gitmojiMap = map[string]string{
	"feat":     "✨",  // Sparkles para nuevas características
	"fix":      "🐛",  // Bug para correcciones
	"docs":     "📚",  // Libros para documentación
	"style":    "💄",  // Lápiz labial para cambios de estilo/CSS
	"refactor": "♻️", // Reciclar para refactorizaciones
	"test":     "✅",  // Check mark para tests
	"chore":    "🔧",  // Llave inglesa para tareas de mantenimiento
	"perf":     "⚡️", // Rayo para mejoras de rendimiento
	"ci":       "👷",  // Trabajador de construcción para CI
	"build":    "📦",  // Paquete para cambios en el sistema de build
	"revert":   "⏪️", // Rebobinar para reverts
}
