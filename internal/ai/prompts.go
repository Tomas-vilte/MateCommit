package ai

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/thomas-vilte/matecommit/internal/models"
)

const (
	issueReferenceInstructionsES = `Si hay un issue asociado (#{{.IssueNumber}}), DEBES incluir la referencia en el título del commit:
       - Para features/mejoras: "tipo: mensaje (#{{.IssueNumber}})"
       - Para bugs: "fix: mensaje (#{{.IssueNumber}})" o "fix(scope): mensaje (fixes #{{.IssueNumber}})"
       - Ejemplos válidos:
         ✅ feat: add dark mode support (#{{.IssueNumber}})
         ✅ fix: resolve authentication error (fixes #{{.IssueNumber}})
         ✅ feat(api): implement caching layer (#{{.IssueNumber}})
       - No omitas la referencia del issue #{{.IssueNumber}}.`

	issueReferenceInstructionsEN = `There is an associated issue (#{{.IssueNumber}}), you MUST include the reference in the commit title:
       - For features/improvements: "type: message (#{{.IssueNumber}})"
       - For bugs: "fix: message (#{{.IssueNumber}})" or "fix(scope): message (fixes #{{.IssueNumber}})"
       - Valid examples:
         ✅ feat: add dark mode support (#{{.IssueNumber}})
         ✅ fix: resolve authentication error (fixes #{{.IssueNumber}})
         ✅ feat(api): implement caching layer (#{{.IssueNumber}})
       - NEVER omit the reference to issue #{{.IssueNumber}}.`
)

// PromptData holds the parameters for template rendering
type PromptData struct {
	Count           int
	Files           string
	Diff            string
	Ticket          string
	History         string
	Instructions    string
	IssueNumber     int
	RelatedIssues   string
	IssueInfo       string
	RepoOwner       string
	RepoName        string
	PreviousVersion string
	CurrentVersion  string
	LatestVersion   string
	ReleaseDate     string
	Changelog       string
	PRContent       string
	TechnicalInfo   string
}

// RenderPrompt renders a prompt template with the provided data
func RenderPrompt(name, tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("error parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template %s: %w", name, err)
	}

	return buf.String(), nil
}

const (
	prPromptTemplateEN = `# Task
  Act as a Senior Tech Lead and generate a Pull Request summary.
  # PR Content
  {{.PRContent}}
  # Golden Rules (Constraints)
  1. **No Hallucinations:** If it's not in the diff, DO NOT invent it.
  2. **Tone:** Professional, direct, technical. Use first person ("I implemented", "I added").
  3. **Markdown Format (STRICT):** The "body" field MUST be valid Markdown with REAL newline characters (\n) between headers, paragraphs, and list items. Each "## Header" goes on its own line followed by a blank line. Each checklist item ("- [ ] ...") goes on its own line. NEVER merge multiple sections or checklist items into a single paragraph of running text.
  # Instructions
  1. Title: Catchy but descriptive (max 80 chars).
  2. Key Changes: Filter the noise. Explain the *technical impact*, not just the code change.
  3. Labels: Choose wisely (feature, fix, refactor, docs, infra, test, breaking-change).`

	prPromptTemplateES = `# Tarea
  Actuá como un Desarrollador Senior y genera un resumen del Pull Request.
  # Contenido del PR
  {{.PRContent}}
  # Reglas de Oro (Constraints)
  1. **Cero alucinaciones:** Si algo no está explícito en el diff, no lo inventes.
  2. **Tono:** Profesional, cercano y directo. Usa primera persona ("Implementé", "Agregué", "Corregí"). Evita el lenguaje robótico ("Se ha realizado").
  3. **Formato Markdown (ESTRICTO):** El campo "body" DEBE ser Markdown válido con saltos de línea reales (\n) entre encabezados, párrafos e items de lista. Cada "## Encabezado" va en su propia línea seguido de una línea en blanco. Cada checkbox ("- [ ] ...") va en su propia línea. NUNCA mezcles varias secciones o checkboxes en un solo párrafo de texto corrido.
  # Instrucciones
  1. Título: Descriptivo y conciso (máx 80 caracteres).
  2. Cambios Clave: Filtrá el ruido. Explicá el *impacto* técnico y el propósito, no solo qué línea cambió.
  3. Etiquetas: Elegí con criterio (feature, fix, refactor, docs, infra, test, breaking-change).

  IMPORTANTE: Responde en ESPAÑOL. Todo el contenido del JSON debe estar en español.`
)

