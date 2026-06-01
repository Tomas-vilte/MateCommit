package config

type AI string

const (
	AIGemini AI = "gemini"
	AIOpenAI AI = "openai"
)

type Model string

const (
	ModelGeminiV15Pro   Model = "gemini-1.5-pro"
	ModelGeminiV15Flash Model = "gemini-1.5-flash"
	ModelGeminiV25Flash Model = "gemini-2.5-flash"
	ModelGeminiV31Pro   Model = "gemini-3.1-pro-preview"
	ModelGeminiV35Flash Model = "gemini-3.5-flash"

	// TODO: Add more models for OpenAI or others...
	ModelGPTV4o Model = "gpt-4o"
)

func SupportedAIs() []AI {
	return []AI{
		AIGemini,
	}
}

func ModelsForAI(ai AI) []Model {
	switch ai {
	case AIGemini:
		return []Model{
			ModelGeminiV15Flash,
			ModelGeminiV25Flash,
			ModelGeminiV35Flash,
			ModelGeminiV15Pro,
			ModelGeminiV31Pro,
		}
	default:
		return []Model{}
	}
}

func DefaultModelForAI(ai AI) Model {
	models := ModelsForAI(ai)
	if len(models) == 0 {
		return ""
	}
	return models[0]
}
