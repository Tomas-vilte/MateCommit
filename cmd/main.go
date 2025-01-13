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

	app := &cli.Command{
		Name:    "mate-commit",
		Usage:   "🧉 Asistente inteligente para generar mensajes de commit",
		Version: "1.0.0",
		Description: `MateCommit te ayuda a generar mensajes de commit significativos usando IA.
		Ejemplos:
		   mate-commit suggest                    # Genera 3 sugerencias en el idioma predeterminado
		   mate-commit s -n 5 -l es              # Genera 5 sugerencias en español
		   mate-commit config show               # Muestra la configuración actual`,
		Commands: []*cli.Command{
			createSuggestCommand(cfg, gitService),
			createConfigCommand(),
		},
	}

	if err := app.Run(ctx, os.Args); err != nil {
		displayError(err)
	}
}

func createSuggestCommand(cfg *config.Config, gitService *git.GitService) *cli.Command {
	return &cli.Command{
		Name:        "suggest",
		Aliases:     []string{"s"},
		Usage:       "💡 Genera sugerencias de mensajes de commit",
		Description: "Analiza tus cambios y sugiere mensajes de commit apropiados",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"n"},
				Value:   3,
				Usage:   "Número de sugerencias (1-10)",
			},
			&cli.StringFlag{
				Name:    "lang",
				Aliases: []string{"l"},
				Value:   cfg.DefaultLang,
				Usage:   "Idioma (en, es)",
			},
			&cli.BoolFlag{
				Name:    "no-emoji",
				Aliases: []string{"ne"},
				Usage:   "Deshabilitar emojis",
			},
			&cli.IntFlag{
				Name:    "max-length",
				Aliases: []string{"ml"},
				Value:   72,
				Usage:   "Longitud máxima del mensaje",
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			// Validar que haya cambios para commitear
			if !gitService.HasStagedChanges() {
				return fmt.Errorf("❌ No hay cambios staged para commitear.\n💡 Usa 'git add' para agregar tus cambios primero")
			}

			count := command.Int("count")
			if count < 1 || count > 10 {
				return fmt.Errorf("❌ El número de sugerencias debe estar entre 1 y 10")
			}

			commitConfig := &config.CommitConfig{
				Locale:    config.GetLocaleConfig(command.String("lang")),
				MaxLength: int(command.Int("max-length")),
				UseEmoji:  !command.Bool("no-emoji"),
			}

			geminiService, err := gemini.NewGeminiService(ctx, cfg.GeminiAPIKey, commitConfig)
			if err != nil {
				return fmt.Errorf("❌ Error al inicializar Gemini: %w", err)
			}

			fmt.Println("🔍 Analizando cambios...")
			commitService := services.NewCommitService(gitService, geminiService)
			suggestions, err := commitService.GenerateSuggestions(ctx, int(count), cfg.Format)
			if err != nil {
				return fmt.Errorf("❌ Error al generar sugerencias: %w", err)
			}

			displaySuggestions(suggestions, commitConfig.Locale)
			return nil
		},
	}
}

func createConfigCommand() *cli.Command {
	return &cli.Command{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "⚙️  Gestiona la configuración",
		Commands: []*cli.Command{
			{
				Name:  "set-lang",
				Usage: "🌍 Configura el idioma predeterminado",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "lang",
						Aliases:  []string{"l"},
						Usage:    "Idioma (en, es)",
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					lang := command.String("lang")
					if lang != "en" && lang != "es" {
						return fmt.Errorf("❌ Idioma no soportado. Usa 'en' o 'es'")
					}

					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					cfg.DefaultLang = lang
					if err := config.SaveConfig(cfg); err != nil {
						return err
					}

					fmt.Printf("✅ Idioma configurado a: %s\n", lang)
					return nil
				},
			},
			{
				Name:  "show",
				Usage: "📋 Muestra la configuración actual",
				Action: func(ctx context.Context, command *cli.Command) error {
					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					fmt.Println("\n📋 Configuración actual:")
					fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━\n")
					fmt.Printf("🌍 Idioma: %s\n", cfg.DefaultLang)
					fmt.Printf("😊 Emojis: %v\n", cfg.UseEmoji)
					fmt.Printf("📏 Longitud máxima: %d\n", cfg.MaxLength)

					if cfg.GeminiAPIKey == "" {
						fmt.Println("🔑 API Key: ❌ No configurada")
						fmt.Println("\n💡 Tip: Configura tu API key con:")
						fmt.Println("   mate-commit config set-api-key --key <tu_api_key>")
					} else {
						fmt.Println("🔑 API Key: ✅ Configurada")
					}
					return nil
				},
			},
			{
				Name:  "set-api-key",
				Usage: "🔑 Configura la API Key de Gemini",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "key",
						Aliases:  []string{"k"},
						Usage:    "Tu API Key de Gemini",
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					apiKey := command.String("key")
					if len(apiKey) < 10 {
						return fmt.Errorf("❌ API Key inválida")
					}

					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					cfg.GeminiAPIKey = apiKey
					if err := config.SaveConfig(cfg); err != nil {
						return err
					}

					fmt.Println("✅ API Key configurada correctamente")
					fmt.Println("💡 Ahora puedes usar 'mate-commit suggest' para generar sugerencias")
					return nil
				},
			},
		},
	}
}

func displaySuggestions(suggestions []string, locale config.CommitLocale) {
	fmt.Printf("\n📝 %s\n", locale.HeaderMsg)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	for i, suggestion := range suggestions {
		fmt.Printf("\n%d. %s\n", i+1, suggestion)
	}
	fmt.Printf("\n💡 %s\n", locale.UsageMsg)
	fmt.Printf("  git commit -m \"[número de sugerencia]\"\n")
}

func displayError(err error) {
	log.Fatalf("❌ Error: %v", err)
}
