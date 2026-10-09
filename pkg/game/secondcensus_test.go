package game

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func censusNode(opcode, id uint32, slots ...[2]uint32) alm.ScriptNode {
	n := alm.ScriptNode{Opcode: opcode, ID: id}
	for i, s := range slots {
		n.Type[i], n.Value[i] = s[0], s[1]
	}
	return n
}

func censusUnit(v uint32) [2]uint32 { return [2]uint32{4, v} }
func censusInt(v uint32) [2]uint32  { return [2]uint32{1, v} }

func TestSecondRefCauseSeparatesPartyArtifactsFromRealGaps(t *testing.T) {
	authored := []alm.Unit{{UnitID: 7}}
	for _, tc := range []struct {
		ref  mapload.ScriptUnresolved
		want SecondCause
	}{
		{mapload.ScriptUnresolved{Value: 10002}, CauseFixture},
		{mapload.ScriptUnresolved{Value: 11000}, CauseFixture},
		{mapload.ScriptUnresolved{Value: 8001}, CauseDynamic},
		{mapload.ScriptUnresolved{Value: 7}, CauseWithdrawn},
		{mapload.ScriptUnresolved{Value: 99}, CauseAbsent},
		{mapload.ScriptUnresolved{Value: 11001}, CauseAbsent},
		{mapload.ScriptUnresolved{Kind: mapload.ScriptRefStructure, Value: 10002}, CauseStructure},
	} {
		if got := SecondRefCause(tc.ref, authored); got != tc.want {
			t.Errorf("cause of %+v = %v, want %v", tc.ref, got, tc.want)
		}
	}
}

func TestSecondOmissionsAttributeEachTriggerToItsGreatestCause(t *testing.T) {
	src := alm.Script{
		Conditions: []alm.ScriptNode{
			censusNode(65538, 1, censusInt(0)),
			censusNode(5, 2, censusUnit(10002)),
			censusNode(5, 3, censusUnit(8001)),
			censusNode(5, 4, censusUnit(10003)),
		},
		Actions: []alm.ScriptNode{
			censusNode(2, 10, censusInt(1)),
			censusNode(19, 11, censusUnit(10002), [2]uint32{3, 1}),
		},
		Triggers: []alm.ScriptTrigger{
			{Left: [3]uint32{2}, Right: [3]uint32{1}, Acts: [4]uint32{10}},
			{Left: [3]uint32{3, 4}, Right: [3]uint32{1, 1}, Acts: [4]uint32{10}},
			{Left: [3]uint32{1}, Right: [3]uint32{1}, Acts: [4]uint32{10, 11}},
		},
	}
	refs := mapload.ScriptRefs{Hero: 5, HasHero: true}
	_, rep, err := mapload.CompileROM2ScriptFrom(src, refs)
	if err != nil {
		t.Fatal(err)
	}
	var c SecondMapCensus
	secondOmissions(&c, src, rep, nil)
	if want := (SecondOmission{Nodes: 3, Triggers: 1, ActionTriggers: 1}); c.Omitted[CauseFixture] != want {
		t.Errorf("fixture omissions = %+v, want %+v", c.Omitted[CauseFixture], want)
	}
	if want := (SecondOmission{Nodes: 1, Triggers: 1}); c.Omitted[CauseDynamic] != want {
		t.Errorf("dynamic omissions = %+v, want %+v: a trigger with a real cause is not a party artifact", c.Omitted[CauseDynamic], want)
	}
	if c.FixtureOnly() {
		t.Error("a dynamic reference reported as a party artifact")
	}

	_, full, err := mapload.CompileROM2ScriptFrom(src, secondSaturatedRefs(refs, 5))
	if err != nil {
		t.Fatal(err)
	}
	var saturated SecondMapCensus
	secondOmissions(&saturated, src, full, nil)
	if saturated.Omitted[CauseFixture] != (SecondOmission{}) || saturated.Omitted[CauseDynamic] != c.Omitted[CauseDynamic] {
		t.Errorf("saturated omissions = %+v, want only the dynamic cause", saturated.Omitted)
	}
}

