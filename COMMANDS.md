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
# Configurar el idioma a español
matecommit config set-lang --lang es

# Configurar tu API key de Gemini
matecommit config set-api-key --key tu-api-key
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

#### Ver toda la configuración
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

#### Configurar API Key
```bash
# Configurar la API key de Gemini
matecommit config set-api-key --key tu-api-key
```

### Configuración de IA

Ahora podés elegir entre diferentes IAs y modelos:

```bash
# Ver las IAs disponibles
matecommit config set-ai-active

# Activar Gemini
matecommit config set-ai-active gemini

# Configurar el modelo de Gemini
matecommit config set-ai-model gemini gemini-1.5-pro

# O si preferís OpenAI
matecommit config set-ai-active openai
matecommit config set-ai-model openai gpt-4
```

### Integración con Jira

Si laburás con Jira, tenés estas opciones:

```bash
# Configurar las credenciales
matecommit config jira \
  --base-url https://tu-empresa.atlassian.net \
  --api-key tu-api-key \
  --email tu@email.com

# Activar la integración
matecommit config ticket enable

# Desactivar la integración
matecommit config ticket disable
```

### Idiomas

```bash
# Cambiar el idioma default
matecommit config set-lang --lang es  # español
matecommit config set-lang --lang en  # inglés

# O usar otro idioma solo para una sugerencia
matecommit s -l en  # sugerencia en inglés
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

### Configuración de VCS

Configura proveedores de control de versiones (GitHub, GitLab, etc.):
```bash
+# Configurar un proveedor VCS (ej: GitHub)
+matecommit config set-vcs \
  --provider github \
  --token tu-token \
  --owner tu-usuario \
  --repo tu-repositorio
  
# Establecer el proveedor VCS activo
matecommit config set-active-vcs --provider github
# Resumir un Pull Request (requiere VCS configurado)
matecommit summarize-pr --pr-number 42
matecommit spr -n 42  # alias corto
```

### Ejemplo 4: Resumen de PR con VCS
```bash
matecommit spr -n 42
✅ PR #42 actulizado: Implementacion de repository
```

## Tips y Trucos

1. **Alias Rápidos**:
   - Usá `s` en lugar de `suggest`
   - `config show` te muestra todo de una

2. **Mejores Prácticas**:
   - Siempre hacé `git add` antes de usar MateCommit
   - Si no te convence ninguna sugerencia, apretá 0 y pedí más

3. **Personalización**:
   - Probá diferentes IAs hasta encontrar la que mejor te funcione
   - Podés tener diferentes modelos configurados para cada IA
   - Los emojis son opcionales pero le dan más onda 😎

4. **Integración con Jira**:
   - Activala solo si trabajás con tickets
   - Te agrega automáticamente el número de ticket en los commits

5. **Idiomas**:
   - Podés tener un idioma default y usar otro para commits específicos
   - El análisis técnico se adapta al idioma elegido

¿Necesitás más ayuda? Siempre podés usar:
```bash
matecommit --help
# o para un comando específico
matecommit config --help
```