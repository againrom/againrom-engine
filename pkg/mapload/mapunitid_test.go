package mapload_test

// The authored map id, from the map record and from an original save's own
// party record to the simulation entity. Check opcode 9 answers with this
// value at run time, so a placement whose id does not reach the entity makes
// that arm answer 0 for every unit on the map and no test in pkg/sim can see
// it: that package builds its entities by hand.
//
// The digest fixtures in fromalm_test.go and gridform_test.go do not witness
// this. Every placement they carry leaves alm.Unit.UnitID at 0, so the two
// bytes the record grew are zeros there and a loader that dropped the field
// would reproduce every one of those pins exactly.

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestThePlacementsAuthoredMapIDReachesTheEntity is the FromALM direction. The
// ids are distinct from the entity ids the loader mints, and one placement
// declares none, so an entity carrying its own index or a blanket zero fails.
func TestThePlacementsAuthoredMapIDReachesTheEntity(t *testing.T) {
	t.Parallel()

	m := &alm.Map{
		Width: 24, Height: 24,
		Tiles:   make([]uint16, 24*24),
		Overlay: make([]uint8, 24*24),
		Units: []alm.Unit{
			{X: 10 << 8, Y: 10 << 8, ClassID: 7, UnitID: 51},
			{X: 11 << 8, Y: 10 << 8, ClassID: 7, UnitID: 0xfffe},
			{X: 12 << 8, Y: 10 << 8, ClassID: 7},
		},
	}
	w := mapload.FromALM(m)
	ents := w.Entities()
	if len(ents) != 3 {
		t.Fatalf("the loaded world holds %d entities, want 3", len(ents))
	}
	want := []uint16{51, 0xfffe, 0}
	for i, e := range ents {
		if e.MapUnitID != want[i] {
			t.Errorf("entity %d (index %d) carries map unit id %d, want the record's own %d",
				e.ID, i, e.MapUnitID, want[i])
		}
	}
	// And the id survives the byte form, because check opcode 9 answers it on
	// every tick of a mission and a mission is saved mid-run.
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := back.Entities(); !reflect.DeepEqual(got, ents) {
		t.Errorf("the round-tripped entities are %+v, want %+v", got, ents)
	}
}

// TestAPartyMembersAuthoredMapIDIsTheRestoredRecordsOwn is the StartMission
// direction, and it is the same fact ScriptUnits already binds the other way
// round: a member restored from an original save answers to the map record's
// identifier word, and a generated member answers to none.
//
// TestScriptUnitsBindsARestoredCharactersMapRecord pins the compile-time
// direction, id to entity. This is the run-time direction, entity to id, and
// the two have to agree or a script that names a character by his map record
// and a check that reads his pursued target's id describe different people.
func TestAPartyMembersAuthoredMapIDIsTheRestoredRecordsOwn(t *testing.T) {
	t.Parallel()

	drop := mapload.Cell{X: 12, Y: 12}
	m := startMap(t, 40, 40, drop)
	m.Units = []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 7, UnitID: 51}}

	// THE FOURTH MEMBER IS A SIEGE MERCENARY, which StartMission builds
	// through a different branch: tavern types 1 and 2 resolve through the
	// placement block instead of the generated-hero path, and that branch
	// constructs its own entity literal. Without him the other branch's write
	// is unwitnessed, and a member restored from a save that names his map
	// record would answer 0 to check opcode 9.
	party := []mapload.PartyMember{
		{Class: 100, Saved: &mapload.Saved{Cell: mapload.Cell{X: 10, Y: 11}, HP: 40, MaxHP: 40,
			MapUnitID: 21}},
		{Class: 100},
		{Class: 100, Saved: &mapload.Saved{Cell: mapload.Cell{X: 10, Y: 12}, HP: 40, MaxHP: 40}},
		{Class: 100, MercenaryType: 1, Saved: &mapload.Saved{Cell: mapload.Cell{X: 11, Y: 12},
			HP: 40, MaxHP: 40, MapUnitID: 33}},
	}
	w, _, err := mapload.StartMission(m, nil, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}

	byID := map[sim.EntityID]uint16{}
	for _, e := range w.Entities() {
		byID[e.ID] = e.MapUnitID
	}
	for i, want := range []uint16{21, 0, 0, 33} {
		id := mapload.PartyEntity(m, i)
		got, ok := byID[id]
		if !ok {
			t.Fatalf("party member %d has no entity %d in the started world", i, id)
		}
		if got != want {
			t.Errorf("party member %d (entity %d) carries map unit id %d, want %d",
				i, id, got, want)
		}
	}
	// The map's own placement keeps its record id in the same world, so the
	// two writers do not overwrite one another.
	if got := byID[0]; got != 51 {
		t.Errorf("the map placement carries map unit id %d, want 51", got)
	}
	// And the compile-time table agrees with the run-time field, id for id.
	if got := mapload.ScriptUnits(m, party); !reflect.DeepEqual(got,
		map[uint16]sim.EntityID{51: 0, 21: mapload.PartyEntity(m, 0), 33: mapload.PartyEntity(m, 3)}) {
		t.Errorf("ScriptUnits = %v, want the map's 51, the restored member's 21 and the "+
			"restored mercenary's 33", got)
	}
}
