# Arquitectura de Proveedores de IA

Este documento explica la arquitectura de proveedores de IA en MateCommit y cómo agregar nuevos proveedores.

## Diseño Actual

### Componentes Principales

1. **`ports.CostAwareAIProvider`** (interfaz): Define el contrato que todos los proveedores deben cumplir
2. **`ai.CostAwareWrapper`**: Wrapper agnóstico que maneja caché, presupuesto y tracking de costos
3. **Provider específico** (ej: `gemini.GeminiProvider`): Implementación base de la interfaz para cada proveedor
4. **Servicios específicos** (ej: `gemini.GeminiCommitSummarizer`): Lógica de negocio que usa el provider

### Flujo de Datos

```
Usuario → Servicio (CommitSummarizer) → CostAwareWrapper → Provider (Gemini) → API Externa
                                             ↓
                                    Caché + Presupuesto + Tracking
```

## Patrón de Implementación

### 1. Proveedor Actual: Gemini

```go
// internal/infrastructure/ai/gemini/base.go
type GeminiProvider struct {
    Client *genai.Client
    model  string
}

func (g *GeminiProvider) CountTokens(ctx context.Context, prompt string) (int, error) {
    // Implementación específica de Gemini
}

func (g *GeminiProvider) GetModelName() string { return g.model }
func (g *GeminiProvider) GetProviderName() string { return "gemini" }
```

### 2. Servicio que usa el Provider

```go
type GeminiCommitSummarizer struct {
    *GeminiProvider  // Embedding: hereda los métodos de la interfaz
    wrapper *ai.CostAwareWrapper
    config  *config.Config
    trans   *i18n.Translations
}

func NewGeminiCommitSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*GeminiCommitSummarizer, error) {
    client, _ := genai.NewClient(...)

    // Crear servicio con provider embedado
    service := &GeminiCommitSummarizer{
        GeminiProvider: NewGeminiProvider(client, modelName),
        config:         cfg,
        trans:          trans,
    }

    // Crear wrapper pasando el servicio como provider
    wrapper, _ := ai.NewCostAwareWrapper(ai.WrapperConfig{
        Provider:              service,  // Implementa CostAwareAIProvider vía embedding
        BudgetDaily:           budgetDaily,
        Trans:                 trans,
        EstimatedOutputTokens: 800,
    })

    service.wrapper = wrapper
    return service, nil
}
```

## Cómo Agregar un Nuevo Proveedor

### Ejemplo: OpenAI

#### Paso 1: Crear el Provider Base

```go
// internal/infrastructure/ai/openai/base.go
package openai

import (
    "context"
    "github.com/Tomas-vilte/MateCommit/internal/domain/ports"
    "github.com/openai/openai-go"
)

type OpenAIProvider struct {
    Client *openai.Client
    model  string
}

func NewOpenAIProvider(client *openai.Client, model string) *OpenAIProvider {
    return &OpenAIProvider{
        Client: client,
        model:  model,
    }
}

// Implementar la interfaz ports.CostAwareAIProvider

func (o *OpenAIProvider) CountTokens(ctx context.Context, prompt string) (int, error) {
    // Usar tiktoken o la API de OpenAI para contar tokens
    // Implementación específica de OpenAI
}

func (o *OpenAIProvider) GetModelName() string {
    return o.model
}

func (o *OpenAIProvider) GetProviderName() string {
    return "openai"
}

// Verificar que implementa la interfaz
var _ ports.CostAwareAIProvider = (*OpenAIProvider)(nil)
```

#### Paso 2: Crear un Servicio

