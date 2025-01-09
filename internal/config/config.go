package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	GeminiAPIKey string
	ProjectPath  string
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	return &Config{
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
	}, nil
}
