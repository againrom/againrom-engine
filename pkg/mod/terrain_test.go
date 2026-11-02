package mod

import (
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
