package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentScriptFixture(t *testing.T) (*FrontEnd, Snapshot, *sim.World) {
	t.Helper()
	f, snapshot, source := currentPlayerSlotFixture(t, 1)
	program, err := sim.NewScript([]sim.ScriptCheck{
		{Op: sim.ScriptCheckConstant, Register: 10, Args: [10]int32{1}},
		{Op: sim.ScriptCheckRelation, Register: 40, Player: 5, Player2: 1, HasPlayer: true, HasPlayer2: true},
		{Op: sim.ScriptCheckVIP, Register: 41, Unit: 1, HasUnit: true},
		{Op: sim.ScriptCheckAlive, Register: 42, Unit: 701, HasUnit: true},
	}, []sim.ScriptInstant{
		{Op: sim.ScriptInstantIncVariable, Args: [10]int32{70}},
		{Op: sim.ScriptInstantSetVariable, Args: [10]int32{10, 17}},
		{Op: sim.ScriptInstantWin}, {Op: sim.ScriptInstantLose},
	}, []sim.ScriptTrigger{{Instants: [4]int32{0, 1, 2, 3}, Once: true, Latch: 18}})
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewControlledScriptWorld(source, program)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 256 && !w.ScriptLatched(18); i++ {
		sim.Step(w, nil)
	}
	won, lost := w.ScriptCounters()
	if !w.ScriptLatched(18) || w.ScriptRegister(10) != 17 || w.ScriptRegister(70) != 1 || won != 1 || lost != 1 {
		t.Fatal("live program did not establish independent register/latch/counter discriminators")
	}
	snapshot.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return f, snapshot, w
}

func coldCurrentScript(t *testing.T, f *FrontEnd, raw []byte) *FrontEnd {
	t.Helper()
	cold := cellStateFront(t)
	cold.Campaign, cold.Table = f.Campaign, f.Table
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("current mission program").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal("cold current program LOAD", town, err)
	}
	return cold
}

func checkCurrentScriptState(t *testing.T, want, got *sim.World) {
	t.Helper()
	a, b := want.Script(), got.Script()
	if !slices.Equal(a.Checks(), b.Checks()) || !slices.Equal(a.Instants(), b.Instants()) || !slices.Equal(a.Triggers(), b.Triggers()) ||
		want.ScriptRegisters() != got.ScriptRegisters() || want.Outcome() != got.Outcome() {
		t.Fatal("current program, complete registers or outcome changed")
	}
	wa, la := want.ScriptCounters()
	wb, lb := got.ScriptCounters()
	if wa != wb || la != lb {
		t.Fatal("current script counters changed")
	}
	for i := int32(0); i < 1000; i++ {
		if want.ScriptLatched(i) != got.ScriptLatched(i) {
			t.Fatal("current trigger latch changed", i)
		}
	}
	aIDs, bIDs := []sim.EntityID{}, []sim.EntityID{}
	for _, e := range want.Entities() {
		aIDs = append(aIDs, e.ID)
	}
	for _, e := range got.Entities() {
		bIDs = append(bIDs, e.ID)
	}
	if !slices.Equal(aIDs, bIDs) {
		t.Fatal("script actor identities changed after manifest restore", aIDs, bIDs)
	}
	if *want.CurrentPolicy().EntityIDFloor != *got.CurrentPolicy().EntityIDFloor {
		t.Fatal("script-only absent endpoint changed the actor identity floor")
	}
}

func TestCurrentScriptProgramSAVCyclesAndNextPass(t *testing.T) {
	f, snapshot, expected := currentScriptFixture(t)
	before := expected.Hash()
	raw, err := f.ExportCurrentSave(snapshot, "live program absent from installed map")
	if err != nil || expected.Hash() != before {
		t.Fatal("SAVE changed live World", err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := coldCurrentScript(t, f, raw)
		checkCurrentScriptState(t, expected, cold.live.world)
		passed := false
		for tick := 0; tick < 256; tick++ {
			a, b := sim.StepTraced(expected, nil), sim.StepTraced(cold.live.world, nil)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("cycle %d next script trace differs at tick %d: %+v / %+v", cycle, tick, a, b)
			}
			checkCurrentScriptState(t, expected, cold.live.world)
			if a.Pass {
				if len(a.Checks) != 4 || len(a.Firings) != 0 || cold.live.world.ScriptRegister(70) != 1 {
					t.Fatal("next pass lost a current check or refired latch 18", a)
				}
				passed = true
				break
			}
		}
		if !passed {
			t.Fatal("next script pass was not observed")
		}
		if cycle == 0 {
			snapshot, _, err = cold.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err = cold.ExportCurrentSave(snapshot, "second current program save")
			if err != nil {
				t.Fatal(err)
			}
			f = cold
		}
	}
}

