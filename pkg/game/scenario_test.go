package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// scenarioBounds is this file's own map size. Every cell a fixture names is
// inside it and the numbers are stated here rather than taken from anything
// production carries.
var scenarioBounds = sim.Bounds{Width: 24, Height: 24}

// scenarioFighter is a unit that can be hit and can hit back. AlwaysHits removes
// the roll, so a scenario that waits for a unit to fall is decided by the
// scenario language rather than by a draw.
func scenarioFighter(id sim.EntityID, owner uint32, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Owner: owner, HP: 20, MaxHP: 20,
		ScanRange: 5, DamageBase: 3, AlwaysHits: true}
}

// scenarioWorld is a PlayWorld over a world built here, with both reference
// tables populated so uNN, pN and eNN all resolve.
func scenarioWorld(t *testing.T, ents ...sim.Entity) *PlayWorld {
	t.Helper()
	w, err := sim.NewWorld(1, scenarioBounds, sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return &PlayWorld{
		World:  w,
		Script: map[uint16]sim.EntityID{57: 1},
		Party:  []sim.EntityID{0},
	}
}

// writeScenario writes a scenario file and reads it back through the production
// reader, so a test drives the same parse, the same strictness and the same
// validation a command line would.
func writeScenario(t *testing.T, source string) HeadlessScenario {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := ReadHeadlessScenario(path)
	if err != nil {
		t.Fatalf("ReadHeadlessScenario: %v", err)
	}
	return s
}

func TestAMissionScenarioRunsEndToEndWithNoInstallPresent(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4), scenarioFighter(1, 9, 18, 18))
	s := writeScenario(t, `{
  "version": 2,
  "stage": "mission",
  "mission": 10,
  "steps": [
    {"command": "report", "name": "placed"},
    {"command": "assert_unit", "unit": "p0", "expect": {"alive": true, "x": 4, "y": 4}},
    {"command": "order", "unit": "p0", "order": "move", "x": 12, "y": 12},
    {"command": "wait_until", "ticks": 400, "until": {"unit": "p0", "within": {"x": 12, "y": 12, "radius": 1}}},
    {"command": "assert_unit", "unit": "p0", "expect": {"within": {"x": 12, "y": 12, "radius": 1}}},
    {"command": "assert_world", "world": {"outcome": "undecided", "unsupported_at_most": 0, "fallen_at_most": 0}},
    {"command": "report", "name": "arrived"}
  ]
}`)

	var machine, human bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, &human); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, human.String())
	}

	events := scenarioEvents(t, machine.Bytes())
	if len(events) != len(s.Steps) {
		t.Fatalf("%d event(s) for %d step(s)", len(events), len(s.Steps))
	}
	first, last := events[0], events[len(events)-1]
	if first.Name != "placed" || last.Name != "arrived" {
		t.Fatalf("report names = %q, %q", first.Name, last.Name)
	}
	if len(first.World.Units) != 2 || len(last.World.Units) != 2 {
		t.Fatalf("report steps carry %d and %d unit row(s), want 2 each",
			len(first.World.Units), len(last.World.Units))
	}
	// A step that is not a report carries the summary and no unit rows.
	if events[1].World.Units != nil {
		t.Fatalf("an assert step emitted %d unit row(s)", len(events[1].World.Units))
	}
	if last.World.Tick <= first.World.Tick {
		t.Fatalf("tick did not advance: %d -> %d", first.World.Tick, last.World.Tick)
	}
	// The party reference resolved to entity 0 and that entity moved. Reading
	// it off the report rather than off the world is deliberate: the machine
	// stream is the contract a consumer reads, so the test reads what ships.
	var moved bool
	for _, u := range last.World.Units {
		if u.Ref == "p0" {
			moved = u.X != 4 || u.Y != 4
		}
	}
	if !moved {
		t.Fatalf("p0 never left (4,4): %+v", last.World.Units)
	}
}

