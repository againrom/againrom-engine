package game_test

// Tests for the campaign entry. Every fixture is a synthetic archive written to
// a temp dir; no game install is read.

import (
	"encoding/binary"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/vfs"
)

// The address is the campaign identity, the number in decimal, and the map
// extension — and a number that is not a mission is refused before anything is
// opened.
func TestMissionMapAddress(t *testing.T) {
	for _, tc := range []struct {
		n    int
		want string
	}{{10, "scenario/10.alm"}, {1, "scenario/1.alm"}, {151, "scenario/151.alm"}} {
		got, ok := game.MissionMap(tc.n)
		if !ok || got != tc.want {
			t.Errorf("MissionMap(%d) = %q, %v; want %q, true", tc.n, got, ok, tc.want)
		}
	}
	for _, n := range []int{0, -1, -10} {
		if got, ok := game.MissionMap(n); ok {
			t.Errorf("MissionMap(%d) = %q, true; want no mission", n, got)
		}
	}
}

// missionFS is a container filesystem over a scenario archive holding the given
// entries.
func missionFS(t *testing.T, files []synth.File) *vfs.FS {
	t.Helper()
	dir := installDir(t, map[string][]byte{game.ScenarioArchive: synth.Archive(files)})
	fsys, err := vfs.Open([]string{dir + "/" + game.ScenarioArchive}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return fsys
}

// The whole path, end to end: a number reaches an archived map and comes back as
// a world with the party in it.
func TestStartMissionBuildsAWorldFromANumber(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 40, Height: 40})},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	})

	m, err := game.StartMission(fsys, 10, nil, mapload.DifficultyNormal,
		[]mapload.PartyMember{{Class: 1}, {Class: 2}})
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if m.Number != 10 || m.Address != "scenario/10.alm" {
		t.Errorf("mission %d at %q, want 10 at scenario/10.alm", m.Number, m.Address)
	}
	if m.Map == nil || m.Map.Width != 40 || m.Map.Height != 40 {
		t.Fatalf("the decoded map did not come back: %+v", m.Map)
	}
	if b := m.World.Bounds(); b.Width != 40 || b.Height != 40 {
		t.Errorf("world bounds %+v, want the map's 40x40", b)
	}
	if len(m.Start.Cells) != 2 || m.Start.Cells[0] != m.Start.Drop {
		t.Fatalf("start = %+v, want two cells with the hero on the drop cell", m.Start)
	}
	if got, want := len(m.World.Entities()), len(m.Map.Units)+2; got != want {
		t.Errorf("%d entities, want %d — the party is appended to the placements", got, want)
	}
}

// Mission 40's persistent npc22 companion is an owner-authored critical
// objective. This fixture carries only the decoded identity shape; no shipped
// map, character row, display name, or save bytes enter the test.
func TestMission40LosesWhenItsPersistentCompanionDies(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "40.alm", Data: synth.ALM(synth.ALMOptions{Width: 40, Height: 40})},
	})
	for _, tc := range []struct {
		name  string
		saved *mapload.Saved
	}{
		{"fresh carry", nil},
		{"original-save-shaped carry", &mapload.Saved{Cell: mapload.Cell{X: 12, Y: 13}, HP: 20, MaxHP: 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			party := []mapload.PartyMember{
				{ID: "hero", StartingHero: true, Class: 100},
				{ID: "npc:22", CompanionNPC: 22, PlayerCharacter: true, Class: 101, Saved: tc.saved},
				{ID: "temporary:3", Temporary: true, Class: 102},
			}
			m, err := game.StartMission(fsys, 40, nil, mapload.DifficultyNormal, party)
			if err != nil {
				t.Fatalf("StartMission: %v", err)
			}

			finish := func(id sim.EntityID) {
				hp := int32(0)
				found := false
				for _, e := range m.World.Entities() {
					if e.ID == id {
						hp = e.HP
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("world has no party entity %d", id)
				}
				// Check 18 represents completed teardown at -10. The debug Kill
				// command stops at the now-healable -1 band, so drive the same
				// damage path directly to the completed boundary.
				sim.Step(m.World, []sim.Command{{Kind: sim.KindDamage, Entity: id, X: hp + 10}})
			}

			// Another member's completed death does not satisfy the objective.
			finish(m.Start.IDs[2])
			for i := 0; i < 40; i++ {
				sim.Step(m.World, nil)
			}
			if got := m.World.Outcome(); got != sim.OutcomeUndecided {
				t.Fatalf("non-critical death decided mission as %v", got)
			}

			finish(m.Start.IDs[1])
			for i := 0; i < 40 && m.World.Outcome() == sim.OutcomeUndecided; i++ {
				sim.Step(m.World, nil)
			}
			if got := m.World.Outcome(); got != sim.OutcomeLost {
				t.Fatalf("critical companion death outcome = %v, want lost", got)
			}
		})
	}
}

