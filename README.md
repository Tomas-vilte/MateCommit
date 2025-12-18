***
# MateCommit

![Build Status](https://github.com/Tomas-vilte/MateCommit/workflows/Go%20CI/badge.svg) ![Report Card](https://goreportcard.com/badge/github.com/Tomas-vilte/MateCommit) [![codecov](https://codecov.io/gh/Tomas-vilte/MateCommit/branch/master/graph/badge.svg?token=6O798E12DC)](https://codecov.io/gh/Tomas-vilte/MateCommit)

Bienvenido a **MateCommit**. Este proyecto nació para simplificar el flujo de trabajo con Git y sacar un poco de la fricción del día a día.

## ¿De qué se trata?

Básicamente, si te da fiaca pensar nombres descriptivos para los commits o sentís que perdés tiempo en eso, esta herramienta te da una mano. Analiza los cambios que tenés en staging y te sugiere títulos coherentes usando IA, para que vos te ocupes del código (y del mate).

### Características principales
- **Sugerencias inteligentes**: Analiza el diff y te propone mensajes de commit con sentido.
- **Integración con GitHub**: Se lleva bien con tu flujo de trabajo actual.
- **Motor de IA**: Funciona con Gemini y soporta varios de sus modelos (Flash, Pro, etc.).
- **Idiomas**: Podés pedirle las sugerencias tanto en español como en inglés.
- **Resumen de Pull Requests**: Te arma una descripción del PR basándose en todos los commits y cambios que hiciste.
- **Gestión de Releases**: Automatiza el versionado y la generación del changelog.

## Instalación

Tenés dos formas de instalarlo, elegí la que te quede más cómoda.

### Opción 1: Usando el binario (Recomendado)

1. **Descargá el ejecutable** desde la sección de releases para tu sistema operativo (Linux, Windows o Mac).

2. **Dale permisos de ejecución** (si estás en Linux o Mac):
   ```bash
   chmod +x matecommit-linux-amd64
   ```

3. **Movelo a tu PATH** para poder ejecutarlo desde cualquier lado:
   ```bash
   sudo mv matecommit-linux-amd64 /usr/local/bin/matecommit
   ```

4. **Configuración inicial**:
   Corré el comando de inicialización para dejar todo listo (API keys, preferencias, etc.):
   ```bash
   matecommit config init
   ```

### Opción 2: Desde el código fuente

Si preferís compilarlo vos mismo:

1. **Cloná el repositorio**:
   ```bash
   git clone https://github.com/Tomas-vilte/MateCommit.git
   ```

2. **Bajá las dependencias**:
   ```bash
   cd MateCommit
   go mod tidy
   ```

3. **Compilá el binario**:
   ```bash
   go build -o matecommit ./cmd/main.go
   ```

## Documentación

Para no hacer este README eterno, separé la guía detallada de uso en otro archivo. En [COMMANDS.md](COMMANDS.md) vas a encontrar:

- Cómo hacer la configuración completa paso a paso.
- Explicación de todos los comandos (`suggest`, `release`, etc.).
- Cómo integrar la herramienta con Jira.
- Ejemplos de output y algunos trucos.

## Modelos de IA

Actualmente la herramienta funciona con **Gemini** (Google). Probamos y soporta bien las versiones `1.5-flash`, `1.5-pro` y `2.0-flash`.

Tengo en el roadmap integrar **GPT-4** y **Claude** más adelante para dar más opciones.

## Licencia

El código es abierto bajo licencia MIT. Fijate el archivo [LICENSE](./LICENSE) para más detalles.

## Contribuciones

Si querés sumar algo, arreglar un bug o mejorar la documentación, sos más que bienvenido. Pegale una mirada a la guía de contribución en `CONTRIBUTING.md` para ver cómo nos manejamos.