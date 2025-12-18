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
Generá release notes profesionales para un CHANGELOG.md siguiendo el estándar "Keep a Changelog".

# Datos del Release
- Repo: %s/%s
- Versiones: %s -> %s (%s)

# Changelog (Diff)
%s

# Instrucciones Críticas

## 1. FILTRADO DE RUIDO TÉCNICO
**IGNORAR completamente** estos tipos de commits (no incluirlos en ninguna sección):
- Cambios en mocks o tests internos (ej: "Implementa GetIssue en MockVCSClient")
- Refactors internos que no afectan funcionalidad (ej: "Refactor: extract helper function")
- Updates menores de dependencias (ej: "chore: update go.mod")
- Cambios de documentación interna o comentarios
- Fixes de typos en código o variables internas

**SÍ INCLUIR** solo cambios que impactan al usuario final:
- Nuevas features visibles
- Mejoras de performance o UX
- Correcciones de bugs que afectaban funcionalidad
- Breaking changes
- Updates importantes de dependencias (cambios de versión mayor)

## 2. AGRUPACIÓN INTELIGENTE
**AGRUPAR** commits relacionados bajo un concepto unificador:

❌ **MAL** (lista cruda de commits):
- "feat: agregar spinners"
- "feat: agregar colores"
- "feat: mejorar feedback visual"

✅ **BIEN** (agrupado con valor):
- "UX Renovada: Agregamos spinners, colores y feedback visual en todas las operaciones largas para que no sientas que la terminal se colgó"

**Reglas de agrupación:**
- Si 3+ commits tocan el mismo módulo/feature → agrupar en un solo highlight
- Priorizar el VALOR para el usuario, no los detalles técnicos
- Máximo 5-7 highlights por release (no listar 15+ ítems)

## 3. IDIOMA Y TONO
**ESPAÑOL ARGENTINO PROFESIONAL:**
- Tono: Conversacional pero técnico, como un email entre devs
- Primera persona plural: "Agregamos", "Mejoramos", "Implementamos"
- Evitar spanglish completamente (nada de "fixeamos" o "pusheamos")
- Evitar jerga forzada, mantener profesionalismo

**Ejemplos de tono correcto:**
- ✅ "Automatizamos la generación del CHANGELOG.md"
- ✅ "Mejoramos la detección automática de issues"
- ❌ "Se implementó la feature de changelog" (muy formal/pasivo)
- ❌ "Agregamos un fix re-copado" (muy informal)

## 4. ESTRUCTURA Y NARRATIVA
Cada release debe contar una historia:
- **Summary:** Explicar el foco principal del release (ej: "En esta versión nos enfocamos en mejorar la UX y automatizar el proceso de releases")
- **Highlights:** Agrupar por tema (UX, Automatización, Performance, etc.)
- Cada highlight debe responder: "¿Qué ganó el usuario con esto?"

## 5. FORMATO DE SALIDA
IMPORTANTE: TODO en español. JSON válido sin markdown.

{
  "title": "Título conciso y descriptivo (ej: 'Mejoras de UX y Automatización')",
  "summary": "2-3 oraciones explicando el foco del release en primera persona plural. Debe dar contexto de por qué estos cambios importan.",
  "highlights": [
    "Highlight 1: Agrupación de features relacionadas con explicación de valor",
    "Highlight 2: Otra mejora importante",
    "Highlight 3: Correcciones relevantes"
  ],
  "breaking_changes": ["Descripción clara del breaking change y cómo migrar" (o [] si no hay)],
  "contributors": "Gracias a @user1, @user2" o "N/A"
}

# Ejemplo de Calidad Esperada

**Input (commits crudos):**
- feat: add spinners to long operations
- feat: add colors to output
- feat: improve visual feedback
- feat(mock): implement GetIssue in MockVCSClient
- fix: correct spinner formatting
- chore: update dependencies

