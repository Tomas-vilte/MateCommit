package ai

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
	promptTemplateWithTicketEN = `
    Instructions:
    1. Generate %d commit message suggestions based on the provided code changes and ticket information.
    2. Each suggestion MUST follow the format defined in the "Suggestion Format" section.
    3. **Critically analyze code changes in detail and rigorously compare them against the "Acceptance Criteria" provided in the "Ticket Information" section.**
    4. **For each acceptance criterion, explicitly determine if it is fully met, partially met, or not met by the code changes.**
    5. **In the "🎯 Requirements Analysis" section, provide a detailed breakdown of the acceptance criteria status. For each criterion that is NOT fully met, list it under "❌ Missing Criteria" and provide specific, actionable improvement suggestions under "💡 Improvement Suggestions" to fully meet the criterion.**
    6. Use appropriate commit types:
        - feat: New features
        - fix: Bug fixes
        - refactor: Code restructuring
        - test: Adding or modifying tests
        - docs: Documentation updates
        - chore: Maintenance tasks
    7. Keep commit messages under 100 characters.
    8. Provide specific, actionable improvement suggestions, especially related to meeting acceptance criteria.

    Suggestion Format:
    =========[ Suggestion ]=========
    [number]. [Ordinal] suggestion:
    🔍 Analyzing changes...

    📊 Code Analysis:
    - Changes Overview: [Brief overview of what changed in the code]
    - Primary Purpose: [Main goal of these changes]
    - Technical Impact: [How these changes affect the codebase]

    📝 Suggestions:
    ━━━━━━━━━━━━━━━━━━━━━━━
    Commit: [type]: [message]
    📄 Modified files:
       - [list of modified files, separated by newline and indented]
    Explanation: [commit explanation]

    🎯 Requirements Analysis:
    ⚠️ Criteria Status Overview: [Overall status: e.g., "Partially Met - Some criteria are pending."]
    ❌ Missing Criteria:
       - [Criterion 1]: [Detailed explanation of why it's missing or partially met]
       - [Criterion 2]: [Detailed explanation of why it's missing or partially met]
       - ... (List all criteria not fully met)
    💡 Improvement Suggestions:
       - [Suggestion for Criterion 1]: [Specific action to fully meet Criterion 1]
       - [Suggestion for Criterion 2]: [Specific action to fully meet Criterion 2]
       - ... (Suggestions for all missing/partially met criteria)
    ━━━━━━━━━━━━━━━━━━━━━━━

    Now, generate %d similar suggestions based on the following information.

    Modified files:
    %s

    Diff:
    %s

    Ticket Information:
    %s
    `

	promptTemplateWithTicketES = `
    Instrucciones:
    1. Generá %d sugerencias de mensajes de commit basadas en los cambios de código proporcionados y la información del ticket.
    2. Cada sugerencia DEBE seguir el formato definido en la sección "Formato de Sugerencia".
    3. **Analizá críticamente los cambios de código en detalle y comparalos rigurosamente con los "Criterios de Aceptación" proporcionados en la sección "Información del Ticket".**
    4. **Para cada criterio de aceptación, determiná explícitamente si se cumple completamente, parcialmente o no se cumple con los cambios de código.**
    5. **En la sección "🎯 Análisis de Criterios de Aceptación", proporcioná un desglose detallado del estado de los criterios de aceptación. Para cada criterio que NO se cumpla completamente, listalo bajo "❌ Criterios Faltantes" y proporcioná sugerencias de mejora específicas y accionables bajo "💡 Sugerencias de Mejora" para cumplir completamente el criterio.**
    6. Usá tipos de commit apropiados:
        - feat: Nuevas funcionalidades
        - fix: Correcciones de bugs
        - refactor: Reestructuración de código
        - test: Agregar o modificar pruebas
        - docs: Actualizaciones de documentación
        - chore: Tareas de mantenimiento
    7. Mantené los mensajes de commit en menos de 100 caracteres.
    8. Proporcioná sugerencias de mejora específicas y accionables, especialmente relacionadas con el cumplimiento de los criterios de aceptación.

    Formato de Sugerencia:
    =========[ Sugerencia ]=========
    [número]. [Ordinal] sugerencia:
    🔍 Analizando cambios...

    📊 Análisis de Código:
    - Resumen de Cambios: [Breve resumen de qué cambió en el código]
    - Propósito Principal: [Objetivo principal de estos cambios]
    - Impacto Técnico: [Cómo estos cambios afectan la base de código]

    📝 Sugerencias:
    ━━━━━━━━━━━━━━━━━━━━━━━
    Commit: [tipo]: [mensaje]
    📄 Archivos modificados:
       - [lista de archivos modificados, separados por nueva línea e indentados]
    Explicación: [explicación del commit]

    🎯 Análisis de Criterios de Aceptación:
    ⚠️ Resumen del Estado de Criterios: [Estado general: ej., "Cumplimiento Parcial - Algunos criterios están pendientes."]
    ❌ Criterios Faltantes:
       - [Criterio 1]: [Explicación detallada de por qué falta o se cumple parcialmente]
       - [Criterio 2]: [Explicación detallada de por qué falta o se cumple parcialmente]
       - ... (Listar todos los criterios no cumplidos completamente)
    💡 Sugerencias de Mejora:
       - [Sugerencia para Criterio 1]: [Acción específica para cumplir completamente el Criterio 1]
       - [Sugerencia para Criterio 2]: [Acción específica para cumplir completamente el Criterio 2]
       - ... (Sugerencias para todos los criterios faltantes/parcialmente cumplidos)
    ━━━━━━━━━━━━━━━━━━━━━━━

    Ahora, generá %d sugerencias similares basándote en la siguiente información.

    Archivos modificados:
    %s

    Diff:
    %s

    Información del Ticket:
    %s
    `
)

