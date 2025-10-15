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

func (c *ConfigCommandFactory) CreateCommand(t *i18n.Translations, cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   t.GetMessage("config_command_usage", 0, nil),
		Commands: []*cli.Command{
			c.newShowCommand(t, cfg),
			c.newInitCommand(t, cfg),
			c.newEditCommand(t, cfg),
		},
	}
}
