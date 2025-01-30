package gemini

const (
	// Template en inglés con ticket
	promptTemplateWithTicketEN = `
    Instructions:
    1. Generate %d commit message suggestions based on the provided code changes and ticket information.
    2. Each suggestion MUST follow the format defined in the "Suggestion Format" section.
    3. Analyze code changes in detail to provide accurate suggestions.
    4. Compare code changes against acceptance criteria, flag any missing implementations and suggest specific improvements.
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
    
    🎯 Requirements Analysis:
    %s
    ━━━━━━━━━━━━━━━━━━━━━━━

    Now, generate %d similar suggestions based on the following information.

    Modified files:
    %s
    
    Diff:
    %s

    Ticket Information:
    %s
    `

	// Template en inglés sin ticket
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

	// Template en español con ticket
	promptTemplateWithTicketES = `
    Instrucciones:
    1. Generá %d sugerencias de mensajes de commit basadas en los cambios de código proporcionados y la información del ticket.
    2. Cada sugerencia DEBE seguir el formato definido en la sección "Formato de Sugerencia".
    3. Analizá los cambios de código en detalle para proporcionar sugerencias precisas.
    4. Compará los cambios de código con los criterios de aceptación, señalá cualquier implementación faltante y sugerí mejoras específicas.
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
    
    🎯 Análisis de Criterios de Aceptación:
    %s
    ━━━━━━━━━━━━━━━━━━━━━━━

    Ahora, generá %d sugerencias similares basándote en la siguiente información.

    Archivos modificados:
    %s
    
    Diff:
    %s

    Información del Ticket:
    %s
    `

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
)
