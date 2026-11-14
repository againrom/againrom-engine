package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// fixtureModsDir holds the mods the runtime's tests load.
var fixtureModsDir = filepath.Join("..", "modrt", "testdata", "mods")

// fixtureMods names every fixture mod folder.
func fixtureMods(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(fixtureModsDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) != 18 {
		t.Fatalf("%d fixture mods, want 18: %v", len(names), names)
	}
	return names
}

// copyFixture copies the fixture mod name into a new folder. A mod that
// declares missions gets the map and texts folder cmd/modmission would copy
// out of an install.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.CopyFS(dir, os.DirFS(filepath.Join(fixtureModsDir, name))); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(mod.MissionsFile))); err == nil {
		writeTestPlayInput(t, assembledMap(dir, "en", 10), synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"}))
		if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(assembledMission("en", 10).Pages("en"))), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// contextEditor is the workspace editor over a new map with an install
// table its mods' forks are built from.
func contextEditor(t *testing.T) (*MapInspector, *mapInspectionEdits) {
	t.Helper()
	f, e := workspaceFixture(t)
	withInstallTable(f.x)
	return f.x, e.Document().Edits.(*mapInspectionEdits)
}

// openInW opens the mod at dir the way W's folder field does.
func openInW(t *testing.T, s *mapInspectionEdits, dir string) *mapInspectionEdits {
	t.Helper()
	d, err := s.SetWorkspaceField("package", "folder-open", dir)
	if err != nil || d == nil {
		t.Fatalf("open %s: %v", dir, err)
	}
	return d.Edits.(*mapInspectionEdits)
}

// overviewLines is the lines of the overview section headed heading.
func overviewLines(page ui.ModOverviewPage, heading string) []string {
	for _, s := range page.Sections {
		if s.Heading == heading || strings.HasPrefix(s.Heading, heading+" (") {
			return s.Lines
		}
	}
	return nil
}

// W opens every fixture mod, whatever kinds it holds: a mod with an enabled
// mission opens that mission's map, any other over the map in the editor
// and on its overview. The overview's report is the lines -check prints
// for the mod, loaded the way -check loads it; a mod the run refuses shows
// the refusal instead.
func TestEditorOpensEveryFixtureMod(t *testing.T) {
	for _, name := range fixtureMods(t) {
		t.Run(name, func(t *testing.T) {
			x, s := contextEditor(t)
			dir := copyFixture(t, name)
			n := openInW(t, s, dir)
			if x.mod == nil || x.mod.dir != dir || x.mod.packed {
				t.Fatalf("mod %+v", x.mod)
			}
			want := s.source
			entry, enabled := x.mod.firstEnabled()
			if enabled {
				want = filepath.Join(dir, filepath.FromSlash(entry.Map))
			}
			page := n.ModOverview()
			if n.source != want || page.First == enabled {
				t.Fatalf("opened %s, want %s; overview first %v", n.source, want, page.First)
			}
			mods, err := mod.Resolve(filepath.Dir(dir), []string{name})
			if err != nil {
				t.Fatal(err)
			}
			report := []string{x.word("editor.mod.overview.none")}
			res, err := modrt.Load(mods, BaseID(InspectInstall(x.archives.Root)), nil, modrt.Options{})
			if err == nil {
				report = ModCheckLines(res)
			} else if issues := overviewLines(page, x.word("editor.mod.overview.issues")); !slices.Contains(issues, x.word("editor.mod.readonly")+" "+err.Error()) {
				t.Fatalf("refusals %q, want %q", issues, err)
			}
			if got := overviewLines(page, x.word("editor.mod.overview.report")); !slices.Equal(got, report) {
				t.Fatalf("report\n%q\nwant\n%q", got, report)
			}
			if got := overviewLines(page, x.word("editor.mod.overview.files")); !slices.Contains(got, mod.ManifestName) {
				t.Fatalf("files %q", got)
			}
		})
	}
}

// W shows the overview of a mod that holds only terrain first, its kinds
// and its -check report, and the window's key turns to the mod's fields.
func TestEditorShowsTheOverviewOfAModWithoutMissions(t *testing.T) {
	f, e := workspaceFixture(t)
	withInstallTable(f.x)
	if err := workspaceField(e, "package", 0, snowFieldMod(t, nil)); err != nil {
		t.Fatal(err)
	}
	x := f.x
	lines := strings.Join(e.PopupLines(), "\n")
	for _, want := range []string{x.word("editor.mod.overview.kinds"), "terrain: " + mod.TerrainFile, x.word("editor.mod.overview.report"), "1 mod(s) active on", x.word("editor.mod.overview.files")} {
		if !strings.Contains(lines, want) {
			t.Fatalf("overview lacks %q:\n%s", want, lines)
		}
	}
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Join(e.PopupLines(), "\n"); strings.Contains(lines, x.word("editor.mod.overview.report")) {
		t.Fatalf("the window's key kept the overview:\n%s", lines)
	}
}

