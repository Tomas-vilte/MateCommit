# Mejoras de UX: Feedback visual, colores y validación de commits

Falta un poco de feedback visual. Actualmente, cuando tiramos el comando para generar sugerencias, la terminal se queda "congelada" unos 5 o 10 segundos sin decir nada y no sabés si está procesando o si se colgó el proceso. Además, los commits salen directo sin mucha validación previa.

La idea es mejorar la experiencia de uso agregando indicadores de carga (spinners), un manejo de colores más claro para los logs y una instancia de revisión antes de confirmar los cambios.

## Propuesta de solución

El objetivo es implementar un paquete nuevo `internal/ui` para centralizar toda la lógica de presentación. Estuve viendo de usar `briandowns/spinner` para las esperas y `fatih/color` para los textos.

Los puntos principales a atacar son:

### 1. Feedback en operaciones asíncronas
Necesitamos spinners para que el usuario sepa qué está pasando, específicamente en:
*   La generación de sugerencias (llamadas a la API de IA).
*   Cuando se incluye el contexto de un issue.
*   Durante el staging de archivos y la creación del commit.

### 2. Sistema de colores y logs
El output monocromático hace difícil escanear rápido la terminal. La idea es estandarizar helpers:
*   Verde para operaciones exitosas.
*   Rojo para errores.
*   Amarillo para warnings.
*   Cyan para información general.

### 3. Preview del diff
Esto es clave para evitar errores. Antes de confirmar el commit, el CLI debería preguntar si queremos ver los cambios. Si ponemos que sí, ejecutamos un `git diff --color` para revisar qué estamos subiendo.

### 4. Errores más amigables
En lugar de tirar el error crudo, estaría bueno detectar casos comunes (falta la API Key, token de GitHub vencido, no hay cambios en staging) y mostrar una "Sugerencia de solución" clara para que el usuario sepa cómo arreglarlo rápido.

## Cambios esperados en el código

Básicamente habría que tocar estos archivos:

*   `go.mod`: Agregar las dependencias nuevas.
*   `internal/ui/ui.go`: Crear el paquete con los helpers de UI.
*   `internal/cli/command/suggests_commits/suggests_commits.go`: Implementar los spinners.
*   `internal/cli/command/handler/suggestions.go`: Meter la lógica de colores y el preview.
*   `internal/cli/command/pull_requests/summarize.go`: Agregar feedback visual también acá.
*   `internal/i18n/locales/*.toml`: **IMPORTANTE**. Todos los textos nuevos (mensajes de spinners, prompts, errores) deben estar internacionalizados. No hardcodear strings en inglés o español en el código. Usar `i18n.GetMessage` para todo.

## Ejemplo del flujo deseado

La interacción en la terminal debería quedar más o menos así:

```bash
━━━━━━━━━━━━━━━━━━━━━━━
🚀 Generando Sugerencias de Commit
━━━━━━━━━━━━━━━━━━━━━━━

ℹ Detectado issue #42
⠙ Generando sugerencias con IA...
✓ 3 sugerencias generadas (2.3s)

Selecciona una opción: 1

ℹ Commit seleccionado: feat: optimize AI prompts (#42)
ℹ Archivos: 15

¿Ver cambios antes de commitear? (y/n): y

[muestra diff con colores]

¿Confirmar commit? (y/n): y

⠙ Creando commit...
✓ Commit creado exitosamente
```

## Criterios de Aceptación

*   [x] Los spinners funcionan correctamente en todas las llamadas asíncronas.
*   [x] Los colores son consistentes en todos los comandos.
*   [x] Se puede ver el diff antes de confirmar el commit.
*   [x] Los errores comunes muestran una sugerencia de solución.
*   [x] El output es legible tanto en terminales claras como oscuras.