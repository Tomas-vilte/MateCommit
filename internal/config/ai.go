package config

type AI string

const (
	AIGemini AI = "gemini"
	AIOpenAI AI = "openai"
)

type Model string

const (
	ModelGeminiV25Pro       Model = "gemini-2.5-pro"
	ModelGeminiV25Flash     Model = "gemini-2.5-flash"
	ModelGeminiV25FlashLite Model = "gemini-2.5-flash-lite"

	// TODO: Agregar mas modelos para openai o otros...
	ModelGPTV4o     Model = "gpt-4o"
	ModelGPTV4oMini Model = "gpt-4o-mini"
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
			ModelGeminiV25Pro,
			ModelGeminiV25Flash,
			ModelGeminiV25FlashLite,
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