func scenarioEvents(t *testing.T, out []byte) []HeadlessEvent {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(out))
	var events []HeadlessEvent
	for dec.More() {
		var e HeadlessEvent
		if err := dec.Decode(&e); err != nil {
			t.Fatalf("decode event stream: %v", err)
		}
		if e.World == nil {
			t.Fatalf("step %d emitted no world state", e.Step)
		}
		events = append(events, e)
	}
	return events
}

func TestUnitReferencesResolveThroughScriptPartyAndEntity(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4), scenarioFighter(1, 9, 6, 4))
	for ref, want := range map[string]sim.EntityID{"p0": 0, "u57": 1, "e1": 1} {
		got, err := p.Resolve(ref)
		if err != nil || got != want {
			t.Fatalf("Resolve(%q) = %d, %v; want %d", ref, got, err, want)
		}
	}
	for _, ref := range []string{"", "x1", "u99", "p4", "uabc", "p-1"} {
		if _, err := p.Resolve(ref); err == nil {
			t.Fatalf("Resolve(%q) accepted a reference that names nothing", ref)
		}
	}
}

// TestAWaitThatNeverHoldsFailsAndNamesTheCondition is AC-3.
//
// A ceiling spent with the condition unmet has to be the failure. Reverting
// waitUntil's ceiling arm to a plain return makes this test the one that
// notices, because every later assertion in such a scenario still passes over a
// world that simply did not move.
func TestAWaitThatNeverHoldsFailsAndNamesTheCondition(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4))
	s := writeScenario(t, `{
  "version": 2, "stage": "mission", "mission": 10,
  "steps": [{"command": "wait_until", "ticks": 3, "until": {"unit": "p0", "within": {"x": 20, "y": 20, "radius": 0}}}]
}`)
	err := RunPlayScenario(p, s, new(bytes.Buffer), new(bytes.Buffer))
	if err == nil || !strings.Contains(err.Error(), "did not hold within 3 tick(s)") {
		t.Fatalf("RunPlayScenario = %v, want the unmet condition and its ceiling", err)
	}
	if !strings.Contains(err.Error(), "p0 within 0 of (20,20)") {
		t.Fatalf("the failure does not name the condition: %v", err)
	}
}

// TestAWorldAssertionReadsTheScriptGapCensus is AC-4.
//
// unsupported_at_most is the number pipeline/check-milestone.sh measures over a
// shipped mission. A scenario can now carry it, so a decode that regresses fails
// a written scenario rather than only a script somebody remembers to run.
func TestAWorldAssertionReadsTheScriptGapCensus(t *testing.T) {
	// Instant opcode 11, the item transfer, is one this build does not run,
	// which is what makes the compiled script report a gap. Opcode 2 held this
	// role until 0169 implemented it. The number is this fixture's own and is
	// asserted below through the census rather than assumed.
	script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: 11}}, []sim.ScriptTrigger{{Instants: [4]int32{0}}})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, scenarioBounds, sim.ModeCanonical, nil,
		[]sim.Entity{scenarioFighter(0, sim.SelfSlot, 4, 4)}, script)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	if n := len(w.Script().Unsupported()); n != 1 {
		t.Fatalf("the fixture reports %d unsupported arm(s), want 1 — instant op 11 is now implemented "+
			"or is no longer reported, and this test's premise has gone", n)
	}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	s := writeScenario(t, `{
  "version": 2, "stage": "mission", "mission": 10,
  "steps": [{"command": "assert_world", "world": {"unsupported_at_most": 0}}]
}`)
	err = RunPlayScenario(p, s, new(bytes.Buffer), new(bytes.Buffer))
	if err == nil || !strings.Contains(err.Error(), "1 arm(s) this build cannot run") {
		t.Fatalf("RunPlayScenario = %v, want the gap census reported", err)
	}
}

