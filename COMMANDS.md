# Guía de la CLI de MateCommit 🧉

Bienvenido a la guía de MateCommit. Acá vas a encontrar todo lo que necesitás saber para usar la CLI .

## Índice
- [Empezando](#empezando)
- [Comandos Principales](#comandos-principales)
  - [Sugerencias de Commits](#sugerencias-de-commits)
  - [Configuración Básica](#configuración-básica)
  - [Configuración de IA](#configuración-de-ia)
  - [Integración con Jira](#integración-con-jira)
  - [Configuración de VCS](#configuración-de-vcs)
  - [Idiomas](#idiomas)
- [Ejemplos con Salidas](#ejemplos-con-salidas)
- [Tips y Trucos](#tips-y-trucos)

## Empezando

MateCommit es simple de usar. La idea es que te ayude a hacer commits más copados sin tener que pensar mucho en los mensajes.

### Instalación Básica

```bash
# Configuración interactiva completa (recomendado)
matecommit config init

# O si solo querés ver la configuración actual
matecommit config show
```

## Comandos Principales

### Sugerencias de Commits

El comando más importante es `suggest` (o `s` para hacerla corta):

```bash
# Generar 3 sugerencias (default)
matecommit suggest

# Generar 5 sugerencias
matecommit s -n 5

# Sugerencias en inglés
matecommit s -l en

# Sin emojis
matecommit s --no-emoji
```

### Configuración Básica

#### Configuración interactiva completa
```bash
matecommit config init
```

Este comando te guía paso a paso para configurar:
- 🌍 **Idioma**: Español o inglés
- 🤖 **IA**: API key de Gemini y modelo
- 🔧 **VCS**: Token de GitHub para resúmenes de PR
- 🎫 **Tickets**: Integración con Jira (opcional)

#### Ver configuración actual
```bash
matecommit config show
```

Te va a mostrar algo así:
```
📋 Configuración actual
━━━━━━━━━━━━━━━━━━━━━━━
🌍 Idioma: es
😊 Emojis: true
🔑 Clave API: ✅ Configurada
IA Activa: gemini
Modelos de IA configurados:
- gemini: gemini-1.5-pro
```

#### Editar configuración manualmente
```bash
matecommit config edit
```

Abre el archivo de configuración en tu editor preferido para editarlo manualmente.

### Configuración de IA

La configuración de IA se hace a través del comando `config init`:

```bash
# Configuración interactiva que incluye IA
matecommit config init
```

Durante el proceso te va a preguntar:
- 🤖 **API Key de Gemini**: Tu clave para usar Gemini
- 🧠 **Modelo**: Qué modelo usar (gemini-1.5-flash, gemini-1.5-pro, etc.)

**Nota**: Actualmente solo soporta Gemini, pero próximamente vamos a agregar OpenAI y Claude.

### Integración con Jira

La configuración de Jira también se hace con `config init`:

```bash
# Configuración interactiva que incluye Jira
matecommit config init
```

Durante el proceso te va a preguntar si querés habilitar Jira y te pedirá:
- 🌐 **Base URL**: La URL de tu instancia de Jira
- 📧 **Email**: Tu email de Jira
- 🔑 **API Token**: Tu token de API de Jira

### Configuración de VCS

La configuración de VCS se hace con `config init`:

```bash
# Configuración interactiva que incluye VCS
matecommit config init
```

Durante el proceso te va a preguntar si querés habilitar VCS y te pedirá:
- 🔑 **Token de GitHub**: Tu Personal Access Token (recomendamos classic tokens)

**Importante para repositorios de organizaciones**: 
- Usá **Personal access tokens (classic)** en lugar de fine-grained tokens
- Los classic tokens funcionan mejor con organizaciones sin necesidad de aprobación

Una vez configurado, podés usar:
```bash
# Resumir un Pull Request
matecommit summarize-pr --pr-number 42
matecommit spr -n 42  # alias corto
```

### Idiomas

```bash
# Configurar idioma default (se hace en config init)
matecommit config init  # te pregunta el idioma

# O usar otro idioma solo para una sugerencia
matecommit s -l en  # sugerencia en inglés
matecommit s -l es  # sugerencia en español
```

## Ejemplos con Salidas

### Ejemplo 1: Flujo básico
```bash
# Agregar cambios
git add .

# Pedir sugerencias
matecommit s

🔍 Analizando cambios...
📝 Sugerencias:
━━━━━━━━━━━━━━━━━━━━━━━
feat: agrega soporte para múltiples idiomas ✨
📄 Archivos modificados:
   - translations/es.json
   - translations/en.json
💡 Agregué archivos de traducción para español e inglés

👉 Ingresá tu selección: 1

✅ Commit creado con éxito
```

### Ejemplo 2: Sugerencia con análisis técnico
```bash
matecommit s

📊 Análisis de Código:
- Resumen de Cambios: Actualización de configuración
- Propósito Principal: Agregar soporte para nuevos modelos de IA
- Impacto Técnico: Mejora la flexibilidad del sistema

💡 feat: agrega soporte para modelos de IA adicionales

📄 Archivos modificados:
   - config/ai_models.go
   - config/providers.go
```

### Ejemplo 3: Integración con Jira
```bash
matecommit s

🎯 Análisis de Requerimientos:
⚠️ Estado de los Criterios: Parcialmente cumplidos
❌ Criterios Faltantes:
   - Falta documentación de API
   - Pendiente actualizar tests

💡 feat(PROJ-123): implementa nuevo endpoint de usuarios
```

### Ejemplo 4: Resumen de PR con VCS
```bash
matecommit spr -n 42
✅ PR #42 actualizado: Implementación de repository
```

## Tips y Trucos

1. **Alias Rápidos**:
   - Usá `s` en lugar de `suggest`
   - Usá `spr` en lugar de `summarize-pr`
   - `config show` te muestra todo de una

2. **Mejores Prácticas**:
   - Siempre hacé `git add` antes de usar MateCommit
   - Si no te convence ninguna sugerencia, apretá 0 y pedí más
   - Usá classic tokens para GitHub en lugar de fine-grained tokens

3. **Personalización**:
   - Probá diferentes modelos de Gemini hasta encontrar el que mejor te funcione
   - Los emojis son opcionales pero le dan más onda 😎
   - Podés tener un idioma default y usar otro para commits específicos

4. **Integración con Jira**:
   - Activala solo si trabajás con tickets
   - Te agrega automáticamente el número de ticket en los commits

5. **Configuración**:
   - Usá `config init` para configurar todo de una vez
   - Si algo no te gusta, podés editarlo con `config edit`
   - Siempre podés volver a ejecutar `config init` para cambiar algo

6. **Repositorios de Organización**:
   - Para repos de organizaciones, usá Personal Access Tokens (classic)
   - Los fine-grained tokens requieren aprobación de la organización

¿Necesitás más ayuda? Siempre podés usar:
```bash
matecommit --help
# o para un comando específico
matecommit config --help
matecommit suggest --help
matecommit summarize-pr --help
```