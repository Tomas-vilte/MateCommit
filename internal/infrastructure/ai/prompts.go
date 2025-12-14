package ai

import (
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

// Issue reference instructions
const (
	issueReferenceInstructionsES = `Si hay un issue asociado (#%d), DEBES incluir la referencia en el título del commit:
       - Para features/mejoras: "tipo: mensaje (#%d)"
       - Para bugs: "fix: mensaje (#%d)" o "fix(scope): mensaje (fixes #%d)"
       - Ejemplos válidos:
         ✅ feat: add dark mode support (#%d)
         ✅ fix: resolve authentication error (fixes #%d)
         ✅ feat(api): implement caching layer (#%d)
       - NUNCA omitas la referencia del issue #%d.`

	issueReferenceInstructionsEN = `There is an associated issue (#%d), you MUST include the reference in the commit title:
       - For features/improvements: "type: message (#%d)"
       - For bugs: "fix: message (#%d)" or "fix(scope): message (fixes #%d)"
       - Valid examples:
         ✅ feat: add dark mode support (#%d)
         ✅ fix: resolve authentication error (fixes #%d)
         ✅ feat(api): implement caching layer (#%d)
       - NEVER omit the reference to issue #%d.`
)

// Templates para Pull Requests
const (
	prPromptTemplateEN = `# Task
  Generate a comprehensive Pull Request summary.

  # PR Content
  %s

  # Instructions
  1. Create concise title (max 80 chars)
  2. Identify 3-5 key changes with purpose and impact
  3. Suggest relevant labels from: feature, fix, refactor, docs, infra, test, breaking-change

  # Output Format
  Respond with ONLY valid JSON:
  {
    "title": "PR title here",
    "body": "Detailed markdown body with:\n- Overview\n- Key changes\n- Technical impact",
    "labels": ["label1", "label2"]
  }

  Generate the summary now.`

	prPromptTemplateES = `# Tarea
  Genera un resumen completo del Pull Request.

  # Contenido del PR
  %s

  # Instrucciones
  1. Crea un título conciso (máx 80 caracteres)
  2. Identifica 3-5 cambios clave con propósito e impacto
  3. Sugiere etiquetas relevantes de: feature, fix, refactor, docs, infra, test, breaking-change

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

  Responde SOLO con JSON válido:
  {
    "title": "título del PR",
    "body": "cuerpo detallado en markdown con:\n- Resumen\n- Cambios clave\n- Impacto técnico",
    "labels": ["etiqueta1", "etiqueta2"]
  }

  Genera el resumen ahora.`
)

// Templates para Commits con ticket
const (
	promptTemplateWithTicketEN = `# Task
  Generate %d commit message suggestions based on code changes and ticket requirements.

  # Modified Files
  %s

  # Code Changes
  %s

  # Ticket Context
  %s

  # Issue Reference Instructions
  %s

  # Instructions
  1. Analyze changes against acceptance criteria
  2. Use conventional commit types: feat, fix, refactor, test, docs, chore
  3. Keep commit messages under 100 characters
  4. Include issue reference if provided

  # Output Format
  Respond with ONLY valid JSON array. Each suggestion must have:
  {
    "title": "commit message",
    "desc": "detailed explanation", 
    "files": ["file1.go", "file2.go"],
    "analysis": {
      "overview": "brief summary",
      "purpose": "main goal",
      "impact": "technical impact"
    },
    "requirements": {
      "status": "full_met | partially_met | not_met",
      "missing": ["criterion 1", "criterion 2"],
      "suggestions": ["improvement 1", "improvement 2"]
    }
  }

  Generate %d suggestions now.`

	promptTemplateWithTicketES = `# Tarea
  Genera %d sugerencias de mensajes de commit basadas en los cambios de código y requisitos del 
  ticket.

  # Archivos Modificados
  %s

  # Cambios en el Código
  %s

  # Contexto del Ticket
  %s

  # Instrucciones de Referencia de Issues
  %s

  # Instrucciones
  1. Analiza los cambios contra los criterios de aceptación
  2. Usa tipos de commit convencionales: feat, fix, refactor, test, docs, chore
  3. Mantén los mensajes de commit en menos de 100 caracteres
  4. Incluye referencia al issue si se proporciona

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.
  EXCEPTO el campo "status" que debe ser uno de los valores permitidos exactos.

  Responde SOLO con un array JSON válido. Cada sugerencia debe tener:
  {
    "title": "mensaje del commit",
    "desc": "explicación detallada",
    "files": ["archivo1.go", "archivo2.go"],
    "analysis": {
      "overview": "resumen breve",
      "purpose": "objetivo principal",
      "impact": "impacto técnico"
    },
    "requirements": {
      "status": "full_met | partially_met | not_met",
      "missing": ["criterio 1", "criterio 2"],
      "suggestions": ["mejora 1", "mejora 2"]
    }
  }

  Genera %d sugerencias ahora.`
)

// Templates para Commits sin ticket
const (
	promptTemplateWithoutTicketES = `# Tarea
  Genera %d sugerencias de mensajes de commit basadas en los cambios de código.

  # Archivos Modificados
  %s

  # Cambios en el Código
  %s

  # Instrucciones de Referencia de Issues
  %s

  # Instrucciones
  1. Analiza los cambios en detalle
  2. Enfócate en aspectos técnicos y mejores prácticas
  3. Usa tipos de commit convencionales: feat, fix, refactor, test, docs, chore
  4. Mantén los mensajes en menos de 100 caracteres
  5. Incluye referencia al issue si se proporciona

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

  Responde SOLO con un array JSON válido. Cada sugerencia debe tener:
  {
    "title": "mensaje del commit",
    "desc": "explicación detallada",
    "files": ["archivo1.go", "archivo2.go"],
    "analysis": {
      "overview": "resumen breve",
      "purpose": "objetivo principal",
      "impact": "impacto técnico"
    }
  }

  %s

  Genera %d sugerencias ahora.`

	promptTemplateWithoutTicketEN = `# Task
  Generate %d commit message suggestions based on code changes.

  # Modified Files
  %s

  # Code Changes
  %s

  # Issue Reference Instructions
  %s

  # Instructions
  1. Analyze changes in detail
  2. Focus on technical aspects and best practices
  3. Use conventional commit types: feat, fix, refactor, test, docs, chore
  4. Keep messages under 100 characters
  5. Include issue reference if provided

  # Output Format
  Respond with ONLY valid JSON array. Each suggestion must have:
  {
    "title": "commit message",
    "desc": "detailed explanation",
    "files": ["file1.go", "file2.go"],
    "analysis": {
      "overview": "brief summary",
      "purpose": "main goal",
      "impact": "technical impact"
    }
  }

  %s

  Generate %d suggestions now.`
)

// Templates para Releases - MARKDOWN + JSON
const (
	releasePromptTemplateES = `# Tarea
Genera release notes en primera persona con tono técnico pero cercano (voseo argentino).

# Información del Release
- Repositorio: %s/%s
- Versión anterior: %s
- Nueva versión: %s
- Tipo de bump: %s

# Cambios
%s

# Instrucciones
1. Basate EXCLUSIVAMENTE en los cambios listados arriba
2. Primera persona con voseo: "Implementé", "Mejoré", "Arreglé"
3. NO inventes features, comandos o funcionalidades
4. Si hay contributors, mencioná con @username
5. Referencias a issues/PRs cuando corresponda

# Formato de Salida
IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

Responde SOLO con JSON válido:
{
  "title": "título conciso (max 60 chars)",
  "summary": "2-3 oraciones en primera persona",
  "highlights": ["highlight 1", "highlight 2", "highlight 3"],
  "breaking_changes": ["cambio 1" o "Ninguno"],
  "contributors": "Gracias a @user1, @user2" o "N/A"
}

Genera las release notes ahora.`

	releasePromptTemplateEN = `# Task
Generate release notes in first person with friendly, technical tone.

# Release Information
- Repository: %s/%s
- Previous version: %s
- New version: %s
- Bump type: %s

# Changes
%s

# Instructions
1. Base EVERYTHING on the changes listed above
2. First person: "I added", "I implemented", "I improved", "I fixed"
3. DO NOT invent features, commands, or functionality
4. Credit contributors with @username when applicable
5. Reference issues/PRs when relevant

# Output Format
Respond with ONLY valid JSON:
{
  "title": "concise title (max 60 chars)",
  "summary": "2-3 sentences in first person",
  "highlights": ["highlight 1", "highlight 2", "highlight 3"],
  "breaking_changes": ["change 1" or "None"],
  "contributors": "Thanks to @user1, @user2" or "N/A"
}

Generate the release notes now.`
)

// GetPRPromptTemplate devuelve el template adecuado según el idioma
func GetPRPromptTemplate(lang string) string {
	switch lang {
	case "es":
		return prPromptTemplateES
	default:
		return prPromptTemplateEN
	}
}

// GetCommitPromptTemplate devuelve el template para commits según el idioma y si hay ticket
func GetCommitPromptTemplate(lang string, hasTicket bool) string {
	switch {
	case lang == "es" && hasTicket:
		return promptTemplateWithTicketES
	case lang == "es" && !hasTicket:
		return promptTemplateWithoutTicketES
	case hasTicket:
		return promptTemplateWithTicketEN
	default:
		return promptTemplateWithoutTicketEN
	}
}

// GetReleasePromptTemplate devuelve el template para releases según el idioma
func GetReleasePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return releasePromptTemplateES
	default:
		return releasePromptTemplateEN
	}
}

