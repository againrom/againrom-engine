package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func secondGameScript(t *testing.T, checks []ScriptCheck, instants []ScriptInstant, triggers []ScriptTrigger) *Script {
	t.Helper()
	s, err := NewROM2Script(checks, instants, triggers)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSecondGameMessageExecutionOrderAndObserverParity(t *testing.T) {
	for _, sourceClock := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy clock", true: "source clock"}[sourceClock], func(t *testing.T) {
			testSecondGameMessageExecutionOrder(t, sourceClock)
		})
	}
}

func testSecondGameMessageExecutionOrder(t *testing.T, sourceClock bool) {
	t.Helper()
	s := secondGameScript(t, nil, []ScriptInstant{
		{Op: 2, Args: [10]int32{4}}, {Op: 36, Args: [10]int32{1, 2}}, {Op: 2, Args: [10]int32{3}},
	}, []ScriptTrigger{{Instants: acts(2, 1, 0)}})
	w := scriptWorld(t, s, nil)
	w.hasSessionClock = sourceClock
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var plain World
	if err := plain.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	var events []int32
	for n := 0; n < 23; n++ {
		events = append(events, StepReported(w, nil).ScriptMessages...)
		Step(&plain, nil)
		a, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		b, err := plain.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) || w.Hash() != plain.Hash() {
			t.Fatalf("report changed canonical state at tick%d", n)
		}
	}
	if !reflect.DeepEqual(events, []int32{3, 253, 4, 3, 4}) {
		t.Fatalf("actual messages=%v", events)
	}
}

func TestSecondGameProgramReplacementKeepsDialectAndExecutionState(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: 35, Args: [10]int32{800, 29}}}, nil)
	withVIP, err := s.WithVIP(7)
	if err != nil || withVIP.Dialect() != ScriptROM2 || len(withVIP.Unsupported()) != 0 {
		t.Fatalf("VIP changed dialect/support: %v %v", withVIP, err)
	}
	w := scriptWorld(t, s, nil)
	w.runInstant(s.Instants()[0])
	w.latches[3] = 1
	if err := w.RestoreScriptProgram(withVIP); err != nil {
		t.Fatal(err)
	}
	if value, ok := w.ROM2ScenarioValue(800); !ok || value != 29 || w.latches[3] != 1 || w.script.Dialect() != ScriptROM2 {
		t.Fatal("same-dialect replacement reset execution state")
	}
	rom1, err := NewScript(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreScriptProgram(rom1); err == nil || w.script.Dialect() != ScriptROM2 {
		t.Fatal("restore accepted a dialect change")
	}
	controlled, err := NewControlledScriptWorld(w, s)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := controlled.ROM2ScenarioValue(800); !ok || value != 0 || controlled.script.Dialect() != ScriptROM2 || controlled.latches[3] != 0 {
		t.Fatal("controlled replacement did not start fresh ROM2 execution")
	}
}

func TestSecondGameObjectiveModeStoresAndNotifications(t *testing.T) {
	for _, tc := range []struct {
		old, mode, want int32
		event           []int32
	}{
		{0, 1, 1, nil}, {3, 1, 3, nil}, {0, 2, 3, []int32{253}}, {1, 2, 3, []int32{253}},
		{3, 2, 3, nil}, {4, 2, 4, nil}, {4, 4, 5, []int32{254}}, {5, 4, 5, []int32{254}}, {-1, -9, -9, nil},
	} {
		s := secondGameScript(t, nil, []ScriptInstant{{Op: 35, Args: [10]int32{753, tc.old}}, {Op: 36, Args: [10]int32{1, tc.mode}}}, []ScriptTrigger{{Instants: acts(0, 1), Once: true}})
		w := scriptWorld(t, s, nil)
		var messages []int32
		for n := 0; n < 7; n++ {
			messages = append(messages, StepReported(w, nil).ScriptMessages...)
		}
		if got, ok := w.ROM2ScenarioValue(753); !ok || got != tc.want || !reflect.DeepEqual(messages, tc.event) {
			t.Fatalf("old%d mode%d value%d/%v messages%v", tc.old, tc.mode, got, ok, messages)
		}
	}
	for _, index := range []int32{-1, 1024, 1<<31 - 1, -1 << 31} {
		w := scriptWorld(t, secondGameScript(t, nil, nil, nil), nil)
		w.runInstant(ScriptInstant{Op: 35, Args: [10]int32{index, 77}})
		w.runInstant(ScriptInstant{Op: 36, Args: [10]int32{index, 4}})
		if _, ok := w.ROM2ScenarioValue(index); ok {
			t.Fatalf("admitted index%d", index)
		}
	}
}