func TestTheReachedCensusCountsWhatTheDriveWalkedInto(t *testing.T) {
	// Opcode 11, the item transfer, is the arm this build does not run; opcode
	// 2 held that role until 0169 implemented it.
	script, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: 11}}, []sim.ScriptTrigger{{Instants: [4]int32{0}}})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, scenarioBounds, sim.ModeCanonical, nil,
		[]sim.Entity{scenarioFighter(0, sim.SelfSlot, 4, 4)}, script)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	s := writeScenario(t, `{
  "version": 2, "stage": "mission", "mission": 10,
  "steps": [
    {"command": "report", "name": "before"},
    {"command": "wait_ticks", "ticks": 64},
    {"command": "report", "name": "after"}
  ]
}`)
	var machine bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, new(bytes.Buffer)); err != nil {
		t.Fatalf("RunPlayScenario: %v", err)
	}
	events := scenarioEvents(t, machine.Bytes())
	before, after := events[0].World, events[len(events)-1].World
	if before.Unsupported != 1 || after.Unsupported != 1 {
		t.Fatalf("compiled count = %d then %d, want 1 throughout", before.Unsupported, after.Unsupported)
	}
	if before.Reached != 0 {
		t.Fatalf("reached count = %d before a tick has run, want 0", before.Reached)
	}
	if after.Reached == 0 {
		t.Fatal("the drive fired the trigger holding the unrunnable instant and reached nothing")
	}
	found := false
	for _, row := range after.Census {
		if row.Kind == "instant" && row.Op == 11 {
			found = row.Times == after.Reached
		}
	}
	if !found {
		t.Fatalf("the census does not account for the %d arrival(s): %+v", after.Reached, after.Census)
	}
	// A digest is recorded on every observation and moves with the world.
	if before.Hash == 0 || before.Hash == after.Hash {
		t.Fatalf("hash = %016x then %016x, want two different non-zero digests", before.Hash, after.Hash)
	}
}

