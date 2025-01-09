package main

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"log"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error al cargar configuraciones:", err)
	}
	ctx := context.Background()

	gitService := git.NewGitService()
	geminiService := gemini.NewGeminiService(ctx, cfg.GeminiAPIKey)
	commitService := services.NewCommitService(gitService, geminiService)

	message, err := commitService.GenerateAndCommit(ctx)
	if err != nil {
		log.Fatal("Error generando commit:", err)
	}

	fmt.Printf("Commit Generado: %s\n", message)
}
