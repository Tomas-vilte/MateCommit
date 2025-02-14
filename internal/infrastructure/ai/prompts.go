package ai

// Templates para Pull Requests
const (
	prPromptTemplateEN = `Hey, could you create a PR summary following this **Markdown format**?

	**Response Format:**
	
	## PR Title
	A **short, descriptive title** (max 80 chars) capturing the PR's essence. Example: "fix: Image loading error". **Ensure the title is informative even without reading the full summary.**
	
	## Key Changes
	Describe **the 3 most significant changes** in the PR. For each change, include:
	- **Change Description:** What was modified?
	- **Purpose of Change:** Why was this done? What problem does it solve or improvement does it introduce?
	- **Technical Impact (if relevant):** How does this affect the codebase, architecture, or performance?
	
	**Focus on changes most relevant to understanding the PR's main goal and impact.** Avoid minor implementation details.
	
	## Suggested Tags
	Comma-separated **relevant tags** from: feature, fix, refactor, docs, infra, test. Example: fix,infra. **Select tags that best represent the changes.**
	
	**Additional Context (user-provided):**
	%s
	
	**PR Content (Diff):**
	%s
	
	**Instructions:**
	* **Prioritize important and relevant information.** Avoid unnecessary details.
	* **Be concise and clear in each section.** Use precise language, avoid excessive jargon.
	* **Ensure the summary is understandable to someone unfamiliar with code details.**
	* **Use "Additional Context" (if provided) to guide key changes selection and enrich explanations.**
	
	Thanks a ton! You're awesome!`

	prPromptTemplateES = `Che, necesito que me **devuelvas la respuesta en el siguiente formato Markdown**:

	**Formato de la Respuesta:**
	
	## Título del PR
	Un título **breve y descriptivo** (máx 80 caracteres) que capture la esencia del PR.  Ej: "fix: Error al cargar imágenes". **Procura que el título sea informativo incluso sin leer el resto del resumen.**
	
	## Cambios clave
	Describe **los 3 cambios más importantes y significativos** del PR.  Para cada cambio, incluye **brevemente**:
	- **Descripción del cambio:** ¿Qué se modificó?
	- **Propósito del cambio:** ¿Por qué se hizo este cambio? ¿Qué problema resuelve o qué mejora introduce?
	- **Impacto técnico (si aplica y es relevante):**  ¿Cómo afecta este cambio a la base de código, la arquitectura o el rendimiento?
	
	**Prioriza los cambios que sean más relevantes para entender el objetivo principal del PR y su impacto.**  Evita detalles de implementación menores.
	
	## Etiquetas sugeridas
	Lista de etiquetas **relevantes y concisas**, separadas por coma.  Opciones: feature, fix, refactor, docs, infra, test.  Ej: fix,infra. **Selecciona las etiquetas que mejor representen el tipo de cambios realizados en el PR.**
	
	**Contexto adicional (opcional, proporcionado por el usuario):**
	%s
	
	**Contenido del PR (Diff):**
	%s
	
	**Instrucciones adicionales:**
	
	*   **Concéntrate en la información más importante y relevante.**  Evita detalles innecesarios.
	*   **Sé conciso y claro en cada sección.**  Utiliza lenguaje preciso y evita jerga técnica excesiva a menos que sea necesario.
	*   **Asegúrate de que el resumen sea comprensible para alguien que no esté familiarizado con los detalles del código.**
	*   **Si se proporciona "Contexto adicional", úsalo para guiar la selección de los "Cambios clave" y para enriquecer la "Explicación" de cada cambio.**
	
	¡Gracias máquina!
`
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
