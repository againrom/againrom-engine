package modrt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const townsMain = "def init(game, settings):\n    game.data.add(\"data/towns.toml\")\n"

// The fixture mod's layer loads through game.data.add: two records, East's
// description read from the mod folder under its merged name, and the load
// order the mods state.
func TestExampleModLayersTwoTowns(t *testing.T) {
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "east-town"), os.DirFS(filepath.Join("testdata", "mods", "east-town"))); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "east-town", "towns"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "east-town", "towns", "east.json"), []byte(`{"town": "east"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(dir, []string{"east-town"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Load(entries, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	rs := res.Towns.Records
	if len(rs) != 2 || rs[0].Entry.ID != 1 || rs[0].Entry.Name != "West" || rs[1].Entry.ID != 2 {
		t.Fatalf("records %+v", rs)
	}
	east := rs[1]
	if east.Entry.Description != "east-town:towns/east.json" || east.Description == nil ||
		string(east.Description.Data) != `{"town": "east"}` || east.Description.Path != "towns/east.json" || east.Dir == "" {
		t.Fatalf("East's description %+v", east.Description)
	}
	if east.Entry.OpensAfter != 20 || len(east.Entry.Missions) != 1 || east.Entry.Missions[0] != 31 {
		t.Fatalf("East %+v", east.Entry)
	}
	if res.Towns.Order == nil || !res.Items.Empty() {
		t.Fatal("the layer carries no order, or the mod adds more than towns")
	}
}

// A refusal of the towns file names the mod, the file and the line; a
// description file the mod does not hold is refused at its line.
func TestTownsFileRefusalsNameModFileAndLine(t *testing.T) {
	for name, c := range map[string]struct {
		towns, want string
	}{
		"unknown key":        {"[[town]]\nid = 2\nsize = 3\n", `mod "m": data/towns.toml:3: unknown key "size"`},
		"missing file":       {"[[town]]\nid = 2\ndescription = \"towns/none.json\"\n", `mod "m": data/towns.toml:3: description "towns/none.json": towns/none.json: no such file in the mod folder`},
		"outside the folder": {"[[town]]\nid = 2\ndescription = \"../x.json\"\n", `mod "m": data/towns.toml:3: description "../x.json"`},
	} {
		_, err := Load(makeMods(t, map[string]map[string]string{"m": {"main.star": townsMain, "data/towns.toml": c.towns}}), "rom1-en", nil, Options{})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}
