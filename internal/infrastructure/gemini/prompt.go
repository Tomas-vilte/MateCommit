package gemini

const (
	promptTemplateEN = `Generate %d commit message suggestions. Respond with the following structure for EACH suggestion:
	=========[ Suggestion ]=========
	[number]. [Ordinal] suggestion:
	Commit: [type]: [message]
	Files: [list of modified files, separated by comma]
	Explanation: [commit explanation]
	
	Example:
	=========[ Suggestion ]=========
	1. First suggestion:
	Commit: ✨ feat: add config option for commit suggestion generation
	Files: main.go, config.go
	Explanation: Added a new configuration option to enable commit suggestion generation.
	
	=========[ Suggestion ]=========
	2. Second suggestion:
	Commit: 🐛 fix: resolve login issues
	Files: auth.go, login.go
	Explanation: Fixed an issue where users were unable to log in due to a validation error.
	
	=========[ Suggestion ]=========
	3. Third suggestion:
	Commit: 📚 docs: update documentation for API endpoints
	Files: api.md
	Explanation: Updated the documentation for all available API endpoints.
	
	Now, generate %d similar suggestions based on the following information.
	
	Modified files:
	%s
	Diff:
	%s
	
	Instructions:
	1. Each commit message must follow the exact template above.
	2. Commit messages should be clear and concise.
	3. Limit each commit message to 100 characters.
	4. Ensure that the commit type matches the change (e.g., feat, fix, refactor, chore).
	5. Use a variety of commit types (feat, fix, docs, chore, refactor, etc).
	6. The ordinal must be correct (e.g., "First", "Second", "Third", etc.)
	`

	promptTemplateES = `Generá %d sugerencias de mensajes de commit. Respondé con la siguiente estructura para CADA sugerencia:
	=========[ Sugerencia ]=========
	[número]. [Ordinal] sugerencia:
	Commit: [tipo]: [mensaje]
	Archivos: [lista de archivos modificados, separados por coma]
	Explicación: [explicación del commit]
	
	Ejemplo:
	=========[ Sugerencia ]=========
	1. Primera sugerencia:
    Commit: ✨ feat: Agregar opción de configuración para generación de sugerencias de commit
	Archivos: main.go, config.go
	Explicación: Se agregó una nueva opción de configuración para habilitar la generación de sugerencias de commit.
	
	=========[ Sugerencia ]=========
	2. Segunda sugerencia:
	Commit: 🐛 fix: Corregir problemas de inicio de sesión
	Archivos: auth.go, login.go
	Explicación: Se corrigió un problema por el cual los usuarios no podían iniciar sesión debido a un error de validación.
	
	=========[ Sugerencia ]=========
	3. Tercera sugerencia:
	Commit: 📚 docs: Actualizar documentación para endpoints de la API
	Archivos: api.md
	Explicación: Se actualizó la documentación para todos los endpoints de la API disponibles.
	
	Ahora, generá %d sugerencias similares basándote en la siguiente información.
	
	Archivos modificados:
	%s
	Diff:
	%s
	
	Instrucciones:
	1. Cada mensaje de commit tiene que seguir la estructura exacta de arriba.
	2. Los mensajes de commit tienen que ser claros y concisos.
	3. Limitá cada mensaje de commit a 100 caracteres.
	4. Asegurate de que el tipo de commit coincida con el cambio (e.g., feat, fix, refactor, chore).
	5. Usá una variedad de tipos de commit (feat, fix, docs, chore, refactor, etc).
	6. El ordinal tiene que ser correcto (e.g., "Primera", "Segunda", "Tercera", etc.)`
)
