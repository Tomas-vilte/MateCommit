# Referencia de la CLI de MateCommit 🧉

Escribí esta guía para explicarte no solo *qué* hace cada comando, sino cómo laburan por detrás. El diseño de la herramienta es modular, lo que me permite ir sumando modelos de IA y plataformas nuevas sin que se rompa todo el flujo que ya venís usando.

Cada comando tiene su propio `--help`, así que si esta guía en algún momento queda desactualizada respecto al código, esa es la fuente de verdad.

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
2.  **Contexto**: Armo un prompt para el proveedor (Gemini, por ahora) con el resumen del diff y los archivos.
3.  **Manejo de archivos grandes**: Si tu diff es gigante, no te tiro un error por la cabeza. Uso un algoritmo que prioriza los cambios lógicos más importantes para mantenerme dentro de los límites del modelo sin perder calidad.
4.  **Plus de contexto**: Si le pasás la flag `--issue`, voy a buscar el título y la descripción del ticket para que la IA entienda el "porqué" real de tus cambios.

**Flags disponibles:**

`--count` / `-n` (int)
> Cuántas opciones querés que te tire de una. (Default: 3, Máximo: 10)

`--lang` / `-l` (string)
> Si querés forzar un idioma para ese commit puntual (ej. si laburás en un repo en inglés pero tu config está en español).

`--issue` (int)
> Trae toda la info de un issue específico para darle más "inteligencia" a la sugerencia. Ojo, no tiene alias corto.

`--no-emoji` / `--ne`
> Saca los emojis si necesitás un historial de commits bien sobrio y técnico.

`--interactive` / `-i`
> En vez de mandar todo lo que tenés en stage, te deja elegir a mano qué archivos entran en el resumen que le mandás a la IA. Sirve cuando stageaste más de un cambio lógico junto y no tenés ganas de separarlo en varios `git add`.

`--dry-run` / `-d`
> Te muestra la lista de archivos, las estadísticas del diff y una estimación de costo en tokens — sin llamar a la IA ni tocar tu repo. Bueno para chequear antes de gastar una consulta real.

**Tip de uso**: Si tirás `matecommit suggest -n 5 -l en`, te genera 5 opciones en inglés al toque, sin importar qué tengas configurado por defecto.

---

## 2. Gestión de PRs e Issues

### `summarize-pr` / `spr`
Lo uso cuando tengo que cerrar un PR y me da paja escribir todo el resumen, el plan de pruebas y buscar si hay cambios disruptivos.

**Uso:**
```bash
matecommit summarize-pr --pr-number <id>
```

**El flujo es simple:**
1.  **Metadata**: Levanta los commits, comentarios y el diff directo de la API de tu VCS (GitHub, por ahora).
2.  **Síntesis**: El LLM lee toda la historia del PR y te arma un resumen cohesivo, con plan de pruebas y aviso de breaking changes.
3.  **Push**: Actualiza el título, la descripción y las labels del PR directamente en la plataforma por vos.

**Flags disponibles:**

`--pr-number` / `-n` (int)
> El número del PR que querés resumir. Obligatorio.

`--hint` / `-H` (string)
> Cualquier cosa extra que quieras que la IA tenga en cuenta, algo que no salga claro del diff solo.

### `issue` / `i`
Todo lo relacionado a crear y gestionar issues vive acá adentro. Odio tener que salir de la terminal y abrir el navegador solo para crear un ticket.

#### `issue generate` / `g`
Transforma lo que estás haciendo en un issue bien escrito, con labels inferidas automáticamente a partir de las que ya existen en tu repo.

**De dónde saca la info (elegís una):**

`--from-diff` / `-d`
> Usa tus cambios actuales en stage como base para describir el problema o la tarea.

`--from-pr` / `-p` / `--pr` (int)
> Genera el issue a partir de un Pull Request que ya existe — sirve para abrir un issue de seguimiento después del hecho.

`--description` / `-m` (string)
> Le contás en lenguaje natural qué querés y la IA lo redacta.

**Otras flags:**

`--hint` / `-h` (string)
> Contexto extra para la IA, además de la fuente que hayas elegido arriba.

`--template` / `-t` (string)
> Forzá un template de issue específico en vez de dejar que la IA infiera el tipo.

`--auto-template`
> Dejá que la IA elija sola el template que mejor encaje, si no especificaste uno.

`--no-labels`
> Se salta la inferencia de labels por completo.

`--assign-me` / `-a`
> Te asigna el issue creado a vos.

