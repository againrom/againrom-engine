package game

import (
	"bytes"
	"encoding/json"
	"testing"

	"againrom/pkg/formats/sav"
)

type terminalHealthWords struct {
	health int16
	maxHP  uint16
	stage  uint8
}

func terminalHealthWire(t *testing.T, raw []byte) map[uint16]terminalHealthWords {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	words, err := unitScalarExpected(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint16]terminalHealthWords{}
	for _, actor := range words.records {
		id := uint16(actor.values["T08"])
		if id == 0 {
			continue
		}
		if _, exists := out[id]; exists {
			t.Fatalf("duplicate ordinary actor map ID %d", id)
		}
		out[id] = terminalHealthWords{int16(actor.values["Health"]), uint16(actor.values["HealthMax"]), uint8(actor.values["Stage"])}
	}
	return out
}

func requireTerminalHealthWire(t *testing.T, raw []byte, want map[uint16]terminalHealthWords) {
	t.Helper()
	got := terminalHealthWire(t, raw)
	for id, expected := range want {
		if actual, present := got[id]; !present || actual != expected {
			t.Errorf("ordinary actor %d Health/HealthMax/Stage = %+v present=%t, want %+v", id, actual, present, expected)
		}
	}
}

func TestTerminalHealthMaximumSourceWordsSurviveTwoSAVCycles(t *testing.T) {
	zero := uint32(0)
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 197, name: "Living control"}
	late := &poolFixtureActor{mapID: 92, cell: 0x0807, hp: uint16(65536 - 237), maxHP: 421, stage: 4}
	terminal := &poolFixtureActor{mapID: 93, cell: 0x0a09, hp: uint16(65536 - 10017), maxHP: 613, stage: 5, runtime: &zero}
	fresh := func() *FrontEnd { return currentPoolFixtureFront(t, 91, 92, 93) }
	f := fresh()
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, []*poolFixtureActor{late, terminal}))
	raw = completeCurrentDeadDocument(t, f, raw)
	want := map[uint16]terminalHealthWords{
		91: {23, 197, 0},
		92: {-237, 421, 4},
		93: {-10017, 613, 5},
	}
	requireTerminalHealthWire(t, raw, want)
	f = openCurrentRetainedRuntime(t, f, raw)
	for cycle := range 2 {
		raw = currentRuntimeSave(t, f)
		requireTerminalHealthWire(t, raw, want)
		cold := openCurrentRetainedRuntime(t, fresh(), raw)
		for _, id := range []uint16{91, 92} {
			actor := poolEntity(t, cold.live.world, id)
			if actor.HP != int32(want[id].health) || actor.MaxHP != int32(want[id].maxHP) {
				t.Errorf("cycle %d cold actor %d HP/MaxHP = %d/%d, want %d/%d", cycle, id, actor.HP, actor.MaxHP, want[id].health, want[id].maxHP)
			}
		}
		for _, actor := range cold.live.world.Entities() {
			if actor.MapUnitID == 93 {
				t.Fatal("terminal actor returned to the live population")
			}
		}
		f = cold
	}
}