// A packed mod opens read-only over the map in the editor, its context
// built as a folder's is; every W edit and Save refuses through the one
// read-only test and the packed file stays as it was. Opening another mod
// is no write.
func TestEditorOpensAPackedModReadOnly(t *testing.T) {
	x, s := contextEditor(t)
	_, raw, err := mod.Pack(snowFieldMod(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	packed := filepath.Join(t.TempDir(), mod.PackedName("snow-field"))
	if err := os.WriteFile(packed, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	n := openInW(t, s, packed)
	if !x.mod.packed || n.source != s.source {
		t.Fatalf("packed %v, opened %s", x.mod.packed, n.source)
	}
	if x.table.ModTerrain() == nil {
		t.Fatal("the packed mod's context holds no terrain groups")
	}
	if page := n.ModOverview(); !page.First || !slices.ContainsFunc(overviewLines(page, x.word("editor.mod.overview.issues")), func(l string) bool {
		return strings.HasPrefix(l, x.word("editor.mod.packed"))
	}) {
		t.Fatalf("overview %+v", page)
	}
	if got := n.ModTarget(); got != "snow-field ("+x.word("editor.mod.packedtarget")+")" {
		t.Fatalf("target %q", got)
	}
	refused := x.word("editor.mod.packed")
	copyTo := filepath.Join(t.TempDir(), "copy")
	for _, c := range []struct{ scope, key, value string }{
		{"package", "title", "Other"}, {"package", "folder", copyTo}, {"package", "save", ""},
		{"package", "publish", ""}, {"package", "add", ""}, {"package", "output", t.TempDir()},
		{"package", "source", "1"}, {"pages", "body", "text"}, {"metadata", "name", "x"},
	} {
		if _, err := n.SetWorkspaceField(c.scope, c.key, c.value); err == nil || !strings.Contains(err.Error(), refused) {
			t.Errorf("%s %s: %v", c.scope, c.key, err)
		}
	}
	if _, err := n.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), refused) {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(copyTo); !os.IsNotExist(err) {
		t.Fatalf("a refused copy made %s: %v", copyTo, err)
	}
	if after, err := os.ReadFile(packed); err != nil || !bytes.Equal(after, raw) {
		t.Fatal("the packed mod changed")
	}
	openInW(t, n, copyFixture(t, "east-town"))
	if x.mod.packed || x.mod.id != "east-town" {
		t.Fatalf("mod %+v", x.mod)
	}
}

// A mod that holds only terrain opens, and a stroke with its group paints
// the map in the editor.
func TestEditorPaintsTheGroupOfATerrainOnlyMod(t *testing.T) {
	_, s := contextEditor(t)
	n := openInW(t, s, snowFieldMod(t, nil))
	if !slices.ContainsFunc(n.BrushTerrains(), func(b ui.BrushTerrain) bool { return b.Class == mapedit.GroupBrush(13) }) {
		t.Fatalf("brush rows %+v", n.BrushTerrains())
	}
	cell := image.Point{X: 20, Y: 20}
	if _, err := n.Paint(ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: mapedit.GroupBrush(13), Cells: []image.Point{cell}}); err != nil {
		t.Fatal(err)
	}
	m, err := n.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	if w := m.Tiles[cell.Y*m.Width+cell.X]; alm.TileGroup(w) != 13 {
		t.Fatalf("painted word %#04x", w)
	}
}

// W's party equipment offers the items a mod adds, read from the mod's
// context; the install's table keeps none of them.
func TestEditorPartyEquipmentOffersAModItem(t *testing.T) {
	x, s := contextEditor(t)
	x.setEditorMod(nil)
	x.table = fixtureItemTable(10, 5, 5)
	dir := writeNamedEditorMod(t, t.TempDir(), "star-blade", "")
	writeModText(t, dir, "main.star", "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n")
	writeModText(t, dir, mod.ItemsFile, "[[weapon]]\nkey        = \"star\"\nname       = \"item.star\"\nbase       = \"Dagger\"\ndamage-min = 9\ndamage-max = 14\nrange      = 3\nprice      = 300\n")
	writeModText(t, dir, "text/en/strings.toml", "\"item.star\" = \"Star Blade\"\n")
	code := int64(data.ComposeItemCode(0, 1, 0, 5))
	x.archives.Containers.Overlay(graphicsPrefix+data.ItemIconPath(data.ItemCode(code)), func() ([]byte, error) { return []byte{1}, nil })
	install := x.table
	n := openInW(t, s, dir)
	if x.mod.contextIssue != "" {
		t.Fatal(x.mod.contextIssue)
	}
	if _, err := n.SetWorkspaceField("package", "landing", "1"); err != nil {
		t.Fatal(err)
	}
	var weapon *ui.InspectionField
	fields := n.WorkspaceFields("package")
	for i := range fields {
		if fields[i].Key == "party.0.weapon" {
			weapon = &fields[i]
		}
	}
	if weapon == nil {
		t.Fatal("no party weapon field")
	}
	if !slices.ContainsFunc(weapon.Choices, func(c ui.InspectionChoice) bool {
		return c.Value == code && strings.Contains(c.Label, "Star Blade")
	}) {
		t.Fatalf("weapon choices %+v", weapon.Choices)
	}
	if _, ok := install.Names.NameFor(data.ItemCode(code)); ok {
		t.Fatal("the install's table took the mod's item")
	}
}
