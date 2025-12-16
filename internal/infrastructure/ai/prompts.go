package ai

import (
	"fmt"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
)

const (
	issueReferenceInstructionsES = `Si hay un issue asociado (#%d), DEBES incluir la referencia en el título del commit:
       - Para features/mejoras: "tipo: mensaje (#%d)"
       - Para bugs: "fix: mensaje (#%d)" o "fix(scope): mensaje (fixes #%d)"
       - Ejemplos válidos:
         ✅ feat: add dark mode support (#%d)
         ✅ fix: resolve authentication error (fixes #%d)
         ✅ feat(api): implement caching layer (#%d)
       - No omitas la referencia del issue #%d.`

	issueReferenceInstructionsEN = `There is an associated issue (#%d), you MUST include the reference in the commit title:
       - For features/improvements: "type: message (#%d)"
       - For bugs: "fix: message (#%d)" or "fix(scope): message (fixes #%d)"
       - Valid examples:
         ✅ feat: add dark mode support (#%d)
         ✅ fix: resolve authentication error (fixes #%d)
         ✅ feat(api): implement caching layer (#%d)
       - NEVER omit the reference to issue #%d.`
)

const (
	prPromptTemplateEN = `# Task
  Act as a Senior Tech Lead and generate a Pull Request summary.

  # PR Content
  %s

  # Golden Rules (Constraints)
  1. **No Hallucinations:** If it's not in the diff, DO NOT invent it.
  2. **Tone:** Professional, direct, technical. Use first person ("I implemented", "I added").
  3. **Format:** Raw JSON only. Do not wrap in markdown blocks (like ` + "```json" + `).

  # Instructions
  1. Title: Catchy but descriptive (max 80 chars).
  2. Key Changes: Filter the noise. Explain the *technical impact*, not just the code change.
  3. Labels: Choose wisely (feature, fix, refactor, docs, infra, test, breaking-change).

  # Output Format
  Respond with ONLY valid JSON (no markdown):
  {
    "title": "PR title",
    "body": "Detailed markdown body with:\n- Overview\n- Key changes\n- Technical impact",
    "labels": ["label1", "label2"]
  }

  Generate the summary now.`

	prPromptTemplateES = `# Tarea
  Actuá como un Desarrollador Senior y genera un resumen del Pull Request.

  # Contenido del PR
  %s

  # Reglas de Oro (Constraints)
  1. **Cero alucinaciones:** Si algo no está explícito en el diff, no lo inventes.
  2. **Tono:** Profesional, cercano y directo. Usa primera persona ("Implementé", "Agregué", "Corregí"). Evita el lenguaje robótico ("Se ha realizado").
  3. **Formato:** JSON crudo. No incluyas bloques de markdown.

  # Instrucciones
  1. Título: Descriptivo y conciso (máx 80 caracteres).
  2. Cambios Clave: Filtrá el ruido. Explicá el *impacto* técnico y el propósito, no solo qué línea cambió.
  3. Etiquetas: Elegí con criterio (feature, fix, refactor, docs, infra, test, breaking-change).

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

  Responde SOLO con JSON válido (sin markdown):
  {
    "title": "título del PR",
    "body": "cuerpo detallado en markdown con:\n- Resumen (qué hice y por qué)\n- Cambios clave\n- Impacto técnico",
    "labels": ["etiqueta1", "etiqueta2"]
  }

  Genera el resumen ahora.`
)