// Each way it can fail names the address, and they are distinguishable: the
// three are three different things wrong with an install.
func TestStartMissionFailuresNameTheAddress(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "11.alm", Data: []byte("this is not a map")},
	})

	for _, tc := range []struct {
		name string
		n    int
		want string
	}{
		{"a number that is not a mission", 0, "not a campaign mission number"},
		{"a mission the archive does not hold", 10, "read scenario/10.alm"},
		{"an entry that will not decode", 11, "scenario/11.alm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := game.StartMission(fsys, tc.n, nil, mapload.DifficultyNormal, nil)
			if err == nil {
				t.Fatal("no error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not carry %q", err, tc.want)
			}
		})
	}

	// A refused difficulty is the fourth, and it too names the map it was
	// refused for.
	good := missionFS(t, []synth.File{{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 40, Height: 40})}})
	_, err := game.StartMission(good, 10, nil, mapload.Difficulty(9), nil)
	if err == nil || !strings.Contains(err.Error(), "scenario/10.alm") {
		t.Errorf("difficulty error = %v, want it to name scenario/10.alm", err)
	}
}

// No filesystem is a failure that still names what it would have read, rather
// than a nil dereference.
func TestStartMissionWithNoFilesystem(t *testing.T) {
	_, err := game.StartMission(nil, 10, nil, mapload.DifficultyNormal, nil)
	if err == nil || !strings.Contains(err.Error(), "scenario/10.alm") {
		t.Errorf("error = %v, want it to name the address", err)
	}
}

// The NPC registry reaches the table, so the arm that reads it can resolve at
// all. Loading it is what a front-end does once at construction.
func TestLoadTableCarriesTheNPCLookup(t *testing.T) {
	units := []synth.DataBinRow{{Name: "one", Params: make([]int32, 38)}}
	humans := []synth.DataBinRow{{Name: "a-human", Params: make([]int32, 38)}}
	tbl, err := game.LoadTable(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: synth.DataBinUnitsTable(units, humans)}},
		[]synth.File{{Path: "npc.reg", Data: synth.NPCReg(map[int32]int32{51: 509, 52: 510})}}))
	if err != nil {
		t.Fatalf("LoadTable: %v", err)
	}
	if got, ok := tbl.NPC.ServerID(51); !ok || got != 509 {
		t.Errorf("ServerID(51) = %d, %v; want 509, true", got, ok)
	}
	if tbl.NPC.Len() != 2 {
		t.Errorf("the lookup holds %d subscripts, want 2", tbl.NPC.Len())
	}
}

// An install without the NPC registry is an install that would put a generic
// peasant where a mission's dialogue names a witch, so it is a startup failure
// and not a table a caller may go on without.
func TestLoadTableRefusesAnInstallWithNoNPCRegistry(t *testing.T) {
	units := []synth.DataBinRow{{Name: "one", Params: make([]int32, 38)}}
	_, err := game.LoadTable(worldAndScenarioFS(t,
		[]synth.File{{Path: "data/data.bin", Data: synth.DataBinUnitsTable(units, nil)}},
		[]synth.File{{Path: "scenario.reg", Data: []byte("something else")}}))
	if err == nil {
		t.Fatal("an install with no npc.reg was accepted")
	}
	if !strings.Contains(err.Error(), game.NPCRegistry) {
		t.Errorf("error %q does not name %q", err, game.NPCRegistry)
	}
}

// The raise list a started mission publishes.
//
// It is the COMPILE'S answer because that is where authored action ids become
// runtime instant slots. What is pinned here is that mission start asks for the
// answer and carries what comes back.
func TestStartMissionCarriesTheCompilesAnswer(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 40, Height: 40})},
	})
	m, err := game.StartMission(fsys, 10, nil, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	// This builder writes a type-7 record whose body the script decoder refuses.
	// A start over such a map still SUCCEEDS — DropCells already answers a body
	// it cannot tile with no cells and no complaint, so refusing here would make
	// one story's tolerance another's error — and the reason is carried rather
	// than discarded.
	if m.RaiseErr == nil {
		t.Fatalf("a map whose script will not decode reported no reason; Raises = %+v", m.Raises)
	}
	if len(m.Raises) != 0 {
		t.Errorf("Raises = %+v, want none for a script that would not decode", m.Raises)
	}
}

