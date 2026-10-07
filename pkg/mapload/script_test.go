package mapload_test

// The binder. Every fixture is an alm.Script built here from the documented
// record layout; nothing reads a map file.

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// node builds one script node out of (type, value) parameter slots, by SLOT —
// which is how the file stores them and is the whole point of several cases
// below.
func node(label string, opcode, id uint32, slots ...[2]uint32) alm.ScriptNode {
	n := alm.ScriptNode{Label: label, Opcode: opcode, ID: id}
	for i, s := range slots {
		n.Type[i] = s[0]
		n.Value[i] = s[1]
	}
	return n
}

// par is one filled parameter slot, and gap one left empty.
func par(typ, val uint32) [2]uint32 { return [2]uint32{typ, val} }

var gap = [2]uint32{0, 0}

// trg builds one trigger record.
func trg(name string, left, right, cmp [3]uint32, acts [4]uint32, once uint32) alm.ScriptTrigger {
	return alm.ScriptTrigger{Name: name, Left: left, Right: right, Cmp: cmp, Acts: acts, Once: once}
}

func compile(t *testing.T, src alm.Script, refs mapload.ScriptRefs) (*sim.Script, mapload.ScriptReport) {
	t.Helper()
	s, rep, err := mapload.CompileScriptFrom(src, refs)
	if err != nil {
		t.Fatalf("CompileScriptFrom: %v", err)
	}
	return s, rep
}

// ---------------------------------------------------------------------------
// AC-3 — the parameter block: plain integers PACKED IN ENCOUNTER ORDER, unit
// references lifted out into their own fields, unset slots skipped by both.
// ---------------------------------------------------------------------------

func TestPlainParametersArePackedInEncounterOrder(t *testing.T) {
	// Par0 is a unit reference, Par1 and Par2 a coordinate pair, Par3..Par8 are
	// empty and Par9 carries a group — the shape 125 shipped nodes have. The
	// coordinates must compile to plain parameters 0 and 1.
	src := alm.Script{
		Conditions: []alm.ScriptNode{
			node("near", 7, 4, par(4, 21), par(5, 36), par(6, 51),
				gap, gap, gap, gap, gap, gap, par(2, 18)),
		},
	}
	s, rep := compile(t, src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 5}})

	got := s.Checks()[0]
	if got.Args[0] != 36 || got.Args[1] != 51 {
		t.Errorf("the plain parameters are %v, want 36 and 51 at 0 and 1 — a binder keeping the "+
			"file's slot numbering would put them at 1 and 2", got.Args[:3])
	}
	if !got.HasUnit || got.Unit != 5 {
		t.Errorf("the unit reference is (%d, %v), want entity 5", got.Unit, got.HasUnit)
	}
	// The group reference at Par9 is carried by nothing here and is NOT reported:
	// no arm this build evaluates reads one, and the arm itself is already in the
	// script's own unsupported report.
	if !rep.Empty() {
		t.Errorf("the report is %+v, want empty", rep)
	}
}

func TestTheThreeTargetUnitBands(t *testing.T) {
	refs := mapload.ScriptRefs{
		Units:   map[uint16]sim.EntityID{21: 5},
		Hero:    9,
		HasHero: true,
	}
	tests := []struct {
		name    string
		value   uint32
		refs    mapload.ScriptRefs
		want    sim.EntityID
		resolve bool
	}{
		{"a placed unit", 21, refs, 5, true},
		{"a placed unit the map does not carry", 22, refs, 0, false},
		{"the hero ordinal", 10001, refs, 9, true},
		{"the hero ordinal with no hero", 10001,
			mapload.ScriptRefs{Units: refs.Units}, 0, false},
		{"a second hero ordinal", 10002, refs, 0, false},
		{"the top of the hero band", 11000, refs, 0, false},
		{"the static name table above it", 11001, refs, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := alm.Script{Conditions: []alm.ScriptNode{
				node("c", 7, 4, par(4, tc.value), par(5, 1), par(6, 2))}}
			s, rep := compile(t, src, tc.refs)
			got := s.Checks()[0]
			if got.HasUnit != tc.resolve || got.Unit != tc.want {
				t.Errorf("resolved to (%d, %v), want (%d, %v)",
					got.Unit, got.HasUnit, tc.want, tc.resolve)
			}
			// An unresolved reference is REPORTED, and a resolved one is not.
			if len(rep.Unresolved) != 0 && tc.resolve {
				t.Errorf("a resolved reference was reported: %+v", rep.Unresolved)
			}
			if len(rep.Unresolved) == 0 && !tc.resolve {
				t.Errorf("an unresolved reference was not reported")
			}
			// It is still BUILT and still owns its register, whichever way it went.
			if got.Register != 0 {
				t.Errorf("the check owns register %d, want 0", got.Register)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-3 — the three passes: registers in list order, ids to subscripts, the
// dropped trigger, the latch by map position, the build-time actions.
// ---------------------------------------------------------------------------

func TestTheThreePasses(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			// The drop table: consumed at build time, never an instant.
			node("Deploy", 0x10002, 100, par(5, 17), par(6, 66)),
			node("Send message", 2, 41, par(1, 15)),
			node("Win", 4, 7),
			// A build-time form the builder knows nothing about: discarded.
			node("Object", 0x10003, 55, par(1, 1)),
			node("Bad", 0x10004, 56),
		},
		Conditions: []alm.ScriptNode{
			node("FALSE", 0x10002, 1, par(7, 0)),
			node("dist", 7, 4, par(4, 21), par(5, 5), par(6, 6)),
			node("three", 0x10002, 12, par(7, 3)),
		},
		Triggers: []alm.ScriptTrigger{
			// Dropped whole: the first pair's LEFT id is zero.
			trg("Deploy Location", [3]uint32{0, 0, 0}, [3]uint32{0, 0, 0},
				[3]uint32{0, 0, 0}, [4]uint32{100, 0, 0, 0}, 1),
			// Built: node ids 4 and 12, actions 41 and 7.
			trg("Snoot", [3]uint32{4, 0, 0}, [3]uint32{12, 0, 0}, [3]uint32{5, 0, 0},
				[4]uint32{41, 7, 0, 0}, 1),
		},
	}
	s, rep := compile(t, src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 3}})

	// Pass 1: two instants, the drop consumed, two discarded.
	if got := s.Instants(); len(got) != 2 || got[0].Op != 2 || got[1].Op != 4 {
		t.Errorf("the instants are %+v, want the message and the win", got)
	}
	if want := []mapload.ScriptCell{{X: 17, Y: 66}}; !reflect.DeepEqual(rep.DropCells, want) {
		t.Errorf("the drop cells are %+v, want %+v", rep.DropCells, want)
	}
	if want := []uint32{0x10003, 0x10004}; !reflect.DeepEqual(rep.DiscardedActions, want) {
		t.Errorf("the discarded actions are %v, want %v", rep.DiscardedActions, want)
	}

	// Pass 2: registers in list order, constants and runtime checks alike.
	checks := s.Checks()
	if len(checks) != 3 {
		t.Fatalf("%d checks, want 3", len(checks))
	}
	for i, c := range checks {
		if c.Register != int32(i) {
			t.Errorf("check %d owns register %d", i, c.Register)
		}
	}
	if checks[0].Op != 0x10002 || checks[0].Args[0] != 0 {
		t.Errorf("the FALSE constant is %+v", checks[0])
	}
	if checks[2].Op != 0x10002 || checks[2].Args[0] != 3 {
		t.Errorf("the 3 constant is %+v", checks[2])
	}

	// Pass 3: one trigger built, one dropped whole, and the latch by MAP
	// position — the built trigger is at map position 1 and compiled index 0.
	if want := []int{0}; !reflect.DeepEqual(rep.DroppedTriggers, want) {
		t.Errorf("the dropped triggers are %v, want %v", rep.DroppedTriggers, want)
	}
	trs := s.Triggers()
	if len(trs) != 1 {
		t.Fatalf("%d triggers built, want 1", len(trs))
	}
	if trs[0].Latch != 1 {
		t.Errorf("the built trigger latches at %d, want its map position 1", trs[0].Latch)
	}
	if !trs[0].Once {
		t.Errorf("the fire-once flag did not cross")
	}
	// Its pair reads the registers node ids 4 and 12 were given: 1 and 2.
	want := sim.ScriptPair{Left: 1, Right: 2, Cmp: 5, Used: true}
	if trs[0].Pairs[0] != want {
		t.Errorf("the pair is %+v, want %+v", trs[0].Pairs[0], want)
	}
	if trs[0].Pairs[1].Used || trs[0].Pairs[2].Used {
		t.Errorf("an unset slot became a pair")
	}
	// And its actions read the subscripts node ids 41 and 7 were given: 0 and 1.
	if trs[0].Instants != [4]int32{0, 1, sim.ScriptNone, sim.ScriptNone} {
		t.Errorf("the instant slots are %v", trs[0].Instants)
	}
}