func TestCurrentScriptProgramLossControls(t *testing.T) {
	f, snapshot, expected := currentScriptFixture(t)
	raw, err := f.ExportCurrentSave(snapshot, "program loss controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"absent", "empty", "VIP", "relation", "latch", "ordinary register"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a.Program == nil {
				t.Fatal("missing program discriminator", err)
			}
			switch change {
			case "absent":
				a.Program = nil
			case "empty":
				a.Program = &currentScriptProgram{}
			case "VIP":
				a.Program.Checks = slices.Delete(a.Program.Checks, 2, 3)
			case "relation":
				a.Program.Checks[1].Player = 4
			case "latch":
				a.Program.Triggers[0].Latch = 19
			case "ordinary register":
				doc.World.Session.Results[10] = 123
			}
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			changed, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			got := coldCurrentScript(t, f, changed).live.world
			if change == "ordinary register" {
				if got.ScriptRegister(10) != 123 || !slices.Equal(got.Script().Checks(), expected.Script().Checks()) {
					t.Fatal("program reset the ordinary current register")
				}
			} else if slices.Equal(got.Script().Checks(), expected.Script().Checks()) && slices.Equal(got.Script().Triggers(), expected.Script().Triggers()) {
				t.Fatal("independent program comparison did not detect the deliberate loss")
			}
			if change == "empty" && !got.Script().Empty() {
				t.Fatal("explicit empty program was replaced by the installed program")
			}
		})
	}
}

func TestCurrentScriptProgramTypedBindings(t *testing.T) {
	p := currentScriptProgram{
		Checks:   []sim.ScriptCheck{{Unit: 0, HasUnit: true, Unit2: 7, HasUnit2: true, Structure: 0, HasStructure: true, Group: 9, Player: 5, Player2: 1, Item: 13}},
		Instants: []sim.ScriptInstant{{Unit: 7, HasUnit: true, Unit2: 0, HasUnit2: true, Structure: 7, HasStructure: true}, {Unit: 99, Structure: 99}},
	}
	if err := p.remap(func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		if structure {
			return id + 100, nil
		}
		return id + 10, nil
	}); err != nil {
		t.Fatal(err)
	}
	c, in := p.Checks[0], p.Instants[0]
	if c.Unit != 10 || c.Unit2 != 17 || c.Structure != 100 || in.Unit != 17 || in.Unit2 != 10 || in.Structure != 107 ||
		c.Group != 9 || c.Player != 5 || c.Player2 != 1 || c.Item != 13 || p.Instants[1].Unit != 99 || p.Instants[1].Structure != 99 {
		t.Fatal("typed actor/structure remap changed presence or non-identity fields", p)
	}
}

func TestCurrentScriptProgramMalformedAndBounded(t *testing.T) {
	empty := `{"Checks":null,"Instants":null,"Triggers":null}`
	for _, raw := range []string{
		`{}`, `{"Checks":{},"Instants":null,"Triggers":null}`,
		`{"Checks":[{"Register":100}],"Instants":null,"Triggers":null}`,
		`{"Checks":[{"Register":0},{"Register":0}],"Instants":null,"Triggers":null}`,
		`{"Checks":null,"Instants":null,"Triggers":[{"Latch":1000,"Instants":[-1,-1,-1,-1]}]}`,
		`{"Checks":null,"Instants":null,"Triggers":[{"Latch":0,"Instants":[0,-1,-1,-1]}]}`,
		`{"Checks":null,"Instants":null,"Triggers":[{"Latch":0,"Instants":[-1,-1,-1,-1],"Inert":true}]}`,
		`{"Checks":null,"Instants":[{"Unknown":1}],"Triggers":null}`,
		`{"Checks":[` + strings.Repeat(`{},`, 100) + `{}],"Instants":null,"Triggers":null}`,
		`{"Checks":null,"Instants":[` + strings.Repeat(`{},`, maxCurrentScriptNodes) + `{}],"Triggers":null}`,
		`{"Checks":null,"Instants":null,"Triggers":[` + strings.Repeat(`{},`, maxCurrentScriptNodes) + `{}]}`,
	} {
		var p currentScriptProgram
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("admitted malformed current program %.120s", raw)
		}
	}
	var p currentScriptProgram
	if err := json.Unmarshal([]byte(empty), &p); err != nil {
		t.Fatal("explicit empty program rejected", err)
	}
	_, _, w := currentScriptFixture(t)
	before := w.Hash()
	p.Checks = []sim.ScriptCheck{{Register: -1}}
	if err := restoreCurrentScriptProgram(&Mission{World: w}, nil, &p); err == nil || w.Hash() != before {
		t.Fatal("invalid program restore was not atomic", err)
	}
	if err := p.remap(func(sim.EntityID, bool) (sim.EntityID, error) { return 0, fmt.Errorf("unbound") }); err != nil {
		t.Fatal("absent references should not be remapped", err)
	}
}