func TestSecondGameDiagnosticInstantKeepsDefaultNoop(t *testing.T) {
	w := scriptWorld(t, secondGameScript(t, nil, []ScriptInstant{{Op: 37}}, nil), nil)
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var obs castObs
	w.runInstantObserved(ScriptInstant{Op: 37, Args: [10]int32{1, 2, 3, 4, 5, 6, 7, 8}}, &obs)
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || len(obs.scriptMessages) != 0 {
		t.Fatal("bad instant altered state or fabricated event")
	}
}

func TestSecondGameCheckCenteredKeepsNativeScalarQuirk(t *testing.T) {
	for _, tc := range []struct {
		args    [10]int32
		transit int32
		fine    uint8
		want    int32
	}{
		{[10]int32{99, 3, 4}, 0, 128, 1}, {[10]int32{3, 4}, 0, 128, 0},
		{[10]int32{99, 3, 4}, 1, 128, 0}, {[10]int32{99, 3, 4}, 0, 127, 0},
	} {
		s := secondGameScript(t, []ScriptCheck{{Op: 27, Register: 0, Unit: 1, HasUnit: true, Args: tc.args}}, nil, nil)
		w := scriptWorld(t, s, []Entity{{ID: 1, X: 3, Y: 4, HP: 10, MaxHP: 10}})
		w.entities[0].Transit = uint16(tc.transit)
		if tc.fine != 128 {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true, Position: SavedActorPosition{FineX: tc.fine, FineY: 128}}}}
		}
		w.scriptPass(nil)
		if got := w.ScriptRegister(0); got != tc.want {
			t.Fatalf("args%v transit%d fine%d result%d want%d", tc.args[:3], tc.transit, tc.fine, got, tc.want)
		}
	}
}

func TestSecondGameEffectChecksReadCurrentMembership(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: 25, Register: 0, Args: [10]int32{257, 1, 0}},
		{Op: 25, Register: 1, Args: [10]int32{1, 2, 2}},
		{Op: 25, Register: 2, Args: [10]int32{8, 8, 0}},
		{Op: 25, Register: 3, Args: [10]int32{1, 2, 1}},
		{Op: 26, Register: 4, Unit: 1, HasUnit: true, Args: [10]int32{44}},
		{Op: 26, Register: 5, Unit: 2, HasUnit: true, Args: [10]int32{12}},
	}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 1, HP: 10, MaxHP: 10}})
	w.effects = []cellEffect{{Key: cellKey(8, 8), Spell: 3, Remaining: 30, Cells: []uint16{cellKey(1, 2)}}, {Key: cellKey(8, 8), Spell: 6, Remaining: 30, Cells: []uint16{cellKey(1, 2)}}}
	w.attached = []attachedEffect{{Target: 1, Spell: 12, Kind: EffectBless, Mode: EffectDuration, Remaining: 30}}
	w.scriptPass(nil)
	for i, want := range []int32{1, 1, 0, 0, 1, 0} {
		if got := w.ScriptRegister(int32(i)); got != want {
			t.Fatalf("register%d=%d want%d", i, got, want)
		}
	}
	for i := range w.effects {
		w.effects[i].Remaining = 1
	}
	w.attached[0].Remaining = 1
	Step(w, nil)
	if len(w.effects) != 0 || len(w.attached) != 0 {
		t.Fatal("ordinary tick retained expired effect membership")
	}
	w.scriptPass(nil)
	for i := 0; i < 6; i++ {
		if got := w.ScriptRegister(int32(i)); got != 0 {
			t.Fatalf("removed membership register%d=%d", i, got)
		}
	}
}

