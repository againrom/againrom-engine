package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondSavedCaster is one VirtualCaster the fixture document holds in the list
// after its Sacks.
type secondSavedCaster struct {
	reference          uint32
	cell               uint16
	x, y, spell, power uint8
	reach              uint8
}

// secondSavedExtraCasters are the casters the next fixture document holds.
var secondSavedExtraCasters []secondSavedCaster

// caster writes a VirtualCaster: the Token head, the reach byte and the
// six-byte block. The second game's Token +0x0c is a word.
func (s *secondStream) caster(c secondSavedCaster, key uint32) {
	s.obj("VirtualCaster")
	s.u16(c.cell)
	s.u16(c.cell)
	s.u8(0x80)
	s.u8(0x80)
	s.u16(0)
	s.u32(0)
	s.u32(0)   // RuntimeID
	s.u16(0)   // T0C
	s.u16(0)   // T0E
	s.u32(0)   // T08
	s.u16(0)   // T18
	s.u32(0)   // T1C
	s.u32(key) // Identity
	s.u32(c.reference)
	s.u8(c.reach)
	s.b = append(s.b, c.spell, c.power, c.x, c.y, 0, 0)
}

const (
	casterFixturePlayerKey = 0x0a000001
	casterFixtureCell      = uint16(21)<<8 | 16
)

func casterFixtureMission() secondMissionContent {
	c := pointerSaveMission()
	c.objects[0].Field0E = 1
	c.casters = []alm.Enchantment{{Tag: 3, X: 40, Y: 41, A: 4, B: 42, SpellRaw: 2 | 99<<16}}
	return c
}

func casterFixtureFromMap() sim.BuildingCaster {
	return sim.BuildingCaster{X: 40, Y: 41, Spell: 2, Power: 99, Reach: 3, Owner: 1, Cell: casterFixtureCell}
}

func withExtraCasters(t *testing.T, casters ...secondSavedCaster) {
	t.Helper()
	secondSavedExtraCasters = casters
	t.Cleanup(func() { secondSavedExtraCasters = nil })
}

func casterLoad(t *testing.T, c secondMissionContent, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if f.live == nil || f.live.world == nil {
		t.Fatalf("the fixture file does not LOAD: %s", app.HeadlessMessage())
	}
	return f, app
}

func fileCasters(t *testing.T, raw []byte) []sav.VirtualCaster {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sf.VirtualCasters()
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func checkCasterList(t *testing.T, name string, got []sim.BuildingCaster, want ...sim.BuildingCaster) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: list %+v, want %+v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: list %+v, want %+v", name, got, want)
		}
	}
}

// A LOAD takes the file's caster list, not the map's; a SAVE writes the world's
// list and its LOAD reads it back. A changed list changes the file. A file whose
// list is gone rebuilds from the map, which the file's list differs from. The
// base does not compile this test.
func TestBuildingCasterSAVWriteThenReadWithLossControls(t *testing.T) {
	withExtraCasters(t, secondSavedCaster{reference: casterFixturePlayerKey, cell: casterFixtureCell, x: 30, y: 31, spell: 5, power: 77, reach: 9})
	want := sim.BuildingCaster{X: 30, Y: 31, Spell: 5, Power: 77, Reach: 9, Owner: 1, Cell: casterFixtureCell}
	c := casterFixtureMission()
	f, app := casterLoad(t, c, pointerSaveFile())
	checkCasterList(t, "the file's LOAD", f.live.world.BuildingCasters(), want)

	saved := secondPageSave(t, f, app)
	read := fileCasters(t, saved)
	if len(read) != 1 || read[0].Spell != 5 || read[0].Power != 77 || read[0].X != 30 || read[0].Y != 31 || read[0].Reach != 9 ||
		read[0].Cell != casterFixtureCell || !read[0].OwnerKnown || read[0].OwnerSlot != 1 {
		t.Fatalf("the SAVE holds %+v, want the file's caster with owner slot 1", read)
	}
	cold, coldApp := casterLoad(t, c, saved)
	checkCasterList(t, "the SAVE's LOAD", cold.live.world.BuildingCasters(), want)
	if resaved := fileCasters(t, secondPageSave(t, cold, coldApp)); len(resaved) != 1 || resaved[0] != read[0] {
		t.Fatalf("the resave holds %+v, want %+v", resaved, read)
	}

	changed := want
	changed.Spell = 6
	f.live.world.DeclareBuildingCasters([]sim.BuildingCaster{changed})
	moved := secondPageSave(t, f, app)
	if bytes.Equal(saved, moved) {
		t.Fatal("a changed caster list wrote the same file")
	}
	if got := fileCasters(t, moved); len(got) != 1 || got[0].Spell != 6 {
		t.Fatalf("the changed list writes %+v", got)
	}

	f.live.world.DeclareBuildingCasters(nil)
	dropped := secondPageSave(t, f, app)
	if got := fileCasters(t, dropped); len(got) != 0 {
		t.Fatalf("a world with no casters wrote %+v", got)
	}
	// The engine's document says its list is empty, so the towers stay down: a
	// lost list would come back from the map as the rebuild's caster.
	loaded, _ := casterLoad(t, c, dropped)
	checkCasterList(t, "a LOAD of a save with every caster gone", loaded.live.world.BuildingCasters())
	if casterFixtureFromMap() == want {
		t.Fatal("the control lists are equal, so a lost list would pass")
	}
}

