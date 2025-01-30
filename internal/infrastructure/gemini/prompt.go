package gemini

const (
	promptTemplateEN = `
	Instructions:
    1. Generate %d commit message suggestions based on the provided code changes and ticket information (if provided).
    2. Each suggestion MUST follow the format defined in the "Suggestion Format" section.
    3. Analyze code changes in detail to provide accurate suggestions.
    4. If a ticket is provided, compare code changes against acceptance criteria, flag any missing implementations and suggest specific improvements.
    5. If no ticket is provided, focus on technical aspects, best practices, code quality and impact on maintainability/performance.
    6. Use appropriate commit types:
        - feat: New features
        - fix: Bug fixes
        - refactor: Code restructuring
        - test: Adding or modifying tests
        - docs: Documentation updates
        - chore: Maintenance tasks
    7. Keep commit messages under 100 characters.
    8. Provide specific, actionable improvement suggestions.

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

    Example:
    =========[ Suggestion ]=========
    1. First suggestion:
    🔍 Analyzing changes...
    
    📊 Code Analysis:
    - Changes Overview: Implementation of Jira API integration and error handling
    - Primary Purpose: Enable ticket information retrieval from Jira
    - Technical Impact: Adds new service layer for Jira integration with proper error handling
    
    📝 Suggestions:
    ━━━━━━━━━━━━━━━━━━━━━━━
    Commit: ✨ feat: Integrate Jira API with error handling and tests
    📄 Modified files:
       - cmd/main.go
       - internal/infrastructure/jira/service.go
       - internal/infrastructure/jira/service_test.go
    Explanation: Added Jira API integration with comprehensive error handling and test coverage
    
    🎯 Requirements Analysis:
    ✅ Criteria Status: Fully met
    ⚠️  Missing Criteria: 
       - Authentication with different token types not implemented
       - Retry mechanism for failed API calls missing
    💡 Improvement Suggestions: 
       - Add support for multiple authentication methods
       - Implement retry strategy for API failures
       - Add detailed logging for debugging
    ━━━━━━━━━━━━━━━━━━━━━━━

    Now, generate %d similar suggestions based on the following information.

    Modified files:
    %s
    
    Diff:
    %s
    %s
    `

	promptTemplateES = `
    Instrucciones:
    1. Generá %d sugerencias de mensajes de commit basadas en los cambios de código proporcionados y la información del ticket (si se proporciona).
    2. Cada sugerencia DEBE seguir el formato definido en la sección "Formato de Sugerencia".
    3. Analizá los cambios de código en detalle para proporcionar sugerencias precisas.
    4. Si se proporciona un ticket, compará los cambios de código con los criterios de aceptación específicamente:
       - Evaluá cada criterio de aceptación individualmente
       - Indicá claramente qué criterios están implementados y cuáles no
       - Proporcioná sugerencias específicas para los criterios no implementados
    5. Si no se proporciona un ticket, concentrate en aspectos técnicos, mejores prácticas, calidad del código e impacto en la mantenibilidad/rendimiento.
    6. Usá tipos de commit apropiados:
        - feat: Nuevas funcionalidades
        - fix: Correcciones de bugs
        - refactor: Reestructuración de código
        - test: Agregar o modificar pruebas
        - docs: Actualizaciones de documentación
        - chore: Tareas de mantenimiento
    7. Mantené los mensajes de commit en menos de 100 caracteres.
    8. Proporcioná sugerencias de mejora específicas y accionables.

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
    ⚠️ Estado de los Criterios: [completamente_cumplidos/parcialmente_cumplidos/no_cumplidos]
    
    ❌ Criterios Pendientes:
    %s
    
    💡 Sugerencias de Mejora:
    - [Lista detallada de mejoras específicas para implementar los criterios pendientes]
    ━━━━━━━━━━━━━━━━━━━━━━━

    Ejemplo de Análisis de Criterios:
    🎯 Análisis de Criterios de Aceptación:
    ⚠️ Estado de los Criterios: Parcialmente cumplidos
    
    ❌ Criterios Pendientes:
    - Conexión a la API de Jira:
      * Falta implementar manejo de token expirado
      * No se detecta implementación de manejo de API no disponible
    - Extracción de Tickets:
      * No se encuentra implementación de almacenamiento en estructura TicketInfo
    
    💡 Sugerencias de Mejora:
    - Implementar manejo de errores para token expirado
    - Agregar retry mechanism para API no disponible
    - Crear estructura TicketInfo y implementar almacenamiento
    ━━━━━━━━━━━━━━━━━━━━━━━

    Ahora, generá %d sugerencias similares basándote en la siguiente información.

    Archivos modificados:
    %s
    
    Diff:
    %s
    %s
    `
)