const (
	promptTemplateWithTicketEN = `# Task
  Act as a Git Specialist and generate {{.Count}} commit message suggestions.
  # Context
  - Modified Files: {{.Files}}
  - Diff: {{.Diff}}
  - Ticket/Issue: {{.Ticket}}
  - Recent History: {{.History}}
  - Issue Instructions: {{.Instructions}}
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
  5. **Requirements Validation (IMPORTANT):**
     - Analyze ONLY the current diff changes against ticket criteria.
     - Mark as "missing" ONLY requirements that are NOT visible in the diff.
     - If recent history shows something was implemented in previous commits, do NOT mark it as missing.
     - If you see file names or function names in the diff indicating prior implementation (e.g., "stats.go", "CountTokens"), assume it exists.
     - Focus on what's missing NOW in the current commit context, not in the entire project.
  Generate {{.Count}} suggestions now.`

	promptTemplateWithTicketES = `# Tarea
  Actuá como un especialista en Git y genera {{.Count}} sugerencias de commits.
  
  # Contexto
  - Archivos: {{.Files}}
  - Diff: {{.Diff}}
  - Ticket/Issue: {{.Ticket}}
  - Historial reciente: {{.History}}
  - Instrucciones Issue: {{.Instructions}}
  # Criterios de Calidad (Guidelines)
  1. **Conventional Commits:** Respeta estrictamente ` + "`tipo(scope): descripción`" + `.
     - Tipos: feat, fix, refactor, perf, test, docs, chore, build, ci.
  2. **Precisión:**
     - ❌ MAL: "fix: arreglos varios en el login" (Muy vago)
     - ✅ BIEN: "fix(auth): manejo de error en token nulo (#42)" (Preciso)
  3. **Scope:** Si tocaste archivos de 'ui', el scope es (ui). Si es 'api', es (api). Si son muchos, no uses scope.
  4. **Primera Persona:** La descripción (\"desc\") escribila como si le contaras a un colega (ej: \"Optimicé la query para mejorar el tiempo de respuesta\").
  5. **Validación de Requerimientos (IMPORTANTE):**
     - Analiza SOLO los cambios del diff actual contra los criterios del ticket.
     - Marca como "missing" ÚNICAMENTE requisitos que NO están visibles en el diff.
     - Si el historial reciente muestra que algo ya se implementó en commits anteriores, NO lo marques como faltante.
     - Si ves nombres de archivos o funciones en el diff que indican implementación previa (ej: "stats.go", "CountTokens"), asume que ya existe.
     - Enfocate en lo que falta AHORA en el contexto del commit actual, no en el proyecto completo.
  Genera {{.Count}} sugerencias ahora.`
)

const (
	promptTemplateWithoutTicketES = `# Tarea
  Actuá como un especialista en Git y genera {{.Count}} sugerencias de commits basadas en el código.
  # Inputs
  - Archivos Modificados: {{.Files}}
  - Cambios (Diff): {{.Diff}}
  - Instrucciones Issues: {{.Instructions}}
  - Historial: {{.History}}
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
  {{.TechnicalInfo}}
  Genera {{.Count}} sugerencias ahora.`

	promptTemplateWithoutTicketEN = `# Task
  Act as a Git Specialist and generate {{.Count}} commit message suggestions based on code changes.
  # Inputs
  - Modified Files: {{.Files}}
  - Code Changes (Diff): {{.Diff}}
  - Issue Instructions: {{.Instructions}}
  - Recent History: {{.History}}
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
  {{.TechnicalInfo}}
  Generate {{.Count}} suggestions now.`
)

