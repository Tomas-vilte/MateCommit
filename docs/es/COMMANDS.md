# Referencia de la CLI de MateCommit 🧉

Escribí esta guía para explicarte no solo *qué* hace cada comando, sino cómo laburan por detrás. El diseño de la herramienta es modular, lo que me permite ir sumando modelos de IA y plataformas nuevas sin que se rompa todo el flujo que ya venís usando.

---

## 1. El motor de sugerencias

### `suggest` / `s`
Es el comando que más uso. Analiza lo que tenés en stage y le pide a la IA que te tire opciones de mensajes de commit que realmente tengan sentido.

**Uso:**
```bash
matecommit suggest [flags]
```

**Cómo funciona la magia:**
1.  **Análisis de Diff**: Ejecuto un `git diff --cached` para ver exactamente qué tocaste.
2.  **Contexto**: Armo un prompt para el proveedor (como Gemini) con el resumen del diff y los archivos.
3.  **Manejo de archivos grandes**: Si tu diff es gigante, no te tiro un error por la cabeza. Uso un algoritmo que prioriza los cambios lógicos más importantes para mantenerme dentro de los límites del modelo sin perder calidad.
4.  **Plus de contexto**: Si le pasás la flag `--issue`, voy a buscar el título y la descripción del ticket para que la IA entienda el "porqué" real de tus cambios.

**Flags disponibles:**

`--count` / `-n` (int)
> Cuántas opciones querés que te tire de una. (Default: 3, Máximo: 10)

`--lang` / `-l` (string)
> Si querés forzar un idioma para ese commit puntual (ej. si laburás en un repo en inglés pero tu config está en español).

`--issue` / `-i` (int)
> Trae toda la info de un issue específico para darle más "inteligencia" a la sugerencia.

`--no-emoji` / `-ne` (bool)
> Saca los emojis si necesitás un historial de commits bien sobrio y técnico.

**Tip de uso**: Si tirás `matecommit suggest -n 5 -l en`, te genera 5 opciones en inglés al toque, sin importar qué tengas configurado por defecto.

---

## 2. Gestión de PRs e Issues

### `summarize-pr` / `spr`
Lo uso cuando tengo que cerrar un PR y me da paja escribir todo el resumen, el plan de pruebas y buscar si hay cambios disruptivos.

**El flujo es simple:**
1.  **Metadata**: Levanta los commits y comentarios desde la API de tu VCS (GitHub, por ahora).
2.  **Síntesis**: El LLM lee toda la historia del PR y te arma un resumen cohesivo.
3.  **Push**: Actualiza la descripción del PR directamente en la plataforma por vos.

### `issue generate` / `g`
Odio tener que salir de la terminal y abrir el navegador solo para crear un ticket. Este comando transforma lo que estás haciendo en un issue profesional.

**De dónde saca la info:**
- **Desde Diff**: Usa tus cambios actuales como base para describir el problema o la tarea.
- **Checkout Automático**: Si usás `--checkout`, después de crear el issue te abre una rama nueva con el nombre correcto para que empieces a laburar ahí mismo.

---

## 3. Automatización de Releases

### `release` / `r`
Construí esto para sacarme de encima el estrés de manejar el versionado semántico (SemVer) a mano.

1.  **Análisis**: Revisa tu historial de commits (basándose en Conventional Commits) y te sugiere si el salto es Patch, Minor o Major.
2.  **Changelog**: Te actualiza el `CHANGELOG.md` automáticamente con lo nuevo.
3.  **Tags**: Crea el tag de git localmente.
4.  **Publicación**: Sube todo a tu VCS y crea el Release con las notas generadas por IA.

---

## 4. Configuración y Sistema

### `config`
Todos tus ajustes se guardan en `~/.config/matecommit/config.yaml`.
*   **Prioridades**: Si tirás una flag en el comando, eso manda por sobre la variable de entorno o el archivo de configuración.
*   **Doctor**: Si algo no anda, tirá `matecommit config doctor`. Chequea conexiones, permisos de tokens y que las APIs respondan.

### `stats`
Como las APIs de IA no son gratis (o tienen límites), agregué un seguimiento de tokens. Así podés ver cuánto venís gastando y no llevarte una sorpresa a fin de mes.

---

## Solución de problemas comunes

**"Las sugerencias no son muy buenas"**
*   *Consejo*: Asegurate de stagear solo los cambios que tengan que ver entre sí. Si metés 5 features distintas en un mismo stage, la IA se marea con el contexto.

**"Error de API"**
*   *Consejo*: Corré el comando `doctor`. Lo más probable es que tu `GEMINI_API_KEY` o `GITHUB_TOKEN` hayan expirado o no tengan los permisos (scopes) necesarios.

---

## Soporte actual

*   **Modelos de IA**: Google Gemini (Por defecto).
*   **VCS**: GitHub.
*   **Issues**: Jira y GitHub Issues.