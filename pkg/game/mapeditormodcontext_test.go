package game

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
)

// contextInspector is an editor over the test play install whose table
// holds the archer rows the archer-girl fixture edits and no speaker.
func contextInspector(t *testing.T) *MapInspector {
	t.Helper()
	x := newTestPlayFixture(t).x
	withInstallTable(x)
	return x
}

// withInstallTable gives the editor x that table, an empty tileset, and the
// install files a table read again from them needs.
func withInstallTable(x *MapInspector) {
	rows := dbCollection{
		{},
		charRow("NPC06_1", 14, 3, 1, "Bow", "", "Helm", "Mail"),
		charRow("NPC06_2", 14, 3, 1, "Bow", "", "Helm", "Mail"),
		charRow("A_PeasantGuard1", 14, 9, 2, "Bow"),
		charRow("A_PeasantGuard2", 14, 9, 2, "Bow"),
	}
	x.table = &mapload.Table{Humans: rows}
	x.tiles = &terrain.Tileset{}
	// A mod that serves install files has the table read again from them.
	x.archives.Containers.Overlay("world/data/data.bin", func() ([]byte, error) { return synth.DataBinUnitsTable(nil, nil), nil })
	x.archives.Containers.Overlay("scenario/npc.reg", func() ([]byte, error) { return synth.NPCReg(nil), nil })
}

// withMission makes the mod at dir also declare a mission, so the editor's
// mission gate admits it, its script adding both files.
func withMission(t *testing.T, dir, kindFile string) {
	t.Helper()
	writeModText(t, dir, mod.MissionsFile, editorModMissions)
	writeMissionMap(t, dir)
	writeModText(t, dir, "main.star", "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n    game.data.add(\""+kindFile+"\")\n")
}

// writeMissionMap writes the map editorModMissions names on an English base.
func writeMissionMap(t *testing.T, dir string) {
	t.Helper()
	writeModText(t, dir, "mission/en/map.alm", string(synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"})))
}

// A map opened under a mod reads the terrain groups, speaker records and
// character catalog the game would apply for it, built by the mod's admitted
// fork; the install's table and tileset stay as they were.
func TestEditorModContextReadsTheKindsTheGameApplies(t *testing.T) {
	t.Run("terrain", func(t *testing.T) {
		x := contextInspector(t)
		install, tiles := x.table, x.tiles
		dir := snowFieldMod(t, func(dir string) { withMission(t, dir, mod.TerrainFile) })
		want, _, err := checkSnowField(t, dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := x.SetMod(dir); err != nil || x.mod.contextIssue != "" {
			t.Fatalf("open: %v %q", err, x.mod.contextIssue)
		}
		if got := x.table.ModTerrain(); got == nil || !reflect.DeepEqual(got.Groups, want.Groups) {
			t.Fatalf("terrain rows %+v, want %+v", got, want.Groups)
		}
		if x.tiles == tiles || x.tiles.Slot(4*13) == nil || x.tiles.Slot(4*31+3) == nil {
			t.Fatal("the tileset holds no strip of the mod's groups")
		}
		if rows := modBrushTerrains(x.table, x.tiles); len(rows) != 2 || rows[0].Class != mapedit.GroupBrush(13) {
			t.Fatalf("brush rows %+v", rows)
		}
		if install.ModTerrain() != nil {
			t.Fatal("the install's table took the mod's rows")
		}
		x.setEditorMod(nil)
		if x.table != install || x.tiles != tiles {
			t.Fatal("closing the mod did not give back the install's context")
		}
	})
	t.Run("speakers", func(t *testing.T) {
		x := contextInspector(t)
		dir := writeEditorMod(t, t.TempDir(), editorModMissions)
		writeMissionMap(t, dir)
		start := speakerNumberStart("ambush")
		writeModText(t, dir, mod.SpeakersFile, speakerRecordText(start, speakerPictureFile(start)))
		if err := os.MkdirAll(filepath.Join(dir, "speakers"), 0o755); err != nil {
			t.Fatal(err)
		}
		speakerPNG(t, filepath.Join(dir, filepath.FromSlash(speakerPictureFile(start))), color.RGBA{R: 10, G: 200, B: 30, A: 255})
		res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
		if err != nil {
			t.Fatal(err)
		}
		records, _, err := mod.MergeSpeakers(res.Speakers)
		if err != nil || len(records) != 1 {
			t.Fatalf("records %v %v", records, err)
		}
		if err := x.SetMod(dir); err != nil || x.mod.contextIssue != "" {
			t.Fatalf("open: %v %q", err, x.mod.contextIssue)
		}
		e, ok := x.table.NPCCatalog.Entry(records[0].ID)
		if !ok || e.Mod != "ambush" || e.Name != "Karl" {
			t.Fatalf("catalog entry %+v %v", e, ok)
		}
	})
	t.Run("characters", func(t *testing.T) {
		x := contextInspector(t)
		dir := filepath.Join(t.TempDir(), "archer-girl")
		if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "modrt", "testdata", "mods", "archer-girl"))); err != nil {
			t.Fatal(err)
		}
		withMission(t, dir, mod.CharactersFile)
		// The archer edit, and guest-chain's sellsword in the catalog.
		edit, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.CharactersFile)))
		if err != nil {
			t.Fatal(err)
		}
		key, err := os.ReadFile(filepath.Join("..", "modrt", "testdata", "mods", "guest-chain", filepath.FromSlash(mod.CharactersFile)))
		if err != nil {
			t.Fatal(err)
		}
		writeModText(t, dir, mod.CharactersFile, string(edit)+"\n"+string(key))
		own, err := os.ReadFile(filepath.Join(dir, "text", "en", "strings.toml"))
		if err != nil {
			t.Fatal(err)
		}
		writeModText(t, dir, "text/en/strings.toml", string(own)+"\"character.sellsword\" = \"Sellsword\"\n")
		res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
		if err != nil {
			t.Fatal(err)
		}
		_, want, _, err := mod.MergeCharacters(res.Characters, installCharacters(x.table))
		if err != nil {
			t.Fatal(err)
		}
		install := x.table
		if err := x.SetMod(dir); err != nil || x.mod.contextIssue != "" {
			t.Fatalf("open: %v %q", err, x.mod.contextIssue)
		}
		if len(want) != 1 || want[0].Key != "sellsword" || !reflect.DeepEqual(x.mod.catalog, want) || !reflect.DeepEqual(characterCatalog(x.table), want) {
			t.Fatalf("catalog %+v, want %+v", x.mod.catalog, want)
		}
		if x.table.Humans == install.Humans || len(x.table.Mods.Characters) != 1 {
			t.Fatal("the context does not hold the rows the game edits")
		}
	})
}
