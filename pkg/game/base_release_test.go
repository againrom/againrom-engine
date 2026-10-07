package game

import (
	"os"
	"strings"
	"testing"

	"againrom/pkg/base"
)

// TestReleaseBaseProfileMatchesTheInstall detects the lawful install named by
// AGAINROM_ASSETS: its main archive is an exact build of the profile its
// language names, the front end carries that profile, and the profile states no
// limit, so mods, the new game and the generation screen are as before.
func TestReleaseBaseProfileMatchesTheInstall(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: base detection needs a lawful install")
	}
	info := InspectInstall(root)
	want := map[string]string{"english": base.ROM1EN, "russian": base.ROM1RU}[info.Language]
	if want == "" {
		t.Fatalf("install language %q is neither english nor russian", info.Language)
	}
	if !info.Valid() || info.Base.ID() != want || !info.Base.Exact {
		t.Fatalf("detected %s (valid %v), want an exact %s", info.Base, info.Valid(), want)
	}
	if BaseID(info) != want {
		t.Fatalf("BaseID = %q, want %q", BaseID(info), want)
	}
	p := info.Base.Profile
	if p.Limits.NoCharacterGeneration || p.Limits.OriginalSaveRefusal != "" || len(p.Limits.Notes) != 0 || p.Mission() != base.DefaultFirstMission {
		t.Fatalf("a release profile states a limit: %+v", p.Limits)
	}
	f := releaseFront(t)
	if f.Base().ID() != want || f.ChargenAssets == nil || f.directNewGame() {
		t.Fatalf("front end base %s, generation art %v", f.Base(), f.ChargenAssets != nil)
	}
	lines := f.BaseLines()
	if len(lines) != 1 || !strings.Contains(lines[0], "base "+want+" (") || !strings.Contains(lines[0], "exact build") {
		t.Fatalf("base lines = %q", lines)
	}
}