// ---------------------------------------------------------------------------
// 0066 T7 — a started mission RUNS its script (AC-25).
//
// The fixture is a type-7 payload assembled here from the documented record
// layout, byte for byte: three counted arrays, each behind its own count word.
// The offsets are written out rather than imported, so a test asserting that a
// mission's world evaluates a script is not reading the layout out of the code
// that decodes it.
// ---------------------------------------------------------------------------

const (
	t7NodeSize    = 796
	t7TriggerSize = 184
)

func t7le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// t7node is one 796-byte action or condition record: an opcode, an identity, and
// (type, value) parameter slots written BY SLOT.
func t7node(opcode, id uint32, slots ...[2]uint32) []byte {
	rec := make([]byte, t7NodeSize)
	binary.LittleEndian.PutUint32(rec[0x40:], opcode)
	binary.LittleEndian.PutUint32(rec[0x44:], id)
	for i, s := range slots {
		binary.LittleEndian.PutUint32(rec[0x4c+4*i:], s[1])
		binary.LittleEndian.PutUint32(rec[0x74+4*i:], s[0])
	}
	return rec
}

// t7trigger is one 184-byte trigger record: one condition pair, its comparison,
// one action slot, and the fire-once word.
func t7trigger(left, right, cmp, act, once uint32) []byte {
	rec := make([]byte, t7TriggerSize)
	binary.LittleEndian.PutUint32(rec[0x80:], left)
	binary.LittleEndian.PutUint32(rec[0x84:], right)
	binary.LittleEndian.PutUint32(rec[0xa8:], cmp)
	binary.LittleEndian.PutUint32(rec[0x98:], act)
	binary.LittleEndian.PutUint32(rec[0xb4:], once)
	return rec
}

// missionScript is a payload a mission can actually run: a message action
// raising event 7, the drop action every shipped map carries, two constants both
// preset to 1, and one fire-once trigger comparing them for equality — so the
// trigger holds on the very first pass.
func missionScript() []byte {
	acts := concatBytes(
		// The message action, at identity 1. Its first plain parameter is the
		// event number, which is what the compile publishes as a raise.
		t7node(2, 1, [2]uint32{1, 7}),
		// The drop action, at identity 2: consumed by the builder, never
		// dispatched, and what puts the party at (17, 20) instead of the
		// fallback.
		t7node(0x10002, 2, [2]uint32{5, 17}, [2]uint32{6, 20}),
	)
	conds := concatBytes(
		t7node(0x10002, 1, [2]uint32{1, 1}), // register 0 = 1
		t7node(0x10002, 2, [2]uint32{1, 1}), // register 1 = 1
	)
	trigs := t7trigger(1, 2, 0, 1, 1) // register 0 == register 1, run action 1, once
	return concatBytes(t7le32(2), acts, t7le32(2), conds, t7le32(1), trigs)
}

func concatBytes(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// AC-25, at the door. A mission opened by number comes back with a world that
// RUNS its script: the trigger holds on the first pass, its latch is set, and
// the announcer raises the event the map authored. Until T7 the program was
// compiled and thrown away, so the latch stayed clear however long the world was
// stepped and no announcement could ever fire on a real install.
func TestStartMissionRunsTheMissionsScript(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{
			Width: 40, Height: 40, Type7Payload: missionScript()})},
	})
	m, err := game.StartMission(fsys, 10, nil, mapload.DifficultyNormal, game.MissionParty(nil, nil, nil))
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if m.RaiseErr != nil {
		t.Fatalf("the fixture's script did not decode: %v", m.RaiseErr)
	}
	if m.World.Script() == nil {
		t.Fatal("the mission's world runs no script — the join is not made")
	}
	if want := (mapload.ScriptRaise{Latch: 0, Event: 7}); len(m.Raises) != 1 || m.Raises[0] != want {
		t.Fatalf("Raises = %+v, want exactly %+v", m.Raises, want)
	}
	if m.Start.Fallback || m.Start.Drop != (mapload.Cell{X: 17, Y: 20}) {
		t.Errorf("start = %+v, want the map's own cell", m.Start)
	}

	// One whole script cycle is 16 ticks and the pass sits on a fixed phase of
	// it, so a cycle guarantees exactly one pass has run. The announcer is
	// sampled per step, which is the granularity a rising edge needs.
	a := game.NewAnnouncer(m.World, m.Raises)
	var raised []int32
	for i := 0; i < 16; i++ {
		sim.Step(m.World, nil)
		raised = append(raised, a.Sample(m.World)...)
	}
	if !m.World.ScriptLatched(0) {
		t.Errorf("latch 0 is clear after a whole script cycle — no trigger was evaluated")
	}
	if len(raised) != 1 || raised[0] != 7 {
		t.Fatalf("raised %v over one script cycle, want exactly [7]", raised)
	}
}

