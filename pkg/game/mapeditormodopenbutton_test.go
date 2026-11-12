package game

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/pathdialog"
	"againrom/pkg/ui"
)

// authoredMod is a mod the editor wrote through W's folder field, in a new
// folder, with the map in the editor as its mission.
func authoredMod(t *testing.T) string {
	t.Helper()
	_, e := workspaceFixture(t)
	folder := filepath.Join(t.TempDir(), "authored-map")
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	return folder
}

// openModByButton opens W in a fresh editor and presses its Open mod button;
// the host's path window answers path.
func openModByButton(t *testing.T, path string) (*ui.MapEditor, *MapInspector, pathdialog.Request) {
	t.Helper()
	f, e := workspaceFixture(t)
	called := installInspectionDialog(e, t.TempDir(), pathdialog.Result{Path: path})
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("open") {
		t.Fatal("W has no Open mod button")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	return e, f.x, finishInspectionDialog(t, e, called, pathdialog.OpenFile)
}

// W's Open mod button asks the host's path window for a mod.toml or a packed
// mod-<id>.res, opens the chosen mod through the folder field's route and
// shows its manifest. Saving it with no edit writes every file as it was.
func TestModWindowOpenButtonOpensAModAndASaveWithoutEditKeepsItsBytes(t *testing.T) {
	folder := authoredMod(t)
	before := modFolderBytes(t, folder)
	e, x, request := openModByButton(t, filepath.Join(folder, mod.ManifestName))
	if len(request.Filters) != 1 || !slices.Equal(request.Filters[0].Extensions, []string{"toml", "res"}) || request.Title != x.word("editor.mod.openmod") {
		t.Fatalf("request %+v", request)
	}
	if x.mod == nil || x.mod.dir != folder || x.mod.packed || !e.ModWindowOpen() {
		t.Fatalf("opened %+v, window %v", x.mod, e.ModWindowOpen())
	}
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{x.word("editor.mod.id"), "authored-map", x.word("editor.mod.version"), x.word("editor.mod.applies-to")} {
		if !strings.Contains(lines, want) {
			t.Fatalf("W lacks %q after the open:\n%s", want, lines)
		}
	}
	if !e.FocusModField("save") {
		t.Fatal("W has no Save mod button")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Join(e.PopupLines(), "\n"); !strings.Contains(lines, x.word("editor.mod.savecomplete")) && !strings.Contains(lines, x.word("editor.mod.saved")) {
		t.Fatalf("W does not say the mod was saved:\n%s", lines)
	}
	after := modFolderBytes(t, folder)
	if !maps.Equal(before, after) {
		t.Fatalf("a save with no edit changed the mod: before %v, after %v", slices.Sorted(maps.Keys(before)), slices.Sorted(maps.Keys(after)))
	}
}

// A packed mod-<id>.res chosen with Open mod opens read-only and stays as it
// was.
func TestModWindowOpenButtonOpensAPackedModReadOnly(t *testing.T) {
	_, raw, err := mod.Pack(authoredMod(t))
	if err != nil {
		t.Fatal(err)
	}
	packed := filepath.Join(t.TempDir(), mod.PackedName("authored-map"))
	if err := os.WriteFile(packed, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	e, x, _ := openModByButton(t, packed)
	if x.mod == nil || !x.mod.packed || x.modReadOnly() == nil || !e.ModWindowOpen() {
		t.Fatalf("opened %+v", x.mod)
	}
	if !strings.Contains(strings.Join(e.PopupLines(), "\n"), x.word("editor.mod.packed")) {
		t.Fatalf("W does not say the mod is packed:\n%s", strings.Join(e.PopupLines(), "\n"))
	}
	if b, err := os.ReadFile(packed); err != nil || !bytes.Equal(b, raw) {
		t.Fatal("the packed mod changed")
	}
}