**Output esperado:**
{
  "title": "Mejoras de Experiencia de Usuario",
  "summary": "En esta versión nos enfocamos en mejorar la experiencia de usuario agregando feedback visual completo. Ya no vas a sentir que la terminal se colgó durante operaciones largas.",
  "highlights": [
    "UX Renovada: Agregamos spinners, colores y feedback visual en todas las operaciones largas (#45)",
    "Correcciones: Mejoramos el formato de los spinners para mejor legibilidad"
  ],
  "breaking_changes": [],
  "contributors": "N/A"
}

Generá las release notes ahora siguiendo estas instrucciones al pie de la letra.`

	releasePromptTemplateEN = `# Task
Generate professional release notes for a CHANGELOG.md following the "Keep a Changelog" standard.

# Release Information
- Repository: %s/%s
- Versions: %s -> %s (%s)

# Changelog (Diff)
%s

# Critical Instructions

## 1. TECHNICAL NOISE FILTERING
**COMPLETELY IGNORE** these types of commits (do not include them in any section):
- Changes to mocks or internal tests (e.g., "Implement GetIssue in MockVCSClient")
- Internal refactors that don't affect functionality (e.g., "Refactor: extract helper function")
- Minor dependency updates (e.g., "chore: update go.mod")
- Internal documentation or comment changes
- Typo fixes in code or internal variables

**DO INCLUDE** only changes that impact the end user:
- New visible features
- Performance or UX improvements
- Bug fixes affecting functionality
- Breaking changes
- Important dependency updates (major version changes)

## 2. INTELLIGENT GROUPING
**GROUP** related commits under a unifying concept:

❌ **BAD** (raw commit list):
- "feat: add spinners"
- "feat: add colors"
- "feat: improve visual feedback"

✅ **GOOD** (grouped with value):
- "Revamped UX: Added spinners, colors, and visual feedback across all long-running operations so you never feel like the terminal froze"