const (
	releasePromptTemplateES = `# Tarea
Generá release notes profesionales para un CHANGELOG.md siguiendo el estándar "Keep a Changelog".
# Datos del Release
- Repo: {{.RepoOwner}}/{{.RepoName}}
- Versiones: {{.CurrentVersion}} -> {{.LatestVersion}} ({{.ReleaseDate}})
# Changelog (Diff)
{{.Changelog}}
# Instrucciones Críticas
## 1. FILTRADO DE RUIDO TÉCNICO
**IGNORAR** commits de mantenimiento interno, typos, docs internos.
**INCLUIR** features, mejoras de UX/Performance, bug fixes y breaking changes.
## 2. AGRUPACIÓN SEMÁNTICA (SECCIONES) - MUY IMPORTANTE
**DEBES** agrupar los cambios en secciones temáticas usando el campo "sections" del esquema JSON.
**Cada sección** debe tener:
- Un título descriptivo y atractivo (puede incluir emoji)
- Una lista de items relacionados

**Ejemplos de buenos títulos de sección:**
- "✨ AI & Generation Improvements" - para mejoras en generación de IA
- "🛠️ Templates & Configuration" - para cambios en templates y config
- "🛡️ Stability & Performance" - para mejoras de estabilidad
- "🎨 User Interface" - para cambios visuales
- "🚀 Performance" - para optimizaciones
- "🔒 Security" - para fixes de seguridad
- "📚 Documentation" - para cambios en docs
- "🔧 Developer Experience" - para mejoras de DX

**Cuándo usar cada tipo:**
- Agrupa cambios relacionados por área funcional (ej: AI, Templates, CLI)
- Si hay muchos cambios pequeños de un tipo, agrúpalos (ej: "Bug Fixes")
- Usa máximo 5-6 secciones para mantener claridad

## 3. ESTILO Y NARRATIVA (IMPORTANTE)
- **Voz:** Usá "Agregamos/Mejoramos" (1ra persona plural). Evita "Se ha implementado".
- **Foco:** Centrate en el BENEFICIO para el usuario, no en la implementación técnica.
- **Formato de items:** Cada item debe ser una oración completa y descriptiva.

## 4. EJEMPLOS DE CALIDAD (GOLD STANDARD)
❌ MAL: "feat: update user schema" (Técnico, aburrido)
✅ BIEN: "Mejoramos el perfil de usuario para soportar múltiples direcciones."
❌ MAL: "fix: fix crash in login" (Vago)
✅ BIEN: "Solucionamos un cierre inesperado al iniciar sesión con Google."

Generá las release notes ahora usando el esquema JSON con secciones semánticas.`

	releasePromptTemplateEN = `# Task
Generate professional release notes for a CHANGELOG.md following the "Keep a Changelog" standard.
# Release Information
- Repository: {{.RepoOwner}}/{{.RepoName}}
- Versions: {{.CurrentVersion}} -> {{.LatestVersion}} ({{.ReleaseDate}})
# Changelog (Diff)
{{.Changelog}}
# Critical Instructions
## 1. TECHNICAL NOISE FILTERING
**IGNORE** internal maintenance, typos, internal docs.
**INCLUDE** features, UX/Performance improvements, bug fixes, and breaking changes.
## 2. SEMANTIC GROUPING (SECTIONS) - VERY IMPORTANT
You MUST group changes into thematic sections using the "sections" field in the JSON schema.
**Each section** must have:
- A descriptive and engaging title (can include emoji)
- A list of related items

**Examples of good section titles:**
- "✨ AI & Generation Improvements" - for AI generation enhancements
- "🛠️ Templates & Configuration" - for template and config changes
- "🛡️ Stability & Performance" - for stability improvements
- "🎨 User Interface" - for visual changes
- "🚀 Performance" - for optimizations
- "🔒 Security" - for security fixes
- "📚 Documentation" - for documentation changes
- "🔧 Developer Experience" - for DX improvements

**When to use each type:**
- Group related changes by functional area (e.g., AI, Templates, CLI)
- If there are many small changes of one type, group them (e.g., "Bug Fixes")
- Use maximum 5-6 sections to maintain clarity

## 3. STYLE AND NARRATIVE (IMPORTANT)
- **Voice:** Use "We added/We improved" (1st person plural). Avoid passive voice.
- **Focus:** Focus on USER BENEFIT, not technical implementation.
- **Item format:** Each item should be a complete, descriptive sentence.

## 4. QUALITY EXAMPLES (GOLD STANDARD)
❌ BAD: "feat: update user schema" (Too technical)
✅ GOOD: "Enhanced user profile to support multiple addresses."
❌ BAD: "fix: fix crash in login" (Vague)
✅ GOOD: "Fixed a crash when logging in via Google."

Generate the release notes now using the JSON schema with semantic sections.`
)

// GetPRPromptTemplate returns the appropriate template based on the language
func GetPRPromptTemplate(lang string) string {
	switch lang {
	case "es":
		return prPromptTemplateES
	default:
		return prPromptTemplateEN
	}
}

// GetCommitPromptTemplate returns the commit template based on language and whether there is a ticket
func GetCommitPromptTemplate(lang string, hasTicket bool) string {
	if lang == "es" {
		if hasTicket {
			return promptTemplateWithTicketES
		}
		return promptTemplateWithoutTicketES
	}

	if hasTicket {
		return promptTemplateWithTicketEN
	}
	return promptTemplateWithoutTicketEN
}

// GetReleasePromptTemplate returns the release template based on the language
func GetReleasePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return releasePromptTemplateES
	default:
		return releasePromptTemplateEN
	}
}