`--checkout` / `-c`
> Crea y hace checkout automático a una rama nueva con el nombre del issue, para que arranques a laburar ahí mismo.

`--dry-run`
> Preview del issue generado, sin crearlo de verdad.

#### `issue link` / `l`
Vincula un PR que ya existe con un issue que ya existe (agrega la referencia "Closes #X" que GitHub entiende).

```bash
matecommit issue link --pr <número-de-pr> --issue <número-de-issue>
```

#### `issue template` / `t`
Maneja los templates que MateCommit usa para estructurar los issues que genera.

- `issue template init` — Te tira el set de templates por defecto (bug report, feature request, tech debt, seguridad, etc.) en `.github/ISSUE_TEMPLATE/`. Pasale `--force` si querés pisar los que ya tenés.
- `issue template list` / `ls` / `l` — Te muestra qué templates hay disponibles en el repo ahora mismo.

#### `issue from-plan`
Si vos (o un asistente de IA) ya escribieron un plan de implementación en un archivo markdown, esto lo desglosa en issues individuales, en vez de hacerte copiar y pegar cada sección a mano.

```bash
matecommit issue from-plan --file PLAN.md
```

`--file` / `-f` (string)
> Ruta al archivo del plan. Obligatorio.

`--labels` / `-l` (string, repetible)
> Labels extra para agregarle a cada issue que se cree a partir del plan.

`--assign-me` / `-a`
> Te asigna a vos los issues creados.

`--dry-run` / `-d`
> Preview de lo que se crearía, sin abrir nada de verdad.

---

## 3. Automatización de Releases

### `release` / `r`
Construí esto para sacarme de encima el estrés de manejar el versionado semántico (SemVer) a mano. En realidad son seis comandos distintos, porque "crear un release" significa cosas distintas según en qué punto del proceso estés.

La mayoría (`preview`, `generate`, `create`, `publish`) esperan que estés parado en `main` o `master` — un release no debería salir de una rama de feature cualquiera. Si te tira error por esto, hacé `git checkout main` primero.

- **`release preview` / `p`** — Te muestra cómo quedaría el próximo release (salto de versión, entradas del changelog) sin crear nada. Bueno para chequear antes de comprometerte a un número de versión.
- **`release generate` / `g`** — Genera las notas del release y las guarda en un archivo (`RELEASE_NOTES.md` por defecto, cambialo con `--output` / `-o`) en vez de publicar nada.
- **`release create` / `c`** — El pipeline completo: analiza los commits desde el último tag, actualiza el `CHANGELOG.md`, bumpea el archivo de versión, crea el tag de git y (opcionalmente) publica.
  - `--auto` / `-y` — Se salta las confirmaciones.
  - `--version` / `-v` — Sobreescribí la versión que se detecta automáticamente (ej. `v1.2.3`).
  - `--publish` — Además publica el release en GitHub una vez creado.
  - `--draft` — Lo publica como borrador (solo tiene sentido junto con `--publish`).
  - `--changelog` — Actualiza el `CHANGELOG.md` y crea el commit automáticamente.
  - `--build-binaries` / `-b` — Compila y sube binarios como assets del release.
  - `--main-path` — Dónde vive tu paquete `main`, si necesita compilar binarios.
- **`release push`** — Pushea un tag que ya existe al remoto. Detecta la versión sola si no le pasás `--version` / `-v`. Si tu remoto tiene un ruleset que bloquea pushes directos, este es el comando que más chances tiene de chocar con eso (ver la nota abajo).
- **`release publish` / `pub`** — Publica un release que ya tiene tag en GitHub. Mismas flags `--version`, `--draft` (acá es `-d`), `--build-binaries` (`-b`) y `--main-path` que `create`.
- **`release edit` / `e`** — Te abre las notas de un release existente en tu editor para retocarlas a mano. Pasale `--ai` / `-a` si querés que la IA las regenere/mejore primero, y `--editor` / `-e` para forzar qué editor usa (por defecto agarra `$EDITOR`, y si no hay, cae a nano/vim).

**Sobre los Rulesets de GitHub**: si tu rama main/master (o tus nombres de tag) están protegidos por un ruleset de GitHub, un push directo va a ser rechazado — y te voy a decir exactamente por qué, con el mensaje real de GitHub incluido. Cuando el push rechazado es el del commit de changelog específicamente, intento pushearlo como una rama nueva y te abro una PR automáticamente en su lugar; solo tenés que mergearla y volver a correr el comando para terminar el release.

---

## 4. Configuración y Sistema

