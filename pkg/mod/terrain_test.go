package mod

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixture's two rows read as groups 13 and 31 with their kinds; each bad
// row is refused at its line.
func TestTerrainRowsReadAndRefuse(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "modrt", "testdata", "mods", "snow-field", filepath.FromSlash(TerrainFile)))
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseTerrain("snow-field", TerrainFile, src)
	if err != nil {
		t.Fatal(err)
	}
	snow, ok := d.Group(13)
	ice, ok2 := d.Group(31)
	if !ok || !ok2 || snow.Class != 2 || snow.Name != "Snow" || !snow.HasMinimap || snow.Minimap.R != 0xe8 || ice.Class != 8 || ice.HasMinimap {
		t.Fatalf("rows %+v", d.Groups)
	}
	for _, c := range []struct{ src, want string }{
		{"[[group]]\ngroup = 12\nkind = \"grass\"\nname = \"x\"\n", "terrain.toml:2: group is 12"},
		{"[[group]]\ngroup = 32\nkind = \"grass\"\nname = \"x\"\n", "group is 32"},
		{"[[group]]\ngroup = 13\nkind = \"lava\"\nname = \"x\"\n", "terrain.toml:3: kind is \"lava\""},
		{"[[group]]\ngroup = 13\nname = \"x\"\n", "has no kind"},
		{"[[group]]\ngroup = 13\nkind = \"sand\"\nname = \"x\"\nminimap = \"red\"\n", "#rrggbb"},
		{"[[group]]\ngroup = 13\nkind = \"sand\"\nname = \"x\"\n[[group]]\ngroup = 13\nkind = \"sand\"\nname = \"y\"\n", "already a row at line 1"},
	} {
		if _, err := ParseTerrain("m", TerrainFile, []byte(c.src)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.src, err, c.want)
		}
	}
	var all Data
	r := DataReader{Mod: "a"}
	if err := readTerrain(r, &all, TerrainFile, src); err != nil {
		t.Fatal(err)
	}
	if err := readTerrain(DataReader{Mod: "b"}, &all, TerrainFile, []byte("[[group]]\ngroup = 31\nkind = \"sand\"\nname = \"x\"\n")); err == nil || !strings.Contains(err.Error(), "one mod's") {
		t.Fatalf("two mods on group 31: %v", err)
	}
}

// The terrain kind's writer in the kind table writes rows that read back
// equal; a changed row is rewritten in place, a removed one leaves, a new one
// is appended, and every byte outside them stays: comments, spacing, order.
func TestTerrainWriterKeepsTheSource(t *testing.T) {
	k, ok := ScriptDataKind(TerrainFile)
	if !ok {
		t.Fatal("no kind reads the terrain file")
	}
	src := []byte("# snow and ice\n\n[[group]]\ngroup   = 13 # first\nkind    = \"grass\"\nname    = \"Snow\"\nminimap = \"#E8F0F8\"\n\n# the cold one\n[[group]]\ngroup = 31\nkind = \"mountain\"\nname = \"Ice\"\n\n[[group]]\ngroup = 20\nkind = \"sand\"\nname = \"Dune\"\n")
	d, err := ParseTerrain("m", TerrainFile, src)
	if err != nil {
		t.Fatal(err)
	}
	same, err := k.Write(TerrainFile, src, Data{Terrain: d})
	if err != nil || string(same) != string(src) {
		t.Fatalf("an unchanged write: %v\n%s", err, same)
	}
	groups := d.Groups
	groups[1].Name, groups[1].HasMinimap, groups[1].Minimap = "Глетчер", true, color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
	groups = append(groups[:2], TerrainGroup{Group: 14, Class: 9, Name: "Lake \"blue\""})
	out, err := k.Write(TerrainFile, src, Data{Terrain: TerrainData{Groups: groups}})
	if err != nil {
		t.Fatal(err)
	}
	want := "# snow and ice\n\n[[group]]\ngroup   = 13 # first\nkind    = \"grass\"\nname    = \"Snow\"\nminimap = \"#E8F0F8\"\n\n# the cold one\n[[group]]\ngroup = 31\nkind = \"mountain\"\nname = \"Глетчер\"\n\nminimap = \"#102030\"\n[[group]]\ngroup = 14\nkind = \"water\"\nname = \"Lake \\\"blue\\\"\"\n"
	if string(out) != want {
		t.Fatalf("written\n%s\nwant\n%s", out, want)
	}
	back, err := ParseTerrain("m", TerrainFile, out)
	if err != nil || len(back.Groups) != 3 || back.Groups[1].Name != "Глетчер" || back.Groups[2].Name != "Lake \"blue\"" || back.Groups[2].Class != 9 {
		t.Fatalf("read back %+v: %v", back.Groups, err)
	}
	fresh, err := k.Write(TerrainFile, nil, Data{Terrain: TerrainData{Groups: groups[2:]}})
	if err != nil || string(fresh) != "[[group]]\ngroup = 14\nkind = \"water\"\nname = \"Lake \\\"blue\\\"\"\n" {
		t.Fatalf("a new file: %v\n%s", err, fresh)
	}
	if _, err := (DataKind{Name: "x"}).Write("x", nil, Data{}); err == nil {
		t.Fatal("a kind with no writer wrote")
	}
}
