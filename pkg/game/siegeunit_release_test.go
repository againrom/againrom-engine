package game

import (
	"encoding/binary"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
)

// siegeUnitsOf returns the Unit actors of a city document's Player groups,
// keyed by the hire type their display-backing word holds, and the number of
// Human actors named like a siege engine.
func siegeUnitsOf(t *testing.T, raw []byte) (map[uint32][]sav.CityObjectData, int) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	city, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	data := city.Data()
	units := map[uint32][]sav.CityObjectData{}
	humans := 0
	for _, g := range data.Objects[data.Players[0]-1].Player.Groups {
		for _, a := range g.Actors {
			object := data.Objects[a-1]
			switch {
			case object.Class == "Unit" && len(object.Unit.Scalar2) == 55:
				typ := binary.LittleEndian.Uint32(object.Unit.Scalar2[47:51])
				units[typ] = append(units[typ], object)
			case object.Class == "Human" && (object.Unit.Name == "Catapult" || object.Unit.Name == "Ballista"):
				humans++
			}
		}
	}
	return units, humans
}

// hireSiegeSquads hires one Catapult and one Ballista through the tavern.
func hireSiegeSquads(t *testing.T, name string) *FrontEnd {
	t.Helper()
	f, screen := hireForSiegeOrderTest(t, name)
	for _, typ := range []int{1, 2} {
		f.Town.mercPool[typ], f.Town.mercCapacity[typ], f.Town.mercEnabled[typ] = 1, 1, true
		if msg, ok := screen.toggleMercenary(typ); !ok {
			t.Fatalf("tavern hire of siege type %d refused: %s", typ, msg)
		}
	}
	return f
}

func siegeMembers(f *FrontEnd) map[uint8]int {
	out := map[uint8]int{}
	for _, m := range f.Carried {
		if nativeCitySiegeMember(m) {
			out[m.MercenaryType]++
		}
	}
	return out
}

