package config

type AI string

const (
	AIGemini AI = "gemini"
	AIOpenAI AI = "openai"
)

type Model string

const (
	ModelGeminiV15Flash Model = "gemini-1.5-flash"
	ModelGeminiV15Pro   Model = "gemini-1.5-pro"
	ModelGeminiV20Flash Model = "gemini-2.0-flash"

	// TODO: Agregar mas modelos para openai o otros...
	ModelGPTV4o     Model = "gpt-4o"
	ModelGPTV4oMini Model = "gpt-4o-mini"
)
