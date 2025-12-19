# Plan de Implementación: Smart Routing & Control de Costos 🧉

Este documento detalla cómo vamos a encarar la **Issue #50** para darle a MateCommit inteligencia financiera y de ruteo, aprovechando la salida de **Gemini 3.0 Flash**.

---

## 🚀 La Nueva Estrella: Gemini 3.0 Flash

Che, la gran novedad es que integramos este modelo que rompe todo. Es rápido como el 2.5 pero razona casi como un Pro.

### 💸 Tabla de Precios (Oficial)
| Tipo de Token | Precio (por 1M) | Detalle |
| :--- | :--- | :--- |
| **Input (Entrada)** | **$0.50** | Lo que le mandamos (diffs, contexto) |
| **Output (Salida)** | **$3.00** | Lo que nos responde (el commit, resumen) |

> **Ojo al piojo:** La salida es 6 veces más cara. Por eso nuestra estimación tiene que ser fina ahí.

---

## 🧠 Smart Routing: El Cerebro

La idea es que no gastes pólvora en chimangos. El sistema va a decidir solo (o sugerirte):

1.  **Diffs Chicos (< 1k tokens):** Se van por **Gemini 2.5 Flash**. Es barato y sobra paño.
2.  **Diffs Grandes (> 10k tokens):** Activamos **Gemini 3.0 Flash**. ¿Por qué? Porque tiene mejor contexto y no alucina cuando le tiras un choclo de código.
3.  **Caching Local:** Si ya hiciste esta pregunta exacta, la sacamos de tu disco. **Costo: $0**.

---

## 🔄 Flujo del Usuario (User Journey)

Así va a ser la experiencia cuando tires un comando:

1.  Vos tirás: `matecommit summarize pr --n 50`
2.  MateCommit **cuenta los tokens** (sin cobrarte nada todavía).
3.  Te canta la justa:
    > "Che, analizar este PR te va a salir **~$0.01 USD**. ¿Le mandamos mecha?" [Y/n]
4.  Si decís que sí, recién ahí llamamos a la API.
5.  Al final, te tiramos la posta: "Costo final real: **$0.0098**".

---

## 🛠️ Cambios Técnicos (Lo que vamos a codear)

### 1. Configuración y Modelos
*   Modificar `internal/config/ai.go` para agregar `gemini-3.0-flash`.

### 2. El "Calculator Service" (`internal/domain/services/cost/`)
Vamos a crear un servicio nuevo que se encargue de:
*   `CountTokens()`: Usar la API para contar exacto.
*   `EstimateCost()`: Calcular $$ basado en la tabla de arriba.
*   `CheckBudget()`: Si tenés un límite diario (ej. $2 USD) y te vas a pasar, te avisamos.
*   **Historial de Actividad Completo**: Guardamos un JSON súper detallado en `~/.matecommit/history.json`:
    *   `timestamp`: Cuándo fue.
    *   `model`: Qué modelo usaste (para ver si el 3.0 te rinde más).
    *   `latency_ms`: Cuánto tardó (para medir velocidad).
    *   `cost_usd`: La dolorosa.
    *   `tokens_saved`: Si hubo caché, cuánto te ahorraste.

### 3. Caché Local (Anti-Crisis)
*   **¿Qué es?** Un archivo en tu compu.
*   **¿Cómo funciona?** Calculamos una "huella digital" (hash) de tu código. Si volvés a pedir lo mismo, leemos el archivo local.
*   **Diferencia con Gemini Cache:** Google ofrece "Context Caching" pero te cobra por guardar. Nosotros hacemos **Caché de Respuesta** en tu disco, que es gratis y más rápido.

### 4. Integración Global
Esto no es solo para PRs, eh. Lo vamos a meter en **todos** los comandos:
*   [ ] `summarize pr`
*   [ ] `suggest commits` (el clásico)
*   [ ] `generate release`
*   [ ] `generate issue`

### 5. Nuevo Comando: `matecommit cost`
Para ver tu resumen mensual: "Este mes gastaste $0.45 USD en 15 PRs".

---

## ✅ Plan de Pruebas

Para estar seguros que no le erramos al vizcachazo:
1.  **Test Unitario:** Verificar que 1 millón de tokens de entrada nos de exactamente $0.50.
2.  **Dry Run:** Correr la CLI, ver la estimación, y compararla con lo que realmente nos cobra Google en el dashboard.

---
*Documento generado automáticamente por tu asistente de IA favorito.* 😉
