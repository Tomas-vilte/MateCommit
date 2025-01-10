package main

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
	"log"
	"os"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error al cargar configuraciones:", err)
	}

	gitService := git.NewGitService()
	geminiService := gemini.NewGeminiService(ctx, cfg.GeminiAPIKey)
	commitService := services.NewCommitService(gitService, geminiService)

	app := &cli.Command{
		Name:  "commit-suggester",
		Usage: "Genera sugerencias de mensajes de commit usando IA",
		Commands: []*cli.Command{
			{
				Name:    "suggest",
				Aliases: []string{"s"},
				Usage:   "Sugiere mensajes de commit basados en los cambios actuales",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "count",
						Aliases: []string{"n"},
						Value:   3,
						Usage:   "Numero de sugerencias a generar",
					},
					&cli.StringFlag{
						Name:    "format",
						Aliases: []string{"f"},
						Value:   "conventional",
						Usage:   "Formato del commit (conventional)",
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					count := command.Int("count")
					format := command.String("format")

					suggestions, err := commitService.GenerateSuggestions(ctx, int(count), format)
					if err != nil {
						return err
					}

					fmt.Println("\n📝 Sugerencias de commit:")
					fmt.Println("------------------------")
					for i, suggestion := range suggestions {
						fmt.Printf("\n%d. %s\n", i+1, suggestion)
					}
					fmt.Println("\n💡 Para usar una sugerencia:")
					fmt.Printf("git commit -m \"[sugerencia elegida]\"\n\n")

					return nil
				},
			},
		},
	}

	if err := app.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
