# MateCommit 🧉

![Build Status](https://github.com/Tomas-vilte/MateCommit/workflows/Go%20CI/badge.svg) ![Report Card](https://goreportcard.com/badge/github.com/Tomas-vilte/MateCommit) [![codecov](https://codecov.io/gh/Tomas-vilte/MateCommit/branch/master/graph/badge.svg?token=6O798E12DC)](https://codecov.io/gh/Tomas-vilte/MateCommit)

¡Bienvenidos a **MateCommit**! Un proyecto que busca hacer tu flujo de trabajo con Git más simple, todo acompañado con un buen mate.

## ¿Qué es esto?

¿Te da paja pensar en el nombre de tu commit? Bueno, **MateCommit** viene a darte una mano. Este proyecto te sugiere títulos para tus commits de manera inteligente, mientras te tomás unos buenos mates. 

### Características 
- 🧉 **Sugerencias inteligentes**: Te ayudamos a elegir los mejores nombres para tus commits
- 💻 **Compatible con GitHub**: Se integra perfectamente con tu flujo de trabajo
- 🤖 **Potenciado por IA**: Actualmente usa Gemini, y soporta varios modelos de Gemini.
- 🌎 **Bilingüe**: Soporta español e inglés
- ⚽ **Fácil de usar**: Simple y efectivo
- 🚀 **Resumenes de Pull Requests**: Ahora podes crear resumenes de Pull Requests, en base a los cambios que hiciste

## Instalación

### Usando el binario

1. **Descargá el binario** desde la [página de releases](https://github.com/Tomas-vilte/MateCommit/releases) para tu sistema:
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
   # Configuración interactiva completa
   matecommit config init
   
   # O si solo querés ver la configuración actual
   matecommit config show
   ```

### Desde el código fuente

1. **Cloná el repositorio**:
   ```bash
   git clone https://github.com/Tomas-vilte/MateCommit.git
   ```

2. **Instalá las dependencias**:
   ```bash
   cd MateCommit
   go mod tidy
   ```

3. **Compilá**:
   ```bash
   go build -o matecommit ./cmd/main.go
   ```

## Documentación de Comandos

Para una guía detallada de todos los comandos disponibles, opciones y ejemplos de uso, consultá el archivo [COMMANDS.md](COMMANDS.md). Ahí encontrarás:

- Configuración completa
- Comandos principales
- Ejemplos con salidas
- Integración con Jira
- Tips y trucos

## Modelos de IA Soportados

### Actual
- 🤖 **Gemini**: 
   - Gemini-1.5-flash
   - Gemini-1.5-pro
   - Gemini-2.0-flash

### Próximamente
- 🔄 **GPT-4**: Integración planificada
- 🔄 **Claude**: Integración planificada

## Licencia

MateCommit está bajo licencia MIT. Podés ver los detalles en el archivo [LICENSE](./LICENSE).

## Contribuciones

¿Querés contribuir? ¡Genial! Consultá nuestra [guía de contribución](CONTRIBUTING.md) para empezar.