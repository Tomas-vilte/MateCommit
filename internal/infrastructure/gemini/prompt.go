package gemini

const (
	promptTemplateEN = `Generate %d commit message suggestions. Respond with the following structure for EACH suggestion:
	=========[ Suggestion ]=========
	[number]. [Ordinal] suggestion:
	Commit: [type]: [message]
	Files: [list of modified files, separated by comma]
	Explanation: [commit explanation]
	Criteria Status: [Indicate if the acceptance criteria are fully met, partially met, or not met.]
	Missing Criteria: [List the specific criteria that are missing, if any.]
	Improvement Suggestions: [Provide suggestions for improvement, if any.]
	
	Example (with emojis):
	=========[ Suggestion ]=========
	1. First suggestion:
	Commit: ✨ feat: add config option for commit suggestion generation
	Files: main.go, config.go
	Explanation: Added a new configuration option to enable commit suggestion generation.
	Criteria Status: fully_met
	Missing Criteria: None
	Improvement Suggestions: None
	
	=========[ Suggestion ]=========
	2. Second suggestion:
	Commit: 🐛 fix: resolve login issues
	Files: auth.go, login.go
	Explanation: Fixed an issue where users were unable to log in due to a validation error.
	Criteria Status: partially_met
	Missing Criteria: Ensure login works with 2FA.
	Improvement Suggestions: Implement two-factor authentication (2FA) in the login module.
	
	=========[ Suggestion ]=========
	3. Third suggestion:
	Commit: 📚 docs: update documentation for API endpoints
	Files: api.md
	Explanation: Updated the documentation for all available API endpoints.
	Criteria Status: not_met
	Missing Criteria: Ensure all endpoints are documented.
	Improvement Suggestions: Add documentation for the missing endpoints.
	
	Now, generate %d similar suggestions based on the following information.
	
	Modified files:
	%s
	Diff:
	%s
	%s  <!-- Aquí se agregará la información del ticket si está disponible -->
	
	Additional Instructions:
	1. Each commit message must follow the exact template above.
	2. Commit messages should be clear and concise.
	3. Limit each commit message to 100 characters.
	4. Ensure that the commit type matches the change (e.g., feat, fix, refactor, chore).
	5. Use a variety of commit types (feat, fix, docs, chore, refactor, etc).
	6. The ordinal must be correct (e.g., "First", "Second", "Third", etc.)
	7. If acceptance criteria are provided, verify if the code meets them. Indicate if the criteria are fully met, partially met, or not met.
	8. If criteria are not fully met, list the specific criteria that are missing and provide suggestions for improvement.
	9. Follow the exact structure for Criteria Status, Missing Criteria, and Improvement Suggestions. Do not mix them with the Explanation.
	10. Use the following format for Criteria Status, Missing Criteria, and Improvement Suggestions:
	    - Criteria Status: [fully_met, partially_met, or not_met]
	    - Missing Criteria: [list of missing criteria, separated by commas]
	    - Improvement Suggestions: [list of suggestions, separated by commas]
	`

	promptTemplateES = `Generá %d sugerencias de mensajes de commit. Respondé con la siguiente estructura para CADA sugerencia:
	=========[ Sugerencia ]=========
	[número]. [Ordinal] sugerencia:
	Commit: [tipo]: [mensaje]
	Archivos: [lista de archivos modificados, separados por coma]
	Explicación: [explicación del commit]
	Estado de los Criterios: [Indicá si los criterios de aceptación se cumplen completamente, parcialmente o no se cumplen.]
	Criterios Faltantes: [Listá los criterios específicos que faltan, si los hay.]
	Sugerencias de Mejora: [Proporcioná sugerencias de mejora, si las hay.]
	
	Ejemplo (con emojis):
	=========[ Sugerencia ]=========
	1. Primera sugerencia:
	Commit: ✨ feat: Agregar opción de configuración para generación de sugerencias de commit
	Archivos: main.go, config.go
	Explicación: Se agregó una nueva opción de configuración para habilitar la generación de sugerencias de commit.
	Estado de los Criterios: completamente_cumplidos
	Criterios Faltantes: Ninguno
	Sugerencias de Mejora: Ninguna
	
	=========[ Sugerencia ]=========
	2. Segunda sugerencia:
	Commit: 🐛 fix: Corregir problemas de inicio de sesión
	Archivos: auth.go, login.go
	Explicación: Se corrigió un problema por el cual los usuarios no podían iniciar sesión debido a un error de validación.
	Estado de los Criterios: parcialmente_cumplidos
	Criterios Faltantes: Asegurar que el inicio de sesión funcione con 2FA.
	Sugerencias de Mejora: Implementar autenticación de dos factores (2FA) en el módulo de inicio de sesión.
	
	=========[ Sugerencia ]=========
	3. Tercera sugerencia:
	Commit: 📚 docs: Actualizar documentación para endpoints de la API
	Archivos: api.md
	Explicación: Se actualizó la documentación para todos los endpoints de la API disponibles.
	Estado de los Criterios: no_cumplidos
	Criterios Faltantes: Asegurar que todos los endpoints estén documentados.
	Sugerencias de Mejora: Agregar documentación para los endpoints faltantes.
	
	Ahora, generá %d sugerencias similares basándote en la siguiente información.
	
	Archivos modificados:
	%s
	Diff:
	%s
	%s  <!-- Aquí se agregará la información del ticket si está disponible -->
	
	Instrucciones adicionales:
	1. Cada mensaje de commit tiene que seguir la estructura exacta de arriba.
	2. Los mensajes de commit tienen que ser claros y concisos.
	3. Limitá cada mensaje de commit a 100 caracteres.
	4. Asegurate de que el tipo de commit coincida con el cambio (e.g., feat, fix, refactor, chore).
	5. Usá una variedad de tipos de commit (feat, fix, docs, chore, refactor, etc).
	6. El ordinal tiene que ser correcto (e.g., "Primera", "Segunda", "Tercera", etc.)
	7. Si se proporcionan criterios de aceptación, verificá si el código los cumple. Indicá si los criterios se cumplen completamente, parcialmente o no se cumplen.
	8. Si los criterios no se cumplen completamente, listá los criterios específicos que faltan y proporcioná sugerencias de mejora.
	9. Seguí la estructura exacta para Estado de los Criterios, Criterios Faltantes y Sugerencias de Mejora. No los mezcles con la Explicación.
	10. Usá el siguiente formato para Estado de los Criterios, Criterios Faltantes y Sugerencias de Mejora:
	    - Estado de los Criterios: [completamente_cumplidos, parcialmente_cumplidos, o no_cumplidos]
	    - Criterios Faltantes: [lista de criterios faltantes, separados por comas]
	    - Sugerencias de Mejora: [lista de sugerencias, separadas por comas]
	`
)