func TestCurrentScriptProgramPresenceKeepsExecutionState(t *testing.T) {
	_, _, w := currentScriptFixture(t)
	before := w.Hash()
	ms := &Mission{World: w}
	if err := restoreCurrentScriptProgram(ms, nil, nil); err != nil || w.Hash() != before {
		t.Fatal("absent supplement changed the legacy program", err)
	}
	regs := w.ScriptRegisters()
	won, lost := w.ScriptCounters()
	outcome := w.Outcome()
	floor := *w.CurrentPolicy().EntityIDFloor
	if err := restoreCurrentScriptProgram(ms, nil, &currentScriptProgram{}); err != nil {
		t.Fatal(err)
	}
	a, b := w.ScriptCounters()
	if !w.Script().Empty() || w.ScriptRegisters() != regs || !w.ScriptLatched(18) || a != won || b != lost || w.Outcome() != outcome || *w.CurrentPolicy().EntityIDFloor != floor {
		t.Fatal("explicit empty program reset execution state or retained the local program")
	}
}

func TestCurrentScriptProgramMissingBindingIsAtomic(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc, Actors: []SnapshotSAVActor{binding}}
	if err := projectItems(state, world); err != nil {
		t.Fatal(err)
	}
	doc = *state.Document
	program, err := sim.NewScript([]sim.ScriptCheck{{Op: sim.ScriptCheckAlive, Register: 41, Unit: 701, HasUnit: true}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := world.RestoreScriptProgram(program); err != nil {
		t.Fatal(err)
	}
	before := world.Hash()
	snapshot := Snapshot{ActorManifest: &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: binding.EntityID, Constructed: true}}}}
	for _, text := range doc.Objects[binding.ObjectIndex-1].Texts {
		if text.Name == "Name" {
			snapshot.ActorManifest.Actors[0].Name = text.Value
		}
	}
	if err := projectCurrentActions(&doc, state, world, snapshot, nil); err != nil || world.Hash() != before {
		t.Fatal("script endpoint projection changed World", err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	missing := slices.IndexFunc(a.Bindings, func(b currentActionBinding) bool { return b.ID == 701 && !b.Structure })
	if missing < 0 || a.Bindings[missing].Object != 0 || !a.Bindings[missing].Missing {
		t.Fatal("script-only absent endpoint lost its explicit binding")
	}
	a.Bindings = slices.Delete(a.Bindings, missing, missing+1)
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	state.Document = &doc
	ms := &Mission{World: world, savedDocument: state}
	loaded, err := readCurrentActions(ms.savedDocument.Document)
	if err != nil || loaded == nil || loaded.Program == nil || len(loaded.Program.Checks) != 1 || loaded.Program.Checks[0].Unit != 701 ||
		slices.ContainsFunc(loaded.Bindings, func(b currentActionBinding) bool { return b.ID == 701 && !b.Structure }) {
		t.Fatal("restore input lacks the program with its deliberately deleted endpoint binding", err)
	}
	if err := restoreOriginalActions(ms, nil); err == nil || !strings.Contains(err.Error(), "current action endpoint lacks a binding") || ms.World != world || world.Hash() != before {
		t.Fatal("missing program endpoint changed World or was silently rebound", err)
	}
}
