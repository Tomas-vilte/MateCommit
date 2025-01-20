package registry

import (
	"fmt"
	cfg "github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type CommandFactory interface {
	CreateCommand(t *i18n.Translations, cfg *cfg.Config) *cli.Command
}

type Registry struct {
	factories map[string]CommandFactory
	config    *cfg.Config
	t         *i18n.Translations
}

func NewRegistry(cfg *cfg.Config, t *i18n.Translations) *Registry {
	return &Registry{
		factories: make(map[string]CommandFactory),
		config:    cfg,
		t:         t,
	}
}

func (r *Registry) Register(name string, factory CommandFactory) error {
	if _, exists := r.factories[name]; exists {
		return fmt.Errorf(r.t.GetMessage("factory_already_registered", 0, map[string]interface{}{
			"FactoryName": name,
		}))
	}
	r.factories[name] = factory
	return nil
}

func (r *Registry) CreateCommands() []*cli.Command {
	commands := make([]*cli.Command, 0, len(r.factories))
	for _, factory := range r.factories {
		commands = append(commands, factory.CreateCommand(r.t, r.config))
	}
	return commands
}