func TestTerminalHealthMaximumCurrentMutationSurvivesRemoval(t *testing.T) {
	living := &poolFixtureActor{mapID: 91, cell: 0x1211, hp: 23, maxHP: 197, name: "Changed living actor", holdings: &holdingFixture{}}
	fresh := func() *FrontEnd { return currentPoolFixtureFront(t, 91) }
	f := fresh()
	source := savedContainer(poolFixtureBody([]*poolFixturePlayer{{groups: [][]*poolFixtureActor{{living}}}, {}}, nil))
	source = completeCurrentDeadDocument(t, f, source)
	f = openCurrentRetainedRuntime(t, f, source)
	actor := poolEntity(t, f.live.world, 91)
	load := actor.CurrentActorLoad()
	if load == nil || load.Inventory.Source.Class == 0 {
		t.Fatal("source actor has no mutable load basis")
	}
	load.Inventory.Source.Stats[9] = 743
	if err := f.live.world.RestoreActorLoad(actor.ID, *load); err != nil {
		t.Fatal(err)
	}
	if changed := poolEntity(t, f.live.world, 91); changed.MaxHP != 743 || changed.HP != 23 {
		t.Fatal("current load mutation did not set the independent maximum", changed.HP, changed.MaxHP)
	}
	if original := terminalHealthWire(t, source)[91]; original.maxHP != 197 {
		t.Fatal("current mutation changed the source maximum", original)
	}
	if err := f.live.world.HeadlessKill(actor.ID); err != nil {
		t.Fatal(err)
	}
	for ticks := 0; ticks < 24000; ticks += 50 {
		if _, present := f.live.world.Entity(actor.ID); !present {
			break
		}
		f.LiveAdvance(50)
	}
	if body, present := f.live.world.Entity(actor.ID); present {
		t.Fatalf("ordinary death did not remove the actor within 24000 ticks: HP=%d MaxHP=%d Decay=%d Dwell=%d tick=%d", body.HP, body.MaxHP, body.Decay, body.Dwell, f.live.world.Tick())
	}
	var health int16
	var stage uint8
	found := false
	for _, dead := range f.live.world.OriginalDeadActors() {
		if dead.ID == actor.ID {
			health, stage, found = dead.Current.HP, dead.Current.Stage, true
		}
	}
	for _, terminal := range f.live.world.CurrentTerminalActors() {
		if terminal.ID == actor.ID {
			health, stage, found = int16(terminal.HP), terminal.Stage, true
		}
	}
	if !found || health >= -10000 || stage != 5 {
		t.Fatal("ordinary death left no terminal health", found, health, stage)
	}
	want := map[uint16]terminalHealthWords{91: {health, 743, stage}}
	for cycle := range 2 {
		raw := currentRuntimeSave(t, f)
		requireTerminalHealthWire(t, raw, want)
		cold := openCurrentRetainedRuntime(t, fresh(), raw)
		for _, entity := range cold.live.world.Entities() {
			if entity.MapUnitID == 91 {
				t.Fatalf("cycle %d terminal actor returned to the live population", cycle)
			}
		}
		f = cold
	}
}

func TestTerminalHealthMaximumOrdinaryEditWinsOverCurrentMarker(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "current marker"
		if legacy {
			name = "legacy marker without maximum"
		}
		t.Run(name, func(t *testing.T) {
			f, raw := nativeTerminalSubjectFixture(t)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			policy, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, object := range doc.DeadActors {
				actor := &doc.Objects[object-1]
				if id, _ := savedStructureValue(actor, "T08"); id == 91 {
					mustSetValue(actor, "HealthMax", 509)
					found = true
				}
			}
			if !found {
				t.Fatal("native terminal actor has no ordinary dead root")
			}
			after, _, _ := sav.NativeActions(doc.State)
			if !bytes.Equal(policy, after) {
				t.Fatal("ordinary maximum edit changed the current supplement")
			}
			if legacy {
				a, err := readCurrentActions(&doc)
				if err != nil || a == nil || a.Values[0].CurrentTerminal == nil {
					t.Fatal("native terminal marker missing", err)
				}
				marker := a.Values[0].CurrentTerminal
				marker.MaxHP, marker.MaxHPPresent = 0, false
				encodedMarker, err := json.Marshal(marker)
				if err != nil || bytes.Contains(encodedMarker, []byte("MaxHP")) {
					t.Fatal("legacy terminal marker carries a maximum", string(encodedMarker), err)
				}
				encoded, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, encoded); err != nil {
					t.Fatal(err)
				}
			}
			raw, err = sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			want := terminalHealthWire(t, raw)
			if got := want[91]; got.health != -602 || got.maxHP != 509 || got.stage != 4 {
				t.Fatal("ordinary wire edit differs from the independent words", got)
			}
			for cycle := range 2 {
				cold := nativeSubjectCold(t, f, raw)
				for _, actor := range cold.live.world.Entities() {
					if actor.MapUnitID == 91 {
						t.Fatalf("cycle %d terminal actor returned to the live population", cycle)
					}
				}
				raw = currentRuntimeSave(t, cold)
				requireTerminalHealthWire(t, raw, want)
				f = cold
			}
		})
	}
}