const (
	promptTemplateWithTicketEN = `# Task
  Act as a Git Specialist and generate %d commit message suggestions.

  # Context
  - Modified Files: %s
  - Diff: %s
  - Ticket/Issue: %s
  - Recent History: %s
  - Issue Instructions: %s

  # Quality Guidelines
  1. **Conventional Commits:** Strictly follow ` + "`type(scope): description`" + `.
     - Types: feat, fix, refactor, perf, test, docs, chore, build, ci.
  2. **Precision:**
     - ❌ BAD: "fix: various fixes in login" (Too vague)
     - ✅ GOOD: "fix(auth): handle null token error (#42)" (Precise)
  3. **Scope:** If you touched 'ui' files, scope is (ui). If 'api', then (api).
  4. **Style:** 
     - Title: Imperative mood ("add", not "added").
     - Description: First person, professional tone ("I optimized the query...").
  5. **Validation:** Analyze changes against ticket criteria.

  # Output Format
  Respond with ONLY valid JSON array (no markdown).

  [
    {
      "title": "type(scope): short message (#N)",
      "desc": "detailed technical explanation in first person", 
      "files": ["file1.go", "file2.go"],
      "analysis": {
        "overview": "brief summary",
        "purpose": "main goal",
        "impact": "technical impact"
      },
      "requirements": {
        "status": "full_met | partially_met | not_met",
        "missing": ["missing test", "missing doc"],
        "completed_indices": [0, 2],
        "suggestions": ["improvement 1", "improvement 2"]
      }
    }
  ]

  Generate %d suggestions now.`

	promptTemplateWithTicketES = `# Tarea
  Actuá como un especialista en Git y genera %d sugerencias de commits.
  
  # Contexto
  - Archivos: %s
  - Diff: %s
  - Ticket/Issue: %s
  - Historial reciente: %s
  - Instrucciones Issue: %s

  # Criterios de Calidad (Guidelines)
  1. **Conventional Commits:** Respeta estrictamente ` + "`tipo(scope): descripción`" + `.
     - Tipos: feat, fix, refactor, perf, test, docs, chore, build, ci.
  2. **Precisión:**
     - ❌ MAL: "fix: arreglos varios en el login" (Muy vago)
     - ✅ BIEN: "fix(auth): manejo de error en token nulo (#42)" (Preciso)
  3. **Scope:** Si tocaste archivos de 'ui', el scope es (ui). Si es 'api', es (api). Si son muchos, no uses scope.
  4. **Primera Persona:** La descripción ("desc") escribila como si le contaras a un colega (ej: "Optimicé la query para mejorar el tiempo de respuesta").
  5. **Validación:** Analiza los cambios contra los criterios del ticket.

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.
  EXCEPTO el campo "status" que debe ser uno de los valores permitidos exactos. JSON crudo, sin markdown.

  Responde SOLO con este array JSON:
  [
    {
      "title": "tipo(scope): mensaje corto (#N)",
      "desc": "explicación técnica detallada y natural",
      "files": ["archivo_modificado.go"],
      "analysis": {
        "overview": "qué cambiaste",
        "purpose": "para qué lo cambiaste",
        "impact": "qué mejora esto"
      },
      "requirements": {
        "status": "full_met | partially_met | not_met",
        "missing": ["falta test", "falta doc"],
        "completed_indices": [0],
        "suggestions": ["agregar test de integración"]
      }
    }
  ]

  Genera %d sugerencias ahora.`
)

const (
	promptTemplateWithoutTicketES = `# Tarea
  Actuá como un especialista en Git y genera %d sugerencias de commits basadas en el código.

  # Inputs
  - Archivos Modificados: %s
  - Cambios (Diff): %s
  - Instrucciones Issues: %s
  - Historial: %s

  # Estrategia de Generación
  1. **Analiza el Diff:** Identifica qué lógica cambió realmente. Ignora cambios de formato/espacios.
  2. **Categoriza:**
     - ¿Nueva feature? -> feat
     - ¿Arreglo de bug? -> fix
     - ¿Cambio de código sin cambio de lógica? -> refactor
     - ¿Solo documentación? -> docs
  3. **Redacta:**
     - Título: Imperativo, max 50 chars si es posible (ej: "agrega validación", no "agregando").
     - Descripción: Primera persona, tono profesional y natural. "Agregué esta validación para evitar X error".

  # Ejemplos de Estilo
  - ❌ "update main.go" (Pésimo, no dice nada)
  - ❌ "se corrigió el error" (Voz pasiva, muy robótico)
  - ✅ "fix(cli): corrijo panic al no tener config" (Bien)

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español. JSON crudo, sin markdown.

  Responde SOLO con un array JSON válido:
  [
    {
      "title": "tipo(scope): mensaje",
      "desc": "explicación clara en primera persona",
      "files": ["archivo1.go", "archivo2.go"],
      "analysis": {
        "overview": "resumen breve",
        "purpose": "objetivo principal",
        "impact": "impacto técnico"
      }
    }
  ]

  %s

  Genera %d sugerencias ahora.`

	promptTemplateWithoutTicketEN = `# Task
  Act as a Git Specialist and generate %d commit message suggestions based on code changes.

  # Inputs
  - Modified Files: %s
  - Code Changes (Diff): %s
  - Issue Instructions: %s
  - Recent History: %s

  # Generation Strategy
  1. **Analyze Diff:** Identify logic changes vs formatting.
  2. **Categorize:**
     - New feature? -> feat
     - Bug fix? -> fix
     - Code change without logic change? -> refactor
     - Docs only? -> docs
  3. **Drafting:**
     - Title: Imperative mood, max 50 chars if possible (e.g., "add validation", not "adding").
     - Description: First person, professional tone. "I added this validation to prevent X error".

  # Style Examples
  - ❌ "update main.go" (Terrible, says nothing)
  - ❌ "error was fixed" (Passive voice)
  - ✅ "fix(cli): handle panic when config is missing" (Perfect)

  # Output Format
  Respond with ONLY valid JSON array (no markdown).

  [
    {
      "title": "type(scope): message",
      "desc": "detailed explanation in first person",
      "files": ["file1.go", "file2.go"],
      "analysis": {
        "overview": "brief summary",
        "purpose": "main goal",
        "impact": "technical impact"
      }
    }
  ]

  %s

  Generate %d suggestions now.`
)

