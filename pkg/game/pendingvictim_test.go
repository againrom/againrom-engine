package game

import (
	"encoding/binary"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func pendingVictimWorld(t *testing.T) *sim.World {
	t.Helper()
	var relations sim.Relations
	relations.Set(sim.SelfSlot, 2, 1)
	relations.Set(2, sim.SelfSlot, 2)
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, TypeID: 1, Group: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Capacity: 300, TokenSize: 1, ScanRange: 6, Reach: 4, DamageBase: 5, AlwaysHits: true, AttackCharge: 40, AttackRelax: 30},
		{ID: 2, Owner: 2, TypeID: 1, Group: 1, X: 24, Y: 20, HP: 200, MaxHP: 200, Capacity: 300, TokenSize: 1},
		{ID: 3, Owner: 2, TypeID: 1, Group: 1, X: 20, Y: 24, HP: 200, MaxHP: 200, Capacity: 300, TokenSize: 1},
	}
	w, err := sim.NewRelatedWorld(17, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{}, actors, nil, relations)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func pendingVictimEntity(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, actor := range w.Entities() {
		if actor.ID == id {
			return actor
		}
	}
	t.Fatalf("pending-victim actor %d disappeared", id)
	return sim.Entity{}
}

func pendingVictimLoadCycle(t *testing.T, w *sim.World) {
	t.Helper()
	sim.Step(w, []sim.Command{sim.Attack(1, 2)})
	actor := pendingVictimEntity(t, w, 1)
	if !actor.HasAttackTarget || actor.AttackTarget != 2 || actor.AttackPhase != sim.AttackCharging {
		t.Fatalf("first Attack did not load the physical cycle: %+v", actor)
	}
	sim.Step(w, []sim.Command{sim.Attack(1, 3)})
	actor = pendingVictimEntity(t, w, 1)
	if !actor.HasAttackTarget || actor.AttackTarget != 2 || actor.AttackPhase != sim.AttackCharging || !actor.HasPendingAttackTarget || actor.PendingAttackTarget != 3 || actor.PendingAttackTargetKind != sim.AttackTargetUnit {
		t.Fatalf("second Attack replaced the loaded victim or lost the requested victim: %+v", actor)
	}
}

func pendingVictimRequireBlows(t *testing.T, worlds ...*sim.World) {
	t.Helper()
	oldBlow := false
	for tick := 0; tick < 240; tick++ {
		for _, w := range worlds {
			sim.Step(w, nil)
		}
		if len(worlds) > 1 && worlds[0].Hash() != worlds[1].Hash() {
			t.Fatalf("pending victim diverged after cold LOAD at successor %d", tick)
		}
		oldHP := pendingVictimEntity(t, worlds[0], 2).HP
		newHP := pendingVictimEntity(t, worlds[0], 3).HP
		if !oldBlow && oldHP != 200 {
			if oldHP != 195 || newHP != 200 {
				t.Fatalf("loaded blow reached the wrong victim: old/new HP %d/%d", oldHP, newHP)
			}
			oldBlow = true
		}
		if newHP != 200 {
			if !oldBlow || oldHP != 195 || newHP != 195 {
				t.Fatalf("requested victim ran before the old cycle completed: old/new HP %d/%d", oldHP, newHP)
			}
			actor := pendingVictimEntity(t, worlds[0], 1)
			if actor.AttackTarget != 3 || actor.HasPendingAttackTarget {
				t.Fatalf("requested victim did not become the active victim: %+v", actor)
			}
			return
		}
	}
	t.Fatal("ordinary ticks did not finish the loaded blow and reach the requested victim")
}

