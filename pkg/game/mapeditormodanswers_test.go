package game

import (
	"slices"
	"strings"
	"testing"
)

// Answers 6 and 7: the overview says what the mod changes as one count
// line per page, the check in a modder's words, and folds the game's own
// check lines under Check details until a press opens them.
func TestOverviewCountsPagesAndFoldsTheGamesCheck(t *testing.T) {
	f, e := workspaceFixture(t)
	withInstallTable(f.x)
	if err := workspaceField(e, "package", 0, snowFieldMod(t, nil)); err != nil {
		t.Fatal(err)
	}
	x := f.x
	lines := e.PopupLines()
	// A count line is a link: "> Terrain: 2" opens the Terrain page.
	for _, want := range []string{"> " + x.word("editor.mod.terrain") + ": 2", x.word("editor.mod.overview.ok"), x.word("editor.mod.overview.more")} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, want) }) {
			t.Errorf("the overview lacks %q:\n%s", want, strings.Join(lines, "\n"))
		}
	}
	for _, gone := range []string{"data/terrain.toml", "mod(s) active", "mod snow-field"} {
		if slices.ContainsFunc(lines, func(l string) bool { return strings.Contains(l, gone) }) {
			t.Errorf("the overview shows %q unfolded:\n%s", gone, strings.Join(lines, "\n"))
		}
	}
}