func TestASyntheticWorldIsBuiltFromTheScenarioItself(t *testing.T) {
	s := writeScenario(t, `{
  "version": 2, "stage": "mission", "assets": "synthetic",
  "world": {"width": 16, "height": 16, "hostile": [[1,9],[9,1]], "units": [
    {"ref": "p0", "x": 2, "y": 2, "owner": 1, "hp": 40, "damage": 4, "scan_range": 5, "always_hits": true},
    {"ref": "u57", "x": 4, "y": 2, "owner": 9, "hp": 8, "damage": 1, "scan_range": 5, "always_hits": true}
  ]},
  "steps": [
    {"command": "order", "unit": "p0", "order": "attack", "target": "u57"},
    {"command": "wait_until", "ticks": 2000, "until": {"unit": "u57", "dead": true}},
    {"command": "assert_unit", "unit": "p0", "expect": {"alive": true, "owner": 1}},
    {"command": "assert_world", "world": {"alive_at_least": 1, "reached_unsupported_at_most": 0}}
  ]
}`)
	play, err := s.World.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := RunPlayScenario(play, s, new(bytes.Buffer), new(bytes.Buffer)); err != nil {
		t.Fatalf("RunPlayScenario: %v", err)
	}

	for _, tc := range []struct{ name, world, want string }{
		{"no units", `{"width":8,"height":8,"units":[]}`, "no units"},
		{"a unit outside the world",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":9,"y":1,"hp":10}]}`, "outside the world"},
		{"a repeated ref",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":10},{"ref":"p0","x":2,"y":1,"hp":10}]}`,
			"is used twice"},
		{"a party with a hole",
			`{"width":8,"height":8,"units":[{"ref":"p1","x":1,"y":1,"hp":10}]}`, "but no p0"},
		{"a unit with no health",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":0}]}`, "hp must be positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var spec HeadlessWorldSpec
			if err := json.Unmarshal([]byte(tc.world), &spec); err != nil {
				t.Fatal(err)
			}
			if _, err := spec.Build(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Build() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestAnAuthoredRotationSpeedReachesTheBuiltEntityOrDefaultsToTheCompatibilitySnap(t *testing.T) {
	var spec HeadlessWorldSpec
	raw := `{"width":8,"height":8,"units":[
		{"ref":"p0","x":1,"y":1,"hp":10,"rotation_speed":19},
		{"ref":"u1","x":2,"y":1,"hp":10}
	]}`
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		t.Fatal(err)
	}
	play, err := spec.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	byID := map[sim.EntityID]sim.Entity{}
	for _, e := range play.World.Entities() {
		byID[e.ID] = e
	}
	named := byID[play.Party[0]]
	if named.RotationSpeed != 19 {
		t.Errorf("p0 named rotation_speed 19, got RotationSpeed=%d", named.RotationSpeed)
	}
	unnamed := byID[play.Script[1]]
	if unnamed.RotationSpeed != 0 {
		t.Errorf("u1 named no rotation_speed, want the compatibility default 0, got RotationSpeed=%d",
			unnamed.RotationSpeed)
	}
}

func TestScenarioVocabularyIsGatedByVersionAndStage(t *testing.T) {
	cases := []struct {
		name string
		in   HeadlessScenario
		want string
	}{
		// THE VERSION GATE IS PER STAGE since 0163, so the reachable version
		// refusal is a command whose OWN stage got it later than the file
		// declares. A version-1 file naming a mission command is refused for its
		// stage first, because a version-1 file cannot declare the mission stage
		// at all -- the case below it.
		{"a front-end command in an older file",
			HeadlessScenario{Version: 2, Steps: []HeadlessStep{
				{Command: "create_character", Character: &HeadlessCharacter{Class: "Mage"}}}},
			`reached stage "frontend" in scenario version 3`},
		{"a front-end condition in an older file",
			HeadlessScenario{Version: 2, Steps: []HeadlessStep{
				{Command: "wait_until", Until: &HeadlessUntil{Screen: "town"}}}},
			`reached stage "frontend" in scenario version 3`},
		{"a play command in an older file",
			HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "report", Name: "x"}}},
			`belongs to stage "mission"`},
		{"a direct mission action in an older file",
			HeadlessScenario{Version: 5, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "place", Unit: "p0", X: int32Pointer(2), Y: int32Pointer(3)}}},
			`reached stage "mission" in scenario version 6`},
		{"a stage in an older file",
			HeadlessScenario{Version: 1, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "save"}}},
			"introduced in version 2"},
		{"a play command on the front-end stage",
			HeadlessScenario{Version: 2, Steps: []HeadlessStep{{Command: "report", Name: "x"}}},
			`belongs to stage "mission"`},
		{"a menu command on the mission stage",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "save"}}},
			`belongs to stage "frontend"`},
		{"a mission number on the front-end stage",
			HeadlessScenario{Version: 2, Mission: 10, Steps: []HeadlessStep{{Command: "save"}}},
			"belong to stage \"mission\""},
		{"a save directory on the mission stage",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10, Saves: "here",
				Steps: []HeadlessStep{{Command: "report", Name: "x"}}},
			"belong to stage \"frontend\""},
		{"a mission stage with no mission",
			HeadlessScenario{Version: 2, Stage: StageMission,
				Steps: []HeadlessStep{{Command: "report", Name: "x"}}},
			"requires a positive mission number"},
		{"an unknown stage",
			HeadlessScenario{Version: 2, Stage: "town", Steps: []HeadlessStep{{Command: "save"}}},
			"unknown stage"},
		{"an unknown difficulty",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10, Difficulty: "brutal",
				Steps: []HeadlessStep{{Command: "report", Name: "x"}}},
			"unknown difficulty"},
		{"an order with no destination",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "order", Unit: "p0", Order: "move"}}},
			`order "move" takes x and y`},
		{"an attack with no victim",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "order", Unit: "p0", Order: "attack"}}},
			`order "attack" takes target`},
		{"a condition naming two forms",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "wait_until", Until: &HeadlessUntil{
					Unit: "p0", Within: &HeadlessCell{}, Outcome: "won"}}}},
			"exactly one of within, dead or outcome"},
		{"an assertion that states nothing",
			HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
				Steps: []HeadlessStep{{Command: "assert_unit", Unit: "p0", Expect: &HeadlessUnitAssertion{}}}},
			"expect states nothing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.in.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.want)
			}
		})
	}

	// The two runners refuse each other's files rather than running half of
	// one: a stage is a property of the whole scenario, so a mission file
	// handed to the front-end runner is a caller error, not a step error.
	mission := HeadlessScenario{Version: 2, Stage: StageMission, Mission: 10,
		Steps: []HeadlessStep{{Command: "report", Name: "x"}}}
	front := HeadlessScenario{Version: 1, Steps: []HeadlessStep{{Command: "save"}}}
	if err := RunHeadlessScenario(&FrontEnd{}, nil, mission, new(bytes.Buffer), new(bytes.Buffer)); err == nil {
		t.Fatal("the front-end runner accepted a mission scenario")
	}
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4))
	err := RunPlayScenario(p, front, new(bytes.Buffer), new(bytes.Buffer))
	if err == nil || !strings.Contains(err.Error(), `stage "frontend"`) {
		t.Fatalf("the mission runner accepted a front-end scenario: %v", err)
	}
}