**Grouping rules:**
- If 3+ commits touch the same module/feature → group into a single highlight
- Prioritize USER VALUE, not technical details
- Maximum 5-7 highlights per release (don't list 15+ items)

## 3. LANGUAGE AND TONE
**PROFESSIONAL ENGLISH:**
- Tone: Conversational yet technical, like an email between developers
- First person plural: "We added", "We improved", "We implemented"
- Maintain professionalism, avoid forced slang

**Examples of correct tone:**
- ✅ "We automated CHANGELOG.md generation"
- ✅ "We improved automatic issue detection"
- ❌ "The changelog feature was implemented" (too formal/passive)
- ❌ "We added a super cool fix" (too informal)

## 4. STRUCTURE AND NARRATIVE
Each release should tell a story:
- **Summary:** Explain the main focus of the release (e.g., "In this release, we focused on improving UX and automating the release process")
- **Highlights:** Group by theme (UX, Automation, Performance, etc.)
- Each highlight should answer: "What did the user gain from this?"

## 5. OUTPUT FORMAT
IMPORTANT: Everything in English. Valid JSON without markdown.

{
  "title": "Concise and descriptive title (e.g., 'UX Improvements and Automation')",
  "summary": "2-3 sentences explaining the release focus in first person plural. Should provide context on why these changes matter.",
  "highlights": [
    "Highlight 1: Grouping of related features with value explanation",
    "Highlight 2: Another important improvement",
    "Highlight 3: Relevant fixes"
  ],
  "breaking_changes": ["Clear description of breaking change and how to migrate" (or [] if none)],
  "contributors": "Thanks to @user1, @user2" or "N/A"
}

# Expected Quality Example

**Input (raw commits):**
- feat: add spinners to long operations
- feat: add colors to output
- feat: improve visual feedback
- feat(mock): implement GetIssue in MockVCSClient
- fix: correct spinner formatting
- chore: update dependencies

**Expected output:**
{
  "title": "User Experience Improvements",
  "summary": "In this release, we focused on improving the user experience by adding complete visual feedback. You'll no longer feel like the terminal froze during long operations.",
  "highlights": [
    "Revamped UX: Added spinners, colors, and visual feedback across all long-running operations (#45)",
    "Fixes: Improved spinner formatting for better readability"
  ],
  "breaking_changes": [],
  "contributors": "N/A"
}

Generate the release notes now following these instructions to the letter.`
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

const (
	issuePromptTemplateEN = `# Task
  Act as a Senior Tech Lead and generate a high-quality GitHub issue based on the provided inputs.

  # Inputs
  %s

  # Golden Rules (Constraints)
  1. **Active Voice:** Write in FIRST PERSON ("I implemented", "I added", "We refactored"). Avoid passive voice like "It was implemented".
  2. **Context First:** Explain the WHY before the WHAT.
  3. **Accurate Categorization:** Always choose at least one primary category: 'feature', 'fix', or 'refactor'. Use 'fix' ONLY for bug corrections. Use 'refactor' for code improvements without logic changes. Use 'feature' for new functionality.
  4. **No Emojis:** Do not use emojis in the title or description. Keep it purely textual and professional.
  5. **Balanced Labeling:** Aim for 2-4 relevant labels. Ensure you include the primary category plus any relevant file-based labels like 'test', 'docs', or 'infra' if applicable.
  6. **Format:** Raw JSON only. Do not wrap in markdown blocks.

  # Description Structure
  The 'description' field must follow this Markdown structure:
  - ### Context (Motivation)
  - ### Technical Details (Architectural changes, new models, etc.)
  - ### Impact (Benefits)

  # Output Format
  Respond with ONLY valid JSON (no markdown):
  {
    "title": "Concise and descriptive title",
    "description": "Markdown body following the structure above",
    "labels": ["label1", "label2"]
  }

  Generate the issue now.`

	issuePromptTemplateES = `# Tarea
  Actuá como un Tech Lead y generá un issue de GitHub profesional basado en los inputs.

  # Entradas (Inputs)
  %s

  # Reglas de Oro (Constraints)
  1. **Voz Activa:** Escribí en PRIMERA PERSONA ("Implementé", "Agregué", "Corregí"). Prohibido usar voz pasiva robótica.
  2. **Contexto Real:** Explicá el POR QUÉ del cambio, no solo qué líneas tocaste.
  3. **Categorización Precisa:** Elegí siempre al menos una categoría principal: 'feature', 'fix', o 'refactor'. Solo usá 'fix' si ves una corrección de un bug. Usá 'refactor' para mejoras de código sin cambios lógicos. Usá 'feature' para funcionalidades nuevas.
  4. **Cero Emojis:** No uses emojis ni en el título ni en el cuerpo del issue. Mantené un estilo sobrio y técnico.
  5. **Etiquetado Equilibrado:** Buscá entre 2 y 4 etiquetas relevantes. Asegurate de incluir la categoría principal más cualquier etiqueta de tipo de archivo como 'test', 'docs', o 'infra' si corresponde.
  6. **Formato:** JSON crudo. No incluyas bloques de markdown (como ` + "```json" + `).

  # Estructura de la Descripción
  El campo "description" tiene que ser Markdown y seguir esta estructura estricta:
  - ### Contexto (¿Cuál es la motivación o el dolor que resuelve esto?)
  - ### Detalles Técnicos (Lista de cambios importantes, modelos nuevos, refactors)
  - ### Impacto (¿Qué gana el usuario o el desarrollador con esto?)

  # Formato de Salida
  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.

  Responde SOLO con JSON válido (sin markdown):
  {
    "title": "título descriptivo y con gancho",
    "description": "Cuerpo en markdown siguiendo la estructura pedida",
    "labels": ["etiqueta1", "etiqueta2"]
  }

  Generá el issue ahora.`
)

// GetIssuePromptTemplate devuelve el template adecuado para generación de issues según el idioma
func GetIssuePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return issuePromptTemplateES
	default:
		return issuePromptTemplateEN
	}
}