func TestSecondGapsCountTheTriggersEachOpcodeHolds(t *testing.T) {
	checks := []sim.ScriptCheck{{Op: 99, Register: 0}}
	instants := []sim.ScriptInstant{{Op: 98}, {Op: sim.ScriptInstantWin}}
	none := sim.ScriptNone
	triggers := []sim.ScriptTrigger{
		{Pairs: [3]sim.ScriptPair{{Left: 0, Right: 1, Used: true}}, Instants: [4]int32{1, none, none, none}, Latch: 3},
		{Pairs: [3]sim.ScriptPair{{Left: 1, Right: 1, Used: true}}, Instants: [4]int32{0, none, none, none}, Latch: 4},
		{Pairs: [3]sim.ScriptPair{{Left: 1, Right: 1, Used: true}}, Instants: [4]int32{0, 1, none, none}, Latch: 5},
	}
	s, err := sim.NewROM2Script(checks, instants, triggers)
	if err != nil {
		t.Fatal(err)
	}
	var c SecondMapCensus
	secondGaps(&c, s)
	want := []SecondOpGap{{Check: true, Op: 99, Nodes: 1, Triggers: 1}, {Op: 98, Nodes: 1, Triggers: 2}}
	if !slices.Equal(c.Gaps, want) || !slices.Equal(c.Inert, []int{3}) {
		t.Fatalf("gaps %+v inert %v, want %+v inert [3]", c.Gaps, c.Inert, want)
	}
	if b := c.Blockers(); !slices.Contains(b, BlockOpcode) {
		t.Errorf("blockers %v name no opcode gap", b)
	}
}

type censusRows int

func (n censusRows) Len() int                { return int(n) }
func (censusRows) EntryName(int) string      { return "" }
func (censusRows) EntryParams(int) []int32   { return nil }
func (censusRows) EntryStrings(int) []string { return nil }

func TestSecondItemRowSelectsTheClassCollection(t *testing.T) {
	table := &mapload.Table{Weapons: censusRows(3), Armors: censusRows(2), MagicItems: censusRows(40)}
	for _, tc := range []struct {
		code uint16
		want bool
	}{
		{uint16(data.ComposeItemCode(0, 1, 0, 2)), true},
		{uint16(data.ComposeItemCode(0, 1, 0, 3)), false},
		{uint16(data.ComposeItemCode(0, 4, 0, 1)), true},
		{uint16(data.ComposeItemCode(0, 2, 0, 1)), false},
		{0x0e1e, true},
		{0x0e30, false},
		{0x0e00, false},
		{0x0d01, false},
	} {
		if got := SecondItemRow(tc.code, table); got != tc.want {
			t.Errorf("item %#04x has a row = %v, want %v", tc.code, got, tc.want)
		}
	}
	if SecondItemRow(0x0e1e, nil) {
		t.Error("no table resolved an item")
	}
}

func TestSecondControllerReachesAndContinuesOnlyItsMissions(t *testing.T) {
	reach := secondEngineReach()
	got := slices.Sorted(func(yield func(int) bool) {
		for n := range reach {
			if !yield(n) {
				return
			}
		}
	})
	if !slices.Equal(got, []int{10, 20, 21, 30, 31, 32}) {
		t.Fatalf("engine reaches %v, want [10 20 21 30 31 32]", got)
	}
	for _, tc := range []struct {
		n                 int
		producer          string
		entry, win        bool
		exits, engineAdds []string
	}{
		{n: 10, producer: "town 1 inn talk", entry: true, win: true, exits: []string{"mission 20:yes", "movie 1:yes"}, engineAdds: []string{"mission 20"}},
		{n: 20, producer: "departure of 10", entry: true, win: true, exits: []string{"town 2:yes", "mission 21 if 772!=0:yes"}, engineAdds: []string{"town 2", "mission 21"}},
		{n: 21, producer: "departure of 20", entry: true, win: true},
		{n: 30, producer: "stage 30 inn talk", entry: true, win: true, exits: []string{"movie 2:yes"}},
		{n: 31, producer: "stage 30 inn talk", entry: true, win: true, exits: []string{"mission 32:yes"}, engineAdds: []string{"mission 32"}},
		{n: 32, producer: "departure of 31", entry: true, win: true},
		{n: 40, win: true, exits: []string{"mission 50:yes", "mission 60:yes"}, engineAdds: []string{"mission 50", "mission 60"}},
		{n: 41, win: true},
		{n: 50, producer: "departure of 40", win: true, exits: []string{"town 3:yes", "unresolved record if 780!=0:no"}, engineAdds: []string{"town 3"}},
		{n: 60, producer: "departure of 40", win: true, exits: []string{"mission 80:yes"}, engineAdds: []string{"mission 80"}},
	} {
		d := secondDeparture(tc.n, reach)
		if d.Producer != tc.producer || d.EngineEntry != tc.entry || d.EngineWin != tc.win ||
			!slices.Equal(censusExits(d), tc.exits) || !slices.Equal(d.EngineAdds, tc.engineAdds) {
			t.Errorf("departure of %d = %+v, exits %v", tc.n, d, censusExits(d))
		}
	}
}

// censusExits prints each exit as "target[ if gates]:engine".
func censusExits(d SecondDeparture) []string {
	var out []string
	for _, e := range d.Exits {
		out = append(out, secondExitList([]SecondExit{e}, e.Engine)+":"+secondYes(e.Engine))
	}
	return out
}