```go
// internal/infrastructure/ai/openai/commit_summarizer_service.go
package openai

import (
    "github.com/Tomas-vilte/MateCommit/internal/infrastructure/ai"
    // ... otros imports
)

type OpenAICommitSummarizer struct {
    *OpenAIProvider  // Embedding del provider base
    wrapper *ai.CostAwareWrapper
    config  *config.Config
    trans   *i18n.Translations
}

func NewOpenAICommitSummarizer(ctx context.Context, cfg *config.Config, trans *i18n.Translations) (*OpenAICommitSummarizer, error) {
    client := openai.NewClient(cfg.AIProviders["openai"].APIKey)
    modelName := string(cfg.AIConfig.Models[config.AIOpenAI])

    service := &OpenAICommitSummarizer{
        OpenAIProvider: NewOpenAIProvider(client, modelName),
        config:         cfg,
        trans:          trans,
    }

    wrapper, err := ai.NewCostAwareWrapper(ai.WrapperConfig{
        Provider:              service,
        BudgetDaily:           cfg.AIConfig.BudgetDaily,
        Trans:                 trans,
        EstimatedOutputTokens: 800,
    })
    if err != nil {
        return nil, err
    }

    service.wrapper = wrapper
    return service, nil
}

func (s *OpenAICommitSummarizer) GenerateSuggestions(ctx context.Context, info models.CommitInfo, count int) ([]models.CommitSuggestion, error) {
    prompt := s.generatePrompt(info, count)

    // Función de generación específica de OpenAI
    generateFn := func(ctx context.Context, p string) (interface{}, *models.TokenUsage, error) {
        resp, err := s.Client.Chat.Completions.Create(ctx, openai.ChatCompletionCreateParams{
            Model: s.GetModelName(),
            Messages: []openai.ChatCompletionMessageParam{
                {Role: "user", Content: p},
            },
        })

        usage := &models.TokenUsage{
            InputTokens:  resp.Usage.PromptTokens,
            OutputTokens: resp.Usage.CompletionTokens,
            TotalTokens:  resp.Usage.TotalTokens,
        }

        return resp, usage, nil
    }

    // El wrapper maneja caché, presupuesto y tracking
    resp, usage, err := s.wrapper.WrapGenerate(ctx, "suggest-commits", prompt, generateFn)
    if err != nil {
        return nil, err
    }

    // Parsear respuesta de OpenAI...
}
```

#### Paso 3: Actualizar la Tabla de Precios

```go
// internal/domain/services/cost/calculator.go
var pricing = ProviderPricing{
    "gemini": {
        "gemini-2.5-flash": {InputPricePerMillion: 0.30, OutputPricePerMillion: 2.50},
        "gemini-3.0-flash": {InputPricePerMillion: 0.50, OutputPricePerMillion: 3.00},
    },
    "openai": {  // NUEVO
        "gpt-4o":      {InputPricePerMillion: 2.50, OutputPricePerMillion: 10.00},
        "gpt-4o-mini": {InputPricePerMillion: 0.15, OutputPricePerMillion: 0.60},
    },
}
```

## Ventajas de Esta Arquitectura

✅ **Extensibilidad**: Agregar un proveedor nuevo es agregar un package con base.go
✅ **DRY**: La lógica de caché, presupuesto y tracking está en un solo lugar (CostAwareWrapper)
✅ **Type-safe**: Las interfaces garantizan que todos los providers tengan los métodos necesarios
✅ **Testeable**: Fácil mockear la interfaz `CostAwareAIProvider` para tests
✅ **Idiomático**: Usa composición (embedding) en lugar de herencia

## Trade-offs Aceptados

⚠️ **Construcción en dos pasos**: El servicio se crea, luego el wrapper, luego se asigna al servicio
⚠️ **Type assertions**: `WrapGenerate` retorna `interface{}`, requiere cast al tipo específico
⚠️ **Sin genéricos**: Se priorizó simplicidad sobre type-safety completa

## Cuándo Usar Qué

- **Nuevo proveedor completo** (OpenAI, Anthropic): Crear `internal/infrastructure/ai/[provider]/base.go`
- **Nuevo servicio en proveedor existente**: Embedar el provider base existente
- **Modificar precios**: Actualizar `internal/domain/services/cost/calculator.go`
- **Cambiar lógica de caché/presupuesto**: Modificar `internal/infrastructure/ai/cost_wrapper.go`