// TestATriggerSlotNamingAnIdThatWasNeverBuiltResolvesToZero is the id-to-subscript
// table's own miss value, and it is a behaviour rather than an error.
func TestATriggerSlotNamingAnIdThatWasNeverBuiltResolvesToZero(t *testing.T) {
	src := alm.Script{
		Actions:    []alm.ScriptNode{node("Win", 4, 7)},
		Conditions: []alm.ScriptNode{node("FALSE", 0x10002, 1, par(7, 0))},
		Triggers: []alm.ScriptTrigger{
			trg("T", [3]uint32{999, 0, 0}, [3]uint32{998, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{997, 0, 0, 0}, 0),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{})
	tr := s.Triggers()[0]
	if tr.Pairs[0].Left != 0 || tr.Pairs[0].Right != 0 {
		t.Errorf("the pair resolved to (%d,%d), want (0,0)", tr.Pairs[0].Left, tr.Pairs[0].Right)
	}
	if tr.Instants[0] != 0 {
		t.Errorf("the action slot resolved to %d, want 0", tr.Instants[0])
	}

	// With NOTHING built, subscript 0 names no instant and the slot is left
	// empty instead — the one case the miss value cannot be taken.
	bare := alm.Script{
		Conditions: []alm.ScriptNode{node("FALSE", 0x10002, 1, par(7, 0))},
		Triggers: []alm.ScriptTrigger{
			trg("T", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{997, 0, 0, 0}, 0),
		},
	}
	s2, _ := compile(t, bare, mapload.ScriptRefs{})
	if got := s2.Triggers()[0].Instants[0]; got != sim.ScriptNone {
		t.Errorf("with no instant built the slot is %d, want none", got)
	}
}

// TestAMapWithNoScriptCompilesToNone: an absent record is the loader's own
// skipped arm and not an error.
func TestAMapWithNoScriptCompilesToNone(t *testing.T) {
	s, rep, err := mapload.CompileScriptFrom(alm.Script{}, mapload.ScriptRefs{})
	if err != nil || s != nil || !rep.Empty() {
		t.Errorf("CompileScriptFrom(empty) = %v, %+v, %v; want nil, empty, nil", s, rep, err)
	}
	s, rep, err = mapload.CompileScript(nil, mapload.ScriptRefs{})
	if err != nil || s != nil || !rep.Empty() {
		t.Errorf("CompileScript(nil) = %v, %+v, %v; want nil, empty, nil", s, rep, err)
	}
}

// TestScriptUnitsPairsEachRecordsIdWithTheEntityItBecomes is AC-2's consumer
// half: the table that turns a below-band Target_Unit into an entity.
func TestScriptUnitsPairsEachRecordsIdWithTheEntityItBecomes(t *testing.T) {
	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0a80, Y: 0x0a80, UnitID: 21},
			{X: 0x0b80, Y: 0x0b80, UnitID: 51},
			// A DUPLICATE: the later record carrying the id keeps it.
			{X: 0x0c80, Y: 0x0c80, UnitID: 21},
			// A duplicate under a lesser owner does not displace a greater one.
			{X: 0x0d80, Y: 0x0d80, UnitID: 51, Owner: 0},
			{X: 0x0e80, Y: 0x0e80, UnitID: 77, Owner: 3},
			{X: 0x0f80, Y: 0x0f80, UnitID: 77, Owner: 2},
		},
	}
	m.Units[1].Owner = 2
	got := mapload.ScriptUnits(m, nil)
	want := map[uint16]sim.EntityID{21: 2, 51: 1, 77: 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ScriptUnits = %v, want %v", got, want)
	}
	// The entity ids really are the record indices FromALM assigns.
	w := mapload.FromALM(m)
	ents := w.Entities()
	if len(ents) != 6 || ents[0].ID != 0 || ents[1].ID != 1 {
		t.Fatalf("the loaded entities are %+v", ents)
	}
	if mapload.ScriptUnits(nil, nil) != nil {
		t.Errorf("ScriptUnits(nil) is not nil")
	}
}

// TestARestoredMembersUnitIdBindsToHisOwnEntity. A resume withdraws the map
// record a restored character was placed from, so the id the script names
// belongs to nobody on the map. The member carries the id and the table
// binds it to the entity he becomes.
func TestARestoredMembersUnitIdBindsToHisOwnEntity(t *testing.T) {
	// The map as the withdrawal leaves it: record 21 is gone.
	m := &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{
			{X: 0x0a80, Y: 0x0a80, UnitID: 51},
			{X: 0x0b80, Y: 0x0b80, UnitID: 52},
		},
	}
	party := []mapload.PartyMember{
		{Class: 100},
		{Class: 100, Saved: &mapload.Saved{Cell: mapload.Cell{X: 10, Y: 11}, MapUnitID: 21}},
	}
	got := mapload.ScriptUnits(m, party)
	want := map[uint16]sim.EntityID{51: 0, 52: 1, 21: mapload.PartyEntity(m, 1)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ScriptUnits = %v, want %v", got, want)
	}
	// The bound id is the entity a started world really gives that member.
	// That PartyEntity answers what StartMission assigns is pinned by
	// TestPartyEntityIsTheIdTheStartAssigns; what is asserted here is that the
	// binding uses the member's own position, so a reordered party binds the
	// id to the character who holds it and not to whoever is at that index.
	if got[21] != sim.EntityID(len(m.Units)+1) {
		t.Errorf("id 21 binds to %d, want the second party member's entity %d",
			got[21], len(m.Units)+1)
	}
	// A member carrying no Saved, and one whose Saved names no unit, add
	// nothing: a fresh start's table is the map's own.
	bare := []mapload.PartyMember{{Class: 100}, {Class: 100, Saved: &mapload.Saved{}}}
	if got := mapload.ScriptUnits(m, bare); !reflect.DeepEqual(got,
		map[uint16]sim.EntityID{51: 0, 52: 1}) {
		t.Errorf("a party carrying no unit id gave %v", got)
	}
}