func TestSecondGameTakeItemVisitsExtantOnMapEntitiesOnce(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: 38, Item: itCure, HasItem: true}}, []ScriptTrigger{{Instants: acts(0), Once: true}})
	ents := []Entity{{ID: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 1, HP: -1, MaxHP: 10}, {ID: 3, X: 2, HP: 10, MaxHP: 10, OffMap: true}}
	stock := []Stock{{ID: 1, Items: []uint16{itCure, itCure, itCure}, OrderedStacks: []ItemStack{StackItem(PlainItem(itCure), 3)}}, {ID: 2, Items: []uint16{itCure}, OrderedStacks: []ItemStack{StackItem(PlainItem(itCure), 1)}}, {ID: 3, Items: []uint16{itCure, itCure}, OrderedStacks: []ItemStack{StackItem(PlainItem(itCure), 2)}}}
	w, err := NewStockedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, ents, s, Relations{}, nil, stock)
	if err != nil {
		t.Fatal(err)
	}
	w.scriptPass(nil)
	for i, want := range []uint32{2, 0, 2} {
		stacks, ok := w.CarriedStacks(EntityID(i + 1))
		if !ok {
			t.Fatal("missing entity")
		}
		got := uint32(0)
		for _, stack := range stacks {
			got += stack.Count
		}
		if got != want {
			t.Fatalf("entity%d count%d want%d", i+1, got, want)
		}
	}
}

func TestSecondGameFailureReasonWinsAndFirstGameDialectStaysInert(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: 4}, {Op: 4}, {Op: 5, Args: [10]int32{19}}}, []ScriptTrigger{{Instants: acts(0, 1, 2), Once: true}})
	w := scriptWorld(t, s, nil)
	scriptTicks(w, 16, nil)
	if won, lost := w.ScriptCounters(); won != 2 || lost != 19 || w.Outcome() != OutcomeLost {
		t.Fatalf("counters%d/%d outcome%v", won, lost, w.Outcome())
	}
	first, err := NewScript(s.Checks(), []ScriptInstant{{Op: 35}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Dialect() != ScriptROM1 || len(first.Unsupported()) != 1 {
		t.Fatal("ROM1 admitted ROM2 operation")
	}
}

func TestSecondGameDeclaredArmsExecute(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: 23, Register: 0, Args: [10]int32{779}},
		{Op: 24, Register: 1, Args: [10]int32{1}},
		{Op: 25, Register: 2},
		{Op: 26, Register: 3, Unit: 1, HasUnit: true},
		{Op: 27, Register: 4, Unit: 1, HasUnit: true},
	}, []ScriptInstant{
		{Op: 35, Args: [10]int32{779, -7}},
		{Op: 36, Args: [10]int32{1, 2}},
		{Op: 37}, {Op: 38, Item: 0xe1e, HasItem: true},
		{Op: 39, Group: 9, HasGroup: true},
	}, []ScriptTrigger{{Instants: acts(0, 1, 2, 3), Once: true}, {Instants: acts(4), Once: true, Latch: 1}})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("declared ROM2 arms unsupported: %+v", gaps)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Group: 9, Owner: 2}})
	scriptTicks(w, 23, nil)
	if got := w.ScriptRegister(0); got != -7 {
		t.Fatalf("check23 bank result=%d, want -7", got)
	}
	if got := w.ScriptRegister(1); got != 3 {
		t.Fatalf("check24 objective result=%d, want 3", got)
	}
}