// GetIssueReferenceInstructions returns issue reference instructions based on the language
func GetIssueReferenceInstructions(lang string) string {
	switch lang {
	case "es":
		return issueReferenceInstructionsES
	default:
		return issueReferenceInstructionsEN
	}
}

const (
	templateInstructionsES = `## Template del Proyecto
 
 El proyecto tiene un template específico. DEBES seguir su estructura y formato al generar el contenido.`

	templateInstructionsEN = `## Project Template
 
 The project has a specific template. You MUST follow its structure and format when generating the content.`

	prTemplateInstructionsES = `## Template de PR del Proyecto

El proyecto tiene un template específico de PR. DEBES seguir su estructura y formato al generar la descripción del PR.

IMPORTANTE: Generá la descripción del PR siguiendo la estructura y formato mostrado en el template arriba. Completá cada sección basándote en los cambios de código y el contexto proporcionado.`

	prTemplateInstructionsEN = `## Project PR Template

The project has a specific PR template. You MUST follow its structure and format when generating the PR description.

IMPORTANT: Generate the PR description following the structure and format shown in the template above. Fill in each section based on the code changes and context provided.`
)

// GetTemplateInstructions returns template instructions based on the language
func GetTemplateInstructions(lang string) string {
	switch lang {
	case "es":
		return templateInstructionsES
	default:
		return templateInstructionsEN
	}
}

// GetPRTemplateInstructions returns PR template instructions based on the language
func GetPRTemplateInstructions(lang string) string {
	switch lang {
	case "es":
		return prTemplateInstructionsES
	default:
		return prTemplateInstructionsEN
	}
}