// GetIssueReferenceInstructions devuelve las instrucciones de referencias de issues según el idioma
func GetIssueReferenceInstructions(lang string) string {
	switch lang {
	case "es":
		return issueReferenceInstructionsES
	default:
		return issueReferenceInstructionsEN
	}
}

const (
	prIssueContextInstructionsES = `
  **IMPORTANTE - Contexto de Issues/Tickets:**
  Este PR está relacionado con los siguientes issues:
  %s

  **INSTRUCCIONES OBLIGATORIAS:**
  1. DEBES incluir AL INICIO del resumen (primeras líneas) las referencias de cierre:
     - Si resuelve bugs: "Fixes #N"
     - Si implementa features: "Closes #N"
     - Si solo relaciona: "Relates to #N"
     - Formato: "Closes #39, Fixes #41" (separados por comas)

  2. En la sección de cambios clave, menciona explícitamente cómo cada cambio aborda el issue

  3. Usa el formato correcto para que GitHub auto-enlace los issues en la sección "Development"

  **Ejemplo de formato correcto:**
  Closes #39

  - **Primer cambio clave:**
    - Propósito: Resolver el problema reportado en #39...
    - Impacto técnico: ...
  `

	prIssueContextInstructionsEN = `
  **IMPORTANT - Issue/Ticket Context:**
  This PR is related to the following issues:
  %s

  **MANDATORY INSTRUCTIONS:**
  1. You MUST include at the BEGINNING of the summary (first lines) the closing references:
     - If fixing bugs: "Fixes #N"
     - If implementing features: "Closes #N"
     - If just relating: "Relates to #N"
     - Format: "Closes #39, Fixes #41" (comma separated)

  2. In the key changes section, explicitly mention how each change addresses the issue

  3. Use the correct format so GitHub auto-links the issues in the "Development" section

  **Example of correct format:**
  Closes #39

  - **First key change:**
    - Purpose: Resolve the problem reported in #39...
    - Technical impact: ...
  `
)

