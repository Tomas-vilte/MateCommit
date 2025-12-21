package routing

type ModelSelector struct{}

func NewModelSelector() *ModelSelector {
	return &ModelSelector{}
}

// SelectBestModel selects the optimal model based on the operation and token count
//
// Smart Routing Strategy:
//   - Small operations (< 1k tokens): Flash-Lite (most economical)
//   - Medium operations (1k-10k tokens): Flash (balance cost/quality)
//   - Large operations (> 10k tokens): 3.0 Flash (better context, avoids hallucinations)
//   - Releases/Issues: 3.0 Flash (maximum writing quality)
//
// SelectBestModel selects the optimal model based on the operation and token count
func (m *ModelSelector) SelectBestModel(operation string, estimatedTokens int) string {
	if operation == "generate-release" || operation == "generate-issue" {
		return "gemini-3-pro-preview"
	}

	if estimatedTokens > 15000 {
		return "gemini-3-flash-preview"
	}

	return "gemini-2.5-flash"
}

// GetRationale returns the translation key that explains why a model was chosen
func (m *ModelSelector) GetRationale(selectedModel string) string {
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