func TestPendingAttackCurrentSAVKeepsActiveAndRequestedVictims(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("pending victim SAV").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	w := pendingVictimWorld(t)
	pendingVictimLoadCycle(t, w)
	snapshot.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	snapshot.SavedDocument, snapshot.ActorManifest = nil, nil
	before := w.Hash()
	raw, err := f.ExportCurrentSave(snapshot, "pending physical victim")
	if err != nil || w.Hash() != before {
		t.Fatal("generated current SAVE failed or changed World", err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actor := generatedActorRecord(t, &doc, 1)
	activeKey := savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 2), "Identity")
	pendingKey := savedRecordValueForTest(t, *generatedActorRecord(t, &doc, 3), "Identity")
	if activeKey == 0 || pendingKey == 0 || activeKey == pendingKey {
		t.Fatal("generated victims lack distinct nonzero SAV identities")
	}
	if got := savedRecordValueForTest(t, *actor, "U5C"); got != activeKey {
		t.Fatalf("U5C active victim = %#x, want %#x", got, activeKey)
	}
	if got := binary.LittleEndian.Uint32(savedRecordRawForTest(t, *actor, "U158")[0xc:]); got != pendingKey {
		t.Fatalf("U158 requested victim = %#x, want %#x", got, pendingKey)
	}
	liveActor := pendingVictimEntity(t, w, 1)
	if binary.LittleEndian.Uint32(savedRecordRawForTest(t, *actor, "U58")) != 5 || savedRecordValueForTest(t, *actor, "U6C") != uint32(liveActor.AttackCountdown) {
		t.Fatal("SAVE changed the loaded physical phase or countdown")
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("pending current actions are absent", err)
	}
	var held sim.ActorContinuation
	for _, row := range actions.Actions.Actors {
		if row.Entity == 1 {
			held = row
		}
	}
	if held.AttackTarget != 2 || !held.HasPendingAttackTarget || held.PendingAttackTarget != 3 || held.PendingAttackTargetKind != sim.AttackTargetUnit || held.AttackPhase != liveActor.AttackPhase || held.AttackCountdown != liveActor.AttackCountdown {
		t.Fatalf("supplement conflated active and requested victims: %+v", held)
	}
	cold := generatedCurrentCold(t, raw)
	assertCurrentWorldEqual(t, w, cold.live.world, "pending victim cold LOAD")
	loss, err := sav.CloneDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	lostActions, err := readCurrentActions(&loss)
	if err != nil || lostActions == nil {
		t.Fatal("loss control lacks current actions", err)
	}
	for i := range lostActions.Actions.Actors {
		row := &lostActions.Actions.Actors[i]
		if row.Entity == 1 {
			row.HasPendingAttackTarget, row.PendingAttackTarget, row.PendingAttackTargetKind = false, 0, sim.AttackTargetUnit
		}
	}
	payload, err := json.Marshal(lostActions)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&loss.State, payload); err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(savedRecordRawForTest(t, *generatedActorRecord(t, &loss, 1), "U158")[0xc:]); got != pendingKey {
		t.Fatal("loss control changed the original requested-victim word")
	}
	lossRaw, err := sav.EncodeDocumentData(loss)
	if err != nil {
		t.Fatal(err)
	}
	lossy := generatedCurrentCold(t, lossRaw)
	lossActor := pendingVictimEntity(t, lossy.live.world, 1)
	if lossActor.HasPendingAttackTarget || lossActor.AttackTarget != 2 || lossActor.AttackPhase != liveActor.AttackPhase || lossActor.AttackCountdown != liveActor.AttackCountdown {
		t.Fatalf("typed LOAD did not replace the wire queue with its absent native queue: %+v", lossActor)
	}
	for range 240 {
		sim.Step(lossy.live.world, nil)
		if got := pendingVictimEntity(t, lossy.live.world, 3).HP; got != 200 {
			t.Fatalf("requested victim survived its removal from native actions: HP %d", got)
		}
	}
	if pendingVictimEntity(t, lossy.live.world, 2).HP >= 200 {
		t.Fatal("loss control did not execute the surviving loaded blow")
	}
	pendingVictimRequireBlows(t, w, cold.live.world)
}

func pendingVictimOriginalDocument() *SnapshotSAVDocument {
	state := &SnapshotSAVDocument{Document: &sav.DocumentData{}}
	for i, key := range []uint32{0x700001, 0x700002, 0x700003} {
		order := make([]byte, 148)
		record := sav.DocumentRecordData{Class: "Unit", Values: []sav.DocumentValueData{
			{Name: "Identity", Value: key}, {Name: "Stage"}, {Name: "U5C"}, {Name: "U6C"}, {Name: "U40"}, {Name: "U48"}, {Name: "U136"},
		}, Raw: []sav.DocumentRawData{
			{Name: "U54", Bytes: make([]byte, 4)}, {Name: "U58", Bytes: make([]byte, 4)}, {Name: "U50", Bytes: make([]byte, 4)}, {Name: "U158", Bytes: order},
		}}
		if i == 0 {
			record.Values[2].Value, record.Values[3].Value = 0x700002, 20
			binary.LittleEndian.PutUint32(record.Raw[0].Bytes, 3)
			binary.LittleEndian.PutUint32(record.Raw[1].Bytes, 5)
			binary.LittleEndian.PutUint32(record.Raw[2].Bytes, 3)
			order[8], order[9] = 2, 1
			binary.LittleEndian.PutUint32(order[0xc:], 0x700003)
		}
		state.Document.Objects = append(state.Document.Objects, record)
		state.Actors = append(state.Actors, SnapshotSAVActor{EntityID: sim.EntityID(i + 1), ObjectIndex: uint16(i + 1)})
	}
	return state
}

