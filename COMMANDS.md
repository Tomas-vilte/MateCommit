***

# Guía de la CLI de MateCommit

Hola, te dejo esta guía para que le saques el jugo a la CLI de MateCommit. Acá vas a encontrar todo lo necesario para empezar a usarla y configurarla a tu gusto.

## Índice
- [Empezando](#empezando)
- [Comandos Principales](#comandos-principales)
   - [Sugerencias de Commits](#sugerencias-de-commits)
   - [Configuración](#configuracion)
   - [Idiomas](#idiomas)
   - [Gestión de Releases](#gestion-de-releases)
- [Ejemplos de uso](#ejemplos-de-uso)
- [Tips y consejos](#tips-y-consejos)

## Empezando

MateCommit es bastante simple. La idea es que te ayude a armar commits prolijos sin que tengas que dar muchas vueltas pensando el mensaje.

### Instalación Básica

Para arrancar, lo mejor es correr la configuración interactiva:

```bash
# Configuración interactiva completa (recomendado)
matecommit config init

# Si querés ver qué tenés configurado actualmente
matecommit config show
```

### Autocompletado (Shell Completion)

Para activar el autocompletado en tu terminal (Bash o Zsh) y que la herramienta te sugiera comandos y flags, ejecutá:

```bash
matecommit completion install
```

Esto va a detectar tu shell y agregar la configuración necesaria en tu `.bashrc` o `.zshrc`. 
Una vez hecho esto, **reiniciá tu terminal** o ejecutá `source ~/.bashrc` (o el archivo que corresponda).

Ahora probá escribir `matecommit serv` y apretá `TAB`. ¡Magia! ✨
También funciona para flags: `matecommit suggest --<TAB>`.

## Comandos Principales

### Sugerencias de Commits

El comando que más vas a usar es `suggest` (o `s` si no querés escribir tanto). Básicamente analiza tus cambios y te tira opciones.

```bash
# Generar 3 sugerencias (el default)
matecommit suggest

# Si querés más variedad (ej: 5 opciones)
matecommit s -n 5

# Si necesitás los mensajes en inglés
matecommit s -l en

# Si preferís un output limpio sin emojis en el commit
matecommit s --no-emoji
```

### Configuración

Tenés varias formas de configurar la herramienta, dependiendo de qué tanto quieras personalizar.

#### Setup Rápido (Para arrancar ya)

Si estás apurado y solo querés que ande, usá el flag `--quick`. Solo te va a pedir la API Key de Gemini.

```bash
matecommit config init --quick
# o más corto
matecommit config init -q
```

**Lo que hace este comando:**
- Te pide la API key de Gemini.
- Configura el modelo recomendado (gemini-2.5-flash) por defecto.
- Deja todo listo para usar en menos de un minuto.

#### Setup Completo (Para tener el control total)

Si preferís revisar cada detalle, mandale el init completo. Es un wizard interactivo.

```bash
matecommit config init --full
# o simplemente
matecommit config init
```

**Acá vas a poder configurar:**

1. **IA (Gemini):** API key y qué modelo específico querés usar (flash, pro, etc.).
2. **Idioma:** Si querés que te hable en español o inglés por defecto.
3. **VCS (GitHub):** Token de GitHub. Esto es clave si querés usar las funciones de resumen de PRs o releases.
4. **Tickets (Jira):** Si usás Jira, podés configurar la URL y credenciales para que te linkee los tickets solo.
5. **Releases:** Opciones como actualizar automáticamente el `CHANGELOG.md` (`update_changelog`).

#### Ver configuración actual

Para chequear cómo quedó todo configurado:

```bash
matecommit config show
```

#### Editar configuración manualmente

Si sos de los que prefieren tocar los archivos directamente:

```bash
matecommit config edit
```

Esto te abre el archivo `config.yaml` en tu editor predeterminado.

#### Diagnóstico (Doctor)

Si algo no te anda, tirá este comando para ver qué pasa:

```bash
matecommit doctor
```

Verifica que tengas conexión, que las keys sean válidas, que git esté instalado, etc.

#### Cómo obtener las API Keys

**Gemini (Requerido):**
1. Entrá a Google AI Studio.
2. Logueate y generá una API Key nueva.
3. Copiala (empieza con `AIza...`).

**GitHub (Opcional, pero recomendado):**
1. Andá a Settings > Developer settings > Personal access tokens > Tokens (classic).
2. Generá un token nuevo con scope `repo` y `read:org`.
3. Usá ese token (empieza con `ghp_...`).

**Importante:** Usá los "Classic tokens", suelen dar menos problemas de permisos que los fine-grained para este tipo de herramientas.

### Idiomas

Podés configurar el idioma base en el `config init`, pero si justo necesitás un commit en otro idioma, podés forzarlo con el flag `-l`:

```bash
matecommit s -l en  # Genera sugerencias en inglés
matecommit s -l es  # Genera sugerencias en español
```

## Gestión de Releases

MateCommit trae un gestor de releases integrado. Automatiza el versionado, genera el changelog con IA y publica en GitHub.

### Comandos de Release

El comando base es `release` (o `r`).

#### Vista previa (Preview)
Antes de romper nada, fijate qué cambios entrarían en el release:

```bash
matecommit release preview
# o el alias
matecommit r p
```
Te muestra la versión actual, la siguiente sugerida y un borrador de las notas.

#### Generar notas
Si solo querés el changelog en un archivo:

```bash
matecommit release generate
# Guardar en un archivo específico
matecommit r g -o CHANGELOG.md
```

#### Crear release (Tag local)
Esto crea el tag de git en tu máquina.

```bash
# Con confirmación
matecommit release create

# Directo sin preguntar (útil para scripts)
matecommit r c --auto

# Crear y subir a GitHub de una
matecommit r c --publish

# Actualizar CHANGELOG.md localmente y crear release
matecommit r c --changelog
```

**Nota sobre `--changelog`:**
Este flag genera el contenido del changelog, actualiza tu archivo `CHANGELOG.md` local (haciendo prepend), y realiza automáticamente un `git add CHANGELOG.md` y un `git commit` antes de crear el tag. Esto asegura que el changelog actualizado sea parte de la versión liberada.

Podés configurar esto para que sea el comportamiento por defecto editando tu config (`matecommit config edit`) y seteando `update_changelog: true`.

#### Publicar en GitHub
Si ya tenés el tag local y querés armar el release en GitHub:

```bash
matecommit release publish
# Publicar como draft (para revisar antes de hacer público)
matecommit r pub --draft
```

#### Editar release existente
Si el release ya está creado pero querés mejorar las notas:

```bash
matecommit release edit -v v1.2.3
# Regenerar las notas con IA
matecommit r e -v v1.2.3 --ai
```

### Flujo de trabajo sugerido

1. **Revisión:** Ejecutá `matecommit r p` para ver qué se viene.
2. **Creación:** Si está todo ok, mandale `matecommit r c --publish`.
3. **Drafts:** Si no estás seguro, usá el flag `--draft` al publicar para revisarlo en la web de GitHub antes de soltarlo.

## Ejemplos de uso

### Flujo básico
```bash
# 1. Pedís sugerencias
matecommit s

# Output:
# Analizando cambios...
# Sugerencias:
# 1. feat: agrega soporte para múltiples idiomas
#    Archivos: translations/es.json, translations/en.json
#    Explicación: Agregué archivos de traducción base.
#
# Seleccioná una opción: 1
# Commit creado con éxito.
```

### Integración con Jira
Si tenés Jira configurado, la herramienta detecta el contexto y formatea el commit acorde:

```bash
matecommit s

# Output:
# feat(PROJ-123): implementa nuevo endpoint de usuarios
```

### Resumen de PR
Si tenés el token de GitHub, podés pedir un resumen de un Pull Request:

```bash
matecommit spr -n 42
# Output:
# PR #42 actualizado: Implementación de repository pattern
```

## Tips y consejos

1. **Usá los alias:** No escribas `matecommit suggest` cada vez. Usá `matecommit s`. Lo mismo para `r` (release) o `spr` (resumen de PR).
2. **Feedback:** Si ninguna sugerencia te convence, seleccioná la opción 0 para generar nuevas o editar una manualmente.
3. **Repos de Organización:** Si laburás en una organización de GitHub, acordate de usar el Personal Access Token (Classic) y autorizarlo en la organización (SSO), si no te va a tirar error de permisos.
4. **Release rápido:** Si confiás en la herramienta, `matecommit r c --publish --auto` hace todo el trabajo sucio (tag, notas, push y release) en un solo paso.