// TestAWithdrawnUnitsProximityCheckMeasuresTheRestoredCharacter is 0148 AC-1,
// end to end over a started world.
//
// THE DEFECT IT PINS. 0147 withdraws the map record a restored character was
// placed from and accepted, as a disclosed divergence, that a script arm naming
// that record would resolve to no entity. An unresolved check is still built and
// still takes its register, and returns before writing anything — so the
// register keeps its initial zero and `r <= 3` is satisfied. The trigger fires
// on a distance nobody measured.
//
// THE TWO ARMS ARE THE SAME MAP, THE SAME SCRIPT AND THE SAME PARTY. Only the
// unit table differs: ScriptUnits with the party, which is this story, against
// ScriptUnits without it, which is what 0147 built. The second is what fires.
func TestAWithdrawnUnitsProximityCheckMeasuresTheRestoredCharacter(t *testing.T) {
	// The map as the withdrawal leaves it: nothing answers to unit 21.
	drop := mapload.Cell{X: 17, Y: 20}
	m := startMap(t, 60, 60, drop)

	// The restored character stands far from the point the check names, which
	// is the whole of what makes the two arms differ: measured, the distance
	// is large; unmeasured, the register is zero.
	saved := mapload.Cell{X: 50, Y: 51}
	party := []mapload.PartyMember{
		{Class: 100, Saved: &mapload.Saved{Cell: saved, HP: 40, MaxHP: 40, MapUnitID: 21}},
	}

	// dist(unit 21 -> (2,3)) <= const 3, winning.
	src := alm.Script{
		Conditions: []alm.ScriptNode{
			node("dist", 7, 1, par(4, 21), par(5, 2), par(6, 3)),
			node("three", 0x10002, 2, par(7, 3)),
		},
		Actions: []alm.ScriptNode{node("win", 4, 10)},
		Triggers: []alm.ScriptTrigger{trg("t", [3]uint32{1}, [3]uint32{2},
			[3]uint32{uint32(sim.ScriptCmpLE)}, [4]uint32{10}, 1)},
	}

	for _, tc := range []struct {
		name string
		refs []mapload.PartyMember
		want sim.Outcome
	}{
		{"bound to the restored character", party, sim.OutcomeUndecided},
		{"unbound, which is what 0147 built", nil, sim.OutcomeWon},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := compile(t, src, mapload.ScriptRefs{
				Units: mapload.ScriptUnits(m, tc.refs),
				Hero:  mapload.PartyEntity(m, 0), HasHero: true,
			})
			w, _, err := mapload.StartMissionScripted(m, nil, mapload.DifficultyNormal, party, s)
			if err != nil {
				t.Fatalf("StartMissionScripted: %v", err)
			}
			// One whole script cycle is 16 ticks, so a cycle guarantees
			// exactly one evaluation pass has run.
			for i := 0; i < 16; i++ {
				sim.Step(w, nil)
			}
			if got := w.Outcome(); got != tc.want {
				t.Errorf("outcome = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAMembersBindingWinsOverASurvivingRecord. The two collide only where a
// claimed record was not withdrawn, and the commandable figure is the one a
// script arm should measure.
func TestAMembersBindingWinsOverASurvivingRecord(t *testing.T) {
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 0x0a80, Y: 0x0a80, UnitID: 21}}}
	party := []mapload.PartyMember{
		{Class: 100, Saved: &mapload.Saved{Cell: mapload.Cell{X: 10, Y: 11}, MapUnitID: 21}},
	}
	got := mapload.ScriptUnits(m, party)
	if got[21] != mapload.PartyEntity(m, 0) {
		t.Errorf("id 21 binds to %d, want the party member's entity %d",
			got[21], mapload.PartyEntity(m, 0))
	}
}

// ---------------------------------------------------------------------------
// AC-8 — a compiled map's unimplemented arms, end to end through the binder.
// ---------------------------------------------------------------------------

// The check node was "population" (opcode 8) until 0122 implemented it as the
// population count; the sentinel moved to opcode 9 and then, when 1029
// implemented 9 as the pursued unit's authored map id, to opcode 20. Opcode 20
// is the sentinel rather than 4, 16 or 21 because those three are the whole of
// the milestone census's remaining count and a research experiment on them is
// queued, so any of them can become supported; no shipped campaign map authors
// a check of opcode 20 at all. The parameter shape is a player reference either
// way, so the binder path under test is unchanged.
func TestTheCompiledScriptReportsTheArmsThisBuildCannotEvaluate(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			// The unrunnable action was "Send message" (opcode 2) until 0169
			// implemented it; the sentinel moved to opcode 11, the item
			// transfer, which no shipped map authors.
			node("Transfer item", 11, 41, par(1, 15)), // not implemented
			node("Win", 4, 7),                         // implemented
		},
		Conditions: []alm.ScriptNode{
			node("not implemented", 20, 1, par(3, 1)), // not implemented
			node("FALSE", 0x10002, 2, par(7, 0)),      // implemented
		},
		Triggers: []alm.ScriptTrigger{
			trg("Kill", [3]uint32{1, 0, 0}, [3]uint32{2, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{41, 7, 0, 0}, 1),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{})

	gaps := s.Unsupported()
	if len(gaps) != 2 {
		t.Fatalf("Unsupported() is %+v, want the transfer arm and the sentinel check arm", gaps)
	}
	if got := s.InertTriggers(); !reflect.DeepEqual(got, []int32{0}) {
		t.Errorf("InertTriggers() is %v, want [0] — the trigger reads the unimplemented check's register", got)
	}

	// And the behaviour that matters: it does NOT fire. Left evaluated, its
	// unwritten register would read 0, the authored constant is 0, and the map
	// would be won on the first pass of the mission.
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 0}}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for i := 0; i < 40; i++ {
		sim.Step(w, nil)
	}
	if won, _ := w.ScriptCounters(); won != 0 {
		t.Errorf("the inert trigger fired: the win counter is %d", won)
	}
	if got := w.Outcome(); got != sim.OutcomeUndecided {
		t.Errorf("the outcome is %d, want undecided", got)
	}
}

