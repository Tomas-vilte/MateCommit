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
	ModelGeminiV3Pro    Model = "gemini-3-pro-preview"
	ModelGeminiV3Flash  Model = "gemini-3-flash-preview"

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
			ModelGeminiV3Flash,
			ModelGeminiV15Pro,
			ModelGeminiV3Pro,
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
