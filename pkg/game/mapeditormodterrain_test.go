package game

import (
	"bytes"
	"image"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/internal/witness"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/bmp"
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

// squaresPicture is a test-made picture of across by down squares of 32
// pixels, every pixel a colour of its own.
func squaresPicture(across, down int) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, across*terrain.CellSize, down*terrain.CellSize))
	for y := range pic.Bounds().Dy() {
		for x := range pic.Bounds().Dx() {
			pic.SetRGBA(x, y, color.RGBA{uint8(x * 2), uint8(y * 3), uint8(x + y), 0xff})
		}
	}
	return pic
}

// groundPNG is the host file name holding pic as a PNG.
func groundPNG(t *testing.T, name string, pic image.Image) string {
	t.Helper()
	source, _ := assetFile(t, name, witness.Bytes(t, pic))
	return source
}

// keysOf is m's keys in order.
func keysOf[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// groupBrush reports the brush offering mod group g.
func groupBrush(s *mapInspectionEdits, g int) bool {
	return slices.ContainsFunc(s.BrushTerrains(), func(b ui.BrushTerrain) bool { return b.Class == mapedit.GroupBrush(g) })
}

// near reports a colour within a few steps of want on each channel.
func near(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	d := func(v uint32, w uint8) int { return max(int(v>>8)-int(w), int(w)-int(v>>8)) }
	return max(d(r, want.R), d(g, want.G), d(b, want.B)) <= 24
}

// Add ground takes one picture of 3 by 2 squares in more than 256 colours:
// four 8-bit strips the bitmap decoder reads back, 6 frames in all cut in
// reading order, the first free group, one row of land kind 1 with the
// picture's name and average colour, as one step. The page lists the game's
// ten grounds with pictures and no group number. The kind choice is a step;
// a ground the open map paints is refused deletion naming the map; Delete
// takes the row and its strips; Undo restores each step; the saved mod loads
// with the ground.
func TestTerrainPageAddsAGroundFromOnePicture(t *testing.T) {
	s, dir := terrainPage(t)
	before := tree(t, dir)
	// The grid opens with the game's grounds in the brush's order; the
	// fixture's tileset draws some of them, and each it draws is the cell's
	// picture with every variation as the cell's hover pictures.
	game, swatches := 0, s.inspector.terrainSwatches()
	grid, _ := terrainField(s, terrainGridKey)
	for i, c := range grid.Choices {
		if want := int64(-i - 1); i < len(swatches) && c.Value == want && c.Label != "" && (grid.Icons[want] == nil) == (swatches[i] == nil) &&
			len(grid.Variations[want]) <= len(mapedit.Variants(mapedit.Palette[i].Class)) {
			game++
		}
	}
	add, ok := terrainField(s, "newground")
	if game != len(mapedit.Palette) || !ok || !add.Action || !slices.Equal(add.Pick, []string{"png", "bmp"}) || add.Meaning == "" {
		t.Fatalf("the empty page: %d game grounds, add %+v", game, add)
	}
	pic := squaresPicture(3, 2)
	var sum [3]int
	for i := 0; i < len(pic.Pix); i += 4 {
		sum[0], sum[1], sum[2] = sum[0]+int(pic.Pix[i]), sum[1]+int(pic.Pix[i+1]), sum[2]+int(pic.Pix[i+2])
	}
	n := len(pic.Pix) / 4
	average := color.RGBA{uint8((sum[0] + n/2) / n), uint8((sum[1] + n/2) / n), uint8((sum[2] + n/2) / n), 0xff}
	steps := len(s.undoWorkspace)
	setTerrain(t, s, "newground", groundPNG(t, "Glacier.png", pic))
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("the add took %d steps", len(s.undoWorkspace)-steps)
	}
	files := s.workspace.Project.Files
	frames := 0
	var palette color.Palette
	for b, want := range []int{2, 2, 1, 1} {
		data := files[modStripFile(13, b)]
		img, err := bmp.DecodePaletted(data)
		if err != nil || len(data) < 30 || data[28] != 8 {
			t.Fatalf("strip %d: %v", b, err)
		}
		strip, err := terrain.ReadStrip(data)
		if err != nil || len(strip.SubCells) != want {
			t.Fatalf("strip %d: %v, want %d frames", b, err, want)
		}
		frames += len(strip.SubCells)
		if palette == nil {
			palette = img.Palette
		} else if !slices.Equal(palette, img.Palette) || len(palette) > 256 {
			t.Fatalf("strip %d has a colour table of its own", b)
		}
		// Square i is strip i mod 4, frame i / 4: its top-left pixel.
		for k := range want {
			i := b + 4*k
			at := image.Pt(i%3*terrain.CellSize, i/3*terrain.CellSize)
			if c := strip.SubCells[k].At(0, 0); !near(c, pic.RGBAAt(at.X, at.Y)) {
				t.Errorf("strip %d frame %d starts %v, want square %d's %v", b, k, c, i, pic.RGBAAt(at.X, at.Y))
			}
		}
	}
	rows, err := mod.ParseTerrain("glacier", mod.TerrainFile, files[mod.TerrainFile])
	if g, ok := rows.Group(13); err != nil || frames != 6 || len(rows.Groups) != 1 || !ok || g.Name != "Glacier" || g.Class != 1 || !g.HasMinimap || g.Minimap != average {
		t.Fatalf("%d frames; rows %+v, want average %v: %v", frames, rows.Groups, average, err)
	}
	if !groupBrush(s, 13) {
		t.Fatal("the brush does not offer the new ground")
	}
	for _, f := range s.WorkspaceFields(ui.ModTerrainScope) {
		if strings.Contains(f.Label, "13") || strings.Contains(f.Group, "13") {
			t.Fatalf("a field shows the group number: %+v", f)
		}
	}
	// The new ground is the grid's last cell, pressed, named under its
	// picture, with its six squares as its variations; its controls stand
	// above the grid under its name.
	grid, _ = terrainField(s, terrainGridKey)
	last := grid.Choices[len(grid.Choices)-1]
	if f, ok := terrainField(s, "13.name"); !ok || f.Group != "Glacier" || last.Value != 13 || last.Label != "Glacier" || grid.Value != 13 ||
		grid.Icons[13] == nil || len(grid.Variations[13]) != 6 || len(grid.Choices) != len(mapedit.Palette)+1 {
		t.Fatalf("the ground's cell %+v of %d, %d variations, name %+v", last, len(grid.Choices), len(grid.Variations[13]), f)
	}
	if _, err := s.SetWorkspaceField(ui.ModTerrainScope, terrainGridKey, "-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := terrainField(s, "13.name"); ok || len(s.undoWorkspace) != steps+1 {
		t.Fatal("a press on a game ground keeps the mod ground's controls or makes a draft step")
	}
	if _, err := s.SetWorkspaceField(ui.ModTerrainScope, terrainGridKey, "13"); err != nil {
		t.Fatal(err)
	}
	setTerrain(t, s, "13.kind", "9")
	rows, _ = mod.ParseTerrain("glacier", mod.TerrainFile, s.workspace.Project.Files[mod.TerrainFile])
	if g, _ := rows.Group(13); g.Class != 9 || len(s.undoWorkspace) != steps+2 {
		t.Fatalf("the kind: %+v, %d steps", g, len(s.undoWorkspace)-steps)
	}

	cell := image.Point{X: 20, Y: 20}
	if _, err := s.Paint(ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: mapedit.GroupBrush(13), Cells: []image.Point{cell}}); err != nil {
		t.Fatal(err)
	}
	_, err = s.SetWorkspaceField(ui.ModTerrainScope, "13.remove", "")
	if own := s.ownMap(); own == "" || err == nil || !strings.Contains(err.Error(), "Glacier") || !strings.Contains(err.Error(), own) {
		t.Fatalf("the delete of a painted ground: %v, want the map %q", err, own)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	setTerrain(t, s, "13.remove", "")
	strip := func(k string) bool {
		return s.workspace.Project.Files[k] != nil && strings.HasPrefix(k, mod.FilesDir+"/")
	}
	if _, ok := s.workspace.Project.Files[mod.TerrainFile]; ok || groupBrush(s, 13) || slices.ContainsFunc(keysOf(s.workspace.Project.Files), strip) {
		t.Fatalf("the delete left files %v", keysOf(s.workspace.Project.Files))
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if f, ok := terrainField(s, "13.kind"); !ok || f.Value != 9 || !groupBrush(s, 13) {
		t.Fatalf("the undo of the delete: %+v", f)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if f, _ := terrainField(s, "13.kind"); f.Value != 1 {
		t.Fatalf("the undo of the kind: %+v", f)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := terrainField(s, "13.name"); ok || groupBrush(s, 13) || len(s.undoWorkspace) != steps {
		t.Fatal("the undo of the add keeps the ground")
	}
	if !slices.Equal(keysOf(before), keysOf(tree(t, dir))) {
		t.Fatal("the page wrote into the mod folder before a save")
	}
	for range 2 {
		if _, err := s.Redo(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := checkSnowField(t, dir)
	g, ok := loaded.Group(13)
	if err != nil || !ok || g.Class != 9 || g.Frames != [4]int{2, 2, 1, 1} {
		t.Fatalf("the saved mod loads %+v: %v", loaded.Groups, err)
	}
}

// A picture the editor cannot use is refused in the page's words with no
// draft: a size that is not whole squares, more than 56 squares, bytes that
// are no picture, no file. A BMP of one square makes four strips of it.
func TestTerrainPageRefusesAPictureItCannotCut(t *testing.T) {
	s, _ := terrainPage(t)
	steps := len(s.undoWorkspace)
	notPicture, _ := assetFile(t, "notes.png", []byte("not a picture"))
	for _, c := range []struct{ source, want string }{
		{groundPNG(t, "wide.png", image.NewRGBA(image.Rect(0, 0, 50, 32))), "50 by 32 pixels"},
		{groundPNG(t, "many.png", squaresPicture(8, 8)), "64 squares"},
		{notPicture, "cannot read this picture"},
		{"", "pick the ground's picture"},
	} {
		if _, err := s.SetWorkspaceField(ui.ModTerrainScope, "newground", c.source); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.source, err, c.want)
		}
	}
	if len(s.undoWorkspace) != steps {
		t.Fatalf("a refusal left %d steps", len(s.undoWorkspace)-steps)
	}
	one, _ := assetFile(t, "Moss.bmp", synth.BMP24(squaresPicture(1, 1)))
	setTerrain(t, s, "newground", one)
	for b := range 4 {
		strip, err := terrain.ReadStrip(s.workspace.Project.Files[modStripFile(13, b)])
		if err != nil || len(strip.SubCells) != 1 {
			t.Fatalf("strip %d of one square: %v", b, err)
		}
	}
	if f, ok := terrainField(s, "13.name"); !ok || f.TextValue != "Moss" {
		t.Fatalf("the name %+v", f)
	}
}

// A mod made on the page alone saves through the project writer, keeping a
// hand comment of its terrain file. The published mod passes the launch's
// check, needs no entry file, and a map painted with its ground plays under
// it with the ground's land kind.
func TestTerrainPageModSavesPublishesAndPlays(t *testing.T) {
	s, dir := terrainPage(t)
	made := tree(t, dir)
	setTerrain(t, s, "newground", groundPNG(t, "Glacier.png", squaresPicture(3, 2)))
	setTerrain(t, s, "13.kind", "8")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	saved := tree(t, dir)
	if saved[mod.EntryName] != made[mod.EntryName] {
		t.Fatalf("the save changed the entry file:\n%s", saved[mod.EntryName])
	}
	want := append(keysOf(made), mod.TerrainFile)
	for b := range 4 {
		want = append(want, modStripFile(13, b))
	}
	slices.Sort(want)
	if got := keysOf(saved); !slices.Equal(got, want) {
		t.Fatalf("the saved folder %v, want %v", got, want)
	}

	// A comment written by hand stays through the page's next edit and save.
	rows := filepath.Join(dir, filepath.FromSlash(mod.TerrainFile))
	writeFileAll(t, rows, append([]byte("# made on the page\n"), saved[mod.TerrainFile]...))
	s = openInW(t, s, dir)
	setTerrain(t, s, "13.name", "Ice")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(rows); !strings.HasPrefix(string(b), "# made on the page\n[[group]]\ngroup = 13\n") || !strings.Contains(string(b), `name = "Ice"`) {
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
	layer := f.modFileLayer(res.Files.Order, nil)
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
		t.Fatal("the ground's mountain kind did not reach the cell's planes")
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
		out[b], _ = os.ReadFile(filepath.Join(dir, filepath.FromSlash(modStripFile(g, b))))
	}
	return out
}

// groupWhole fails the test unless the page reads group 13 whole, offers it
// to the brush, and a save writes the strips want.
func groupWhole(t *testing.T, s *mapInspectionEdits, dir string, want [4][]byte) {
	t.Helper()
	if _, ok := terrainField(s, "13.state"); ok {
		t.Fatal("the page says the ground is not ready")
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

// Deleting a saved ground, saving and undoing brings it back whole with its
// strips' bytes, and the next save writes them back; the same after a redo
// and a second undo. A ground the folder holds with a strip missing reads
// as not ready.
func TestTerrainPageUndoPastASaveRestoresADeletedGround(t *testing.T) {
	s, dir := terrainPage(t)
	setTerrain(t, s, "newground", groundPNG(t, "Glacier.png", groupPictureOf(t)))
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	strips := stripBytes(dir, 13)
	s = openInW(t, s, dir)
	setTerrain(t, s, "13.remove", "")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := stripBytes(dir, 13); slices.ContainsFunc(got[:], func(b []byte) bool { return b != nil }) {
		t.Fatalf("the save kept strips of the deleted ground")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	groupWhole(t, s, dir, strips)
	setTerrain(t, s, "13.remove", "")
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	for _, step := range []func() (*ui.InspectionDocument, error){s.Undo, s.Redo, s.Undo} {
		if _, err := step(); err != nil {
			t.Fatal(err)
		}
	}
	groupWhole(t, s, dir, strips)
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(modStripFile(13, 3)))); err != nil {
		t.Fatal(err)
	}
	s = openInW(t, s, dir)
	if f, ok := terrainField(s, "13.state"); !ok || f.TextValue == "" {
		t.Fatalf("a ground with a strip missing %+v", f)
	}
}

// groupPictureOf is a picture of 2 by 2 squares.
func groupPictureOf(t *testing.T) image.Image {
	t.Helper()
	return squaresPicture(2, 2)
}