// ---------------------------------------------------------------------------
// AC-1 — the whole road: bytes -> alm -> compiled program.
// ---------------------------------------------------------------------------

// TestTheWholeRoadFromMapBytes runs a script through alm.Open's own decode and
// out the other side, so the two halves are checked against each other rather
// than each against its own fixture.
func TestTheWholeRoadFromMapBytes(t *testing.T) {
	m := scriptFixtureMap(t)
	s, rep, err := mapload.CompileScript(m, mapload.ScriptRefs{
		Units: mapload.ScriptUnits(m, nil), Hero: 7, HasHero: true,
	})
	if err != nil {
		t.Fatalf("CompileScript: %v", err)
	}
	if len(s.Checks()) != 2 || len(s.Instants()) != 1 || len(s.Triggers()) != 1 {
		t.Fatalf("compiled %d checks, %d instants, %d triggers; want 2, 1, 1",
			len(s.Checks()), len(s.Instants()), len(s.Triggers()))
	}
	// The hero-band reference resolved through the caller's own answer.
	if c := s.Checks()[0]; !c.HasUnit || c.Unit != 7 {
		t.Errorf("the hero reference is (%d, %v), want entity 7", c.Unit, c.HasUnit)
	}
	if c := s.Checks()[0]; c.Args[0] != 66 || c.Args[1] != 16 {
		t.Errorf("the coordinate pair compiled to %v, want 66 and 16", c.Args[:2])
	}
	if want := []mapload.ScriptCell{{X: 17, Y: 66}}; !reflect.DeepEqual(rep.DropCells, want) {
		t.Errorf("the drop cells are %+v, want %+v", rep.DropCells, want)
	}
}

// ---------------------------------------------------------------------------
// The byte fixture for the road test: a whole .alm file with a type-7 payload,
// assembled here from the documented layout. It is the ONLY fixture in this file
// that goes through alm.Open, and it exists so that the decoder's answer and the
// binder's expectation are checked against each other rather than each against
// its own literal.
// ---------------------------------------------------------------------------

func sle32(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

func sjoin(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// srec wraps a payload in a 20-byte record header.
func srec(typeID uint32, payload []byte) []byte {
	return sjoin(sle32(7), sle32(20), sle32(uint32(len(payload))), sle32(typeID),
		sle32(0xBFC02B6D), payload)
}

// snode and strg build the two type-7 record shapes.
func snode(opcode, id uint32, slots ...[2]uint32) []byte {
	r := make([]byte, 796)
	copy(r[0x040:], sle32(opcode))
	copy(r[0x044:], sle32(id))
	for i, s := range slots {
		copy(r[0x04c+4*i:], sle32(s[1]))
		copy(r[0x074+4*i:], sle32(s[0]))
	}
	return r
}

func strg(left, right, cmp [3]uint32, acts [4]uint32, once uint32) []byte {
	r := make([]byte, 184)
	for i := 0; i < 3; i++ {
		copy(r[0x80+8*i:], sle32(left[i]))
		copy(r[0x84+8*i:], sle32(right[i]))
		copy(r[0xa8+4*i:], sle32(cmp[i]))
	}
	for i := 0; i < 4; i++ {
		copy(r[0x98+4*i:], sle32(acts[i]))
	}
	copy(r[0xb4:], sle32(once))
	return r
}

// scriptFixtureMap is a 4x4 map carrying one placed unit and a script whose win
// chain is the campaign's own shape: a drop location, a hero-band distance test
// against an authored constant, and a winning instant.
func scriptFixtureMap(t *testing.T) *alm.Map {
	t.Helper()

	meta := make([]byte, 632)
	copy(meta[0x00:], sle32(4))  // width
	copy(meta[0x04:], sle32(4))  // height
	copy(meta[0x1c:], sle32(0))  // #type5
	copy(meta[0x20:], sle32(0))  // #type4
	copy(meta[0x24:], sle32(1))  // #type6
	copy(meta[0x2c:], sle32(0))  // #type8 records
	copy(meta[0x30:], "fixture") // name

	unit := make([]byte, 70)
	copy(unit[0x00:], sle32(0x0180)) // X = cell 1
	copy(unit[0x04:], sle32(0x0180)) // Y = cell 1
	copy(unit[0x40:], []byte{21, 0}) // the id a Target_Unit names

	// Actions: the drop table, then the win.
	actions := sjoin(
		snode(0x10002, 100, [2]uint32{5, 17}, [2]uint32{6, 66}),
		snode(4, 7),
	)
	// Conditions: a hero-band distance test, then the authored 3.
	conds := sjoin(
		snode(7, 4, [2]uint32{4, 10001}, [2]uint32{5, 66}, [2]uint32{6, 16}),
		snode(0x10002, 12, [2]uint32{7, 3}),
	)
	triggers := sjoin(
		strg([3]uint32{4, 0, 0}, [3]uint32{12, 0, 0}, [3]uint32{5, 0, 0},
			[4]uint32{7, 0, 0, 0}, 1),
	)
	type7 := sjoin(sle32(2), actions, sle32(2), conds, sle32(1), triggers)

	payload := map[uint32][]byte{
		0: meta,
		1: make([]byte, 2*4*4),
		2: make([]byte, 4*4),
		3: make([]byte, 4*4),
		4: nil,
		5: nil,
		6: unit,
		7: type7,
		8: nil,
		9: sle32(0),
	}
	blobs := [][]byte{sjoin(sle32(0x0052374D), sle32(20), sle32(999), sle32(10), sle32(990))}
	for _, tid := range []uint32{0, 1, 2, 3, 5, 4, 9, 8, 6, 7} {
		blobs = append(blobs, srec(tid, payload[tid]))
	}

	m, err := alm.Open(sjoin(blobs...))
	if err != nil {
		t.Fatalf("alm.Open of the fixture: %v", err)
	}
	return m
}

// ---------------------------------------------------------------- raises (0066)

// The raise list is what a consumer announces from, and it exists because the
// compiled program cannot be read for the same answer without inventing
// announcements the map never authored. These pin both halves.

func TestCompileReportsEveryRaiseWithItsLatch(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("Msg1", 2, 10, par(7, 1)),
			node("Win", 4, 11),
			node("Msg2", 2, 12, par(7, 5)),
		},
		Conditions: []alm.ScriptNode{node("FALSE", 0x10002, 1, par(7, 0))},
		Triggers: []alm.ScriptTrigger{
			trg("t0", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{10, 0, 0, 0}, 1),
			trg("t1", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{12, 11, 0, 0}, 0),
		},
	}
	_, rep := compile(t, src, mapload.ScriptRefs{})

	want := []mapload.ScriptRaise{{Latch: 0, Event: 1}, {Latch: 1, Event: 5}}
	if !reflect.DeepEqual(rep.Raises, want) {
		t.Fatalf("Raises = %+v, want %+v — one row per message slot, in trigger then slot order",
			rep.Raises, want)
	}
}

