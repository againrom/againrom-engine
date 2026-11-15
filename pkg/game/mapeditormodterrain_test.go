package game

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// terrainPage is the editor with W open on a new mod the editor made in an
// empty folder from the map it holds.
func terrainPage(t *testing.T) (*mapInspectionEdits, string) {
	t.Helper()
	_, s := contextEditor(t)
	dir := filepath.Join(t.TempDir(), "authored-map")
	d, err := s.SetWorkspaceField("package", "folder", dir)
	if err != nil || d == nil {
		t.Fatalf("make %s: %v", dir, err)
	}
	return d.Edits.(*mapInspectionEdits), dir
}

// terrainField is the terrain page's field key; false when the page has none.
func terrainField(s *mapInspectionEdits, key string) (ui.InspectionField, bool) {
	for _, f := range s.WorkspaceFields(ui.ModTerrainScope) {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

// setTerrain makes one edit on the terrain page, failing the test on a refusal.
func setTerrain(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	if _, err := s.SetWorkspaceField(ui.ModTerrainScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
}

// importStrips imports column b's strip of group g for each b given, a strip
// of frames frames.
func importStrips(t *testing.T, s *mapInspectionEdits, g, frames int, columns ...int) {
	t.Helper()
	for _, b := range columns {
		source, _ := assetFile(t, fmt.Sprintf("strip%d.bmp", b), stripBMP(terrain.CellSize, frames))
		setTerrain(t, s, fmt.Sprintf("%d.strip.%d", g, b), source)
	}
}

// keysOf is m's keys in order.
func keysOf[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// groupBrush reports the brush offering mod group g.
func groupBrush(s *mapInspectionEdits, g int) bool {
	return slices.ContainsFunc(s.BrushTerrains(), func(b ui.BrushTerrain) bool { return b.Class == mapedit.GroupBrush(g) })
}

// The page offers the first free group from 13, adds it, and refuses a
// number outside 13..31, a taken number, a value the loader refuses and a
// strip the loader's strip rule refuses, each with no draft.
func TestTerrainPageAddsAGroupAndRefusesWhatTheLoaderRefuses(t *testing.T) {
	s, _ := terrainPage(t)
	if f, ok := terrainField(s, "new"); !ok || f.TextValue != "13" {
		t.Fatalf("the offered group %+v", f)
	}
	for _, n := range []string{"12", "32", "x"} {
		if _, err := s.SetWorkspaceField(ui.ModTerrainScope, "new", n); err == nil || !strings.Contains(err.Error(), "from 13 to 31") {
			t.Errorf("group %s: %v", n, err)
		}
	}
	setTerrain(t, s, "add", "")
	if f, ok := terrainField(s, "new"); !ok || f.TextValue != "14" {
		t.Fatalf("the next offered group %+v", f)
	}
	if _, err := s.SetWorkspaceField(ui.ModTerrainScope, "new", "13"); err == nil || !strings.Contains(err.Error(), "already has group 13") {
		t.Fatalf("a taken group: %v", err)
	}
	setTerrain(t, s, "13.name", "Glacier")
	setTerrain(t, s, "13.kind", "8")
	setTerrain(t, s, "13.minimap", "#A0C0E0")
	rows, err := mod.ParseTerrain("glacier", mod.TerrainFile, s.workspace.Project.Files[mod.TerrainFile])
	if g, ok := rows.Group(13); err != nil || !ok || g.Name != "Glacier" || g.Class != 8 || !g.HasMinimap || g.Minimap.B != 0xe0 {
		t.Fatalf("the draft rows %+v: %v", rows.Groups, err)
	}
	steps := len(s.undoWorkspace)
	for _, c := range []struct{ key, value, want string }{
		{"13.minimap", "blue", "#rrggbb"},
		{"13.name", strings.Repeat("x", 40), "1 to 32 bytes"},
		{"13.name", "", "1 to 32 bytes"},
		{"20.name", "Dune", "no field 20.name"},
	} {
		if _, err := s.SetWorkspaceField(ui.ModTerrainScope, c.key, c.value); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s = %q: %v, want %q", c.key, c.value, err, c.want)
		}
	}
	for _, c := range []struct {
		width, frames int
		want          string
	}{{33, 1, "33 px wide"}, {terrain.CellSize, terrain.MaxStripFrames + 1, "at most 14"}} {
		source, _ := assetFile(t, "bad.bmp", stripBMP(c.width, c.frames))
		_, err := s.SetWorkspaceField(ui.ModTerrainScope, "13.strip.0", source)
		if err == nil || !strings.Contains(err.Error(), "strip 1 of group 13") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("a strip %dx%d: %v, want %q", c.width, c.frames, err, c.want)
		}
	}
	if len(s.undoWorkspace) != steps || slices.ContainsFunc(keysOf(s.workspace.Project.Files), func(k string) bool { return strings.HasPrefix(k, mod.FilesDir+"/") }) {
		t.Fatalf("a refusal left a draft: %d steps, files %v", len(s.undoWorkspace)-steps, keysOf(s.workspace.Project.Files))
	}
	if f, ok := terrainField(s, "13.state"); !ok || !strings.Contains(f.TextValue, "0 of 4") {
		t.Fatalf("the state of a group with no strips %+v", f)
	}
}

// A group and its strips are draft steps of the one history: the brush
// offers the group once its four strips are in, before any save, and paints
// it; undo takes the last strip and the group out of the brush and redo
// brings them back. A second import of a strip asks first.
func TestTerrainPageImportsUndoesAndOffersTheBrushBeforeSave(t *testing.T) {
	s, dir := terrainPage(t)
	before := tree(t, dir)
	steps := len(s.undoWorkspace)
	setTerrain(t, s, "add", "")
	importStrips(t, s, 13, 2, 0, 1, 2)
	if groupBrush(s, 13) {
		t.Fatal("the brush offers a group with three strips")
	}
	importStrips(t, s, 13, 2, 3)
	if len(s.undoWorkspace) != steps+5 || !groupBrush(s, 13) {
		t.Fatalf("steps %d, want %d; the brush %+v", len(s.undoWorkspace)-steps, 5, s.BrushTerrains())
	}
	if f, ok := terrainField(s, "13.strip.3"); !ok || !strings.Contains(f.Meaning, "2") || len(f.Pick) != 1 {
		t.Fatalf("the held strip's field %+v", f)
	}
	if f, ok := terrainField(s, "13.state"); !ok || !strings.Contains(f.TextValue, "All four") {
		t.Fatalf("the state of a whole group %+v", f)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if groupBrush(s, 13) {
		t.Fatal("the brush keeps the group after the undo of its last strip")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if !groupBrush(s, 13) {
		t.Fatal("the redo did not bring the group back to the brush")
	}
	source, _ := assetFile(t, "again.bmp", stripBMP(terrain.CellSize, 3))
	_, err := s.SetWorkspaceField(ui.ModTerrainScope, "13.strip.0", source)
	var ask *ui.WorkspaceConfirmation
	if !errors.As(err, &ask) || ask.Action != modAssetReplaceKey {
		t.Fatalf("a held strip: %v", err)
	}
	if _, err := s.SetWorkspaceField("package", modAssetReplaceKey, "confirmed"); err != nil {
		t.Fatal(err)
	}
	if f, _ := terrainField(s, "13.strip.0"); !strings.Contains(f.Meaning, "3") {
		t.Fatalf("the replaced strip %+v", f)
	}
	cell := image.Point{X: 20, Y: 20}
	if _, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: mapedit.GroupBrush(13), Cells: []image.Point{cell}}); err != nil {
		t.Fatal(err)
	}
	m, err := s.model.Map()
	if err != nil || alm.TileGroup(m.Tiles[cell.Y*m.Width+cell.X]) != 13 {
		t.Fatalf("painted %v", err)
	}
	for range 7 {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := s.workspace.Project.Files[mod.TerrainFile]; ok || groupBrush(s, 13) {
		t.Fatalf("the undo of every step keeps the group: files %v", keysOf(s.workspace.Project.Files))
	}
	if !slices.Equal(keysOf(before), keysOf(tree(t, dir))) {
		t.Fatal("the page wrote into the mod folder before a save")
	}
}

// A mod made on the page alone saves through the project writer, keeping a
// hand comment of its terrain file; a group with fewer than four strips is
// refused at the save in the words of the check. The published mod passes
// the launch's check, needs no entry file, and a map painted with its group
// plays under it with the group's land kind.
func TestTerrainPageModSavesPublishesAndPlays(t *testing.T) {
	s, dir := terrainPage(t)
	made := tree(t, dir)
	setTerrain(t, s, "add", "")
	setTerrain(t, s, "13.name", "Glacier")
	setTerrain(t, s, "13.kind", "8")
	importStrips(t, s, 13, 2, 0, 1, 2, 3)
	setTerrain(t, s, "add", "")
	importStrips(t, s, 14, 1, 0, 1)

	whole := filepath.Join(t.TempDir(), "authored-map")
	for name, b := range made {
		writeFileAll(t, filepath.Join(whole, filepath.FromSlash(name)), []byte(b))
	}
	for name, b := range s.workspace.Project.Files {
		if b != nil {
			writeFileAll(t, filepath.Join(whole, filepath.FromSlash(name)), b)
		}
	}
	_, _, check := checkSnowField(t, whole)
	if check == nil {
		t.Fatal("the check takes a group with two strips")
	}
	if _, err := s.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), check.Error()) {
		t.Fatalf("the save of a group with two strips: %v, want the check's %q", err, check)
	}
	setTerrain(t, s, "14.remove", "")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	if saved[mod.EntryName] != made[mod.EntryName] {
		t.Fatalf("the save changed the entry file:\n%s", saved[mod.EntryName])
	}
	want := append(keysOf(made), mod.TerrainFile)
	for b := range 4 {
		want = append(want, mod.FilesDir+"/"+mod.CommonFiles+"/"+terrain.GroupStripPath(13, b))
	}
	slices.Sort(want)
	if got := keysOf(saved); !slices.Equal(got, want) {
		t.Fatalf("the saved folder %v, want %v", got, want)
	}

	// A comment written by hand stays through the page's next edit and save.
	rows := filepath.Join(dir, filepath.FromSlash(mod.TerrainFile))
	writeFileAll(t, rows, append([]byte("# made on the page\n"), saved[mod.TerrainFile]...))
	s = openInW(t, s, dir)
	setTerrain(t, s, "13.minimap", "#a0c0e0")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rows); !strings.HasPrefix(string(b), "# made on the page\n[[group]]\ngroup = 13\n") || !strings.Contains(string(b), "minimap = \"#a0c0e0\"") {
		t.Fatalf("the terrain file after the second save:\n%s", b)
	}

	out := t.TempDir()
	if _, err := s.SetWorkspaceField("package", "output", out); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("package", "publish", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(out, []string{"authored-map"})
	if err != nil || len(entries) != 1 {
		t.Fatal(entries, err)
	}
	res, err := modrt.Load(entries, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if g, ok := res.Terrain.Group(13); !ok || g.Class != 8 || len(res.Files.Files) != 4 {
		t.Fatalf("the published mod: rows %+v, %d files", res.Terrain.Groups, len(res.Files.Files))
	}
	held, _, err := checkSnowField(t, entries[0].Dir)
	if err != nil {
		t.Fatalf("the check refused the published mod: %v", err)
	}
	// The fixture front's campaign opens no town after the editor's mission,
	// so the packed mod plays on it without its mission rows; the mod's own
	// mission on a whole campaign is the release test's.
	f := modGroupCells(t, nil, held, groupCell{13, 20})
	layer := f.modFileLayer()
	layer.order = res.Files.Order
	if err := layer.serveInstallFiles(f.Archives, res.Files.Files); err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	res.Data.Missions = mod.MissionData{}
	if _, err := f.SetModData(res.Data); err != nil {
		t.Fatalf("the launch's check refused the published mod: %v", err)
	}
	app := f.App("terrain page group")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	m := f.live.mission.state.Map
	if alm.TileGroup(m.Tiles[20*m.Width+20]) != 13 {
		t.Fatalf("the mission's cell %#04x", m.Tiles[20*m.Width+20])
	}
	planes, present := f.live.world.SavedCellPlanes()
	if !present || planes.Static[20*256+20] != 1 {
		t.Fatal("the group's mountain kind did not reach the cell's planes")
	}
	for range 10 {
		f.live.tick()
	}
}

// stripBytes is the four strips of group g in the mod folder dir, by column;
// nil for a strip the folder lacks.
func stripBytes(dir string, g int) [4][]byte {
	var out [4][]byte
	for b := range 4 {
		out[b], _ = os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.FilesDir+"/"+mod.CommonFiles+"/"+terrain.GroupStripPath(g, b))))
	}
	return out
}