// A map whose script will not decode still starts, and its world runs none —
// the carried-error path is unchanged by the join.
func TestStartMissionWithAnUndecodableScriptRunsNone(t *testing.T) {
	fsys := missionFS(t, []synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 40, Height: 40})},
	})
	m, err := game.StartMission(fsys, 10, nil, mapload.DifficultyNormal, game.MissionParty(nil, nil, nil))
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if m.RaiseErr == nil {
		t.Fatal("the default fixture's script decoded; this test needs one that does not")
	}
	if m.World == nil || m.World.Script() != nil {
		t.Error("a mission whose script would not decode was given one to run")
	}
}

// ---------------------------------------------------------------------------
// 0069 — the mission binds its hero.
//
// The fixture is a map that can be WON only by the hero arriving: one condition
// measuring the distance from a fixed point to hero ordinal 1, one constant to
// compare it against, and a fire-once trigger forcing the mission-complete state
// when the distance is within it. The drop action is what moves the hero, so one
// script and two drop cells give the arrived and the not-arrived cases.
// ---------------------------------------------------------------------------

// The objective, and the radius the trigger accepts.
const (
	heroObjX, heroObjY = 30, 30
	heroObjRadius      = 3
)

// heroWinScript is that map's whole type-7 payload. dropX/dropY is where the
// party starts, which is the only thing that varies between the cases.
//
// unit is the Target_Unit value the condition names, so a caller can ask for an
// ordinal the band does not resolve as easily as for the hero.
func heroWinScript(dropX, dropY, unit uint32) []byte {
	acts := concatBytes(
		// Force "Mission Complete", at identity 1.
		t7node(4, 1),
		// The drop action, at identity 2: consumed by the builder.
		t7node(0x10002, 2, [2]uint32{5, dropX}, [2]uint32{6, dropY}),
	)
	conds := concatBytes(
		// Register 0 = distance from (heroObjX, heroObjY) to the named unit.
		// The unit slot is type 4 — Target_Unit — and the two coordinates are
		// the plain parameters the arm reads.
		t7node(7, 1, [2]uint32{4, unit}, [2]uint32{1, heroObjX}, [2]uint32{1, heroObjY}),
		// Register 1 = the constant the distance is compared against.
		t7node(0x10002, 2, [2]uint32{1, heroObjRadius}),
	)
	// Register 0 <= register 1, run action 1, once.
	trigs := t7trigger(1, 2, 5, 1, 1)
	return concatBytes(t7le32(2), acts, t7le32(2), conds, t7le32(1), trigs)
}

// heroWinMission starts that map with the party given. The placements are there
// so the hero's entity id is not zero: a binding that resolved to entity 0 by
// accident would pass every assertion below on a map that places nobody.
func heroWinMission(t *testing.T, dropX, dropY, unit uint32, p []mapload.PartyMember) *game.Mission {
	t.Helper()
	fsys := missionFS(t, []synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{
			Width: 40, Height: 40,
			Units: []synth.ALMUnit{
				{X: 0x0200, Y: 0x0200}, {X: 0x0300, Y: 0x0200}, {X: 0x0400, Y: 0x0200},
			},
			Type7Payload: heroWinScript(dropX, dropY, unit)})},
	})
	m, err := game.StartMission(fsys, 10, nil, mapload.DifficultyNormal, p)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if m.RaiseErr != nil {
		t.Fatalf("the fixture's script did not decode: %v", m.RaiseErr)
	}
	return m
}

// steppedOutcome runs the world for n whole script cycles and reports what it
// decided. A cycle is 16 ticks with the pass on a fixed phase, so n cycles is
// exactly n passes.
func steppedOutcome(w *sim.World, cycles int) sim.Outcome {
	for i := 0; i < 16*cycles; i++ {
		sim.Step(w, nil)
		if w.Outcome() != sim.OutcomeUndecided {
			return w.Outcome()
		}
	}
	return w.Outcome()
}

