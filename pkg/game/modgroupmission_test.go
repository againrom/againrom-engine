package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
)

// groupMissionMod is the snow-field mod with a group 20 row and its strips
// when group20 is set and, when mapBytes is given, a mission replacing
// mission 10 with that map.
func groupMissionMod(t *testing.T, mapBytes []byte, group20 bool) string {
	t.Helper()
	return snowFieldMod(t, func(dir string) {
		entry := "def init(game, settings):\n    game.data.add(\"data/terrain.toml\")\n"
		if mapBytes != nil {
			entry += "    game.data.add(\"data/missions.toml\")\n"
			writeFileAll(t, filepath.Join(dir, "mission", "map.alm"), mapBytes)
			writeFileAll(t, filepath.Join(dir, filepath.FromSlash(mod.MissionsFile)),
				[]byte("[[mission]]\nkind = \"replace\"\nkey = \"snow\"\nnumber = 10\nmap = \"mission/map.alm\"\n"))
		}
		writeFileAll(t, filepath.Join(dir, "main.star"), []byte(entry))
		if !group20 {
			return
		}
		rows := filepath.Join(dir, filepath.FromSlash(mod.TerrainFile))
		b, err := os.ReadFile(rows)
		if err != nil {
			t.Fatal(err)
		}
		writeFileAll(t, rows, append(b, "\n[[group]]\ngroup = 20\nkind  = \"sand\"\nname  = \"Dune\"\n"...))
		for c := 0; c < 4; c++ {
			writeFileAll(t, stripFile(dir, 20, c), stripBMP(terrain.CellSize, 1))
		}
	})
}

// launchMods applies the mods of dir to f as the launch does: their files
// served, the set and the rules taken, then every kind of their data.
func launchMods(t *testing.T, f *FrontEnd, dir string) (modrt.Result, error) {
	t.Helper()
	res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	layer := f.modFileLayer()
	layer.order = res.Files.Order
	if err := layer.serveInstallFiles(f.Archives, res.Files.Files); err != nil {
		t.Fatal(err)
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		t.Fatal(err)
	}
	_, err = f.SetModData(res.Data)
	return res, err
}

// A mod mission whose map uses the mod's groups 13, 20 and 31 passes the
// launch's map check, which runs inside SetModData, and the rule table the
// editor's Publish and Test play run, with the install's terrain cost table
// present. The same map under a mod without group 20 is refused naming the
// map and the group.
func TestModGroupMissionPassesTheMapChecks(t *testing.T) {
	rows, _, err := checkSnowField(t, groupMissionMod(t, nil, true))
	if err != nil {
		t.Fatal(err)
	}
	painted := modGroupCells(t, nil, rows, groupCell{31, 20}, groupCell{13, 21}, groupCell{20, 22})
	mapBytes, err := painted.Archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's unit has owner slot 0, which the rule table refuses
	// whatever the terrain; slot 1 is the map's one side.
	ed, err := mapedit.New(mapBytes)
	if err == nil {
		err = ed.SetUnitField(0, "owner", 1)
	}
	if err != nil {
		t.Fatal(err)
	}
	mapBytes = ed.Bytes()

	f := cellStateFrontWith(t, nil)
	res, err := launchMods(t, f, groupMissionMod(t, mapBytes, true))
	if err != nil {
		t.Fatalf("the launch refused the mod's mission: %v", err)
	}
	if f.Table.ModTerrain() == nil {
		t.Fatal("the launch applied no terrain rows")
	}
	x, err := f.modMapInspector()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := x.terrainCosts(); !ok || err != nil {
		t.Fatalf("the fixture carries no terrain cost table (%t, %v); rule terrain-planes is not reached", ok, err)
	}
	if len(res.Missions.Missions) != 1 {
		t.Fatalf("missions %d", len(res.Missions.Missions))
	}
	m := res.Missions.Missions[0]
	address, _ := MissionMap(m.Number)
	for _, finding := range x.CheckModMap(m, address) {
		if finding.Error {
			t.Errorf("the editor's map check: %s", finding.Line())
		}
	}

	_, err = launchMods(t, cellStateFrontWith(t, nil), groupMissionMod(t, mapBytes, false))
	if err == nil || !strings.Contains(err.Error(), "mission/map.alm") || !strings.Contains(err.Error(), "terrain group 20") {
		t.Fatalf("a map naming a group no mod adds: %v", err)
	}
}
