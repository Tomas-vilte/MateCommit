# Smart Routing & Control de Costos - Guía de Usuario

Este documento explica las nuevas features de inteligencia de costos implementadas en MateCommit.

## 🧠 Smart Routing Automático

MateCommit ahora selecciona automáticamente el modelo óptimo según la complejidad de la tarea:

### Estrategia de Selección

| Operación | Tokens | Modelo Seleccionado | Razón |
|-----------|--------|---------------------|-------|
| `suggest-commits` | < 1,000 | Gemini 2.5 Flash-Lite | Económico para cambios pequeños |
| `suggest-commits` | 1,000-10,000 | Gemini 2.5 Flash | Balance costo/calidad |
| `suggest-commits` | > 10,000 | Gemini 3.0 Flash | Mejor contexto, evita alucinaciones |
| `summarize-pr` | Cualquiera | Según tokens | Mismo criterio que commits |
| `generate-release` | Cualquiera | Gemini 3.0 Flash | Máxima calidad de redacción |
| `generate-issue` | Cualquiera | Gemini 3.0 Flash | Claridad y detalle |

### Ejemplo de Sugerencia

Si estás usando Gemini 2.5 Flash pero tienes un diff grande (> 10k tokens), verás:

```
💡 Sugerencia: Operación grande (> 10k tokens), requiere mejor manejo de contexto
   Modelo sugerido: gemini-3.0-flash (actualmente usando: gemini-2.5-flash)
```

Esta es solo una sugerencia. Puedes cambiar el modelo en tu configuración si lo prefieres.

---

## 💰 Confirmación de Costo

Para operaciones que cuestan más de **$0.005 USD**, MateCommit pedirá confirmación:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💰 Estimación de Costo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Tokens de entrada estimados:  12500
📤 Tokens de salida estimados:   800
💵 Costo estimado:                $0.0077 USD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
¿Desea continuar? [Y/n]:
```

### Respuestas Aceptadas

- **Continuar:** presiona `Enter`, `Y`, `y`, `yes`, `si`, o `s`
- **Cancelar:** presiona `N` o `n`

### Deshabilitar Confirmación

Si no quieres que te pregunte (útil para CI/CD), puedes:

1. **Configuración global:** Agregar a `~/.matecommit/config.toml`:
   ```toml
   [ai_config]
   skip_confirmation = true
   ```

2. **Variable de entorno:**
   ```bash
   export MATECOMMIT_SKIP_CONFIRMATION=true
   ```

---

## 🚨 Alertas de Presupuesto

Si configuraste un presupuesto diario, verás alertas progresivas:

### Alerta al 50% (Amarillo)

```
⚠️  Has usado 52% de tu presupuesto diario ($0.52 / $1.00)
```

### Alerta al 75% (Amarillo Bold)

```
⚠️  ¡Cuidado! Has usado 78% de tu presupuesto diario
   Total gastado: $0.78 / $1.00
```

### Alerta al 90% (Rojo Bold)

```
🚨 ¡ALERTA! Has usado 93% de tu presupuesto diario
   Total gastado: $0.93 / $1.00
   Quedan solo: $0.07
```

### Presupuesto Excedido

Si una operación excedería tu presupuesto:

```
❌ Presupuesto diario excedido
   Gastado hoy:      $0.98
   Costo estimado:   $0.05
   Total sería:      $1.03
   Límite diario:    $1.00
   Exceso:           $0.03

Error: presupuesto diario excedido...
```

### Configurar Presupuesto

Edita `~/.matecommit/config.toml`:

```toml
[ai_config]
budget_daily = 2.00  # $2 USD por día
```

O al crear la configuración:

```bash
matecommit config init
# Cuando pregunte por el presupuesto diario, ingresa: 2.00
```

**Sin presupuesto:** Si no configuras `budget_daily` o lo pones en `0`, no habrá límites.

---

## 📊 Ver Estadísticas

### Estadísticas Diarias

```bash
matecommit stats
```

Salida:
```
📊 Estadísticas Diarias
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

10:30 - suggest-commits: $0.0003
11:45 - summarize-pr: $0.0012
14:20 - generate-release: $0.0045 [CACHE]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Hoy: $0.0060 USD
```

**[CACHE]** indica que la respuesta salió del caché local (costo $0).

### Estadísticas Mensuales

```bash
matecommit stats --monthly
```

Salida:
```
📅 Estadísticas Mensuales - December 2025
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

2025-12-17: $0.0234
2025-12-18: $0.0567
2025-12-19: $0.0060

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total Este Mes: $0.0861 USD
```

### Alias

Puedes usar `cost` como alias:

```bash
matecommit cost          # = matecommit stats
matecommit cost -m       # = matecommit stats --monthly
```

---

## 💾 Caché Local

### Beneficios

El caché guarda respuestas por **24 horas**:

- **Costo:** $0.00 (gratis)
- **Velocidad:** Instantáneo
- **Ubicación:** `~/.matecommit/cache/`

### Cuándo se usa

Si ejecutas **exactamente el mismo comando** dos veces:

```bash
# Primera vez: llama a la API, cuesta $0.0003
matecommit suggest

