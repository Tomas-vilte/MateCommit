<div align="center">
  <img src="/home/enano/.gemini/antigravity/brain/8d348096-a7bc-4507-b3d1-e0ceeee5f35f/matecommit_social_preview_1766175423747.png" alt="MateCommit Logo" width="640">

  # MateCommit

  **Inteligencia Artificial para tu flujo de trabajo en Git**

  MateCommit elimina la fricción de escribir mensajes de commit. Utiliza el poder de Gemini AI para analizar tus cambios y sugerir títulos coherentes y profesionales, permitiéndote enfocarte en lo que realmente importa: tu código.

  [Go Report Card](https://goreportcard.com/report/github.com/Tomas-vilte/MateCommit) | [Licencia](https://opensource.org/licenses/MIT) | [Estado del Build](https://github.com/Tomas-vilte/MateCommit/actions)

</div>

---

### Idiomas
*   [Documentación Oficial (Inglés)](../../README.md)
*   [Traducción al Español (🇦🇷)](#)

---

## Qué ofrece MateCommit
Escribir buenos nombres para los commits es fundamental pero consume tiempo. Esta herramienta automatiza esa tarea analizando el `diff` de tus archivos en staging.

*   **Sugerencias Inteligentes**: Análisis contextual de lógica, no solo nombres de archivos.
*   **Potencia de Gemini**: Optimizado para modelos Flash 1.5 y 2.0 para máxima precisión.
*   **Ciclo de vida de Issues**: Generá issues de GitHub desde código, PRs o descripciones.
*   **Releases Unificados**: Automatización de changelogs, tagging y publicación en un solo paso.
*   **Inteligencia en PRs**: Resúmenes instantáneos para Pull Requests complejos.
*   **Control de Costos**: Seguimiento de gastos y estadísticas de uso de IA en tiempo real.
*   **Herramientas de Eficiencia**: Autocompletado, caché local y herramientas de diagnóstico.


---

## Inicio Rápido

### 1. Instalación
La forma más rápida de instalarlo es a través de Go:

```bash
go install github.com/Tomas-vilte/MateCommit/cmd/matecommit@latest
```

### 2. Configuración
Corré el asistente interactivo para configurar tus API Keys:

```bash
matecommit config init
```

### 3. Uso
Agregá tus cambios y pedí sugerencias:

```bash
git add .
matecommit suggest
```

---

## Funcionalidad Avanzada
Diseñado para entornos profesionales:

*   **Integración con Jira**: Linkeo automático de tickets basado en el contexto.
*   **Resúmenes de PR**: Generación automática de descripciones para Pull Requests.
*   **Automatización de Releases**: Actualización de changelogs y versionado en un solo paso.

Para una guía detallada de comandos, consultá [COMMANDS.md](../../COMMANDS.md).

---

## Contribuciones
Valoramos las contribuciones de calidad. Si querés mejorar el proyecto, revisá nuestra [Guía de Contribución](../../CONTRIBUTING.md).

---

## Licencia
Código abierto bajo licencia MIT. Consultá [LICENSE](../../LICENSE) para más detalles.