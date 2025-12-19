package models

// IssueGenerationRequest contiene la información necesaria para generar una issue.
// Soporta múltiples fuentes de contexto: descripción manual, diff de git, o ambos.
type IssueGenerationRequest struct {
	// Description es la descripción manual proporcionada por el usuario (opcional)
	Description string

	// Diff contiene los cambios locales de git (opcional)
	Diff string

	// ChangedFiles es la lista de archivos modificados (opcional)
	ChangedFiles []string

	// Hint es contexto adicional proporcionado por el usuario para guiar la generación (opcional)
	Hint string

	// Language es el idioma para la generación de contenido (ej: "es", "en")
	Language string
}

// IssueGenerationResult contiene el resultado de la generación de contenido de una issue.
type IssueGenerationResult struct {
	// Title es el título generado para la issue
	Title string

	// Description es la descripción completa generada para la issue
	Description string

	// Labels son las etiquetas sugeridas para la issue
	Labels []string

	// Assignees son los responsables sugeridos para la issue
	Assignees []string

	// Usage contiene los metadatos de uso de tokens de la IA
	Usage *TokenUsage
}

// DiffAnalysis contiene el análisis estructurado del diff para inferencia de labels.
type DiffAnalysis struct {
	// HasGoFiles indica si el diff incluye archivos .go
	HasGoFiles bool

	// HasTestFiles indica si el diff incluye archivos de test
	HasTestFiles bool

	// HasDocFiles indica si el diff incluye archivos de documentación
	HasDocFiles bool

	// HasConfigFiles indica si el diff incluye archivos de configuración
	HasConfigFiles bool

	// HasUIFiles indica si el diff incluye archivos de UI (CSS, HTML, JSX, etc)
	HasUIFiles bool

	// Keywords contiene palabras clave encontradas en el diff (fix, feat, refactor, etc)
	Keywords map[string]bool
}
