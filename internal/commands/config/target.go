package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/thomas-vilte/matecommit/internal/config"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

// resolveTargetConfig decides whether the local (repo) or global config
// should be used, based on the --local/--global flags — falling back to
// auto-detecting a repo config when neither flag is given — and loads (or
// creates) the local config file when needed. Commands that operate on
// either the local or global config (init, set) share this resolution.
func resolveTargetConfig(command *cli.Command, cfg *config.Config, t *i18n.Translations) (target *config.Config, useLocal bool, err error) {
	isLocalExplicit := command.Bool("local")
	isGlobalExplicit := command.Bool("global")

	useLocal = isLocalExplicit
	if !isGlobalExplicit && !isLocalExplicit {
		useLocal = config.GetRepoConfigPath() != ""
	}

	if !useLocal {
		return cfg, false, nil
	}

	localPath := config.GetRepoConfigPath()
	if localPath == "" {
		return nil, true, errors.New(t.GetMessage("config_local.not_in_repo", 0, nil))
	}

	localCfg, loadErr := config.LoadConfig(localPath)
	if loadErr != nil {
		if os.IsNotExist(loadErr) || errors.Is(loadErr, os.ErrNotExist) {
			localCfg, loadErr = config.CreateDefaultConfig(localPath)
			if loadErr != nil {
				return nil, true, fmt.Errorf("error creating local config: %w", loadErr)
			}
		} else {
			return nil, true, fmt.Errorf("error loading local config: %w", loadErr)
		}
	}

	return localCfg, true, nil
}
