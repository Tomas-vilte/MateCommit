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
	prPromptTemplateEN = `Hey, could you whip up a summary for this PR with:

	## PR Title
	A short title (max 80 chars). Example: "fix: Image loading error"
	
	## Key Changes
	- The 3 main changes
	- Purpose of each one
	- Technical impact if applicable
	
	## Suggested Tags
	Comma-separated. Options: feature, fix, refactor, docs, infra, test. Example: fix,infra
	
	PR Content:
	%s
	
	Thanks a bunch, you rock!`

	prPromptTemplateES = `Che, armame un resumen de este PR con:

	## Título del PR
	Un título corto (máx 40 caracteres). Ej: "fix: Error al cargar imágenes"
	
	## Cambios clave
	- Los 3 cambios principales
	- El propósito de cada uno
	- Impacto técnico si aplica
	
	## Etiquetas sugeridas
	Separadas por coma. Opciones: feature, fix, refactor, docs, infra, test. Ej: fix,infra
	
	Contenido del PR:
	%s
	
	¡Gracias máquina!`
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
      "status": "Fully Met | Partially Met | Not Met",
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
      "status": "Completamente Cumplido | Parcialmente Cumplido | No Cumplido",
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

// Templates para Releases
const (
	releasePromptTemplateES = `
  Sos un desarrollador escribiendo las release notes de tu proyecto en primera persona.
    Usá un tono técnico pero cercano, explicando qué hiciste en esta versión.

    Repositorio: %s/%s
    Versión anterior: %s
    Nueva versión: %s
    Tipo de bump: %s

    Cambios en este release:

    %s

    IMPORTANTE - CONTEXTO ADICIONAL:
    El listado anterior incluye no solo commits, sino también:
    - Issues cerrados: Problemas reportados por usuarios que fueron resueltos
    - Pull Requests mergeados: Contribuciones de la comunidad o del equipo
    - Contributors: Personas que participaron en este release
    - Estadísticas de archivos: Magnitud de los cambios
    - Actualizaciones de dependencias: Librerías actualizadas

    Usá esta información para:
    1. Dar crédito a contributors mencionándolos por username (@usuario)
    2. Referenciar issues/PRs específicos cuando sean relevantes (#123)
    3. Mencionar áreas del código más afectadas según las estadísticas
    4. Destacar contribuciones de la comunidad si hay nuevos contributors
    5. Mencionar upgrades importantes de dependencias si afectan al usuario

    REGLAS DE ESTILO:
    - Primera persona: "Implementé", "Mejoré", "Arreglé", "Agregué"
    - Voseo natural: "podés", "tenés", "querés" (en vez de "puedes", "tienes", "quieres")
    - Expresiones naturales: "mucho más simple", "ahora funciona mejor", "sin vueltas"
    - Tono profesional pero directo, como si le explicaras a un colega
    - Sé técnico y preciso, pero accesible
    - NO uses emojis en el contenido de las release notes
    - Dá crédito: "Gracias a @usuario por reportar/contribuir"

    REGLAS CRÍTICAS - PREVENCIÓN DE ALUCINACIONES:
    1. Basate EXCLUSIVAMENTE en los commits, issues, PRs listados arriba
    2. Si la sección de cambios está vacía o solo tiene cambios menores, escribí un resumen breve y honesto
    3. NO inventes features, comandos, flags, o funcionalidades que no aparezcan explícitamente
    4. Solo mencioná issues/PRs que estén en el listado
    5. Solo mencioná contributors que estén en el listado
    6. Si no hay suficiente información, sé honesto y simple
    7. Para EXAMPLES, solo mostrá comandos que realmente existan
    8. Si los cambios son principalmente internos, decilo claramente

    VALIDACIÓN DE CONTENIDO:
    Antes de escribir cada sección, preguntate: "¿Este detalle específico está en la información que me pasaron?"
    Si la respuesta es NO, no lo incluyas.

    Formato de respuesta (IMPORTANTE: Incluí TODAS las secciones):
    
    TÍTULO: <título conciso y descriptivo (máximo 60 caracteres)>
    
    RESUMEN: <2-3 oraciones en primera persona contando los cambios más importantes. Mencioná contributors clave si corresponde>
    
    HIGHLIGHTS:
    - <highlight 1 en primera persona, basado en commits/PRs/issues reales>
    - <highlight 2 en primera persona, basado en commits/PRs/issues reales>
    - <highlight 3 en primera persona, basado en commits/PRs/issues reales>
    (Si hay nuevos contributors o muchos issues cerrados, podés incluirlo como highlight)
    
    QUICK_START:
    <Instrucciones de instalación/actualización en 2-3 líneas. Usá el repositorio real: github.com/%s/%s>
    IMPORTANTE: Este proyecto es un CLI de Go. Usá "go install github.com/%s/%s@<version>" para instalación.
    
    EXAMPLES:
    EXAMPLE_1:
    TITLE: <Título del ejemplo>
    DESCRIPTION: <Breve descripción de qué hace>
    LANGUAGE: bash
    CODE: <código del ejemplo - debe ser un comando real que funcione>
    
    EXAMPLE_2:
    TITLE: <Título del segundo ejemplo>
    DESCRIPTION: <Breve descripción>
    LANGUAGE: bash
    CODE: <código del ejemplo - debe ser un comando real que funcione>
    
    BREAKING_CHANGES:
    - <cambio breaking 1, o "Ninguno" si no hay>
    
    COMPARISONS:
    COMPARISON_1:
    FEATURE: <nombre de la feature que realmente cambió>
    BEFORE: <estado anterior según los commits>
    AFTER: <estado nuevo según los commits>
    
    CONTRIBUTORS:
    <Lista de contributors con agradecimiento. Formato: "Gracias a @user1, @user2 y @user3 por sus contribuciones. Damos la bienvenida a los nuevos contributors: @newuser1, @newuser2">
    (Si hay contributors listados arriba, incluí esta sección. Si no, poné "N/A")
    
    LINKS:
    - Closed Issues: <lista de links a issues cerrados si hay, o "N/A">
    - Merged PRs: <lista de links a PRs mergeados si hay, o "N/A">
	`

	releasePromptTemplateEN = `
  You are a developer writing release notes for your project in first person.
  Write in a friendly, casual tone explaining what you built in this version.

  Repository: %s/%s
  Previous version: %s
  New version: %s
  Bump type: %s

  Changes in this release:

  %s

  STYLE RULES:
  - First person: "I added", "I implemented", "I improved", "I fixed"
  - Professional but accessible tone (you can use expressions like "now", "simpler", "much better")
  - Explain what you did and why it's useful
  - Be technical and precise, but approachable
  - DO NOT use emojis in the release notes content

  CRITICAL RULES - PREVENTING HALLUCINATIONS:
  1. Base everything EXCLUSIVELY on the commits listed above in "Changes in this release"
  2. If the changes section is empty or only has minor changes (e.g., version bump), write a brief and honest summary
  3. DO NOT invent features, commands, flags, or functionality not explicitly present in the commits
  4. DO NOT mention "validators", "linters", "new options" or other features unless they appear in the commits
  5. If you don't have enough info for a specific example, use generic examples of basic project usage
  6. For EXAMPLES, only show commands that actually exist according to the commits. If there are no significant changes, show existing basic usage
  7. For COMPARISONS, only include comparisons if there are concrete changes to compare. If not, use "N/A" or a generic version comparison
  8. If changes are primarily internal or maintenance-related, state that clearly instead of inventing user-visible features

  CONTENT VALIDATION:
  Before writing each section, ask yourself: "Is this specific detail in the commits I was given?"
  If the answer is NO, don't include it.

  Response format (IMPORTANT: Include ALL sections):
  
  TITLE: <concise, descriptive title (max 60 chars)>
  
  SUMMARY: <2-3 sentences in first person highlighting the most important changes. If there are no significant changes, be honest about it>
  
  HIGHLIGHTS:
  - <highlight 1 in first person, based on actual commits>
  - <highlight 2 in first person, based on actual commits>
  - <highlight 3 in first person, based on actual commits>
  (If there aren't enough real highlights, focus on maintenance, stability, or preparation for future features)
  
  QUICK_START:
  <Installation/update instructions in 2-3 lines. Use the real repository: github.com/%s/%s>
  IMPORTANT: This project is a Go CLI. Use "go install github.com/%s/%s@<version>" for installation.
  Do not invent flags or commands that don't exist.
  
  EXAMPLES:
  EXAMPLE_1:
  TITLE: <Example title>
  DESCRIPTION: <Brief description of what it does>
  LANGUAGE: bash
  CODE: <example code - must be a real command that works>
  
  EXAMPLE_2:
  TITLE: <Second example title>
  DESCRIPTION: <Brief description>
  LANGUAGE: bash
  CODE: <example code - must be a real command that works>
  (Only include examples of functionality that actually exists. If there are no new features, show existing basic usage)
  
  BREAKING_CHANGES:
  - <breaking change 1, or "None" if there are no breaking changes>
  (Only list breaking changes if they are explicitly mentioned in the commits)
  
  COMPARISONS:
  COMPARISON_1:
  FEATURE: <name of feature that actually changed>
  BEFORE: <previous state according to commits>
  AFTER: <new state according to commits>
  (If there are no concrete comparisons based on commits, use "N/A" or a generic version comparison)
  
  LINKS:
  (Only include links if they are relevant to this specific release, such as closed issues or related PRs. If there are no relevant links, put "N/A")
  `
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