// A mission entered from the map holds the map's casters: the rebuild a fresh
// entry goes through carries the list. Before the two rebuilds named it, the
// world entered with an empty list, which is what mission 72 measured.
func TestBuildingCasterFreshEntryCarriesTheMapsList(t *testing.T) {
	f, _ := secondThresholdEnter(t, casterFixtureMission(), mapload.DifficultyNormal)
	got := f.live.world.BuildingCasters()
	if len(got) != 1 || got[0].Spell != 2 || got[0].Power != 99 || got[0].X != 40 || got[0].Y != 41 || got[0].Owner != 1 {
		t.Fatalf("a fresh entry holds %+v, want the map's caster", got)
	}
}

// A file with an empty list on a map with such records rebuilds from the map
// through the saved structure's authored index. Fails at the base, which does
// not compile the fixture's caster field.
func TestBuildingCasterEmptyFileListRebuildsFromTheMap(t *testing.T) {
	f, _ := casterLoad(t, casterFixtureMission(), pointerSaveFile())
	checkCasterList(t, "the LOAD", f.live.world.BuildingCasters(), casterFixtureFromMap())
}

// A building down at load rebuilds no caster; a standing one rebuilds it; a
// file record with no Player keeps the map's owner on its cell word. The base
// does not compile this test.
func TestApplyOriginalBuildingCastersRules(t *testing.T) {
	f, _ := casterLoad(t, casterFixtureMission(), pointerSaveFile())
	w := f.live.world
	m := f.live.mission.state.Map
	setHealth := func(health uint16) {
		structures := w.Structures()
		structures[0].Field42, structures[0].MaxHealth = health, 100
		w.DeclareStructures(structures)
	}
	setHealth(50)
	w.DeclareBuildingCasters(nil)
	applyOriginalBuildingCasters(&Mission{Map: m, World: w}, nil)
	checkCasterList(t, "standing", w.BuildingCasters(), casterFixtureFromMap())
	setHealth(0)
	applyOriginalBuildingCasters(&Mission{Map: m, World: w}, nil)
	checkCasterList(t, "down", w.BuildingCasters())
	setHealth(50)
	applyOriginalBuildingCasters(&Mission{Map: m, World: w}, []sav.VirtualCaster{{Reach: 4, Spell: 7, Power: 1, X: 5, Y: 6, Cell: casterFixtureCell}})
	checkCasterList(t, "file", w.BuildingCasters(), sim.BuildingCaster{X: 5, Y: 6, Spell: 7, Power: 1, Reach: 4, Owner: 1, Cell: casterFixtureCell})

	first, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 20, Height: 20}, sim.ModeCanonical, sim.Terrain{}, nil, nil, sim.Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	applyOriginalBuildingCasters(&Mission{Map: m, World: first}, []sav.VirtualCaster{{Reach: 4, Spell: 7, Cell: casterFixtureCell}})
	checkCasterList(t, "first game", first.BuildingCasters())
	if first.BuildsCasters() {
		t.Fatal("a world with no second-game script builds casters")
	}

}
