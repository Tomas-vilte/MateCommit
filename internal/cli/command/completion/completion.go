package completion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tomas-vilte/MateCommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

const bashCompletionScript = `#! /bin/bash

_mate_commit_bash_autocomplete() {
  if [[ "${COMP_WORDS[0]}" != "source" ]]; then
    local cur opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    
    # Construct the command line with previous words and append the completion flag
    # We strip the current word being completed (index COMP_CWORD) to ask for suggestions based on the context so far
    local cmd_context=("${COMP_WORDS[@]:0:$COMP_CWORD}")
    opts=$( "${cmd_context[@]}" --generate-shell-completion )
    
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi
}

complete -o bashdefault -o default -o nospace -F _mate_commit_bash_autocomplete matecommit
`

const zshCompletionScript = `#compdef matecommit

_matecommit() {
  local -a opts
  # Zsh array slicing: 1 to CURRENT-1 (all words before the one under cursor)
  local cmd_context=("${(@)words[1,$CURRENT-1]}")
  opts=("${(@f)$("${cmd_context[@]}" --generate-shell-completion)}")
  _describe 'values' opts
}

compdef _matecommit matecommit
`

const installInfo = `
# MateCommit Shell Completion
if command -v matecommit >/dev/null 2>&1; then
	source <(matecommit completion %s)
fi
`

func NewCompletionCommand(t *i18n.Translations) *cli.Command {
	return &cli.Command{
		Name:        "completion",
		Usage:       t.GetMessage("completion.command_usage", 0, nil),
		Description: t.GetMessage("completion.command_description", 0, nil),
		Commands: []*cli.Command{
			{
				Name:  "bash",
				Usage: t.GetMessage("completion.bash_usage", 0, nil),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Print(bashCompletionScript)
					return nil
				},
			},
			{
				Name:  "zsh",
				Usage: t.GetMessage("completion.zsh_usage", 0, nil),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Print(zshCompletionScript)
					return nil
				},
			},
			{
				Name:  "install",
				Usage: t.GetMessage("completion.install_usage", 0, nil),
				Action: func(ctx context.Context, cmd *cli.Command) error {
					shell := os.Getenv("SHELL")
					home, err := os.UserHomeDir()
					if err != nil {
						return fmt.Errorf("%s", t.GetMessage("completion.error_home_dir", 0, map[string]interface{}{"Error": err.Error()}))
					}

					var configFile string
					var shellName string

					if strings.Contains(shell, "zsh") {
						configFile = filepath.Join(home, ".zshrc")
						shellName = "zsh"
					} else if strings.Contains(shell, "bash") {
						configFile = filepath.Join(home, ".bashrc")
						shellName = "bash"
					} else {
						return fmt.Errorf("%s", t.GetMessage("completion.error_unsupported_shell", 0, map[string]interface{}{"Shell": shell}))
					}

					content := fmt.Sprintf(installInfo, shellName)

					fileContent, err := os.ReadFile(configFile)
					if err == nil && strings.Contains(string(fileContent), "# MateCommit Shell Completion") {
						fmt.Println(t.GetMessage("completion.already_installed", 0, map[string]interface{}{"File": configFile}))
						fmt.Println(t.GetMessage("completion.restart_shell", 0, nil))
						fmt.Printf("  source %s\n", configFile)
						return nil
					}

					f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
					if err != nil {
						return fmt.Errorf("%s", t.GetMessage("completion.error_open_config", 0, map[string]interface{}{"Error": err.Error()}))
					}
					defer func() {
						if err := f.Close(); err != nil {
							return
						}
					}()

					if _, err := f.WriteString(content); err != nil {
						return fmt.Errorf("%s", t.GetMessage("completion.error_write_config", 0, map[string]interface{}{"Error": err.Error()}))
					}

					fmt.Println(t.GetMessage("completion.installed_success", 0, map[string]interface{}{"File": configFile}))
					fmt.Println(t.GetMessage("completion.restart_shell", 0, nil))
					fmt.Printf("  source %s\n", configFile)

					return nil
				},
			},
		},
	}
}
