package completion

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thomas-vilte/matecommit/internal/i18n"
	"github.com/urfave/cli/v3"
)

func generateBashScript(t *testing.T) string {
	t.Helper()

	translations, err := i18n.NewTranslations("en", "../../i18n/locales")
	require.NoError(t, err)

	app := &cli.Command{
		Name:     "test",
		Commands: []*cli.Command{NewCompletionCommand(translations)},
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	require.NoError(t, app.Run(context.Background(), []string{"test", "completion", "bash"}))
	require.NoError(t, w.Close())

	data := make([]byte, 0, 4096)
	buf := make([]byte, 4096)
	for {
		n, readErr := r.Read(buf)
		data = append(data, buf[:n]...)
		if readErr != nil {
			break
		}
	}

	return string(data)
}

// TestBashCompletion_StripsDescriptions verifies the generated bash
// completion function against a real bash process. urfave/cli's
// --generate-shell-completion emits "name:description" lines, and compgen
// -W splits its wordlist on whitespace — so without stripping the
// description first, each of its words leaks in as a bogus extra
// candidate instead of just the real command/flag name.
func TestBashCompletion_StripsDescriptions(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	script := generateBashScript(t)

	dir := t.TempDir()
	scriptFile := filepath.Join(dir, "completion.bash")
	require.NoError(t, os.WriteFile(scriptFile, []byte(script), 0600))

	fakeBin := filepath.Join(dir, "matecommit")
	fakeBinContent := "#!/bin/sh\n" +
		"echo 'config:Manage configuration'\n" +
		"echo 'suggest:Generate commit message suggestions'\n"
	require.NoError(t, os.WriteFile(fakeBin, []byte(fakeBinContent), 0700))

	driver := `
source "` + scriptFile + `"
COMP_WORDS=(matecommit conf)
COMP_CWORD=1
_mate_commit_bash_autocomplete
printf '%s\n' "${COMPREPLY[@]}"
`
	cmd := exec.Command("bash", "-c", driver)
	cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	lines := strings.Fields(strings.TrimSpace(string(out)))
	assert.Equal(t, []string{"config"}, lines,
		"description words must not leak into the completion candidates")
}