func TestPendingAttackOriginalImportKeepsDistinctVictimWords(t *testing.T) {
	w := pendingVictimWorld(t)
	state := pendingVictimOriginalDocument()
	if supplement, err := readCurrentActions(state.Document); err != nil || supplement != nil {
		t.Fatal("literal original-import witness has a native supplement", err)
	}
	if err := importSavedActorActions(&Mission{World: w}, state); err != nil {
		t.Fatal(err)
	}
	actor := pendingVictimEntity(t, w, 1)
	if actor.AttackTarget != 2 || !actor.HasAttackTarget || actor.AttackPhase != sim.AttackCharging || actor.AttackCountdown != 20 || !actor.HasPendingAttackTarget || actor.PendingAttackTarget != 3 || actor.PendingAttackTargetKind != sim.AttackTargetUnit {
		t.Fatalf("original victim words did not restore separate physical and pending targets: %+v", actor)
	}
	pendingVictimRequireBlows(t, w)
}

func TestPendingAttackOriginalImportBoundsAdmission(t *testing.T) {
	for _, control := range []string{"nonzero stage", "other state", "zero progress", "unknown key", "same victim", "self victim", "no active cycle"} {
		t.Run(control, func(t *testing.T) {
			state := pendingVictimOriginalDocument()
			record := &state.Document.Objects[0]
			switch control {
			case "nonzero stage":
				savedObjectSetValue(record, "Stage", 1)
			case "other state":
				binary.LittleEndian.PutUint32(record.Raw[2].Bytes, 0xb)
			case "zero progress":
				record.Raw[3].Bytes[9] = 0
			case "unknown key":
				binary.LittleEndian.PutUint32(record.Raw[3].Bytes[0xc:], 0xdecafbad)
			case "same victim":
				binary.LittleEndian.PutUint32(record.Raw[3].Bytes[0xc:], 0x700002)
			case "self victim":
				binary.LittleEndian.PutUint32(record.Raw[3].Bytes[0xc:], 0x700001)
			case "no active cycle":
				binary.LittleEndian.PutUint32(record.Raw[1].Bytes, 0)
			}
			w := pendingVictimWorld(t)
			if err := importSavedActorActions(&Mission{World: w}, state); err != nil {
				t.Fatal(err)
			}
			if actor := pendingVictimEntity(t, w, 1); actor.HasPendingAttackTarget {
				t.Fatalf("%s admitted an unproved pending victim: %+v", control, actor)
			}
		})
	}
}

func TestPendingAttackCurrentBindingsRequireRequestedEndpoint(t *testing.T) {
	state := pendingVictimOriginalDocument()
	makeActions := func() *currentActionData {
		return &currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: 11, Object: 1}, {ID: 12, Object: 2}, {ID: 13, Object: 3}},
			Actions: sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 11, HasAttackTarget: true, AttackTarget: 12, AttackPhase: sim.AttackCharging, AttackCountdown: 20, HasPendingAttackTarget: true, PendingAttackTarget: 13, PendingAttackTargetKind: sim.AttackTargetUnit}}}}
	}
	mission := &Mission{World: pendingVictimWorld(t), savedDocument: state}
	a := makeActions()
	if _, err := resolveCurrentActions(mission, a); err != nil {
		t.Fatal(err)
	}
	row := a.Actions.Actors[0]
	if row.Entity != 1 || row.AttackTarget != 2 || row.PendingAttackTarget != 3 || !row.HasPendingAttackTarget {
		t.Fatalf("requested endpoint was not remapped through its own object: %+v", row)
	}
	broken := makeActions()
	broken.Bindings = slices.DeleteFunc(broken.Bindings, func(b currentActionBinding) bool { return b.ID == 13 })
	before := mission.World.Hash()
	if _, err := resolveCurrentActions(mission, broken); err == nil || !strings.Contains(err.Error(), "endpoint lacks a binding") || mission.World.Hash() != before {
		t.Fatal("missing requested binding did not fail without changing World", err)
	}
}

func TestPendingAttackRemappingKeepsZeroAndStructureNamespaces(t *testing.T) {
	actions := sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 0, HasAttackTarget: true, AttackTarget: 0, AttackTargetKind: sim.AttackTargetUnit,
		HasPendingAttackTarget: true, PendingAttackTarget: 0, PendingAttackTargetKind: sim.AttackTargetStructure}}}
	if err := actions.RemapActors(func(id sim.EntityID, structure bool) (sim.EntityID, error) {
		if structure {
			return id + 100, nil
		}
		return id + 10, nil
	}); err != nil {
		t.Fatal(err)
	}
	row := actions.Actors[0]
	if row.Entity != 10 || row.AttackTarget != 10 || row.PendingAttackTarget != 100 || !row.HasPendingAttackTarget || row.PendingAttackTargetKind != sim.AttackTargetStructure {
		t.Fatalf("zero requested endpoint joined the active actor namespace: %+v", row)
	}
}