// GetPRIssueContextInstructions devuelve las instrucciones de contexto de issues para PRs
func GetPRIssueContextInstructions(locale string) string {
	if locale == "es" {
		return prIssueContextInstructionsES
	}
	return prIssueContextInstructionsEN
}

// FormatIssuesForPrompt formatea la lista de issues para incluir en el prompt
func FormatIssuesForPrompt(issues []models.Issue, locale string) string {
	if len(issues) == 0 {
		return ""
	}

	var result strings.Builder
	for _, issue := range issues {
		if locale == "es" {
			result.WriteString(fmt.Sprintf("- Issue #%d: %s\n", issue.Number, issue.Title))
			if issue.Description != "" {
				desc := issue.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				result.WriteString(fmt.Sprintf("  Descripción: %s\n", desc))
			}
		} else {
			result.WriteString(fmt.Sprintf("- Issue #%d: %s\n", issue.Number, issue.Title))
			if issue.Description != "" {
				desc := issue.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				result.WriteString(fmt.Sprintf("  Description: %s\n", desc))
			}
		}
	}

	return result.String()
}

// Technical Analysis instructions
const (
	technicalAnalysisES = `Proporciona análisis técnico detallado incluyendo: mejores prácticas aplicadas, impacto en rendimiento/mantenibilidad, y consideraciones de seguridad si aplican.`
	technicalAnalysisEN = `Provide detailed technical analysis including: best practices applied, performance/maintainability impact, and security considerations if applicable.`
)

func GetTechnicalAnalysisInstruction(locale string) string {
	if locale == "es" {
		return technicalAnalysisES
	}
	return technicalAnalysisEN
}

// No Issue Reference instructions
const (
	noIssueReferenceES = `No incluyas referencias de issues en el título.`
	noIssueReferenceEN = `Do not include issue references in the title.`
)

func GetNoIssueReferenceInstruction(locale string) string {
	if locale == "es" {
		return noIssueReferenceES
	}
	return noIssueReferenceEN
}

// Release Note Headers
var (
	releaseHeadersES = map[string]string{
		"breaking":      "CAMBIOS QUE ROMPEN:",
		"features":      "NUEVAS CARACTERÍSTICAS:",
		"fixes":         "CORRECCIONES DE BUGS:",
		"improvements":  "MEJORAS:",
		"closed_issues": "ISSUES CERRADOS:",
		"merged_prs":    "PULL REQUESTS MERGEADOS:",
		"contributors":  "CONTRIBUIDORES",
		"file_stats":    "ESTADÍSTICAS DE ARCHIVOS:",
		"deps":          "ACTUALIZACIONES DE DEPENDENCIAS:",
	}

	releaseHeadersEN = map[string]string{
		"breaking":      "BREAKING CHANGES:",
		"features":      "NEW FEATURES:",
		"fixes":         "BUG FIXES:",
		"improvements":  "IMPROVEMENTS:",
		"closed_issues": "CLOSED ISSUES:",
		"merged_prs":    "MERGED PULL REQUESTS:",
		"contributors":  "CONTRIBUTORS",
		"file_stats":    "FILE STATISTICS:",
		"deps":          "DEPENDENCY UPDATES:",
	}
)

func GetReleaseNotesSectionHeaders(locale string) map[string]string {
	if locale == "es" {
		return releaseHeadersES
	}
	return releaseHeadersEN
}
