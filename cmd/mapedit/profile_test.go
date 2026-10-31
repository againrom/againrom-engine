package main

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/ui"
)

// The editor reads its side panels' state from the player profile the MCP
// record lives in, and a panel key writes the new state back there. The
// file here has the earlier open field; the write has the minimized one.
func TestPanelStateLivesInTheEditorProfile(t *testing.T) {
	profile := t.TempDir()
	t.Setenv("AGAINROM_PROFILE", profile)
	file := filepath.Join(profile, ui.PanelStateFile)
	if err := os.WriteFile(file, []byte(`{"left":{"open":false,"width":300},"right":{"open":false,"width":520}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, nil)
	e.Layout(1280, 800)
	loadPanelProfile(e, "", "", t.TempDir())
	if err := e.Key("objectspanel"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if want := `{"left":{"minimized":false,"width":300},"right":{"minimized":true,"width":520}}` + "\n"; err != nil || string(got) != want {
		t.Fatalf("panel state %q, %v; want %q", got, err, want)
	}
	// A profile that cannot be resolved leaves the defaults and writes nothing.
	t.Setenv("AGAINROM_PROFILE", "relative")
	e = ui.NewMapEditor(nil, nil, nil)
	e.Layout(1280, 800)
	loadPanelProfile(e, "", "", t.TempDir())
	_ = e.Key("objectspanel")
	if _, err := os.Stat(filepath.Join("relative", ui.PanelStateFile)); err == nil {
		t.Fatal("an unresolved profile was written")
	}
}