// wholeSavedGroup is a page with group 13 whole in the saved mod folder,
// strips of 2, 3, 4 and 5 frames, opened again from its folder.
func wholeSavedGroup(t *testing.T) (*mapInspectionEdits, string, [4][]byte) {
	t.Helper()
	s, dir := terrainPage(t)
	setTerrain(t, s, "add", "")
	for b := range 4 {
		importStrips(t, s, 13, 2+b, b)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	strips := stripBytes(dir, 13)
	for b, data := range strips {
		if data == nil {
			t.Fatalf("the save wrote no strip %d", b)
		}
	}
	// A mod opened from its folder holds its strips only as files there.
	return openInW(t, s, dir), dir, strips
}

// groupWhole fails the test unless the page reads group 13 whole, offers it
// to the brush, and a save writes the strips want.
func groupWhole(t *testing.T, s *mapInspectionEdits, dir string, want [4][]byte) {
	t.Helper()
	if f, ok := terrainField(s, "13.state"); !ok || !strings.Contains(f.TextValue, "All four") {
		t.Fatalf("the state of the group %+v", f)
	}
	if !groupBrush(s, 13) {
		t.Fatal("the brush does not offer the group")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatalf("the save after the undo: %v", err)
	}
	got := stripBytes(dir, 13)
	for b := range 4 {
		if !bytes.Equal(got[b], want[b]) {
			t.Errorf("strip %d: %d bytes in the folder, want %d", b, len(got[b]), len(want[b]))
		}
	}
}

// Removing a whole group, saving and undoing brings the group back whole
// with its strips' bytes, and the next save writes them back; the same after
// a redo and a second undo.
func TestTerrainPageUndoPastASaveRestoresARemovedGroup(t *testing.T) {
	s, dir, strips := wholeSavedGroup(t)
	setTerrain(t, s, "13.remove", "")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := stripBytes(dir, 13); slices.ContainsFunc(got[:], func(b []byte) bool { return b != nil }) {
		t.Fatalf("the save kept strips of the removed group")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	groupWhole(t, s, dir, strips)
	setTerrain(t, s, "13.remove", "")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	groupWhole(t, s, dir, strips)
}

// Replacing strip 2 of a whole group, saving and undoing shows the old strip
// and the next save writes the old bytes; the same after a redo and a second
// undo.
func TestTerrainPageUndoPastASaveRestoresAReplacedStrip(t *testing.T) {
	s, dir, strips := wholeSavedGroup(t)
	source, _ := assetFile(t, "new.bmp", stripBMP(terrain.CellSize, 7))
	if _, err := s.SetWorkspaceField(ui.ModTerrainScope, "13.strip.1", source); err == nil {
		t.Fatal("a held strip is replaced without asking")
	}
	if _, err := s.SetWorkspaceField("package", modAssetReplaceKey, "confirmed"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := stripBytes(dir, 13); bytes.Equal(got[1], strips[1]) {
		t.Fatal("the save kept the old strip")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if f, _ := terrainField(s, "13.strip.1"); !strings.Contains(f.Meaning, "3 frames") {
		t.Fatalf("the strip after the undo %+v", f)
	}
	groupWhole(t, s, dir, strips)
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	groupWhole(t, s, dir, strips)
}

// A minimap colour the loader refuses is refused naming its group and the
// loader's reason.
func TestTerrainPageRefusesAMinimapColourInTheLoadersWords(t *testing.T) {
	s, _ := terrainPage(t)
	setTerrain(t, s, "add", "")
	_, err := s.SetWorkspaceField(ui.ModTerrainScope, "13.minimap", "#abcdeg")
	if err == nil || !strings.Contains(err.Error(), "group 13") || !strings.Contains(err.Error(), `"#rrggbb"`) || strings.Contains(err.Error(), "ParseUint") {
		t.Fatalf("the refusal %v", err)
	}
}