// THE MILESTONE, IN FOUR ROWS. A mission whose win requires the hero to reach a
// point must not be won while the hero is elsewhere, and must be won once it
// arrives — and the last two rows are the control that says why, because a test
// asserting only that the mission CAN be won passes just as well against the
// defect this story fixes.
//
// The control is a party of none. Nobody is bound then, by contract, so the
// distance check resolves nothing and writes no register; register 0 keeps its
// initial zero and 0 <= 3 holds, so the mission wins on the first pass with no
// hero in the world at all. That is exactly what EVERY mission did before this
// story, and it is asserted here so the false win cannot come back quietly.
func TestAMissionIsWonOnlyWhenTheHeroArrives(t *testing.T) {
	for _, tc := range []struct {
		name       string
		drop       [2]uint32
		party      []mapload.PartyMember
		wantWin    bool
		wantBound  bool
		whyItHolds string
	}{
		{name: "hero bound, standing far from the objective",
			drop: [2]uint32{8, 8}, party: game.MissionParty(nil, nil, nil),
			wantWin: false, wantBound: true,
			whyItHolds: "the distance is measured and exceeds the radius"},
		{name: "hero bound, standing on the objective",
			drop: [2]uint32{heroObjX, heroObjY}, party: game.MissionParty(nil, nil, nil),
			wantWin: true, wantBound: true,
			whyItHolds: "the distance is measured and is within the radius"},
		{name: "no party, standing far from the objective",
			drop: [2]uint32{8, 8}, party: nil,
			wantWin: true, wantBound: false,
			whyItHolds: "nothing is measured, the register keeps its zero, and zero is within"},
		{name: "no party, standing on the objective",
			drop: [2]uint32{heroObjX, heroObjY}, party: nil,
			wantWin: true, wantBound: false,
			whyItHolds: "nothing is measured, and the unwritten zero is within either way"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := heroWinMission(t, tc.drop[0], tc.drop[1], 10001, tc.party)
			checks := m.World.Script().Checks()
			if got := checks[0].HasUnit; got != tc.wantBound {
				t.Fatalf("the distance check's reference resolved = %v, want %v", got, tc.wantBound)
			}
			got := steppedOutcome(m.World, 4)
			if won := got == sim.OutcomeWon; won != tc.wantWin {
				t.Fatalf("outcome = %d, want won = %v — %s", got, tc.wantWin, tc.whyItHolds)
			}
		})
	}
}

// AC-2 — the bound entity is THE HERO and not merely some entity that exists.
// Asserted by cell as well as by id: an off-by-one that named the second party
// member would carry a plausible id and stand somewhere else.
func TestTheBoundHeroIsThePartysFirstMember(t *testing.T) {
	p := []mapload.PartyMember{{Class: 1}, {Class: 2}, {Class: 3}}
	m := heroWinMission(t, heroObjX, heroObjY, 10001, p)

	c := m.World.Script().Checks()[0]
	if !c.HasUnit {
		t.Fatal("the hero-band reference did not resolve on a mission started with a party")
	}
	if want := mapload.PartyEntity(m.Map, 0); c.Unit != want {
		t.Errorf("the check names entity %d, the party's first member is %d", c.Unit, want)
	}
	var hero sim.Entity
	for _, e := range m.World.Entities() {
		if e.ID == c.Unit {
			hero = e
		}
	}
	if hero.X != m.Start.Cells[0].X || hero.Y != m.Start.Cells[0].Y {
		t.Errorf("the named entity stands at (%d,%d), the start put the hero at %v",
			hero.X, hero.Y, m.Start.Cells[0])
	}
}

// AC-5 — the band is a subscript and this tree binds subscript 1 alone. An
// ordinal above 1 resolves to nobody whether or not a hero was supplied, which
// is why the fence in the contract is a fence and not a gap.
func TestAnOrdinalAboveOneStaysUnresolved(t *testing.T) {
	for _, ordinal := range []uint32{10002, 10006, 11000} {
		m := heroWinMission(t, heroObjX, heroObjY, ordinal, game.MissionParty(nil, nil, nil))
		if c := m.World.Script().Checks()[0]; c.HasUnit {
			t.Errorf("ordinal %d resolved to entity %d; only ordinal 1 is bound", ordinal, c.Unit)
		}
	}
}

// AC-8 — nothing without a party moves. A mission started with no party compiles
// exactly what it compiled before this story, so its world is the one the
// explicit no-hero compile builds, digest for digest.
func TestAMissionWithNoPartyKeepsItsDigest(t *testing.T) {
	m := heroWinMission(t, heroObjX, heroObjY, 10001, nil)

	s, _, err := mapload.CompileScript(m.Map, mapload.ScriptRefs{Units: mapload.ScriptUnits(m.Map, m.Party)})
	if err != nil {
		t.Fatalf("CompileScript: %v", err)
	}
	want, _, err := mapload.StartMissionScripted(m.Map, nil, mapload.DifficultyNormal, nil, s)
	if err != nil {
		t.Fatalf("StartMissionScripted: %v", err)
	}
	if got := m.World.Hash(); got != want.Hash() {
		t.Errorf("a mission with no party hashes %#x, the no-hero compile hashes %#x",
			got, want.Hash())
	}
}