// Every published exit stays, with its gates: both of mission 110's outputs,
// and the opposite bank777/bank778 gates on the outputs of 70 and 80. The
// engine produces each movie exactly under its gates; only mission 50's
// unresolved record stays unproduced.
func TestSecondDepartureKeepsEveryExitAndItsGates(t *testing.T) {
	reach := secondEngineReach()
	for n, want := range map[int][]string{
		70:  {"movie 3 if 777=0 778!=0:yes"},
		80:  {"movie 3 if 778=0 777!=0:yes"},
		110: {"movie 5 if 779!=0:yes", "movie 4 if 779=0:yes"},
		50:  {"town 3:yes", "unresolved record if 780!=0:no"},
	} {
		d := secondDeparture(n, reach)
		if got := censusExits(d); !slices.Equal(got, want) {
			t.Errorf("exits of %d = %v, want %v", n, got, want)
		}
		b := (SecondMapCensus{Mission: n, Departure: d}).Blockers()
		if slices.Contains(b, BlockMovie) || slices.Contains(b, BlockContinuation) != (n == 50) {
			t.Errorf("blockers of %d = %v", n, b)
		}
	}
	if !secondEngineExit(20, SecondExit{Target: "mission 21", Gates: []SecondGate{{772, true}}}) {
		t.Error("the engine's gated mission 21 add is not recognised")
	}
	if secondEngineExit(20, SecondExit{Target: "mission 21"}) || secondEngineExit(20, SecondExit{Target: "mission 21", Gates: []SecondGate{{772, false}}}) {
		t.Error("an exit the engine gates was accepted under another gate")
	}
}

func censusReady() SecondMapCensus {
	return SecondMapCensus{Mission: 20, Run: SecondRun{Ticks: SecondCensusTicks},
		Departure: secondDeparture(20, map[int]bool{20: true})}
}

func TestSecondBlockersNameEachClass(t *testing.T) {
	if b := censusReady().Blockers(); len(b) != 0 {
		t.Fatalf("a supported map has blockers %v", b)
	}
	for _, tc := range []struct {
		want   string
		mutate func(*SecondMapCensus)
	}{
		{BlockRun, func(c *SecondMapCensus) { c.Run.Err = "panic" }},
		{BlockHeadlessLoss, func(c *SecondMapCensus) { c.Run.Outcome = sim.OutcomeLost }},
		{BlockReference, func(c *SecondMapCensus) { c.Omitted[CauseAbsent].Nodes = 1 }},
		{BlockParty, func(c *SecondMapCensus) { c.Omitted[CauseFixture].Triggers = 1 }},
		{BlockDefinition, func(c *SecondMapCensus) { c.NoRowPlacements = 1 }},
		{BlockSpell, func(c *SecondMapCensus) { c.NoSpellRule = []uint16{29} }},
		{BlockEntry, func(c *SecondMapCensus) { c.Departure.EngineEntry = false }},
		{BlockContinuation, func(c *SecondMapCensus) { c.Departure.Exits[1].Engine = false }},
		{BlockMovie, func(c *SecondMapCensus) { c.Departure.Exits = append(c.Departure.Exits, SecondExit{Target: "movie 2"}) }},
		{BlockSave, func(c *SecondMapCensus) { c.Departure.EngineSave = false }},
	} {
		c := censusReady()
		tc.mutate(&c)
		if b := c.Blockers(); !slices.Equal(b, []string{tc.want}) {
			t.Errorf("blockers = %v, want [%s]", b, tc.want)
		}
	}
	if b := (SecondMapCensus{Err: "decode"}).Blockers(); !slices.Equal(b, []string{BlockStart}) {
		t.Errorf("a map that does not start has blockers %v", b)
	}
}

func TestWriteSecondCensusPrintsRowsAndTotals(t *testing.T) {
	blocked := censusReady()
	blocked.Mission, blocked.Omitted[CauseFixture] = 30, SecondOmission{Nodes: 2, Triggers: 1}
	var out bytes.Buffer
	if err := WriteSecondCensus(&out, []SecondMapCensus{censusReady(), blocked}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	header := strings.Count(lines[0], "\t")
	if len(lines) != 4 || strings.Count(lines[1], "\t") != header || strings.Count(lines[2], "\t") != header {
		t.Fatalf("census output does not hold a header, two rows of %d fields and totals:\n%s", header+1, out.String())
	}
	for _, want := range []string{"maps=2", "fixture=2/1/0", "fixture_maps=1", "ready=1"} {
		if !strings.Contains(lines[3], want) {
			t.Errorf("totals %q lack %q", lines[3], want)
		}
	}
	if !strings.HasSuffix(lines[2], "\tparty") {
		t.Errorf("row %q does not end with its blocker", lines[2])
	}
}