### `config` / `c`
Tus ajustes viven en `.matecommit/config.json` si estás dentro de un repo de git (config local), o en `~/.config/matecommit/config.json` si no (config global). La config local siempre gana sobre la global cuando existen las dos — MateCommit no las mezcla campo por campo.

- **`config show`** — Imprime la configuración resuelta (local o global, la que aplique), con la API key oculta.
- **`config init`** — El wizard de configuración. Corrélo sin flags y te pregunta rápida vs. completa; o andá directo a una con `--quick` / `-q` o `--full`. Sumale `--local` / `-l` para que quede en el repo actual en vez de tu config global, o `--global` / `-g` para forzar la global aunque estés dentro de un repo.
- **`config set <clave> <valor>`** — Seteá un valor puntual sin pasar por el wizard, ej. `matecommit config set lang es`. Claves soportadas: `lang`/`language`, `emoji`/`use_emoji`, `count`/`suggestions_count`, `active-ai`, `model`, `active-vcs`, `git.name`, `git.email`. Sumale `--local` / `-l` o `--global` / `-g` si querés ser explícito sobre qué archivo se escribe.
- **`config edit`** — Te abre el archivo de configuración directo en tu editor, para cuando el wizard es más lío del que vale.

### `doctor` / `dr`
Corre un chequeo de salud completo: conexión a internet, que git esté instalado y estés dentro de un repo, tu identidad de git (nombre/email), si tu proveedor de IA activo está bien configurado, tu token de GitHub y sus permisos, y si encuentra un editor para usar. Si algo no anda, este es siempre el primer comando que hay que correr — te dice exactamente qué falta en vez de dejarte adivinar a partir de un stack trace.

```bash
matecommit doctor
```

### `stats` / `cost`
Como las APIs de IA no son gratis, agregué un seguimiento de uso. Cada consulta queda registrada localmente con sus tokens y su costo.

- `matecommit stats` — El uso de hoy.
- `--monthly` / `-m` — El uso de este mes, desglosado por día.
- `--breakdown` / `-b` — El uso agrupado por comando (cuánto de tu gasto es `suggest` vs `summarize-pr` vs el resto).
- `--forecast` / `-f` — Una proyección de cuánto vas a gastar a fin de mes al ritmo actual.

### `cache`
Las respuestas de la IA quedan cacheadas localmente, así repetir la misma consulta (o reintentar después de un fallo) no te gasta otra llamada a la API. `matecommit cache clean` te borra ese cache si querés arrancar de cero — útil si sospechás que una respuesta vieja cacheada es la razón de que una sugerencia te salga rara.

### `completion`
Genera el script de autocompletado para tu shell.

```bash
matecommit completion bash   # imprime el script de bash
matecommit completion zsh    # imprime el script de zsh
matecommit completion install  # detecta tu shell y lo instala solo
```

`completion install` mira tu variable `$SHELL`, te agrega la línea de `source` correspondiente a tu `.bashrc` o `.zshrc`, y te avisa que reinicies la terminal (o hagas `source` del archivo vos mismo). Por ahora solo soporta bash y zsh.

### `update`
Actualiza MateCommit a la última versión. Se fija cómo lo instalaste (`go install`, Homebrew, o un binario suelto) y lo actualiza de la misma forma, para no pelearse con tu gestor de paquetes.

```bash
matecommit update
```

MateCommit también chequea si hay versiones nuevas en segundo plano y te avisa antes de la mayoría de los comandos. Si te resulta molesto, seteá `MATECOMMIT_DISABLE_UPDATE_CHECK=1` y te deja de joder.

---

## Solución de problemas comunes

**"Las sugerencias no son muy buenas"**
*   *Consejo*: Asegurate de stagear solo los cambios que tengan que ver entre sí. Si metés 5 features distintas en un mismo stage, la IA se marea con el contexto.

**"Error de API" / algo no autentica**
*   *Consejo*: Corré `matecommit doctor`. Lo más probable es que tu API key de Gemini o tu token de GitHub hayan expirado, falten, o no tengan los permisos (scopes) necesarios — `doctor` te dice cuál.

**"Me rechazó el push de la nada"**
*   *Consejo*: Si vos (o tu equipo) tienen un ruleset de GitHub protegiendo la rama o el tag, eso es esperable — MateCommit te muestra el motivo real que da GitHub en vez de un error genérico de git. Pusheá a través de una PR, o pedile a quien maneje el ruleset que lo ajuste.

---

## Soporte actual

*   **Modelos de IA**: Google Gemini (Por defecto).
*   **VCS**: GitHub.
*   **Issues**: Jira y GitHub Issues.