func int32Pointer(v int32) *int32 { return &v }

// TestDirectHeadlessActionsReachVictoryThroughTheMissionScript is story
// 1060's mechanism proof. Each direct action changes ordinary world state and
// advances a tick. Only the world's own script raises Victory after the final
// teleport; the scenario has no action that can write an outcome or latch.
func TestDirectHeadlessActionsReachVictoryThroughTheMissionScript(t *testing.T) {
	const item = uint16(77)
	checks := []sim.ScriptCheck{
		{Op: sim.ScriptCheckWithin, Register: 0, Unit: 0, HasUnit: true,
			Args: [10]int32{12, 12, 0}},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
	}
	instants := []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}
	triggers := []sim.ScriptTrigger{{
		Pairs:    [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
		Instants: [4]int32{0, -1, -1, -1}, Once: true,
	}}
	script, err := sim.NewScript(checks, instants, triggers)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	down := scenarioFighter(2, 1, 6, 6)
	down.HP = 0
	sim.PrepareAuthoredBody(&down)
	victim := scenarioFighter(1, 9, 8, 8)
	victim.DyingTime = 500 // kill must be dead to the script without spending this dwell.
	w, err := sim.NewStockedWorld(1060, scenarioBounds, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{
			scenarioFighter(0, sim.SelfSlot, 2, 2),
			victim,
			down,
			scenarioFighter(3, 9, 9, 8),
		}, script, sim.Relations{}, []sim.Sack{{X: 7, Y: 7, Items: []uint16{item}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	p := &PlayWorld{World: w, Script: map[uint16]sim.EntityID{57: 1, 58: 2, 59: 3}, Party: []sim.EntityID{0}}
	s := writeScenario(t, `{
  "version": 6, "stage": "mission", "mission": 10,
  "steps": [
    {"command":"heal","unit":"u58"},
    {"command":"assert_unit","unit":"u58","expect":{"alive":true,"hp_at_least":20}},
    {"command":"kill","unit":"u57"},
    {"command":"assert_unit","unit":"u57","expect":{"alive":false,"hp_at_most":-10}},
    {"command":"kill_player","player":9},
    {"command":"assert_unit","unit":"u59","expect":{"alive":false,"hp_at_most":-10}},
    {"command":"pick_item","unit":"p0","x":7,"y":7},
    {"command":"assert_unit","unit":"p0","expect":{"carries":[{"code":77,"count":1}]}},
    {"command":"place","unit":"p0","x":12,"y":12},
    {"command":"order","unit":"p0","order":"move","x":12,"y":12},
    {"command":"wait_until","ticks":64,"until":{"outcome":"won"}},
    {"command":"assert_world","world":{"outcome":"won"}}
  ]
}`)
	var machine, human bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, &human); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, human.String())
	}
	events := scenarioEvents(t, machine.Bytes())
	if got := events[len(events)-1].World.Outcome; got != "won" {
		t.Fatalf("final outcome = %q, want won", got)
	}
	if events[0].World.Tick == 0 || events[2].World.Tick <= events[0].World.Tick ||
		events[4].World.Tick <= events[2].World.Tick || events[6].World.Tick <= events[4].World.Tick ||
		events[8].World.Tick <= events[6].World.Tick {
		t.Fatalf("direct actions did not each advance the world: ticks %d %d %d %d %d",
			events[0].World.Tick, events[2].World.Tick, events[4].World.Tick,
			events[6].World.Tick, events[8].World.Tick)
	}
}

