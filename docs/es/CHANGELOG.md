# Changelog

All notable changes to this project will be documented in this file.

## [v1.4.0]

[v1.4.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.3.0...v1.4.0

En esta versión nos enfocamos en transformar tu interacción con la herramienta, mejorando drásticamente la experiencia de usuario con feedback visual en tiempo real y optimizando la automatización de tus procesos de release. Además, potenciamos la inteligencia artificial para un análisis más profundo y contextualizado.

### Highlights

- **Experiencia de Usuario Renovada:** Implementamos spinners, colores y una previsualización de cambios (diff) para un feedback visual en tiempo real. Además, agregamos un comando `doctor` para validar tu clave de API de Gemini y mejoramos la previsualización de commits, permitiéndote editar el mensaje antes de confirmar.
- **Automatización Integral de Releases:** Simplificamos y automatizamos la generación de notas de release para `CHANGELOG.md`, la actualización de la versión de la aplicación y el commit automático del changelog. Ahora también podés editar releases existentes, agilizando tu flujo de trabajo.
- **Inteligencia Artificial Contextualizada:** Mejoramos la capacidad de la IA para entender el contexto de tus commits y solicitudes de incorporación (PRs). Ahora detecta automáticamente issues, breaking changes y planes de prueba, enriqueciendo los resúmenes de PR y las notas de release con información más relevante.
- **Análisis de Dependencias Multi-lenguaje:** Agregamos la capacidad de analizar cambios en las dependencias de tus proyectos, incluso en entornos multi-lenguaje, brindando una visión más completa y detallada de cada release.
- **Mejoras en la Interfaz de Línea de Comandos (CLI):** Implementamos autocompletado para comandos y flags, haciendo tu experiencia en la terminal mucho más fluida y eficiente.
- **Correcciones y Estabilidad:** Aseguramos que el resumidor de PRs utilice correctamente la plantilla para las instrucciones de formato JSON, garantizando la consistencia en la generación de resúmenes.

## [v1.3.0] - 2025-12-09

[v1.3.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.2.0...v1.3.0

En esta versión, nos enfocamos en simplificar y automatizar aún más sus flujos de trabajo, desde la gestión de releases hasta la interacción con Pull Requests. También mejoramos la experiencia de configuración y la estabilidad general de la aplicación.

### Highlights

- Simplificamos la gestión de releases y la publicación: Ahora pueden generar y publicar nuevas versiones de forma más fluida con comandos dedicados y un asistente de prompts que utiliza un tono natural y en primera persona, haciendo el proceso más intuitivo y automatizado.
- Renovamos la configuración y la asistencia en la CLI: Introdujimos el comando `config init` para guiarlos en la configuración inicial, un comando `edit` para ajustar fácilmente los parámetros, y un comando `help` para que siempre tengan la información a mano. También optimizamos la guía para la configuración de VCS y agregamos una etiqueta de 'performance' para la IA.
- Potenciamos la interacción con Pull Requests: Ahora detectamos automáticamente la información del repositorio para los comandos de PR, validamos y normalizamos las etiquetas, y manejamos mejor los diffs grandes con un sistema de fallback, asegurando un flujo de trabajo más robusto.
- Mejoramos la localización y claridad de los mensajes: Agregamos mensajes internacionalizados para errores de permisos de token de GitHub y para el procesamiento de diffs grandes en PRs, brindando un feedback más claro y útil en español.
- Actualizamos nuestros modelos de IA: Migramos a los modelos Gemini v2.5, lo que nos permite ofrecer respuestas más precisas y eficientes. También mejoramos la configuración de la IA para que puedan ajustarla a sus necesidades.
- Correcciones de Estabilidad y Calidad: Solucionamos un problema que causaba errores en el servicio de IA en ciertas situaciones, mejoramos la precisión de `git add` y corregimos un error de ortografía en nuestros mensajes en español para una experiencia más pulcra.

## [v1.2.0] - 2025-02-18

[v1.2.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.1.1...v1.2.0

En esta versión de MateCommit, nos complace presentar una nueva funcionalidad que te va a ahorrar tiempo: la capacidad de resumir Pull Requests. Además, aprovechamos para fortalecer la aplicación mejorando el manejo de errores y su adaptabilidad, para que tu experiencia sea aún más fluida y confiable.

### Highlights

- Resumen de Pull Requests: Agregamos el comando `summarize-pr`, una herramienta potente que te permite obtener resúmenes concisos de tus Pull Requests directamente desde la terminal. Para hacer esto posible, implementamos un cliente de GitHub robusto que facilita la interacción con tus repositorios.
- Estabilidad y Adaptabilidad Mejoradas: Optimizamos significativamente el manejo de errores para que la aplicación sea más robusta frente a situaciones inesperadas. También ampliamos la internacionalización, haciendo a MateCommit más adaptable y fácil de usar en distintos contextos.

## [v1.1.1] - 2025-02-06

[v1.1.1]: https://github.com/Tomas-vilte/MateCommit/compare/v1.1.0...v1.1.1

En esta versión nos enfocamos en fortalecer la robustez de la aplicación. Implementamos mejoras clave en el manejo de errores para que las operaciones sean más estables y el feedback, más preciso.

### Highlights

- Manejo de Errores Mejorado: Reforzamos la gestión de errores al agregar archivos al staging. Esto significa que la aplicación es más robusta y te brindará información más clara si surge algún problema durante esta operación.

## [v1.1.0] - 2025-02-05

[v1.1.0]: https://github.com/Tomas-vilte/MateCommit/compare/v1.0.0...v1.1.0

En esta versión de MateCommit, nos enfocamos en expandir las capacidades de nuestra interfaz de línea de comandos. Ahora, los usuarios podrán configurar la inteligencia artificial directamente desde la CLI, lo que facilita una mayor personalización y control sobre sus flujos de trabajo.

### Highlights

- Configuración de IA en la CLI: Agregamos una nueva funcionalidad que permite configurar la inteligencia artificial directamente desde la línea de comandos, ofreciendo una experiencia más integrada y un control detallado para los usuarios avanzados.

## [v1.0.0] - 2025-01-15

[v1.0.0]: https://github.com/Tomas-vilte/MateCommit/releases/tag/v1.0.0

En esta primera versión de MateCommit, nos enfocamos en potenciar tu flujo de trabajo con la integración de inteligencia artificial. Ahora, generar mensajes de commit descriptivos y profesionales es más fácil que nunca, y te damos una bienvenida más cálida al iniciar la herramienta.

### Highlights

- **Sugerencias de Mensajes de Commit con IA:** Implementamos la integración con modelos de IA (como Gemini) para que MateCommit te ofrezca sugerencias inteligentes y relevantes para tus mensajes de commit. Esto te ayuda a mantener un historial de cambios claro y consistente sin esfuerzo.
- **Bienvenida Mejorada en la CLI:** Agregamos un mensaje de saludo al iniciar la aplicación, haciendo la experiencia de usuario más amigable y dándote la bienvenida a MateCommit.

