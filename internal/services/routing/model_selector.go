package routing

type ModelSelector struct{}

func NewModelSelector() *ModelSelector {
	return &ModelSelector{}
}

// SelectBestModel selecciona el modelo óptimo basado en la operación y cantidad de tokens
//
// Estrategia de Smart Routing:
//   - Operaciones pequeñas (< 1k tokens): Flash-Lite (más económico)
//   - Operaciones medianas (1k-10k tokens): Flash (balance costo/calidad)
//   - Operaciones grandes (> 10k tokens): 3.0 Flash (mejor contexto, evita alucinaciones)
//   - Releases/Issues: 3.0 Flash (máxima calidad de redacción)
//
// SelectBestModel selecciona el modelo óptimo basado en la operación y cantidad de tokens
func (m *ModelSelector) SelectBestModel(operation string, estimatedTokens int) string {
	if operation == "generate-release" || operation == "generate-issue" {
		return "gemini-3-pro-preview"
	}

	if estimatedTokens > 15000 {
		return "gemini-3-flash-preview"
	}

	return "gemini-2.5-flash"
}

// GetRationale retorna la clave de traducción que explica por qué se eligió un modelo
func (m *ModelSelector) GetRationale(operation string, selectedModel string) string {
	switch selectedModel {
	case "gemini-1.5-flash":
		return "routing.reason_balance"
	case "gemini-3-flash-preview":
		return "routing.reason_large"
	case "gemini-3-pro-preview":
		return "routing.reason_high_quality"
	default:
		return "routing.reason_default"
	}
}