# Segunda vez (< 24h): lee del caché, cuesta $0
matecommit suggest
```

El hash incluye:
- Proveedor (gemini)
- Modelo (gemini-2.5-flash)
- Prompt completo (diff + contexto)

**Cambió algo?** → Nuevo hash → No usa caché

### Limpiar Caché

```bash
matecommit cache clean
```

Salida:
```
✓ Caché limpiado exitosamente
```

Esto elimina todos los archivos en `~/.matecommit/cache/`.

---

## 🎯 Ejemplos de Uso

### Ejemplo 1: Commit Pequeño

```bash
# Cambio de 3 líneas en un archivo
git add file.go
matecommit suggest
```

Salida:
```
💡 Sugerencia: Operación pequeña (< 1k tokens), modelo económico suficiente
   Modelo sugerido: gemini-2.5-flash-lite (actualmente usando: gemini-2.5-flash)

[Genera sugerencias sin pedir confirmación porque cuesta < $0.005]
```

### Ejemplo 2: PR Grande

```bash
# PR con 50 archivos modificados
matecommit summarize-pr --n 123
```

Salida:
```
💡 Sugerencia: Operación grande (> 10k tokens), requiere mejor manejo de contexto
   Modelo sugerido: gemini-3.0-flash (actualmente usando: gemini-2.5-flash)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💰 Estimación de Costo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Tokens de entrada estimados:  15800
📤 Tokens de salida estimados:   500
💵 Costo estimado:                $0.0129 USD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
¿Desea continuar? [Y/n]: y

[Genera el resumen del PR]
```

### Ejemplo 3: Presupuesto Casi Agotado

```bash
# Ya gastaste $0.95 de $1.00 hoy
matecommit suggest
```

Salida:
```
🚨 ¡ALERTA! Has usado 95% de tu presupuesto diario
   Total gastado: $0.95 / $1.00
   Quedan solo: $0.05

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💰 Estimación de Costo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Tokens de entrada estimados:  800
📤 Tokens de salida estimados:   500
💵 Costo estimado:                $0.0015 USD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
¿Desea continuar? [Y/n]:
```

---

## 📁 Archivos de Datos

### Historial

**Ubicación:** `~/.matecommit/history.json`

```json
[
  {
    "timestamp": "2025-12-19T10:30:00Z",
    "command": "suggest-commits",
    "provider": "gemini",
    "model": "gemini-2.5-flash",
    "tokens_input": 450,
    "tokens_output": 120,
    "cost_usd": 0.0003,
    "duration_ms": 1250,
    "cache_hit": false,
    "hash": "abc123..."
  }
]
```

### Caché

**Ubicación:** `~/.matecommit/cache/[hash].json`

```json
{
  "hash": "abc123...",
  "response": { ... },
  "created_at": "2025-12-19T10:30:00Z"
}
```

**TTL:** 24 horas (auto-limpieza al iniciar)

---

## 💡 Tips para Ahorrar

1. **Usa caché:** Si no cambiaste nada, la segunda ejecución es gratis
2. **Configura presupuesto:** Te protege de gastos inesperados
3. **Presta atención a las sugerencias:** Si te sugiere Flash-Lite, probablemente no necesitas Pro
4. **Limpia archivos grandes antes de commit:** `go.sum`, `package-lock.json` no aportan al análisis

---

## ⚙️ Configuración Avanzada

### Deshabilitar Smart Routing

Si prefieres controlar manualmente el modelo:

```toml
# ~/.matecommit/config.toml
[ai_config]
model = "gemini-3.0-flash"  # Siempre usa este
```

MateCommit seguirá sugiriendo, pero usará el modelo que configuraste.

### Ajustar Umbral de Confirmación

Actualmente hardcodeado en `$0.005`, pero podrías modificar en:

`internal/infrastructure/ai/cost_wrapper.go:116`

```go
if estimatedCost > 0.010 && !w.skipConfirmation {  // Cambiar de 0.005 a 0.010
```

### Cambiar TTL del Caché

`internal/domain/services/cache/cache.go` al construir:

```go
cache.NewCache(48 * time.Hour)  // Cambiar de 24h a 48h
```

---

## 🐛 Troubleshooting

### "Presupuesto excedido" pero no configuré ninguno

Verifica `~/.matecommit/config.toml`:

```toml
[ai_config]
budget_daily = 0  # 0 = sin límite
```

### El caché no funciona

1. Verifica que `~/.matecommit/cache/` existe
2. Chequea permisos: `chmod 755 ~/.matecommit/cache`
3. Limpia y reinicia: `matecommit cache clean`

### Sugerencias de modelo equivocadas

Abre un issue en GitHub con:
- Comando ejecutado
- Cantidad de tokens estimados
- Modelo sugerido vs esperado