// A BUILD-TIME action has no runtime slot. This is not hypothetical: every
// shipped map carries exactly one drop-location action, and the campaign's first
// mission has a trigger that names it.
func TestCompileDoesNotReportARaiseForABuildTimeAction(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("Msg1", 2, 10, par(7, 1)),
			node("Drop", 0x10002, 20, par(7, 17), par(7, 66)),
		},
		Conditions: []alm.ScriptNode{node("FALSE", 0x10002, 1, par(7, 0))},
		Triggers: []alm.ScriptTrigger{
			// The ONLY action this trigger names is the drop location.
			trg("t0", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{20, 0, 0, 0}, 1),
		},
	}
	s, rep := compile(t, src, mapload.ScriptRefs{})

	if len(rep.Raises) != 0 {
		t.Fatalf("Raises = %+v, want none — the trigger names no message action", rep.Raises)
	}
	tr := s.Triggers()
	if got := tr[0].Instants[0]; got != sim.ScriptNone {
		t.Fatalf("the build-time action left runtime slot %d, want none", got)
	}
	if got := s.Instants()[0].Op; got != 2 {
		t.Fatalf("instant 0 is opcode %d; this test needs it to be the message action", got)
	}
	// The drop cell still reached the report by its own route.
	if len(rep.DropCells) != 1 || rep.DropCells[0] != (mapload.ScriptCell{X: 17, Y: 66}) {
		t.Fatalf("DropCells = %+v, want one cell (17,66)", rep.DropCells)
	}
}

// A trigger the builder drops whole contributes nothing, and a map with no
// message action reports no raises at all.
func TestCompileReportsNoRaiseForADroppedOrSilentTrigger(t *testing.T) {
	src := alm.Script{
		Actions:    []alm.ScriptNode{node("Msg1", 2, 10, par(7, 3)), node("Win", 4, 11)},
		Conditions: []alm.ScriptNode{node("FALSE", 0x10002, 1, par(7, 0))},
		Triggers: []alm.ScriptTrigger{
			// Dropped whole: its first pair's left id is zero.
			trg("dropped", [3]uint32{0, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{10, 0, 0, 0}, 1),
			// Built, but names only the win action.
			trg("silent", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{11, 0, 0, 0}, 1),
		},
	}
	_, rep := compile(t, src, mapload.ScriptRefs{})
	if len(rep.Raises) != 0 {
		t.Fatalf("Raises = %+v, want none", rep.Raises)
	}
}

// ---------------------------------------------------------------------------
// AC-2 — the group a check names reaches the compiled check, with a presence
// flag of its own, and moves no plain parameter.
// ---------------------------------------------------------------------------

// TestTheGroupACheckNamesReachesTheCompiledCheck is AC-2's first two clauses:
// a node carrying a group parameter compiles to a check that carries that group
// and says so; a node carrying none compiles to a check that says so too.
//
// The group id 0 case is what separates the two: it is a real group, so the
// difference between "names group 0" and "names no group" cannot live in the
// value.
func TestTheGroupACheckNamesReachesTheCompiledCheck(t *testing.T) {
	src := alm.Script{
		Conditions: []alm.ScriptNode{
			node("count", 1, 1, par(2, 18)),
			node("count zero", 1, 2, par(2, 0)),
			node("no group", 1, 3),
			node("alive", 5, 4, par(4, 21)),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 3}})

	want := []struct {
		group uint32
		has   bool
	}{{18, true}, {0, true}, {0, false}, {0, false}}
	got := s.Checks()
	for i, w := range want {
		if got[i].Group != w.group || got[i].HasGroup != w.has {
			t.Errorf("check %d carries group %d/%v, want %d/%v",
				i, got[i].Group, got[i].HasGroup, w.group, w.has)
		}
	}
	// And the arm that names a unit and no group keeps its unit reference, so
	// the new field did not displace the old ones.
	if !got[3].HasUnit || got[3].Unit != 3 {
		t.Errorf("the unit check carries %v/%d, want true/3", got[3].HasUnit, got[3].Unit)
	}
}

// TestAGroupParameterMovesNoPlainParameter is AC-2's third clause, and it is the
// one that would fail if the group had been admitted to the plain-parameter
// packing instead of given a field.
//
// The node is the shape the corpus actually carries: a build-time literal, a
// GROUP, a unit reference and an integer, in that slot order. The two plain
// values must land in slots 0 and 1 with nothing between them.
func TestAGroupParameterMovesNoPlainParameter(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("group order", 6, 1, par(7, 15), par(2, 2), par(4, 10001), par(1, 3)),
		},
		Conditions: []alm.ScriptNode{
			node("in box", 2, 2, par(2, 4), par(5, 10), par(6, 20), par(5, 30), par(6, 40)),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{})

	if got := s.Instants()[0].Args[:3]; got[0] != 15 || got[1] != 3 || got[2] != 0 {
		t.Errorf("the action's plain parameters are %v, want 15 and 3 packed at 0 and 1", got)
	}
	c := s.Checks()[0]
	if got := c.Args[:5]; got[0] != 10 || got[1] != 20 || got[2] != 30 || got[3] != 40 || got[4] != 0 {
		t.Errorf("the check's plain parameters are %v, want the box packed at 0..3", got)
	}
	if c.Group != 4 || !c.HasGroup {
		t.Errorf("the check carries group %d/%v, want 4/true", c.Group, c.HasGroup)
	}
}

// TestOnlyTheFirstGroupSlotIsTaken pins the rule the binder states: a node
// carrying two group parameters keeps the first, as it keeps the first unit.
// No arm of the vocabulary names two, so this is about what a CUSTOMISED map
// gets rather than about anything that ships — and the alternative, silently
// taking the last, would depend on slot order in a way nothing declares.
func TestOnlyTheFirstGroupSlotIsTaken(t *testing.T) {
	src := alm.Script{
		Conditions: []alm.ScriptNode{node("two groups", 1, 1, par(2, 6), par(2, 9))},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{})
	if c := s.Checks()[0]; c.Group != 6 || !c.HasGroup {
		t.Errorf("the check carries group %d/%v, want 6/true", c.Group, c.HasGroup)
	}
}

// ---------------------------------------------------------------------------
// AC-3 — an INSTANT node's three references reach the compiled instant.
// ---------------------------------------------------------------------------

