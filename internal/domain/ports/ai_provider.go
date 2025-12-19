package ports

import (
	"context"
)

// CostAwareAIProvider define la interfaz para proveedores de IA que soportan tracking de costos.
type CostAwareAIProvider interface {
	// CountTokens cuenta los tokens de un prompt sin hacer la llamada real al modelo.
	// Esto permite estimar el costo antes de ejecutar la generación.
	CountTokens(ctx context.Context, prompt string) (int, error)

	// GetModelName retorna el nombre del modelo actual (ej: "gemini-2.5-flash")
	GetModelName() string

	// GetProviderName retorna el nombre del proveedor (ej: "gemini", "openai", "anthropic")
	GetProviderName() string
}

// TokenCounter es una interfaz más simple para proveedores que solo necesitan contar tokens.
type TokenCounter interface {
	CountTokens(ctx context.Context, content string) (int, error)
}
