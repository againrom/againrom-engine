package game

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// noMapEditor is an editor on the synthetic install with no map open.
func noMapEditor(t *testing.T) (*MapInspector, *ui.MapEditor) {
	t.Helper()
	f := newTestPlayFixture(t)
	f.x.tiles = &terrain.Tileset{}
	f.x.font = &text.Font{Selector: 1}
	withInstallTable(f.x)
	e := f.x.Editor()
	e.Layout(1920, 1080)
	return f.x, e
}

// W opens the mod window in an editor with no map, and a mod that holds no
// mission opens there on its overview.
func TestEditorOpensAModWithNoMapOpen(t *testing.T) {
	x, e := noMapEditor(t)
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.ModWindowOpen() {
		t.Fatalf("W opened no mod window: %q", e.PopupLines())
	}
	dir := snowFieldMod(t, nil)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if e.Document() != nil {
		t.Fatal("a mod with no mission opened a map")
	}
	if x.mod == nil || x.mod.dir != dir || x.table.ModTerrain() == nil {
		t.Fatalf("the mod is not the one the editor works in: %+v", x.mod)
	}
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{x.word("editor.mod.overview.kinds"), "terrain: " + mod.TerrainFile, x.word("editor.mod.overview.report")} {
		if !strings.Contains(lines, want) {
			t.Fatalf("overview lacks %q:\n%s", want, lines)
		}
	}
	_ = e.ChromeImage()
	// What edits a map's own data refuses with no map.
	if err := e.Key("saveas"); err == nil || !strings.Contains(err.Error(), x.word("editor.mod.nomap")) {
		t.Fatalf("Save with no map: %v", err)
	}
	if e.Document() != nil || !e.ModWindowOpen() {
		t.Fatal("Save opened a map or closed the window")
	}
	// A second mod opens in place of the first.
	other := writeEditorMod(t, t.TempDir(), "")
	if err := workspaceField(e, "package", 0, other); err != nil {
		t.Fatal(err)
	}
	if x.mod.dir != other || x.table.ModTerrain() != nil {
		t.Fatalf("the second mod did not replace the first: %+v", x.mod)
	}
}

// A mod with a mission opens that mission's map from an editor with no map.
func TestEditorOpensAMissionModWithNoMapOpen(t *testing.T) {
	x, e := noMapEditor(t)
	dir := writeEditorMod(t, t.TempDir(), editorModMissions)
	writeMissionMap(t, dir)
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if e.Document() == nil || x.mod == nil || x.mod.dir != dir || !e.ModWindowOpen() {
		t.Fatalf("document %v mod %+v window %v", e.Document() != nil, x.mod, e.ModWindowOpen())
	}
}
