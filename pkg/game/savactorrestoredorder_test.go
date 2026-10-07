package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func restoredOrderFixture(t *testing.T) (*SnapshotSAVDocument, *Mission) {
	t.Helper()
	body := sim.Entity{ID: 2, X: 5, Y: 5, HP: -57, MaxHP: 40, Decay: sim.DecayFallen, Dwell: 3}
	attacker := sim.Entity{ID: 1, X: 4, Y: 4, HP: 40, MaxHP: 40}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{attacker, body})
	if err != nil {
		t.Fatal(err)
	}
	order := make([]byte, 148)
	record := func(identity, target, phase, countdown, credit uint32) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: "Human",
			Values: []sav.DocumentValueData{{Name: "Identity", Value: identity}, {Name: "Stage", Value: 0},
				{Name: "U5C", Value: target}, {Name: "U6C", Value: countdown}, {Name: "U40", Value: credit}, {Name: "U48", Value: 0}, {Name: "U136", Value: 0}},
			Raw: []sav.DocumentRawData{
				{Name: "U54", Bytes: []byte{3, 0, 0, 0}},
				{Name: "U58", Bytes: binary.LittleEndian.AppendUint32(nil, phase)},
				{Name: "U50", Bytes: []byte{0xc, 0, 0, 0}},
				{Name: "U158", Bytes: order},
			},
		}
	}
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{record(11, 22, 7, 2, 22), record(22, 0, 0, 0, 0)}},
		Actors:   []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}},
	}
	ms := &Mission{World: w}
	if err := importSavedActorActions(ms, state); err != nil {
		t.Fatal(err)
	}
	return state, ms
}

func restoredOrderFields(t *testing.T, state *SnapshotSAVDocument) (target, phase, countdown, credit uint32) {
	t.Helper()
	r := &state.Document.Objects[0]
	raw, err := savedMotionRaw(r, "U58", 4)
	if err != nil {
		t.Fatal(err)
	}
	get := func(name string) uint32 {
		v, err := savedStructureValue(r, name)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	return get("U5C"), binary.LittleEndian.Uint32(raw), get("U6C"), get("U40")
}

// A restored attack on a body below the targetable floor is dropped by the
// World. A save before the World advances keeps the loaded order fields, so the
// order is not left half cleared; once the World advances they follow the World.
func TestProjectSavedActorActionsKeepsLoadedOrderOfADroppedTargetUntilTheFirstTick(t *testing.T) {
	state, ms := restoredOrderFixture(t)
	for _, e := range ms.World.Entities() {
		if e.ID == 1 && e.HasAttackTarget {
			t.Fatalf("the World kept an attack on a body: %+v", e)
		}
	}
	if err := projectSavedActorActions(state, ms.World); err != nil {
		t.Fatal(err)
	}
	if target, phase, countdown, credit := restoredOrderFields(t, state); target != 22 || phase != 7 || countdown != 2 || credit != 22 {
		t.Fatalf("loaded order fields were rewritten: target %d phase %d countdown %d credit %d", target, phase, countdown, credit)
	}
	sim.Step(ms.World, nil)
	if err := projectSavedActorActions(state, ms.World); err != nil {
		t.Fatal(err)
	}
	if target, phase, countdown, credit := restoredOrderFields(t, state); target != 0 || phase != 0 || countdown != 0 {
		t.Fatalf("order fields outlived the first tick: target %d phase %d countdown %d credit %d", target, phase, countdown, credit)
	}
}