// TestTheThreeReferencesAnInstantNamesReachTheCompiledInstant is AC-3's first
// two clauses at once: a node carrying a unit, a group and a player compiles to
// three PRESENT references holding the node's own values, and a node carrying
// none compiles to three ABSENT ones.
//
// The three values are distinct and none is zero, so a reference read out of one
// of its two neighbours lands on a number this table names elsewhere. The unit
// resolves through the caller's table, so the value the compiled record carries
// is the ENTITY id and not the map's word — 21 goes in and 3 comes out, which is
// what says the unit reference was resolved rather than copied.
func TestTheThreeReferencesAnInstantNamesReachTheCompiledInstant(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("hand a unit over", 19, 1, par(4, 21), par(3, 2)),
			node("hand a group over", 22, 2, par(2, 6), par(3, 5)),
			node("all three", 8, 3, par(4, 21), par(2, 6), par(3, 5)),
			node("none of them", 4, 4),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 3}})

	want := []sim.ScriptInstant{
		{Op: 19, Unit: 3, HasUnit: true, Player: 2, HasPlayer: true},
		{Op: 22, Group: 6, HasGroup: true, Player: 5, HasPlayer: true},
		{Op: 8, Unit: 3, HasUnit: true, Group: 6, HasGroup: true, Player: 5, HasPlayer: true},
		{Op: 4},
	}
	got := s.Instants()
	if len(got) != len(want) {
		t.Fatalf("compiled %d instants, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("instant %d compiles to %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGiveMoneyCompilesItsAmountAndRecipient(t *testing.T) {
	src := alm.Script{Actions: []alm.ScriptNode{
		node("give money", uint32(sim.ScriptInstantGiveMoney), 1, par(3, sim.SelfSlot), par(1, 500)),
	}}
	s, _ := compile(t, src, mapload.ScriptRefs{})
	got := s.Instants()
	if len(got) != 1 {
		t.Fatalf("compiled %d instants, want 1", len(got))
	}
	if got[0].Op != sim.ScriptInstantGiveMoney || !got[0].HasPlayer || got[0].Player != sim.SelfSlot || got[0].Args[0] != 500 {
		t.Fatalf("give-money instant = %+v, want player 1 and amount 500", got[0])
	}
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("give-money instant remains unsupported: %+v", gaps)
	}
}

// TestPlayerZeroNamedIsNotNoPlayer is why the player reference carries a
// presence flag at all.
//
// A roster slot is 1-based, so zero already names nobody — the flag looks
// redundant. It is not, because the two arms that read it treat "the node named
// no player" as WRITE NOTHING and not as write zero, and only a flag keeps those
// apart. A node naming slot zero is a customised map's to make, and it must
// compile to a record the absent case cannot be mistaken for.
func TestPlayerZeroNamedIsNotNoPlayer(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("player zero", 19, 1, par(3, 0)),
			node("no player", 19, 2),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{})
	got := s.Instants()
	if got[0].Player != 0 || !got[0].HasPlayer {
		t.Errorf("the node naming player 0 compiles to %d/%v, want 0/true", got[0].Player, got[0].HasPlayer)
	}
	if got[1].HasPlayer {
		t.Errorf("the node naming no player compiles to present")
	}
	if got[0] == got[1] {
		t.Errorf("a node naming player 0 and a node naming none compile alike")
	}
}

// TestAnInstantsReferencesMoveNoPlainParameter is AC-3's third clause, and the
// one that would fail if the three had been admitted to the plain-parameter
// packing instead of given fields of their own.
//
// The node's slot order interleaves them deliberately — a literal, a unit, an
// integer, a group, an integer, a player — so a binder that packed any reference
// as a plain parameter would shift the three integers and this table would say
// which. The comparison is against the SAME node with its three references
// struck out, which is the claim exactly: their packing is unchanged.
func TestAnInstantsReferencesMoveNoPlainParameter(t *testing.T) {
	withRefs := alm.Script{Actions: []alm.ScriptNode{
		node("interleaved", 3, 1,
			par(7, 15), par(4, 10001), par(1, 3), par(2, 6), par(1, 4), par(3, 5)),
	}}
	plainOnly := alm.Script{Actions: []alm.ScriptNode{
		node("plain only", 3, 1, par(7, 15), par(1, 3), par(1, 4)),
	}}

	a, _ := compile(t, withRefs, mapload.ScriptRefs{})
	b, _ := compile(t, plainOnly, mapload.ScriptRefs{})
	if got, want := a.Instants()[0].Args, b.Instants()[0].Args; got != want {
		t.Errorf("the interleaved node packs %v, want the reference-free node's %v", got, want)
	}
	if got := a.Instants()[0].Args[:4]; got[0] != 15 || got[1] != 3 || got[2] != 4 || got[3] != 0 {
		t.Errorf("the plain parameters are %v, want 15, 3 and 4 packed at 0..2", got)
	}
}

// ---------------------------------------------------------------------------
// 0122 R-4 — the second player reference a CHECK may carry, and that filling
// it disturbs no plain parameter of an unrelated arm.
// ---------------------------------------------------------------------------

// TestAChecksTwoPlayerReferencesMoveNoPlainParameter is R-4: the seenPlayer
// counter T2 adds shares bindParams' one next index with every other case in
// the switch, so a regression here would show as a shifted plain parameter
// on a check that carries a player between two of them — not as a wrong
// player value, which is why the comparison is against the SAME node with
// its two player parameters struck out, on
// TestAnInstantsReferencesMoveNoPlainParameter's own pattern.
func TestAChecksTwoPlayerReferencesMoveNoPlainParameter(t *testing.T) {
	withRefs := alm.Script{Conditions: []alm.ScriptNode{
		node("interleaved", 10, 1,
			par(1, 15), par(3, 21), par(1, 3), par(2, 6), par(1, 4), par(3, 5)),
	}}
	plainOnly := alm.Script{Conditions: []alm.ScriptNode{
		node("plain only", 10, 1, par(1, 15), par(1, 3), par(1, 4)),
	}}

	a, _ := compile(t, withRefs, mapload.ScriptRefs{})
	b, _ := compile(t, plainOnly, mapload.ScriptRefs{})
	if got, want := a.Checks()[0].Args, b.Checks()[0].Args; got != want {
		t.Errorf("the interleaved node packs %v, want the reference-free node's %v", got, want)
	}
	if got := a.Checks()[0].Args[:3]; got[0] != 15 || got[1] != 3 || got[2] != 4 {
		t.Errorf("the plain parameters are %v, want 15, 3 and 4 packed at 0..2", got)
	}
	got := a.Checks()[0]
	if got.Player != 21 || !got.HasPlayer || got.Player2 != 5 || !got.HasPlayer2 {
		t.Errorf("the two player references are %d/%v and %d/%v, want 21/true and 5/true",
			got.Player, got.HasPlayer, got.Player2, got.HasPlayer2)
	}
}