// TestAScenarioCanWaitForAUnitToFall is AC-6: the second condition form, over a
// fight the scenario itself orders.
func TestAScenarioCanWaitForAUnitToFall(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4), scenarioFighter(1, 9, 5, 4))
	s := writeScenario(t, `{
  "version": 2, "stage": "mission", "mission": 10,
  "steps": [
    {"command": "assert_unit", "unit": "u57", "expect": {"alive": true}},
    {"command": "order", "unit": "p0", "order": "attack", "target": "u57"},
    {"command": "wait_until", "ticks": 4000, "until": {"unit": "u57", "dead": true}},
    {"command": "assert_unit", "unit": "u57", "expect": {"alive": false, "hp_at_most": 0}},
    {"command": "assert_world", "world": {"alive_at_least": 1, "fallen_at_most": 1}}
  ]
}`)
	var human bytes.Buffer
	if err := RunPlayScenario(p, s, new(bytes.Buffer), &human); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, human.String())
	}
	if !strings.Contains(human.String(), "u57 dead=true after ") {
		t.Fatalf("the trace does not report the wait it satisfied:\n%s", human.String())
	}
}

func TestACarriesExpectationReadsTheUnitsOwnContainer(t *testing.T) {
	const cure uint16 = 0x0e1e

	// Two triggers: the first fires at once and adds, the second fires when
	// the unit reaches the far cell and takes it back. The condition is the
	// distance test mission 30's own win uses.
	script, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
			{Op: sim.ScriptCheckWithin, Register: 2, Unit: 0, HasUnit: true, Args: [10]int32{12, 12, 1}},
			{Op: sim.ScriptCheckConstant, Register: 3, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{
			{Op: sim.ScriptInstantAddItem, Unit: 0, HasUnit: true, Item: cure, HasItem: true},
			{Op: sim.ScriptInstantTakeItem, Unit: 0, HasUnit: true, Item: cure, HasItem: true},
			{Op: sim.ScriptInstantTakeItem, Item: cure, HasItem: true}, // the unresolved second carrier
			{Op: sim.ScriptInstantWin},
		},
		[]sim.ScriptTrigger{
			{Pairs: [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
				Instants: [4]int32{0, -1, -1, -1}, Once: true, Latch: 0},
			// The arrival trigger is trig[6]'s own shape: one distance test,
			// then take, take, WIN.
			{Pairs: [3]sim.ScriptPair{{Left: 2, Right: 3, Cmp: sim.ScriptCmpEQ, Used: true}},
				Instants: [4]int32{1, 2, 3, -1}, Once: true, Latch: 1},
		})
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, scenarioBounds, sim.ModeCanonical, nil,
		[]sim.Entity{scenarioFighter(0, sim.SelfSlot, 4, 4)}, script)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}

	s := writeScenario(t, `{
  "version": 2,
  "stage": "mission",
  "mission": 30,
  "steps": [
    {"command": "wait_ticks", "ticks": 40},
    {"command": "assert_unit", "unit": "p0", "expect": {"carries": [{"code": 3614, "count": 1}]}},
    {"command": "order", "unit": "p0", "order": "move", "x": 12, "y": 12},
    {"command": "wait_until", "ticks": 4000, "until": {"unit": "p0", "within": {"x": 12, "y": 12, "radius": 1}}},
    {"command": "wait_until", "ticks": 100, "until": {"outcome": "won"}},
    {"command": "assert_unit", "unit": "p0", "expect": {"carries": [{"code": 3614, "absent": true}]}},
    {"command": "assert_world", "world": {"outcome": "won"}}
  ]
}`)
	var machine, human bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, &human); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, human.String())
	}
}

func TestACarriesExpectationFailsWhenTheCodeIsNotThere(t *testing.T) {
	p := scenarioWorld(t, scenarioFighter(0, sim.SelfSlot, 4, 4))
	s := writeScenario(t, `{
  "version": 2,
  "stage": "mission",
  "mission": 30,
  "steps": [
    {"command": "assert_unit", "unit": "p0", "expect": {"carries": [{"code": 3614}]}}
  ]
}`)
	var machine, human bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, &human); err == nil {
		t.Fatal("a carries clause naming a code the unit does not hold was accepted")
	}
}

