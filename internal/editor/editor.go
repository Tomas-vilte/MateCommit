// Package editor resolves which text editor to use for interactive edits
// (config files, release notes, etc.), shared by the commands that need it.
package editor

import (
	"os"
	"os/exec"
)

// candidates is tried, in order, when neither $EDITOR nor $VISUAL is set.
var candidates = []string{"nano", "vim", "vi", "code", "emacs"}

// Resolve returns the user's preferred editor: $EDITOR, then $VISUAL, then
// the first of a list of common editors found on PATH. It returns
// ("", false) when none could be determined.
func Resolve() (string, bool) {
	if e := os.Getenv("EDITOR"); e != "" {
		return e, true
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e, true
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c, true
		}
	}
	return "", false
}
