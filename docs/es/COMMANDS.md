# Referencia de CLI MateCommit

**Manual Técnico y Guía de Uso**

Este documento es la referencia definitiva para exprimir al máximo MateCommit. Acá vas a encontrar el detalle fino de cómo funciona cada comando, qué banderas ("flags") podés usar y cómo integrar la herramienta en tu flujo de trabajo diario.

---

## 1. Inteligencia en Commits

### `suggest` / `s`
Este comando analiza tus cambios en staging (`git diff --cached`) y le pide al modelo de IA que genere mensajes siguiendo Conventional Commits.

**Comando:**
```bash
matecommit suggest [flags]
```

**Detalles Técnicos:**
*   **Context Window**: La herramienta envía el resumen del diff, nombres de archivos y (si lo activás) el contexto del issue linkeado. Si tu diff es gigante, lo truncamos inteligentemente para que entre en el límite de tokens del modelo sin perder la lógica principal.
*   **Precedencia**: Los flags que pases por consola siempre le ganan a lo que tengas en `config.yaml`. (ej: si tenés configurado inglés pero tirás `--lang es`, sale en español).

**Flags:**
| Flag | Corto | Tipo | Descripción |
| :--- | :--- | :--- | :--- |
| `--count` | `-n` | `int` | Cantidad de opciones a generar (1-10). |
| `--lang` | `-l` | `string` | Idioma de salida (ej: `es`, `en`, `pt`). |
| `--issue` | `-i` | `int` | Busca el título/descripción del issue en GitHub para darle contexto a la IA. |
| `--no-emoji` | `-ne` | `bool` | Vuela los emojis del título para mantener un historial sobrio. |

**Ejemplo Avanzado:**
```bash
# Dame 5 sugerencias en español, leé el contexto del issue #42 y sacame los emojis
matecommit suggest -n 5 -l es -i 42 --no-emoji
```

---

## 2. Gestión de Pull Requests

### `summarize-pr` / `spr`
Genera un Resumen, Plan de Pruebas y Alerta de Breaking Changes para un PR existente.

**Comando:**
```bash
matecommit spr --pr-number <id>
```

**Workflow:**
1.  **Fetch**: Traemos la metadata del PR (commits, diffs, issues linkeados) vía API de GitHub.
2.  **Análisis**: Gemini sintetiza los cambios en una narrativa coherente.
3.  **Update**: Parcheamos el cuerpo del PR directamente en GitHub.

**Requisitos:**
*   Necesitás el `GITHUB_TOKEN` configurado.
*   Scopes del token: `repo` (para repos privados) o `public_repo` (para públicos).

---

## 3. Ciclo de Vida de Issues

### `issue generate` / `g`
Crea issues en GitHub usando IA para transformar inputs vagos en reportes profesionales.

**Comando:**
```bash
matecommit issue generate [source-flags] [opciones]
```

**Fuentes (Elegí una):**
*   `--from-diff` / `-d`: Usa tus cambios en staging como base (Ideal para: "Ya arreglé esto, ahora necesito el ticket").
*   `--from-pr` / `-p`: Usa el título/body de un PR para crear un issue de seguimiento.
*   `--description` / `-m`: Usa un texto plano que le pases como input.

**Opciones:**
*   `--template` / `-t`: Apunta a un template específico (ej: `bug_report`). Matchea con los archivos en `.github/ISSUE_TEMPLATE/`.
*   `--checkout` / `-c`: Automatiza la creación de la rama (`git checkout -b issue/123-titulo`) post-generación.
*   `--dry-run`: Te imprime el Markdown en consola sin tocar la API de GitHub.

**Escenario Real:**
*Hiciste un fix rápido pero te colgaste en crear el ticket.*
```bash
git add .
matecommit issue generate --from-diff --template bug_report --assign-me --checkout
```
*Resultado: Crea el issue, te lo asigna a vos y te cambia a la rama correcta.*

---

## 4. Automatización de Releases

### `release` / `r`
Estandarizamos el proceso de release siguiendo [Semantic Versioning](https://semver.org/).

**Subcomandos:**

#### `preview` / `p`
Un "dry-run". Calculamos la próxima versión (ej: `v1.0.0` -> `v1.1.0`) basándonos en el historial de commits y te mostramos el borrador del changelog.

#### `create` / `c`
Ejecuta el release localmente.
1.  Actualiza `CHANGELOG.md` (agrega lo nuevo arriba).
2.  Crea el tag de git.
3.  (Opcional) Pushea los cambios.

**Flags:**
*   `--auto`: Modo no-interactivo (clave para scripts de CI/CD).
*   `--changelog`: Fuerza el commit del archivo de changelog actualizado.
*   `--publish`: Dispara el `git push origin <tag>` inmediatamente.

#### `publish` / `pub`
Sincroniza el tag local con GitHub Releases. Crea la entrada en GitHub con las notas generadas por la IA.

---

## 5. Sistema y Configuración

### `config`
**Ubicación**: `~/.config/matecommit/config.yaml` (o la ruta estándar de tu SO).

*   `init`: El asistente interactivo.
*   `doctor`: Chequeo de salud (Conectividad con Gemini, GitHub, path de Git).

### `stats`
Te muestra una estimación de costos basada en el uso de tokens.
*   **Nota**: Son estimaciones basadas en el precio público de Gemini. La facturación real puede variar.

### `update`
Auto-updater usando GitHub Releases. Reemplaza tu binario actual por la última versión estable.

---

## Variables de Entorno

MateCommit respeta estas variables, que tienen prioridad sobre el archivo de configuración (ideal para CI):

*   `GEMINI_API_KEY`: Tu Key de Google AI Studio.
*   `GITHUB_TOKEN`: Tu Personal Access Token de GitHub.
*   `MATECOMMIT_LANG`: Override del idioma por defecto.