// FormatTemplateForPrompt formats a template for inclusion in an AI prompt.
// It handles both Issue and PR templates with proper language support.
func FormatTemplateForPrompt(template *models.IssueTemplate, lang string, templateType string) string {
	if template == nil {
		return ""
	}

	if lang == "" {
		lang = "en"
	}

	var sb strings.Builder
	isIssue := templateType == "issue"

	if lang == "es" {
		if isIssue {
			sb.WriteString("## Template de Issue del Proyecto\n\n")
			sb.WriteString("El proyecto tiene un template específico de issue. DEBES seguir su estructura y formato al generar el contenido del issue.\n\n")
		} else {
			sb.WriteString("## Template de PR del Proyecto\n\n")
			sb.WriteString("El proyecto tiene un template específico de PR. DEBES seguir su estructura y formato al generar la descripción del PR.\n\n")
		}
	} else {
		if isIssue {
			sb.WriteString("## Project Issue Template\n\n")
			sb.WriteString("The project has a specific issue template. You MUST follow its structure and format when generating the issue content.\n\n")
		} else {
			sb.WriteString("## Project PR Template\n\n")
			sb.WriteString("The project has a specific PR template. You MUST follow its structure and format when generating the PR description.\n\n")
		}
	}

	if template.Name != "" {
		if lang == "es" {
			sb.WriteString(fmt.Sprintf("Nombre del Template: %s\n", template.Name))
		} else {
			sb.WriteString(fmt.Sprintf("Template Name: %s\n", template.Name))
		}
	}

	if template.GetAbout() != "" {
		if lang == "es" {
			sb.WriteString(fmt.Sprintf("Descripción del Template: %s\n", template.GetAbout()))
		} else {
			sb.WriteString(fmt.Sprintf("Template Description: %s\n", template.GetAbout()))
		}
	}

	if template.BodyContent != "" {
		if lang == "es" {
			sb.WriteString("\nEstructura del Template:\n```markdown\n")
		} else {
			sb.WriteString("\nTemplate Structure:\n```markdown\n")
		}
		sb.WriteString(template.BodyContent)
		sb.WriteString("\n```\n\n")
		if isIssue {
			sb.WriteString(GetTemplateInstructions(lang))
		} else {
			sb.WriteString(GetPRTemplateInstructions(lang))
		}
		sb.WriteString("\n\n")
	} else if len(template.Body) > 0 {
		if lang == "es" {
			if isIssue {
				sb.WriteString("\nTipo de Template: GitHub Issue Form (YAML)\n")
			} else {
				sb.WriteString("\nTipo de Template: GitHub PR Template (YAML/Markdown)\n")
			}
			sb.WriteString("El template define campos específicos. A continuación la estructura que DEBES completar:\n\n")
		} else {
			if isIssue {
				sb.WriteString("\nTemplate Type: GitHub Issue Form (YAML)\n")
			} else {
				sb.WriteString("\nTemplate Type: GitHub PR Template (YAML/Markdown)\n")
			}
			sb.WriteString("The template defines specific fields. Below is the structure you MUST complete:\n\n")
		}

		for _, item := range template.Body {
			if item.Type == "markdown" {
				continue
			}

			if item.Attributes.Label != "" {
				sb.WriteString(fmt.Sprintf("### %s\n", item.Attributes.Label))
				if item.Attributes.Description != "" {
					sb.WriteString(fmt.Sprintf("Context: %s\n", item.Attributes.Description))
				}
				if item.Attributes.Placeholder != "" {
					sb.WriteString(fmt.Sprintf("Example: %s\n", item.Attributes.Placeholder))
				}
				sb.WriteString("\n")
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

const (
	prIssueContextInstructionsES = `
  **IMPORTANTE - Contexto de Issues/Tickets:**
  Este PR está relacionado con los siguientes issues:
  {{.RelatedIssues}}

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
  {{.RelatedIssues}}

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

// GetPRIssueContextInstructions returns issue context instructions for PRs
func GetPRIssueContextInstructions(locale string) string {
	if locale == "es" {
		return prIssueContextInstructionsES
	}
	return prIssueContextInstructionsEN
}

// FormatIssuesForPrompt formats the issue list to be included in the prompt
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
  {{.IssueInfo}}

  # Golden Rules (Constraints)
  1. **Active Voice:** Write in FIRST PERSON ("I implemented", "I added", "We refactored"). Avoid passive voice like "It was implemented".
  2. **Context First:** Explain the WHY before the WHAT.
  3. **Accurate Categorization:** Always choose at least one primary category: 'feature', 'fix', or 'refactor'. Use 'fix' ONLY for bug corrections. Use 'refactor' for code improvements without logic changes. Use 'feature' for new functionality.
  4. **No Emojis:** Do not use emojis in the title or description. Keep it purely textual and professional.
  5. **Balanced Labeling:** Aim for 2-4 relevant labels. Ensure you include the primary category plus any relevant file-based labels like 'test', 'docs', or 'infra' if applicable.

  Generate the issue now.`

	issuePromptTemplateES = `# Tarea
  Actuá como un Tech Lead y generá un issue de GitHub profesional basado en los inputs.

  # Entradas (Inputs)
  {{.IssueInfo}}

  # Reglas de Oro (Constraints)
  1. **Voz Activa:** Escribí en PRIMERA PERSONA ("Implementé", "Agregué", "Corregí"). Prohibido usar voz pasiva robótica.
  2. **Contexto Real:** Explicá el POR QUÉ del cambio, no solo qué líneas tocaste.
  3. **Categorización Precisa:** Elegí siempre al menos una categoría principal: 'feature', 'fix', o 'refactor'. Solo usá 'fix' si ves una corrección de un bug. Usá 'refactor' para mejoras de código sin cambios lógicos. Usá 'feature' para funcionalidades nuevas.
  4. **Cero Emojis:** No uses emojis ni en el título ni en el cuerpo del issue. Mantené un estilo sobrio y técnico.
  5. **Etiquetado Equilibrado:** Buscá entre 2 y 4 etiquetas relevantes. Asegurate de incluir la categoría principal más cualquier etiqueta de tipo de archivo como 'test', 'docs', o 'infra' si corresponde.

  Generá el issue ahora. Responde en ESPAÑOL.`

	issueDefaultStructureEN = `
  # Description Structure
  The 'description' field must follow this Markdown structure:
  - ### Context (Motivation)
  - ### Technical Details (Architectural changes, new models, etc.)
  - ### Impact (Benefits)
`

	issueDefaultStructureES = `
  # Estructura de la Descripción
  El campo "description" tiene que ser Markdown y seguir esta estructura estricta:
  - ### Contexto (¿Cuál es la motivación o el dolor que resuelve esto?)
  - ### Detalles Técnicos (Lista de cambios importantes, modelos nuevos, refactors)
  - ### Impacto (¿Qué gana el usuario o el desarrollador con esto?)
`
)

// GetIssuePromptTemplate returns the appropriate issue generation template based on language
func GetIssuePromptTemplate(lang string) string {
	switch lang {
	case "es":
		return issuePromptTemplateES
	default:
		return issuePromptTemplateEN
	}
}

// GetIssueDefaultStructure returns the default structure for issues when no template is provided
func GetIssueDefaultStructure(lang string) string {
	switch lang {
	case "es":
		return issueDefaultStructureES
	default:
		return issueDefaultStructureEN
	}
}
