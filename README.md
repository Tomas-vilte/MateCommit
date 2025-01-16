# MateCommit 🧉

![Build Status](https://github.com/Tomas-vilte/MateCommit/workflows/Go%20CI/badge.svg) ![Report Card](https://goreportcard.com/badge/github.com/Tomas-vilte/MateCommit) [![codecov](https://codecov.io/gh/Tomas-vilte/MateCommit/branch/master/graph/badge.svg?token=6O798E12DC)](https://codecov.io/gh/Tomas-vilte/MateCommit)

¡Bienvenidos a **MateCommit**! Un proyecto que busca hacer tu flujo de trabajo con Git más simple, todo acompañado con un buen mate.

## Índice

1. [¿Qué es esto?](#que-es-esto)
   - [Características](#características)
2. [Guía de Uso](#guía-de-uso)
   - [Idiomas Soportados](#idiomas-soportados)
   - [Comandos Principales](#comandos-principales)
     - [Sugerir Commits](#1-sugerir-commits)
     - [Configuración](#2-configuración)
   - [Flujo de Trabajo](#flujo-de-trabajo)
   - [Ejemplo de Uso Interactivo](#ejemplo-de-uso-interactivo)
3. [Instalación](#instalación)
   - [Usando el binario](#usando-el-binario)
   - [Desde el código fuente](#desde-el-código-fuente)
4. [Modelos de IA Soportados](#modelos-de-ia-soportados)
   - [Actual](#actual)
   - [Próximamente](#próximamente)
5. [Licencia](#licencia)
6. [¿Cómo contribuir?](#como-contribuir)

## ¿Qué es esto?

¿Te da paja pensar en el nombre de tu commit? Bueno, **MateCommit** viene a darte una mano. Este proyecto te sugiere títulos para tus commits de manera inteligente, mientras te tomás unos buenos mates. 

Con **MateCommit**, gestionar tus commits es tan simple como preparar un mate.

### Características 
- 🧉 **Sugerencias inteligentes**: Te ayudamos a elegir los mejores nombres para tus commits
- 💻 **Compatible con GitHub**: Se integra perfectamente con tu flujo de trabajo
- 🤖 **Potenciado por IA**: Actualmente usa Gemini, con planes de soportar más modelos en el futuro
- 🌎 **Bilingüe**: Soporta español e inglés
- ⚽ **Fácil de usar**: Simple y efectivo

## Guía de Uso

### Idiomas Soportados
- Español (es)
- Inglés (en)

### Comandos Principales

#### 1. Sugerir Commits
```bash
matecommit suggest [opciones]
# o usando el alias
matecommit s [opciones]
```

Opciones:
- `-n, --count <número>`: Cantidad de sugerencias (1-10, default: 3)
- `-l, --lang <idioma>`: Idioma (es/en)
- `-ne, --no-emoji`: Deshabilita emojis
- `-ml, --max-length <número>`: Longitud máxima del mensaje (default: 72)

#### 2. Configuración
```bash
# Ver configuración
matecommit config show

# Cambiar idioma
matecommit config set-lang --lang <es|en>

# Configurar API Key
matecommit config set-api-key --key <tu-api-key>
```

### Flujo de Trabajo
1. Hacer cambios en tus archivos
2. `git add` de los cambios
3. `matecommit suggest`
4. Elegir una sugerencia
5. ¡Listo! El commit se crea automáticamente

### Ejemplo de Uso Interactivo

Cuando ejecutes `matecommit suggest`, verás una interfaz interactiva como esta:

```
🔍 Analyzing changes...
📝 Suggestions:
━━━━━━━━━━━━━━━━━━━━━━━
Commit: docs: add CONTRIBUTING.md
📄 Modified files:
   - CONTRIBUTING.md
Explanation: Added a CONTRIBUTING.md file to guide contributors.
━━━━━━━━━━━━━━━━━━━━━━━
Commit: docs: add README.md
📄 Modified files:
   - README.md
Explanation: Added a README.md file explaining the project.
━━━━━━━━━━━━━━━━━━━━━━━
Commit: chore: update main.go
📄 Modified files:
   - cmd/main.go
Explanation: Minor updates to the main application file.
━━━━━━━━━━━━━━━━━━━━━━━
Select an option:
1-N: Use the corresponding suggestion
0: Exit without committing
👉 Enter your selection:
```

Para cada sugerencia, verás:
- 📝 Un título de commit sugerido
- 📄 La lista de archivos modificados
- Una breve explicación de los cambios

Simplemente ingresa el número de la sugerencia que prefieras (1, 2, 3...) y MateCommit:
1. Creará el commit automáticamente con el mensaje seleccionado
2. Te confirmará que el commit se realizó exitosamente
3. ¡Solo te quedará hacer el push cuando estés listo!

## Instalación

### Usando el binario

Para arrancar rápido, podés descargar el binario desde la [página de releases](https://github.com/Tomas-vilte/MateCommit/releases).

Seguí estos pasos:

1. **Descargá el binario** para tu sistema:
   - Linux: `matecommit-linux-amd64`
   - Windows: `matecommit-windows-amd64.exe`
   - Mac: `matecommit-darwin-amd64`

2. **Dale permisos** (Linux/Mac):
   ```bash
   chmod +x matecommit-linux-amd64
   ```

3. **Movelo al PATH**:
   ```bash
   sudo mv matecommit-linux-amd64 /usr/local/bin/matecommit
   ```

4. **Configuración inicial**:
   ```bash
   # Configura tu API key de Gemini
   matecommit config set-api-key --key <tu-api-key>
   
   # Establece tu idioma preferido
   matecommit config set-lang --lang es  # o en para inglés
   ```

### Desde el código fuente

Si preferís compilar el código:

1. **Prerequisitos**:
   - Go instalado
   - Git instalado

2. **Cloná el repositorio**:
   ```bash
   git clone https://github.com/Tomas-vilte/MateCommit.git
   ```

3. **Instalá las dependencias**:
   ```bash
   cd MateCommit
   go mod tidy
   ```

4. **Compilá**:
   ```bash
   go build -o matecommit ./cmd/main.go
   ```

## Modelos de IA Soportados

### Actual
- 🤖 **Gemini**: Modelo principal actual

### Próximamente
- 🔄 **GPT-4**: Integración planificada
- 🔄 **Claude**: Integración planificada

## Licencia

MateCommit está bajo licencia MIT. Podés ver los detalles en el archivo [LICENSE](./LICENSE).

## ¿Cómo contribuir?

¡Nos encanta recibir ayuda! Si querés contribuir:

1. **Leé el CONTRIBUTING.md**: 
   - Dale una mirada al [CONTRIBUTING.md](CONTRIBUTING.md) antes de empezar.

2. **Hacé un fork**: 
   - Creá tu propia copia del proyecto para trabajar.

3. **Cloná tu fork**: 
   ```bash
   git clone https://github.com/tu-usuario/MateCommit.git
   cd MateCommit
   ```

4. **Creá una rama**:
   ```bash
   git checkout -b nombre-de-tu-rama
   ```

5. **Hacé tus cambios**:
   - Implementá las modificaciones necesarias
   - Mantené el estilo del código existente
   - Si agregás soporte para un nuevo modelo de IA, seguí el patrón existente

6. **Tests y validación**:
   - Asegurate de que todo funcione correctamente
   - Agregá tests para las nuevas funcionalidades

7. **Commit y Push**:
   ```bash
   git add .
   git commit -m "Descripción de los cambios"
   git push origin nombre-de-tu-rama
   ```

8. **Crea el Pull Request**:
   - Explicá bien tus cambios y qué mejoran
   - Mencioná si agregás soporte para nuevos modelos


¡Gracias por sumarte al proyecto! 🙌 