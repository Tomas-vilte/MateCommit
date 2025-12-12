package suggests_commits

import (
	"context"
	"fmt"

	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/domain/models"
	"github.com/Tomas-vilte/MateCommit/internal/domain/ports"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v3"
)

type SuggestCommandFactory struct {
	commitService ports.CommitService
	commitHandler ports.CommitHandler
}

func NewSuggestCommandFactory(commitService ports.CommitService, commitHandler ports.CommitHandler) *SuggestCommandFactory {
	return &SuggestCommandFactory{
		commitService: commitService,
		commitHandler: commitHandler,
	}
}

func (f *SuggestCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:        "suggest",
		Aliases:     []string{"s"},
		Usage:       t.GetMessage("suggest_command_usage", 0, nil),
		Description: t.GetMessage("suggest_command_description", 0, nil),
		Flags:       f.createFlags(cfg, t),
		Action:      f.createAction(cfg, t),
	}
}

func (f *SuggestCommandFactory) createFlags(cfg *config.Config, t *i18n.Translations) []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "count",
			Aliases: []string{"n"},
			Value:   cfg.SuggestionsCount,
			Usage:   t.GetMessage("suggest_count_flag_usage", 0, nil),
		},
		&cli.StringFlag{
			Name:    "lang",
			Aliases: []string{"l"},
			Usage:   t.GetMessage("suggest_lang_flag_usage", 0, nil),
			Value:   cfg.Language,
		},
		&cli.BoolFlag{
			Name:    "no-emoji",
			Aliases: []string{"ne"},
			Value:   cfg.UseEmoji,
			Usage:   t.GetMessage("suggest_no_emoji_flag_usage", 0, nil),
		},
		&cli.IntFlag{
			Name:    "issue",
			Aliases: []string{"i"},
			Usage:   t.GetMessage("suggest_issue_flag_usage", 0, nil),
			Value:   0,
		},
	}
}

func (f *SuggestCommandFactory) createAction(cfg *config.Config, t *i18n.Translations) cli.ActionFunc {
	return func(ctx context.Context, command *cli.Command) error {
		emojiFlag := command.Bool("no-emoji")
		if emojiFlag {
			cfg.UseEmoji = false
		} else {
			cfg.UseEmoji = true
		}
		count := command.Int("count")
		if count < 1 || count > 10 {
			msg := t.GetMessage("invalid_suggestions_count", 0, map[string]interface{}{
				"Min": 1,
				"Max": 10,
			})
			return fmt.Errorf("%s", msg)
		}

		cfg.Language = command.String("lang")

		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("error al guardar la configuración: %w", err)
		}

		issueNumber := command.Int("issue")

		p := tea.NewProgram(initialModel(f.commitService, ctx, count, issueNumber, t))
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running spinner: %w", err)
		}

		model := m.(model)
		if model.err != nil {
			msg := t.GetMessage("suggestion_generation_error", 0, map[string]interface{}{"Error": model.err})
			return fmt.Errorf("%s", msg)
		}

		if len(model.suggestions) == 0 {
			// Should verify if error was nil but suggestions empty (handled by service usually returning error if empty?)
			// Service returns error "no suggestions" if empty.
			return nil
		}

		return f.commitHandler.HandleSuggestions(ctx, model.suggestions)
	}
}

// Bubble Tea Model for Spinner

type model struct {
	commitService ports.CommitService
	ctx           context.Context
	count         int
	issueNumber   int
	trans         *i18n.Translations

	spinner     spinner.Model
	loading     bool
	suggestions []models.CommitSuggestion
	err         error
}

type suggestionsMsg []models.CommitSuggestion
type errMsg error

func initialModel(cs ports.CommitService, ctx context.Context, count, issue int, t *i18n.Translations) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		commitService: cs,
		ctx:           ctx,
		count:         count,
		issueNumber:   issue,
		trans:         t,
		spinner:       s,
		loading:       true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchSuggestions)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case suggestionsMsg:
		m.suggestions = msg
		m.loading = false
		return m, tea.Quit
	case errMsg:
		m.err = msg
		m.loading = false
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("cancelled by user")
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return ""
	}
	if m.loading {
		msg := m.trans.GetMessage("analyzing_changes", 0, nil)
		if m.issueNumber > 0 {
			msg = m.trans.GetMessage("issue_including_context", 0, map[string]interface{}{
				"Number": m.issueNumber,
			})
		}
		return fmt.Sprintf("\n %s %s\n\n", m.spinner.View(), msg)
	}
	return ""
}

func (m model) fetchSuggestions() tea.Msg {
	var suggestions []models.CommitSuggestion
	var err error

	if m.issueNumber > 0 {
		suggestions, err = m.commitService.GenerateSuggestionsWithIssue(m.ctx, m.count, m.issueNumber)
	} else {
		suggestions, err = m.commitService.GenerateSuggestions(m.ctx, m.count)
	}

	if err != nil {
		return errMsg(err)
	}
	return suggestionsMsg(suggestions)
}