// TestAnUnresolvableActionUnitReferenceIsReportedAndAbsent is AC-3's last
// clause. The binder already reported an action node's unresolvable unit
// reference and then threw the result away; now it keeps the result, and the
// report must be exactly what it was.
//
// The hero band is what makes this reachable without inventing a broken map: a
// caller with no party resolves nothing in it, which is 196 of the shipped
// corpus's references.
func TestAnUnresolvableActionUnitReferenceIsReportedAndAbsent(t *testing.T) {
	src := alm.Script{Actions: []alm.ScriptNode{
		node("hand the hero over", 19, 1, par(4, 10001), par(3, 1)),
	}}
	s, rep := compile(t, src, mapload.ScriptRefs{})

	in := s.Instants()[0]
	if in.HasUnit || in.Unit != 0 {
		t.Errorf("the unresolved reference compiles to %d/%v, want 0/false", in.Unit, in.HasUnit)
	}
	// The PLAYER beside it still resolved, so an absent unit is one field and not
	// the whole reference block.
	if in.Player != 1 || !in.HasPlayer {
		t.Errorf("the player beside it compiles to %d/%v, want 1/true", in.Player, in.HasPlayer)
	}
	want := []mapload.ScriptUnresolved{{Condition: false, NodeID: 1, Opcode: 19, Value: 10001}}
	if !reflect.DeepEqual(rep.Unresolved, want) {
		t.Errorf("the report names %+v, want %+v", rep.Unresolved, want)
	}
	// And with a hero to resolve against, the same node resolves and is silent.
	s2, rep2 := compile(t, src, mapload.ScriptRefs{Hero: 12, HasHero: true})
	if in := s2.Instants()[0]; !in.HasUnit || in.Unit != 12 {
		t.Errorf("with a hero the reference compiles to %d/%v, want 12/true", in.Unit, in.HasUnit)
	}
	if len(rep2.Unresolved) != 0 {
		t.Errorf("the report names %+v with a hero to resolve against, want nothing", rep2.Unresolved)
	}
}

// ---------------------------------------------------------------------------
// 0129 AC-5 — an ACTION node's SECOND unit reference reaches the compiled
// instant, on the same terms the condition pass has always carried it onto
// ScriptCheck. bindParams already resolved the pair for every node it binds;
// only CompileScriptFrom's action literal was throwing the second half away.
// ---------------------------------------------------------------------------

// TestAnActionNodesTwoUnitReferencesReachTheCompiledInstant is AC-5's first two
// clauses: a node naming two unit parameters compiles to an instant with both
// Unit/HasUnit and Unit2/HasUnit2 set to the two resolved entities, in
// parameter order, and a node naming only one leaves the second absent rather
// than defaulting it to the first's value.
func TestAnActionNodesTwoUnitReferencesReachTheCompiledInstant(t *testing.T) {
	refs := mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 3, 22: 6}}
	src := alm.Script{Actions: []alm.ScriptNode{
		// Give All (opcode 28): the giver first, the receiver second.
		node("give all", uint32(sim.ScriptInstantGiveAll), 1, par(4, 21), par(4, 22)),
		// One unit and a player beside it: no second unit parameter at all.
		node("one unit", 19, 2, par(4, 21), par(3, 5)),
	}}
	s, rep := compile(t, src, refs)

	got := s.Instants()
	if !got[0].HasUnit || got[0].Unit != 3 || !got[0].HasUnit2 || got[0].Unit2 != 6 {
		t.Errorf("the two-unit node compiles to Unit %d/%v, Unit2 %d/%v, want 3/true and 6/true",
			got[0].Unit, got[0].HasUnit, got[0].Unit2, got[0].HasUnit2)
	}
	if !got[1].HasUnit || got[1].Unit != 3 || got[1].HasUnit2 {
		t.Errorf("the one-unit node compiles to Unit %d/%v, Unit2 present %v, want the second absent",
			got[1].Unit, got[1].HasUnit, got[1].HasUnit2)
	}
	// The player beside the lone unit still carried, so the absent second
	// reference is one field going missing and not the whole record collapsing.
	if got[1].Player != 5 || !got[1].HasPlayer {
		t.Errorf("the player beside it is %d/%v, want 5/true", got[1].Player, got[1].HasPlayer)
	}
	if !rep.Empty() {
		t.Errorf("the report is %+v, want empty — both references resolve", rep)
	}
}

// TestAnActionNodesUnresolvableSecondUnitReferenceIsReportedAndAbsent is AC-5's
// third clause: a SECOND parameter that resolves to nothing leaves Unit2
// absent and is reported exactly as an unresolvable FIRST reference already is
// (TestAnUnresolvableActionUnitReferenceIsReportedAndAbsent) — bindParams'
// seenUnit counter does not special-case which slot failed, only which one it
// was.
func TestAnActionNodesUnresolvableSecondUnitReferenceIsReportedAndAbsent(t *testing.T) {
	src := alm.Script{Actions: []alm.ScriptNode{
		// 10002 is a second hero ordinal: this tree has one player, so it names
		// nobody (TestTheThreeTargetUnitBands's own case for that value).
		node("give all", uint32(sim.ScriptInstantGiveAll), 1, par(4, 21), par(4, 10002)),
	}}
	s, rep := compile(t, src, mapload.ScriptRefs{
		Units: map[uint16]sim.EntityID{21: 3}, Hero: 9, HasHero: true,
	})

	in := s.Instants()[0]
	if !in.HasUnit || in.Unit != 3 {
		t.Errorf("the first reference is %d/%v, want 3/true", in.Unit, in.HasUnit)
	}
	if in.HasUnit2 || in.Unit2 != 0 {
		t.Errorf("the unresolved second reference compiles to %d/%v, want 0/false", in.Unit2, in.HasUnit2)
	}
	want := []mapload.ScriptUnresolved{
		{Condition: false, NodeID: 1, Opcode: uint32(sim.ScriptInstantGiveAll), Value: 10002},
	}
	if !reflect.DeepEqual(rep.Unresolved, want) {
		t.Errorf("the report names %+v, want %+v", rep.Unresolved, want)
	}
}

func TestATargetItemParameterCompilesToAPackedItemCode(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("Add Cure", 12, 20, par(4, 10001), par(8, 6)),
			node("Remove Cure", 13, 21, par(4, 10001), par(8, 6)),
			node("Win", 4, 22),
		},
		Conditions: []alm.ScriptNode{node("TRUE", 0x10002, 1, par(7, 1))},
		Triggers: []alm.ScriptTrigger{
			trg("t", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{20, 21, 22, 0}, 1),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{Hero: 9, HasHero: true})

	got := s.Instants()
	if len(got) != 3 {
		t.Fatalf("the program holds %d instant(s), want 3", len(got))
	}
	for i, in := range got[:2] {
		if !in.HasItem || in.Item != 0x0e1e {
			t.Errorf("instant %d carries Item=%#04x HasItem=%v, want 0x0e1e and true — "+
				"0x0e18 + the authored 6", i, in.Item, in.HasItem)
		}
		if in.Unit != 9 || !in.HasUnit {
			t.Errorf("instant %d carries Unit=%d HasUnit=%v, want the resolved hero 9", i, in.Unit, in.HasUnit)
		}
		if in.Args != [10]int32{} {
			t.Errorf("instant %d carries plain parameters %v, want none — an item reference does "+
				"not join the encounter-order packing", i, in.Args)
		}
	}
	// The win names no item, so its flag must be clear. A binder that set the
	// flag on every node would make "named no item" and "named item code zero"
	// the same thing, and the arms answer those two the same way only by
	// accident.
	if got[2].HasItem || got[2].Item != 0 {
		t.Errorf("the win carries Item=%#04x HasItem=%v, want no item at all", got[2].Item, got[2].HasItem)
	}
}