// Templates para Commits sin ticket
const (
	// Template en español sin ticket
	promptTemplateWithoutTicketES = `
    Instrucciones:
    1. Generá %d sugerencias de mensajes de commit basadas en los cambios de código proporcionados.
    2. Cada sugerencia DEBE seguir el formato definido en la sección "Formato de Sugerencia".
    3. Analizá los cambios de código en detalle para proporcionar sugerencias precisas.
    4. Concentrate en aspectos técnicos, mejores prácticas, calidad del código e impacto en la mantenibilidad/rendimiento.
    5. Usá tipos de commit apropiados:
        - feat: Nuevas funcionalidades
        - fix: Correcciones de bugs
        - refactor: Reestructuración de código
        - test: Agregar o modificar pruebas
        - docs: Actualizaciones de documentación
        - chore: Tareas de mantenimiento
    6. Mantené los mensajes de commit en menos de 100 caracteres.
    7. Proporcioná sugerencias de mejora específicas y accionables.

    Formato de Sugerencia:
    =========[ Sugerencia ]=========
    [número]. [Ordinal] sugerencia:
    🔍 Analizando cambios...
    
    📊 Análisis de Código:
    - Resumen de Cambios: [Breve resumen de qué cambió en el código]
    - Propósito Principal: [Objetivo principal de estos cambios]
    - Impacto Técnico: [Cómo estos cambios afectan la base de código]
    
    📝 Sugerencias:
    ━━━━━━━━━━━━━━━━━━━━━━━
    Commit: [tipo]: [mensaje]
    📄 Archivos modificados:
       - [lista de archivos modificados, separados por nueva línea e indentados]
    Explicación: [explicación del commit]
    
    💭 Análisis Técnico:
    %s
    ━━━━━━━━━━━━━━━━━━━━━━━

    Ahora, generá %d sugerencias similares basándote en la siguiente información.

    Archivos modificados:
    %s
    
    Diff:
    %s
    `

	promptTemplateWithoutTicketEN = `
    Instructions:
    1. Generate %d commit message suggestions based on the provided code changes.
    2. Each suggestion MUST follow the format defined in the "Suggestion Format" section.
    3. Analyze code changes in detail to provide accurate suggestions.
    4. Focus on technical aspects, best practices, code quality and impact on maintainability/performance.
    5. Use appropriate commit types:
        - feat: New features
        - fix: Bug fixes
        - refactor: Code restructuring
        - test: Adding or modifying tests
        - docs: Documentation updates
        - chore: Maintenance tasks
    6. Keep commit messages under 100 characters.
    7. Provide specific, actionable improvement suggestions.

    Suggestion Format:
    =========[ Suggestion ]=========
    [number]. [Ordinal] suggestion:
    🔍 Analyzing changes...
    
    📊 Code Analysis:
    - Changes Overview: [Brief overview of what changed in the code]
    - Primary Purpose: [Main goal of these changes]
    - Technical Impact: [How these changes affect the codebase]
    
    📝 Suggestions:
    ━━━━━━━━━━━━━━━━━━━━━━━
    Commit: [type]: [message]
    📄 Modified files:
       - [list of modified files, separated by newline and indented]
    Explanation: [commit explanation]
    
    💭 Technical Analysis:
    %s
    ━━━━━━━━━━━━━━━━━━━━━━━

    Now, generate %d similar suggestions based on the following information.

    Modified files:
    %s
    
    Diff:
    %s
    `
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

  REGLAS DE ESTILO:
  - Primera persona: "Implementé", "Mejoré", "Arreglé", "Agregué"
  - Voseo natural: "podés", "tenés", "querés" (en vez de "puedes", "tienes", "quieres")
  - Expresiones naturales: "mucho más simple", "ahora funciona mejor", "sin vueltas"
  - Tono profesional pero directo, como si le explicaras a un colega
  - Sé técnico y preciso, pero accesible
  - NO uses emojis en el contenido de las release notes

  REGLAS CRÍTICAS - PREVENCIÓN DE ALUCINACIONES:
  1. Basate EXCLUSIVAMENTE en los commits listados arriba en "Cambios en este release"
  2. Si la sección de cambios está vacía o solo tiene cambios menores (ej: bump de versión), escribí un resumen breve y honesto
  3. NO inventes features, comandos, flags, o funcionalidades que no aparezcan explícitamente en los commits
  4. NO menciones "validadores", "linters", "nuevas opciones" u otras features a menos que estén en los commits
  5. Si no hay suficiente información para un ejemplo específico, usá ejemplos genéricos del uso básico del proyecto
  6. Para EXAMPLES, solo mostrá comandos que realmente existan según los commits. Si no hay cambios significativos, mostrá el uso básico existente
  7. Para COMPARISONS, solo incluí comparaciones si hay cambios concretos que comparar. Si no hay, usá "N/A" o una comparación genérica de versiones
  8. Si los cambios son principalmente internos o de mantenimiento, decilo claramente en vez de inventar features visibles al usuario

  VALIDACIÓN DE CONTENIDO:
  Antes de escribir cada sección, preguntate: "¿Este detalle específico está en los commits que me pasaron?"
  Si la respuesta es NO, no lo incluyas.

  Formato de respuesta (IMPORTANTE: Incluí TODAS las secciones):
  
  TÍTULO: <título conciso y descriptivo (máximo 60 caracteres)>
  
  RESUMEN: <2-3 oraciones en primera persona contando los cambios más importantes. Si no hay cambios significativos, sé honesto al respecto>
  
  HIGHLIGHTS:
  - <highlight 1 en primera persona, basado en commits reales>
  - <highlight 2 en primera persona, basado en commits reales>
  - <highlight 3 en primera persona, basado en commits reales>
  (Si no hay suficientes highlights reales, enfocate en mantenimiento, estabilidad o preparación para futuras features)
  
  QUICK_START:
  <Instrucciones de instalación/actualización en 2-3 líneas. Usá el repositorio real: github.com/%s/%s>
  IMPORTANTE: Este proyecto es un CLI de Go. Usá "go install github.com/%s/%s@<version>" para instalación.
  No inventes flags o comandos que no existan.
  
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
  (Solo incluí ejemplos de funcionalidad que realmente exista. Si no hay nuevas features, mostrá el uso básico existente)
  
  BREAKING_CHANGES:
  - <cambio breaking 1, o "Ninguno" si no hay>
  (Solo listá breaking changes si están explícitamente mencionados en los commits)
  
  COMPARISONS:
  COMPARISON_1:
  FEATURE: <nombre de la feature que realmente cambió>
  BEFORE: <estado anterior según los commits>
  AFTER: <estado nuevo según los commits>
  (Si no hay comparaciones concretas basadas en los commits, usá "N/A" o una comparación genérica de versiones)
  
  LINKS:
  (Solo incluí links si son relevantes para esta release específica, como issues cerrados o PRs relacionados. Si no hay links relevantes, poné "N/A")
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
