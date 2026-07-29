package editor

import "testing"

func TestResolve(t *testing.T) {
	t.Run("prefers EDITOR", func(t *testing.T) {
		t.Setenv("EDITOR", "my-editor")
		t.Setenv("VISUAL", "my-visual")

		ed, found := Resolve()
		if !found || ed != "my-editor" {
			t.Errorf("Resolve() = (%q, %v), want (\"my-editor\", true)", ed, found)
		}
	})

	t.Run("falls back to VISUAL when EDITOR is unset", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "my-visual")

		ed, found := Resolve()
		if !found || ed != "my-visual" {
			t.Errorf("Resolve() = (%q, %v), want (\"my-visual\", true)", ed, found)
		}
	})

	t.Run("falls back to PATH candidates when neither is set", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		t.Setenv("VISUAL", "")

		ed, found := Resolve()
		if !found {
			t.Skip("no candidate editor available on PATH in this environment")
		}
		if ed == "" {
			t.Error("Resolve() returned found=true with an empty editor name")
		}
	})
}