func TestOnlyTheFirstTargetItemSlotIsTaken(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{node("two items", 12, 20, par(4, 10001), par(8, 6), par(8, 11))},
		Conditions: []alm.ScriptNode{
			node("TRUE", 0x10002, 1, par(7, 1)),
		},
		Triggers: []alm.ScriptTrigger{
			trg("t", [3]uint32{1, 0, 0}, [3]uint32{1, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{20, 0, 0, 0}, 1),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{Hero: 9, HasHero: true})

	if got := s.Instants()[0]; got.Item != 0x0e1e {
		t.Errorf("the node carries Item=%#04x, want 0x0e1e — the FIRST slot's 6, not the second's 11",
			got.Item)
	}
}

// ---------------------------------------------------------------------------
// 1029 B1 — the ITEM reference on a CONDITION node: the same decode 0156 built
// for an action, carried onto the compiled check instead of dropped.
// ---------------------------------------------------------------------------

// TestAConditionsTargetItemParameterCompilesToAPackedItemCode is B1 over the
// check record, on TestATargetItemParameterCompilesToAPackedItemCode's own
// pattern above. bindParams has decoded typeItem for both node kinds since
// 0156; what changed here is that the check builder stops dropping it.
//
// THE COMPILED CODE IS 0x0e1e AND NOT 6, for the same reason: check opcode 17
// compares the record's code against a container element's code with no mask,
// so a binder that copied the authored value across would put class 0 on the
// record and the arm would answer 0 for every shipped node that authors one.
func TestAConditionsTargetItemParameterCompilesToAPackedItemCode(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{node("Win", 4, 22)},
		Conditions: []alm.ScriptNode{
			node("Has Cure", 17, 1, par(4, 10001), par(8, 6)),
			node("TRUE", 0x10002, 2, par(7, 1)),
		},
		Triggers: []alm.ScriptTrigger{
			trg("t", [3]uint32{1, 0, 0}, [3]uint32{2, 0, 0}, [3]uint32{0, 0, 0},
				[4]uint32{22, 0, 0, 0}, 1),
		},
	}
	s, _ := compile(t, src, mapload.ScriptRefs{Hero: 9, HasHero: true})

	got := s.Checks()
	if len(got) != 2 {
		t.Fatalf("the program holds %d check(s), want 2", len(got))
	}
	if !got[0].HasItem || got[0].Item != 0x0e1e {
		t.Errorf("the condition carries Item=%#04x HasItem=%v, want 0x0e1e and true — "+
			"0x0e18 + the authored 6", got[0].Item, got[0].HasItem)
	}
	if got[0].Unit != 9 || !got[0].HasUnit {
		t.Errorf("the condition carries Unit=%d HasUnit=%v, want the resolved hero 9",
			got[0].Unit, got[0].HasUnit)
	}
	if got[0].Args != [10]int32{} {
		t.Errorf("the condition carries plain parameters %v, want none — an item reference does "+
			"not join the encounter-order packing", got[0].Args)
	}
	// The constant names no item, so its flag must be clear: "named no item"
	// and "named item code zero" are two records, and the arm answers 0 for
	// the first of them by its own branch rather than by comparison.
	if got[1].HasItem || got[1].Item != 0 {
		t.Errorf("the constant carries Item=%#04x HasItem=%v, want no item at all",
			got[1].Item, got[1].HasItem)
	}
	// And the trigger reading it is live, which is the whole point of B1: an
	// item reference the check record dropped left check opcode 17 unable to
	// answer, and this build no longer reports it as a gap.
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Errorf("Unsupported() is %+v, want none — check opcode 17 is implemented", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Errorf("InertTriggers() is %v, want none", inert)
	}
}

// ---------------------------------------------------------------------------
// THE TWO NUMBER SPACES A CONSUMER MUST NOT JOIN BY VALUE. A compiled check's
// subscript is its register subscript, assigned in condition list order. The
// authored node id is the map's own number, it is what ScriptReport.Unresolved
// carries, and the shipped corpus does not run it 0..n-1 over the conditions
// alone -- map 81 decodes 35 conditions whose ids reach 41.
//
// A consumer holding both reports has no other way to line them up, so the two
// facts below are what makes the join possible at all: subscript i is condition
// i with no skips, and Unresolved names the AUTHORED id beside the RAW
// reference value. cmd/almtool's `script` verb joins on exactly this.
// ---------------------------------------------------------------------------

// TestConditionSubscriptIsListOrderAndUnresolvedCarriesTheAuthoredNodeID pins
// both. The fixture's authored ids are deliberately neither 0-based nor
// ascending, so a compile that used the authored id as a subscript, or sorted
// the conditions, fails rather than agreeing by coincidence. Its middle
// condition names an unresolvable hero-band unit, which is also the case that
// would expose a builder that skipped a failed node instead of giving it a
// register.
func TestConditionSubscriptIsListOrderAndUnresolvedCarriesTheAuthoredNodeID(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{
		node("first", 17, 7, par(4, 21)),
		node("second", 7, 4, par(4, 10002)),
		node("third", 15, 19, par(4, 21)),
	}}
	refs := mapload.ScriptRefs{Units: map[uint16]sim.EntityID{21: 3}}
	s, rep := compile(t, src, refs)

	checks := s.Checks()
	if len(checks) != 3 {
		t.Fatalf("the compile makes %d check(s) from 3 conditions, want 3", len(checks))
	}
	// Subscript i is condition i: the register is the subscript, the opcodes
	// arrive in list order, and the failed middle node still took its register.
	for i, want := range []int32{17, 7, 15} {
		if checks[i].Op != want {
			t.Errorf("check %d carries opcode %d, want %d (list order)", i, checks[i].Op, want)
		}
		if checks[i].Register != int32(i) {
			t.Errorf("check %d takes register %d, want %d", i, checks[i].Register, i)
		}
	}
	// The two that resolved did, and the middle one did not -- so a consumer
	// cannot read the failure off HasUnit's position either.
	if !checks[0].HasUnit || checks[0].Unit != 3 {
		t.Errorf("check 0 resolves to %d/%v, want 3/true", checks[0].Unit, checks[0].HasUnit)
	}
	if checks[1].HasUnit {
		t.Errorf("check 1 resolves to %d/%v, want absent", checks[1].Unit, checks[1].HasUnit)
	}
	if !checks[2].HasUnit || checks[2].Unit != 3 {
		t.Errorf("check 2 resolves to %d/%v, want 3/true", checks[2].Unit, checks[2].HasUnit)
	}

	// The report names the AUTHORED id, 4, and not the subscript, 1. Those are
	// different numbers here on purpose: a report carrying the subscript would
	// pass a test whose fixture used 0-based ascending ids.
	want := []mapload.ScriptUnresolved{{Condition: true, NodeID: 4, Opcode: 7, Value: 10002}}
	if !reflect.DeepEqual(rep.Unresolved, want) {
		t.Fatalf("the report names %+v, want %+v", rep.Unresolved, want)
	}
	if rep.Unresolved[0].NodeID == 1 {
		t.Errorf("the report names the subscript rather than the authored node id")
	}
}
