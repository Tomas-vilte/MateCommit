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
	formatInstructions := `Usa el formato de Conventional Commits (feat/fix/docs/etc) y agrega un emoji al inicio de cada tipo de commit:
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
		Cuando modifiques el archivo .gitignore usa el tipo "chore" o "docs".
        `

	prompt := fmt.Sprintf(`Genera %d sugerencias diferentes de mensajes de commit, incluyendo una explicación concisa del motivo del commit y los archivos que están incluidos en el commit. Considera que los cambios en estos archivos deben ir en el mismo commit, a menos que se detecte que no están relacionados. No incluyas encabezados como "Sugerencia 1:", "Sugerencia 2:", etc.
            Archivos modificados:
            %s
            Diff:
            %s
            Instrucciones:
            1. %s
            2. Cada mensaje de commit no debe exceder 72 caracteres
            3. Los mensajes de commit deben ser claros y descriptivos
            4. No incluyas puntos finales.
            5. Genera exactamente %d sugerencias diferentes.
            6. Cada sugerencia debe estar en una línea nueva, con el siguiente formato:
               [tipo]: [mensaje de commit]
               Archivos: [lista de archivos separados por comas]
               Explicación: [explicación concisa de por qué se eligió ese mensaje de commit]
            7. No incluyas ninguna lista numerada ni texto introductorio. No incluyas encabezados de "Sugerencia n:",  ni "**Mensaje de commit n:**"
			8. Si detectas que algunos archivos no están lógicamente relacionados, separa sus cambios en commits adicionales, siguiendo el mismo formato.
			9. Cuando modifiques el archivo .gitignore usa el tipo "chore" o "docs".`,
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
	//log.Printf("Respuesta del modelo:\n%s", responseText) // Log response from the model
	suggestions := processResponse(responseText)

	suggestions = ensureConventionalFormat(suggestions)

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
					formattedContent.WriteString(fmt.Sprintf("%v", part))
				}
			}
		}
	}
	return formattedContent.String()
}

// processResponse toma la respuesta generada y la divide en sugerencias
func processResponse(response string) []string {
	lines := strings.Split(response, "\n")
	var suggestions []string
	var currentCommitSuggestion string
	var currentFiles string
	var currentExplanation string
	var mode string // "", commit", "files", "explanation"

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if strings.Contains(trimmedLine, ":") && mode == "" && !strings.HasPrefix(trimmedLine, "**Mensaje de commit") {
			if currentCommitSuggestion != "" {
				suggestions = append(suggestions, fmt.Sprintf("Commit: %s\nArchivos: %s\nExplicación: %s", currentCommitSuggestion, currentFiles, currentExplanation))
			}
			currentCommitSuggestion = trimmedLine
			mode = "files"
			currentFiles = ""
			currentExplanation = ""
		} else if strings.HasPrefix(trimmedLine, "Archivos:") {
			currentFiles = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "Archivos:"))
			mode = "explanation"
		} else if strings.HasPrefix(trimmedLine, "Explicación:") {
			currentExplanation = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "Explicación:"))
			if currentCommitSuggestion != "" {
				suggestions = append(suggestions, fmt.Sprintf("Commit: %s\nArchivos: %s\nExplicación: %s", currentCommitSuggestion, currentFiles, currentExplanation))
				currentCommitSuggestion = ""
				currentFiles = ""
				currentExplanation = ""
				mode = ""
			}
		} else if mode == "files" {
			currentFiles += trimmedLine
		} else if mode == "explanation" {
			currentExplanation += trimmedLine
		}
	}

	if currentCommitSuggestion != "" {
		suggestions = append(suggestions, fmt.Sprintf("Commit: %s\nArchivos: %s\nExplicación: %s", currentCommitSuggestion, currentFiles, currentExplanation))
	}
	//log.Printf("Sugerencias después de processResponse: %+v", suggestions)
	return suggestions
}

func ensureConventionalFormat(suggestions []string) []string {
	var formatted []string
	for _, suggestion := range suggestions {
		parts := strings.SplitN(suggestion, "\n", 3)
		if len(parts) < 3 {
			continue
		}

		// Extraer las líneas relevantes
		commitLine := parts[0]
		filesLine := parts[1]
		explanationLine := parts[2]

		// Remover el "Commit: " prefix si existe
		commitContent := strings.TrimPrefix(commitLine, "Commit: ")

		// Separar el tipo y mensaje
		commitParts := strings.SplitN(commitContent, ":", 2)
		if len(commitParts) < 2 {
			continue
		}

		// Limpiar el tipo de commit de cualquier "Commit: " residual
		commitType := strings.TrimSpace(strings.TrimPrefix(commitParts[0], "Commit:"))
		commitMessage := strings.TrimSpace(commitParts[1])

		// Encontrar el emoji correcto para el tipo de commit
		typeWithoutEmoji := ""
		emoji := ""
		for commitTypeMap, commitEmoji := range gitmojiMap {
			if strings.Contains(strings.ToLower(commitType), commitTypeMap) {
				typeWithoutEmoji = commitTypeMap
				emoji = commitEmoji
				break
			}
		}

		// Si se encontró un tipo válido, usar ese, si no, usar el tipo original
		if typeWithoutEmoji != "" {
			formatted = append(formatted, fmt.Sprintf("Commit: %s %s: %s\n%s\n%s",
				emoji, typeWithoutEmoji, strings.TrimSpace(commitMessage),
				filesLine, explanationLine))
		} else {
			// Si no se encontró un tipo válido, usar feat como default
			formatted = append(formatted, fmt.Sprintf("Commit: ✨ feat: %s\n%s\n%s",
				strings.TrimSpace(commitMessage),
				filesLine, explanationLine))
		}
	}
	//log.Printf("Sugerencias después de ensureConventionalFormat: %+v", formatted)
	return formatted
}

var gitmojiMap = map[string]string{
	"feat":     "✨",
	"fix":      "🐛",
	"docs":     "📚",
	"style":    "💄",
	"refactor": "♻️",
	"test":     "✅",
	"chore":    "🔧",
	"perf":     "⚡️",
	"ci":       "👷",
	"build":    "📦",
	"revert":   "⏪️",
}