const (
	releasePromptTemplateES = `# Tarea
Genera release notes actuando como un Desarrollador Senior.

# Datos del Release
- Repo: %s/%s
- Versiones: %s -> %s (%s)

# Changelog (Diff)
%s

# Instrucciones de Precisión
1. **Verdad ante todo:** Si el changelog solo muestra actualizaciones de dependencias, no inventes features. Poné "Mantenimiento de dependencias".
2. **Agrupación Inteligente:**
   - Si ves muchos commits de "fix", agrupalos en un highlight si están relacionados.
   - Si es un bump de versión sin cambios de código, decilo claro: "Release de mantenimiento para actualizar versiones internas".
3. **Estilo de Redacción:**
   - **Tono:** Profesional, directo y humano (similar a como escribirías en Slack o un email técnico).
   - **Primera Persona:** Usa "Implementé", "Mejoramos", "Agregué". Evita la voz pasiva ("Se ha implementado").
   - **Highlights:** Tienen que explicar el valor real del cambio, no solo describir el código.

# Formato de Salida
IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español. JSON crudo.

Responde SOLO con JSON válido:
{
  "title": "título descriptivo y real (ej: 'Mejoras en performance y correcciones')",
  "summary": "2-3 oraciones en primera persona contando de qué trata este release (ej: 'En esta versión me enfoqué en mejorar la UX...')",
  "highlights": ["highlight 1 (en primera persona)", "highlight 2", "highlight 3"],
  "breaking_changes": ["descripción del cambio" (o array vacío [] si no hay)],
  "contributors": "Gracias a @user1, @user2" o "N/A"
}

Genera las release notes ahora.`

	releasePromptTemplateEN = `# Task
Generate release notes acting as a Technical Product Owner.

# Release Information
- Repository: %s/%s
- Previous version: %s
- New version: %s
- Bump type: %s

# Changes
%s

# Precision Instructions
1. **Truthfulness:** If the changelog only shows dependency updates, DO NOT invent features. State "Dependency maintenance".
2. **Smart Grouping:**
   - Group related "fix" commits into a single highlight.
   - If it's a version bump without code changes, state it clearly.
3. **Style:**
   - First person ("We released", "I improved", "I added").
   - Tone: Professional, technical, yet friendly.
   - "Highlights" must sell the value, not just describe the code.

# Output Format
Respond with ONLY valid JSON (no markdown):
{
  "title": "Catchy but real title (e.g., 'Performance improvements and fixes')",
  "summary": "2-3 sentences in first person summarizing this release.",
  "highlights": ["highlight 1", "highlight 2", "highlight 3"],
  "breaking_changes": ["change 1" (or empty array [] if none)],
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

  **INSTRUCCIONES CLAVES:**
  1. DEBES incluir AL INICIO del resumen (primeras líneas) las referencias de cierre:
     - Si resuelve bugs: "Fixes #N"
     - Si implementa features: "Closes #N"
     - Si solo relaciona: "Relates to #N"
     - Formato: "Closes #39, Fixes #41" (separados por comas)

  2. En la sección de cambios clave, menciona explícitamente cómo cada cambio impacta en el issue.

  3. Usa el formato correcto para que GitHub enlace los issues automáticamente.

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

const (
	technicalAnalysisES = `Proporciona un análisis técnico detallado incluyendo: buenas prácticas aplicadas, impacto en rendimiento/mantenibilidad, y consideraciones de seguridad si aplican.`
	technicalAnalysisEN = `Provide detailed technical analysis including: best practices applied, performance/maintainability impact, and security considerations if applicable.`
)

func GetTechnicalAnalysisInstruction(locale string) string {
	if locale == "es" {
		return technicalAnalysisES
	}
	return technicalAnalysisEN
}

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
