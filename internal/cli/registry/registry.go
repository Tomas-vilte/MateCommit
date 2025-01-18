package registry

import (
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/config"
	"github.com/Tomas-vilte/MateCommit/internal/cli/command/suggest"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type CommandRegistry struct {
	config *cfg.Config
	t      *i18n.Translations
}

func NewCommandRegistry(cfg *cfg.Config, t *i18n.Translations) *CommandRegistry {
	return &CommandRegistry{
		config: cfg,
		t:      t,
	}
}

func (r *CommandRegistry) RegisterCommands() []*cli.Command {
	return []*cli.Command{
		suggest.NewCommand(r.config, r.t),
		config.NewCommand(r.t, r.config),
	}
}