// A hired Catapult and Ballista are written as the Unit actors the original
// writes: the Units row of the engine, the hire constructor's state word,
// insert index and flags, and the engine's weapon as the held weapon. A cold
// LOAD holds each once, the mission enters with both, and the second SAVE
// writes the same two Units again.
func TestReleaseCitySiegeHireWritesUnitActor(t *testing.T) {
	f := hireSiegeSquads(t, "Siege Unit")
	types := siegeTypes(t, f)
	raw := currentTownSave(t, f)

	units, humans := siegeUnitsOf(t, raw)
	if humans != 0 {
		t.Fatalf("the SAV holds %d Human actors named like a siege engine, want none", humans)
	}
	for typ, name := range map[uint32]string{1: "Catapult", 2: "Ballista"} {
		got := units[typ]
		if len(got) != 1 {
			t.Fatalf("SAV holds %d Unit actors for hire type %d, want 1", len(got), typ)
		}
		u := got[0].Unit
		row := int(u.Token[16])
		if row == 0 || row >= f.Table.Units.Len() || f.Table.Units.EntryName(row) != name {
			t.Fatalf("hire type %d is on Units row %d, want the %s row", typ, row, name)
		}
		if !types[int32(binary.LittleEndian.Uint16(u.Token[17:19]))] {
			t.Fatalf("hire type %d token type %d is not a siege type", typ, binary.LittleEndian.Uint16(u.Token[17:19]))
		}
		if u.Name != "" || u.SpellbookFlag != 0 || len(u.XP) != 0 || len(u.Equipment) != 0 {
			t.Fatalf("hire type %d holds name %q, spellbook %d, XP %d, equipment %d", typ, u.Name, u.SpellbookFlag, len(u.XP), len(u.Equipment))
		}
		if state := binary.LittleEndian.Uint32(u.Scalar1[4:8]); state != siegeHireStateWord {
			t.Fatalf("hire type %d state word %#x, want %#x", typ, state, siegeHireStateWord)
		}
		if u.ContainerFlag != 1 || u.ContainerTails[0] != siegeHireInsertIndex || len(u.Container) != 0 {
			t.Fatalf("hire type %d container flag %d tails %v items %d", typ, u.ContainerFlag, u.ContainerTails, len(u.Container))
		}
		if u.Scalar2[41] != siegeHireU136 || binary.LittleEndian.Uint16(u.Token[23:25]) != 0 {
			t.Fatalf("hire type %d U136 %d, token mask %d", typ, u.Scalar2[41], binary.LittleEndian.Uint16(u.Token[23:25]))
		}
		if u.Reference74 == 0 {
			t.Fatalf("hire type %d holds no weapon", typ)
		}
	}

	cold, coldApp, coldStore := cityGroupsColdApp(t, raw)
	if got := siegeMembers(cold); got[1] != 1 || got[2] != 1 || len(got) != 2 {
		t.Fatalf("cold LOAD holds siege members %v, want one of each type", got)
	}
	cityRosterSame(t, f, cold)
	if again, _ := siegeUnitsOf(t, cityRosterF2Save(t, coldApp, coldStore, "siege-again")); len(again[1]) != 1 || len(again[2]) != 1 {
		t.Fatalf("the cold town's SAVE writes %d and %d siege Units", len(again[1]), len(again[2]))
	}

	if err := coldApp.OpenMission(cold.MissionOpener(cold.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	liveSiegeHires(t, cold, types)
	_, mission := writeOrdinarySAV(t, cold, "siege-mission.sav")
	file, err := sav.Open(mission)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int32]int{}
	for _, a := range graph.Actors {
		if a.Class == "Unit" && types[int32(a.TypeID)] && a.OwnerSlot == 1 {
			seen[int32(a.TypeID)]++
		}
	}
	for typ := range types {
		if seen[typ] != 1 {
			t.Fatalf("mission SAVE holds %d player siege Units of type %d, want 1", seen[typ], typ)
		}
	}

	// Loss control: a town without the Ballista squad writes no Ballista Unit.
	cityGroupsToggle(t, cold, 2)
	if after, _ := siegeUnitsOf(t, currentTownSave(t, cold)); len(after[2]) != 0 || len(after[1]) != 1 {
		t.Fatalf("after returning the Ballista squad the SAV holds %d Ballista and %d Catapult Units", len(after[2]), len(after[1]))
	}
}

// The original's resave of the city kit holds the engine's legacy Human
// actor with Units row 0 beside its own rebuilt Unit. LOAD reads one siege
// member for the Ballista hire and none for the Human actor.
func TestReleaseOriginalCityResaveSiegeUnit(t *testing.T) {
	path := os.Getenv("AGAINROM_OWNER_KIT_RESAVE_CITY")
	if path == "" {
		t.Skip("no AGAINROM_OWNER_KIT_RESAVE_CITY: the owner's city resave is not committed")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if units, humans := siegeUnitsOf(t, raw); len(units[2]) != 1 || humans != 1 {
		t.Fatalf("resave holds %d Ballista Units and %d Human siege actors, want 1 and 1", len(units[2]), humans)
	}
	f, _, _ := cityGroupsColdApp(t, raw)
	if got := siegeMembers(f); got[2] != 1 || len(got) != 1 {
		t.Fatalf("LOAD holds siege members %v, want one Ballista", got)
	}
}

// A saved Unit on a siege row is a hired siege engine and persists; the legacy
// Human actor with Units row 0 and a Unit on any other row do not.
func TestReleaseOriginalSiegeUnitRecognition(t *testing.T) {
	f := releaseFront(t)
	rows := map[string]byte{}
	other := byte(0)
	for i := 1; i < f.Table.Units.Len(); i++ {
		switch name := f.Table.Units.EntryName(i); name {
		case "Catapult", "Ballista":
			rows[name] = byte(i)
		default:
			if other == 0 {
				other = byte(i)
			}
		}
	}
	for name, typ := range map[string]int{"Catapult": 1, "Ballista": 2} {
		got, ok := originalSiegeHire(sav.Character{Class: "Unit", DefRow: rows[name]}, f.Table)
		if !ok || got != typ || !persistentOriginalCharacter(sav.Character{Class: "Unit", DefRow: rows[name]}, f.Table) {
			t.Fatalf("%s Unit read as type %d, hire %v", name, got, ok)
		}
	}
	for _, c := range []sav.Character{
		{Class: "Human", DefRow: 0, Name: "Ballista"},
		{Class: "Unit", DefRow: other},
		{Class: "Unit", DefRow: 0},
	} {
		if _, ok := originalSiegeHire(c, f.Table); ok || persistentOriginalCharacter(c, f.Table) {
			t.Fatalf("%+v read as a hired siege engine", c)
		}
	}
}
