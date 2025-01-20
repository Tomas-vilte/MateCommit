package config

import (
	"github.com/Tomas-vilte/MateCommit/internal/config"
	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

type ConfigCommandFactory struct {
}

func NewConfigCommandFactory() *ConfigCommandFactory {
	return &ConfigCommandFactory{}
}

func (f *ConfigCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   t.GetMessage("config_command_usage", 0, nil),
		Commands: []*cli.Command{
			f.newSetLangCommand(t, cfg),
			f.newShowCommand(t, cfg),
			f.newSetAPIKeyCommand(t, cfg),
		},
	}
}
