package main

import (
	"context"
	"fmt"
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/gemini"
	"github.com/Tomas-vilte/MateCommit/internal/infrastructure/git"
	"github.com/Tomas-vilte/MateCommit/internal/services"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"strings"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Error al cargar configuraciones:", err)
	}

	fmt.Println("Probando release")

	t, err := i18n.NewTranslations(cfg.DefaultLang)
	if err != nil {
		log.Fatal("Error al inicializar traducciones:", err)
	}

	gitService := git.NewGitService()

	app := &cli.Command{
		Name:        "mate-commit",
		Usage:       t.GetMessage("app_usage", 0, nil),
		Version:     "1.0.0",
		Description: t.GetMessage("app_description", 0, nil),
		Commands: []*cli.Command{
			createSuggestCommand(cfg, gitService, t),
			createConfigCommand(t),
		},
	}

	if err := app.Run(ctx, os.Args); err != nil {
		displayError(err)
	}
}

func createSuggestCommand(cfg *config.Config, gitService *git.GitService, t *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:        "suggest",
		Aliases:     []string{"s"},
		Usage:       t.GetMessage("suggest_command_usage", 0, nil),
		Description: t.GetMessage("suggest_command_description", 0, nil),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "count",
				Aliases: []string{"n"},
				Value:   3,
				Usage:   t.GetMessage("suggest_count_flag_usage", 0, nil),
			},
			&cli.StringFlag{
				Name:    "lang",
				Aliases: []string{"l"},
				Value:   cfg.DefaultLang,
				Usage:   t.GetMessage("suggest_lang_flag_usage", 0, nil),
			},
			&cli.BoolFlag{
				Name:    "no-emoji",
				Aliases: []string{"ne"},
				Usage:   t.GetMessage("suggest_no_emoji_flag_usage", 0, nil),
			},
			&cli.IntFlag{
				Name:    "max-length",
				Aliases: []string{"ml"},
				Value:   72,
				Usage:   t.GetMessage("suggest_max_length_flag_usage", 0, nil),
			},
		},
		Action: func(ctx context.Context, command *cli.Command) error {
			// Validar que haya cambios para commitear
			if !gitService.HasStagedChanges() {
				msg := t.GetMessage("no_staged_changes", 0, nil)
				return fmt.Errorf("%s", msg)
			}

			count := command.Int("count")
			if count < 1 || count > 10 {
				msg := t.GetMessage("invalid_suggestions_count", 0, map[string]interface{}{
					"Min": 1,
					"Max": 10,
				})
				return fmt.Errorf("%s", msg)
			}

			commitConfig := &config.CommitConfig{
				Locale:    config.GetLocaleConfig(command.String("lang")),
				MaxLength: int(command.Int("max-length")),
				UseEmoji:  !command.Bool("no-emoji"),
			}

			geminiService, err := gemini.NewGeminiService(ctx, cfg.GeminiAPIKey, commitConfig, t)
			if err != nil {
				msg := t.GetMessage("gemini_init_error", 0, map[string]interface{}{
					"Error": err,
				})
				return fmt.Errorf("%s", msg)
			}

			fmt.Println(t.GetMessage("analyzing_changes", 0, nil))
			commitService := services.NewCommitService(gitService, geminiService)
			suggestions, err := commitService.GenerateSuggestions(ctx, int(count), cfg.Format)
			if err != nil {
				msg := t.GetMessage("suggestion_generation_error", 0, map[string]interface{}{
					"Error": err,
				})
				return fmt.Errorf("%s", msg)
			}

			displaySuggestions(suggestions, gitService, t)
			return nil
		},
	}
}

