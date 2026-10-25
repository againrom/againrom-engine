package game

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

const replaceModID = "ambush"

// replaceMod writes, in a fresh mods folder, the mod ambush that replaces
// campaign mission 20 with the map and texts it copies out of the install
// under test, as cmd/modmission does. It returns the mods folder.
func replaceMod(t *testing.T) string { return replaceModNumber(t, 20) }

// replaceModNumber is replaceMod for campaign mission number.
func replaceModNumber(t *testing.T, number int) string {
	t.Helper()
	install := os.Getenv("AGAINROM_ASSETS")
	if install == "" {
		t.Skip("no AGAINROM_ASSETS: the replaced mission needs a lawful install")
	}
	root := t.TempDir()
	dir := filepath.Join(root, replaceModID)
	files := map[string]string{
		"mod.toml": "id = \"" + replaceModID + "\"\ntitle = \"Ambush\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\n" +
			"description = \"Mission 20 on an edited map.\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n",
		"main.star": "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n",
		"data/missions.toml": "[[mission]]\nkind = \"replace\"\nkey = \"ambush\"\nnumber = " + strconv.Itoa(number) + "\n" +
			"map = \"mission/{lang}/map.alm\"\ntexts = \"mission/{lang}/text\"\ntitle = \"ambush.title\"\n",
		"text/en/strings.toml": "\"ambush.title\" = \"Ambush\"\n",
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lang := modrt.LanguageFor(BaseID(InspectInstall(install)))
	if _, err := AssembleModMission(install, lang, number, dir); err != nil {
		t.Fatal(err)
	}
	return root
}

// replacedUnit is the first interior creature of m whose unit id no other
// record shares: its record index and id.
func replacedUnit(t *testing.T, m *alm.Map) (int, uint16) {
	t.Helper()
	ids := map[uint16]int{}
	for _, u := range m.Units {
		ids[u.UnitID]++
	}
	for i, u := range m.Units {
		x, y := int(u.X>>8), int(u.Y>>8)
		if u.ClassID >= 0x1a && u.UnitID != 0 && ids[u.UnitID] == 1 && x >= 10 && y >= 10 && x < int(m.Width)-10 && y < int(m.Height)-10 {
			return i, u.UnitID
		}
	}
	t.Fatal("mission 20 places no interior creature with a unit id of its own")
	return 0, 0
}

// replaceRoute plays mission 10 through the production completion path and
// opens the mission its win leads to, which must be mission 20.
func replaceRoute(t *testing.T, f *FrontEnd) {
	t.Helper()
	party := f.ChargenParty(ui.ChargenResult{Name: "Ambush", Choices: []int{0, 0, 0}, Stats: []int{31, 27, 24, 29}})
	app := f.App("replaced mission")
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	next, _, err := f.LiveCompleteCampaign()
	if err != nil || next != 20 {
		t.Fatalf("mission 10's win answered %d: %v", next, err)
	}
	if err := app.OpenMission(f.MissionOpener(next)); err != nil {
		t.Fatal(err)
	}
}

func liveUnitCount(f *FrontEnd, id uint16) int {
	n := 0
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == id {
			n++
		}
	}
	return n
}

func mapHasUnit(m *alm.Map, id uint16) bool {
	for _, u := range m.Units {
		if u.UnitID == id {
			return true
		}
	}
	return false
}

// TestReleaseModReplacesACampaignMission proves, on the lawful install, the
// editor-to-play loop of a replaced mission: the editor deletes one unit of
// the installed mission 20 map and its Save writes the map into the mod's
// mission entry; the ordinary route from mission 10 then builds mission 20
// from the mod's map, a SAVE and cold LOAD under the mod keep it, and a LOAD
// without the mod refuses. The same route without the mod builds the
// installed map.
func TestReleaseModReplacesACampaignMission(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := replaceMod(t)
	install := os.Getenv("AGAINROM_ASSETS")
	x, err := NewMapInspector(install)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.archives.Containers.ReadFile("scenario/20.alm")
	if err != nil {
		t.Fatal(err)
	}
	installed, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	index, id := replacedUnit(t, installed)
	if err := x.SetMod(filepath.Join(dir, replaceModID)); err != nil {
		t.Fatal(err)
	}
	e := x.Editor()
	if err := e.Open("scenario/20.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	if d, err := e.Document().Edits.Delete("Unit", index); err != nil || d == nil {
		t.Fatalf("delete unit %d: %v", index, err)
	}
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("Save into the mod: %v", err)
	}

	// Loss control: without the mod the route builds the installed map.
	plain := guestFront(t, "")
	replaceRoute(t, plain)
	if plain.live.mission.state.Address != "scenario/20.alm" || !mapHasUnit(plain.live.mission.state.Map, id) || liveUnitCount(plain, id) != 1 {
		t.Fatalf("loss control: the installed mission 20 does not play unit %d", id)
	}
	unmarked := guestMissionSave(t, plain)

	f := guestFront(t, dir, replaceModID)
	replaceRoute(t, f)
	built := f.live.mission.state
	if built.Address != "scenario/20.alm" || mapHasUnit(built.Map, id) || liveUnitCount(f, id) != 0 {
		t.Fatalf("mission 20 was not built from the mod's map: unit %d plays", id)
	}
	if got, want := len(f.live.world.Entities()), len(plain.live.world.Entities())-1; got != want {
		t.Fatalf("the replaced mission holds %d entities, want %d", got, want)
	}

	// SAVE and cold LOAD under the mod keep the mod's map and world.
	saved := guestMissionSave(t, f)
	h := guestFront(t, dir, replaceModID)
	open, town, err := h.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold LOAD of the replaced mission: town %v %v", town, err)
	}
	if err := h.App("replaced mission load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if h.live.mission.state.Number != 20 || mapHasUnit(h.live.mission.state.Map, id) || liveUnitCount(h, id) != 0 {
		t.Fatalf("the cold LOAD did not keep the mod's mission 20")
	}
	if got, want := len(h.live.world.Entities()), len(f.live.world.Entities()); got != want {
		t.Fatalf("the cold LOAD holds %d entities, want %d", got, want)
	}

	// A LOAD without the mod refuses, naming it.
	if _, _, err := guestFront(t, "").RestoreOriginal(saved); !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), replaceModID) {
		t.Fatalf("LOAD without the mod: %v", err)
	}
	// An unmarked save of the installed mission 20 is refused under the mod
	// even where unmarked saves are accepted.
	k := guestFront(t, dir, replaceModID)
	k.Table.Mods.AcceptUnmarked = true
	if _, _, err := k.RestoreOriginal(unmarked); !errors.Is(err, ErrModMark) || !strings.Contains(err.Error(), "replaces that mission's map") {
		t.Fatalf("unmarked LOAD under the mod: %v", err)
	}
}
