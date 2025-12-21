# Guía de Contribución para MateCommit

¡Gracias por tu interés en contribuir a MateCommit! Valoramos mucho el tiempo y esfuerzo dedicados a mejorar este proyecto. Para mantener la calidad del código y la coherencia técnica, solicitamos seguir estas pautas generales.

---

## Proceso de Desarrollo

### 1. Preparación
*   Realiza un **Fork** del repositorio en GitHub.
*   Clona tu fork localmente.
*   Asegúrate de tener las dependencias al día ejecutando `go mod tidy`.
*   Crea una rama descriptiva para tu cambio: `git checkout -b feature/nombre-de-la-funcionalidad`.

### 2. Estándares de Código
*   **Formato**: El código debe seguir las convenciones estándar de Go (`go fmt`).
*   **Pruebas**: No se aceptarán cambios que rompan los tests existentes. Si agregas una funcionalidad, se espera que incluyas pruebas unitarias o de integración que la validen.
*   **Documentación**: Si tu cambio afecta el uso de la herramienta, actualiza los archivos `README.md` o `COMMANDS.md` correspondientes.

### 3. Mensajes de Commit
Practicamos lo que predicamos. Por favor, utiliza el formato de [Conventional Commits](https://www.conventionalcommits.org/):
*   `feat: descripción de nueva funcionalidad`
*   `fix: corrección de un error`
*   `refactor: mejora del código sin cambios lógicos`

---

## Envío de Pull Requests (PR)

Al abrir un PR, asegúrate de proporcionar el contexto suficiente:
*   **Descripción**: Explica qué cambios realizaste y por qué son necesarios. Evita descripciones genéricas.
*   **Issues**: Vincula cualquier issue relacionado utilizando palabras clave como "Fixes #34".
*   **Verificación**: Asegúrate de que todas las comprobaciones del CI (Integración Continua) pasen correctamente.

---

## Administración y Versiones
La gestión de etiquetas (tags) y lanzamientos de versiones (releases) es responsabilidad exclusiva de los mantenedores del proyecto para garantizar la estabilidad de las versiones distribuidas.

¡Esperamos con interés tus propuestas y mejoras! Gracias por ayudarnos a hacer de MateCommit una herramienta profesional para la comunidad.