func createConfigCommand(t *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   t.GetMessage("config_command_usage", 0, nil),
		Commands: []*cli.Command{
			{
				Name:  "set-lang",
				Usage: t.GetMessage("config_set_lang_usage", 0, nil),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "lang",
						Aliases:  []string{"l"},
						Usage:    t.GetMessage("config_set_lang_flag_usage", 0, nil),
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					lang := command.String("lang")
					if lang != "en" && lang != "es" {
						msg := t.GetMessage("unsupported_language", 0, nil)
						return fmt.Errorf("%s", msg)
					}

					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					cfg.DefaultLang = lang
					if err := config.SaveConfig(cfg); err != nil {
						return err
					}

					fmt.Printf("%s\n", t.GetMessage("language_configured", 0, map[string]interface{}{
						"Lang": lang,
					}))
					return nil
				},
			},
			{
				Name:  "show",
				Usage: t.GetMessage("config_show_usage", 0, nil),
				Action: func(ctx context.Context, command *cli.Command) error {
					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					fmt.Println(t.GetMessage("current_config", 0, nil))
					fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━\n")
					fmt.Printf("%s\n", t.GetMessage("language_label", 0, map[string]interface{}{
						"Lang": cfg.DefaultLang,
					}))
					fmt.Printf("%s\n", t.GetMessage("emojis_label", 0, map[string]interface{}{
						"Emoji": cfg.UseEmoji,
					}))
					fmt.Printf("%s\n", t.GetMessage("max_length_label", 0, map[string]interface{}{
						"MaxLength": cfg.MaxLength,
					}))

					if cfg.GeminiAPIKey == "" {
						fmt.Println(t.GetMessage("api.key_not_set", 0, nil))
						fmt.Println(t.GetMessage("api.key_tip", 0, nil))
						fmt.Println(t.GetMessage("api.key_config_command", 0, nil))
					} else {
						fmt.Println(t.GetMessage("api.key_set", 0, nil))
					}
					return nil
				},
			},
			{
				Name:  "set-api-key",
				Usage: t.GetMessage("commands.set_api_key_usage", 0, nil),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "key",
						Aliases:  []string{"k"},
						Usage:    t.GetMessage("flags.gemini_api_key", 0, nil),
						Required: true,
					},
				},
				Action: func(ctx context.Context, command *cli.Command) error {
					apiKey := command.String("key")
					if len(apiKey) < 10 {
						msg := t.GetMessage("api.invalid_key", 0, nil)
						return fmt.Errorf("%s", msg)
					}

					cfg, err := config.LoadConfig()
					if err != nil {
						return err
					}

					cfg.GeminiAPIKey = apiKey
					if err := config.SaveConfig(cfg); err != nil {
						return err
					}

					fmt.Println(t.GetMessage("api.key_configured", 0, nil))
					fmt.Println(t.GetMessage("api.key_configuration_help", 0, nil))
					return nil
				},
			},
		},
	}
}

func displaySuggestions(suggestions []models.CommitSuggestion, gitService *git.GitService, t *i18n.Translations) {
	fmt.Printf("%s\n", t.GetMessage("commit.header_message", 0, nil))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")

	for _, suggestion := range suggestions {
		fmt.Printf("%s\n", suggestion.CommitTitle)
		fmt.Println(t.GetMessage("commit.file_list_header", 0, nil))
		for _, file := range suggestion.Files {
			fmt.Printf("   - %s\n", file)
		}
		fmt.Printf("%s\n", suggestion.Explanation)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	}

	fmt.Println(t.GetMessage("commit.select_option_prompt", 0, nil))
	fmt.Println(t.GetMessage("commit.option_commit", 0, nil))
	fmt.Println(t.GetMessage("commit.option_exit", 0, nil))

	if err := handleCommitSelection(suggestions, gitService, t); err != nil {
		msg := t.GetMessage("commit.error_creating_commit", 0, nil)
		fmt.Printf("%s", msg)
	}
}

func displayError(err error) {
	log.Fatal(err)
}

func handleCommitSelection(suggestions []models.CommitSuggestion, gitService *git.GitService, t *i18n.Translations) error {
	var selection int
	fmt.Print(t.GetMessage("commit.prompt_selection", 0, nil))
	_, err := fmt.Scan(&selection)
	if err != nil {
		msg := t.GetMessage("commit.error_reading_selection", 0, map[string]interface{}{
			"Error": err,
		})
		return fmt.Errorf("%s", msg)
	}

	if selection == 0 {
		fmt.Println(t.GetMessage("commit.operation_canceled", 0, nil))
		return nil
	}

	if selection < 1 || selection > len(suggestions) {
		msg := t.GetMessage("commit.invalid_selection", 0, map[string]interface{}{
			"Number": len(suggestions),
		})
		return fmt.Errorf("%s", msg)
	}
	suggestions[0].CommitTitle = strings.TrimPrefix(suggestions[0].CommitTitle, "Commit: ")
	selectedSuggestion := suggestions[selection-1]

	if err := gitService.CreateCommit(selectedSuggestion.CommitTitle); err != nil {
		return err
	}

	fmt.Printf("%s\n", t.GetMessage("commit.commit_successful", 0, map[string]interface{}{
		"CommitTitle": selectedSuggestion.CommitTitle,
	}))
	return nil
}