func TestTheShippedSpellScenarioRunsEndToEndWithNoInstallPresent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scenarios", "0154-synthetic-spells.json"))
	if err != nil {
		t.Fatalf("reading the shipped scenario: %v", err)
	}
	var s HeadlessScenario
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("decoding the shipped scenario: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("the shipped scenario does not validate: %v", err)
	}
	play, err := s.World.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var out, trace bytes.Buffer
	if err := RunPlayScenario(play, s, &out, &trace); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, trace.String())
	}
}

func TestTheSpellVocabularyRefusesTheFormsItCannotMean(t *testing.T) {
	for _, tc := range []struct{ name, world, want string }{
		{"an arm this build has no name for",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":10}],
			  "spells":[{"id":1,"arm":"curse"}]}`, "want damage, heal or nothing"},
		{"a spell id of zero",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":10}],
			  "spells":[{"id":0,"arm":"damage"}]}`, "names no row"},
		{"a repeated spell id",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":10}],
			  "spells":[{"id":1,"arm":"damage"},{"id":1,"arm":"heal"}]}`, "is used twice"},
		{"health above the maximum it is measured against",
			`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":20,"max_hp":10}]}`,
			"is above max_hp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var spec HeadlessWorldSpec
			if err := json.Unmarshal([]byte(tc.world), &spec); err != nil {
				t.Fatal(err)
			}
			if _, err := spec.Build(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Build() = %v, want error containing %q", err, tc.want)
			}
		})
	}

	for _, tc := range []struct{ name, step, want string }{
		{"a cast naming no spell",
			`{"command":"order","unit":"p0","order":"cast","target":"u57"}`,
			`takes a spell`},
		{"a cast naming spell 0",
			`{"command":"order","unit":"p0","order":"cast","target":"u57","spell":0}`,
			`names no row`},
		{"an autocast naming no spell",
			`{"command":"order","unit":"p0","order":"autocast"}`,
			`takes a spell`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var step HeadlessStep
			if err := json.Unmarshal([]byte(tc.step), &step); err != nil {
				t.Fatal(err)
			}
			if err := step.validateOrder(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateOrder() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestAUnitAuthoredHurtStaysHurt(t *testing.T) {
	var spec HeadlessWorldSpec
	if err := json.Unmarshal([]byte(
		`{"width":8,"height":8,"units":[{"ref":"p0","x":1,"y":1,"hp":20,"max_hp":60}]}`), &spec); err != nil {
		t.Fatal(err)
	}
	play, err := spec.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	e := play.World.Entities()[0]
	if e.HP != 20 || e.MaxHP != 60 {
		t.Errorf("the authored unit is %d/%d, want 20/60", e.HP, e.MaxHP)
	}
}

func TestPickItemWalksToTheSackAndTakesItUnderfoot(t *testing.T) {
	const item = uint16(77)
	w, err := sim.NewStockedWorld(1060, scenarioBounds, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{scenarioFighter(0, sim.SelfSlot, 2, 2)}, nil, sim.Relations{},
		[]sim.Sack{{X: 7, Y: 7, Items: []uint16{item}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	p := &PlayWorld{World: w, Party: []sim.EntityID{0}}
	s := writeScenario(t, `{
  "version": 6, "stage": "mission", "mission": 10,
  "steps": [
    {"command":"pick_item","unit":"p0","x":7,"y":7},
    {"command":"assert_unit","unit":"p0","expect":{"x":7,"y":7,"carries":[{"code":77,"count":1}]}}
  ]
}`)
	var machine, human bytes.Buffer
	if err := RunPlayScenario(p, s, &machine, &human); err != nil {
		t.Fatalf("RunPlayScenario: %v\n%s", err, human.String())
	}
	if len(w.Sacks()) != 0 {
		t.Fatalf("the sack stayed on the ground: %+v", w.Sacks())
	}
}
