package game

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// stripBMP is a test-made terrain strip: w px wide, frames cells of 32 px
// tall, an 8-bit picture with its own colour table.
func stripBMP(w, frames int) []byte {
	pal := color.Palette{color.RGBA{0, 0, 0, 0xff}, color.RGBA{0xe8, 0xf0, 0xf8, 0xff}}
	img := image.NewPaletted(image.Rect(0, 0, w, frames*terrain.CellSize), pal)
	for i := range img.Pix {
		img.Pix[i] = uint8(i & 1)
	}
	return synth.BMP8(img)
}

// snowFieldMod copies the snow-field fixture into a new folder and writes
// the four strips of groups 13 (two frames each) and 31 (one frame each);
// edit changes the folder before it is read.
func snowFieldMod(t *testing.T, edit func(dir string)) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "snow-field")
	from := filepath.Join("..", "modrt", "testdata", "mods", "snow-field")
	for _, rel := range []string{"mod.toml", "main.star", filepath.FromSlash(mod.TerrainFile)} {
		b, err := os.ReadFile(filepath.Join(from, rel))
		if err != nil {
			t.Fatal(err)
		}
		writeFileAll(t, filepath.Join(dir, rel), b)
	}
	for _, g := range []struct{ group, frames int }{{13, 2}, {31, 1}} {
		for b := 0; b < 4; b++ {
			writeFileAll(t, stripFile(dir, g.group, b), stripBMP(terrain.CellSize, g.frames))
		}
	}
	if edit != nil {
		edit(dir)
	}
	return dir
}

func stripFile(dir string, g, b int) string {
	return filepath.Join(dir, mod.FilesDir, mod.CommonFiles, filepath.FromSlash(terrain.GroupStripPath(g, b)))
}

func writeFileAll(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// checkSnowField runs the mod and the check of its strips and rows over an
// empty install, answering the terrain rows and the container the strips
// are served on.
func checkSnowField(t *testing.T, dir string) (mod.TerrainData, *vfs.FS, error) {
	t.Helper()
	res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
	if err != nil {
		return mod.TerrainData{}, nil, err
	}
	fs, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := &modFiles{fs: fs, order: res.Files.Order, owners: map[string]modFileEntry{}}
	if err := b.serveInstallFiles(&Archives{Containers: fs, mods: b}, res.Files.Files); err != nil {
		return mod.TerrainData{}, nil, err
	}
	if res.Set.Mods[0].StateFree {
		t.Error("a mod that adds terrain groups is state-free; it changes game state")
	}
	rows, err := modTerrain(res.Terrain, b.owners, fs)
	return rows, fs, err
}

// The check accepts a mod adding group 13 and group 31 with their strips,
// the tileset holds the new slots, and each row carries its strips' frames.
func TestModTerrainGroupsAreAccepted(t *testing.T) {
	rows, fs, err := checkSnowField(t, snowFieldMod(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	snow, ok := rows.Group(13)
	ice, ok2 := rows.Group(31)
	if !ok || !ok2 || snow.Frames != [4]int{2, 2, 2, 2} || ice.Frames != [4]int{1, 1, 1, 1} || snow.Class != 2 || ice.Class != 8 {
		t.Fatalf("rows %+v", rows.Groups)
	}
	tiles := terrain.LoadTileset(fs)
	for _, g := range []int{13, 31} {
		for b := 0; b < 4; b++ {
			if tiles.Slot(4*g+b) == nil {
				t.Errorf("group %d column %d: no slot", g, b)
			}
		}
	}
	if ref := terrain.Resolve(uint16(13<<6 | 0x11)); tiles.Slot(ref.Slot).SubCell(ref.Sub) == nil {
		t.Error("a snow word draws no cell")
	}
	if got := len(groupWords(snow)); got != 8 {
		t.Errorf("snow brush words %d, want 8", got)
	}
}

// The check refuses a strip that breaks the strip rule, a strip of a group
// no row adds and a row whose group lacks a strip, naming the mod and file.
func TestModTerrainCheckRefuses(t *testing.T) {
	for _, c := range []struct {
		name string
		edit func(dir string)
		want []string
	}{
		{"a strip 33 px wide", func(dir string) { writeFileAll(t, stripFile(dir, 13, 1), stripBMP(33, 2)) }, []string{`mod "snow-field"`, "tile4-05.bmp", "32 px wide"}},
		{"a strip of 15 frames", func(dir string) { writeFileAll(t, stripFile(dir, 13, 2), stripBMP(32, 15)) }, []string{"tile4-06.bmp", "15 frames"}},
		{"a group with no row", func(dir string) { writeFileAll(t, stripFile(dir, 14, 0), stripBMP(32, 1)) }, []string{`mod "snow-field"`, "tile4-08.bmp", "terrain group 14"}},
		{"a row with no strip", func(dir string) { os.Remove(stripFile(dir, 31, 3)) }, []string{`mod "snow-field"`, "terrain.toml", "group 31 has no strip for column 3"}},
	} {
		_, _, err := checkSnowField(t, snowFieldMod(t, c.edit))
		for _, w := range c.want {
			if err == nil || !strings.Contains(err.Error(), w) {
				t.Errorf("%s: %v, want %q", c.name, err, w)
			}
		}
	}
}

// The editor's brush of a mod group paints that group's words on a new map
// and the saved bytes carry the group in their words and mask; without the
// mod's rows the brush is refused.
func TestEditorPaintsAModGroup(t *testing.T) {
	rows, _, err := checkSnowField(t, snowFieldMod(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := alm.NewMap(alm.DefaultNewMap())
	if err != nil {
		t.Fatal(err)
	}
	ed, err := mapedit.New(doc.Write())
	if err != nil {
		t.Fatal(err)
	}
	table := &mapload.Table{}
	table.Mods.Terrain = rows
	op := ui.BrushStroke{Plane: ui.BrushPlaneTerrain, Value: mapedit.GroupBrush(13), Cells: []image.Point{{X: 20, Y: 20}}}
	if _, err := paintStroke(ed, op, table); err != nil {
		t.Fatal(err)
	}
	m, err := ed.Map()
	if err != nil {
		t.Fatal(err)
	}
	if w := m.Tiles[20*m.Width+20]; alm.TileGroup(w) != 13 || m.Meta.Bitmask&(1<<13) == 0 {
		t.Fatalf("painted word %#04x mask %#x", w, m.Meta.Bitmask)
	}
	if err := mapload.CheckTileGroups(m, table); err != nil {
		t.Fatalf("the game's load check refuses the painted map: %v", err)
	}
	if err := mapload.CheckTileGroups(m, &mapload.Table{}); err == nil || !strings.Contains(err.Error(), "terrain group 13") {
		t.Fatalf("without the mod: %v", err)
	}
	if _, err := paintStroke(ed, op, &mapload.Table{}); err == nil {
		t.Fatal("a mod group's brush painted with no mod open")
	}
	terrains := modBrushTerrains(table, nil)
	if len(terrains) != 2 || terrains[0].Name != "Snow" || terrains[1].Class != mapedit.GroupBrush(31) {
		t.Fatalf("terrain list rows %+v", terrains)
	}
}
