package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A dying pursuer's own order block previously never reached the writer at
// all: project skipped any not-Alive entity outright, so its attack target,
// phase and countdown were left at whatever the document already held. Once
// it does reach the writer, its own U54 must stay whatever the motion
// producer already wrote there rather than the fictional "currently
// attacking" action word the alive arm forces, because the per-tick dying
// branch clears that field every tick and never reaches the order machine.
func TestProjectSavedActorActionsWritesADyingPursuersFrozenOrder(t *testing.T) {
	target := sim.Entity{ID: 2, X: 5, Y: 5, HP: 40, MaxHP: 40}
	// The attacker is built already dying and with no order (fine for the
	// constructor: a dying body with no target is not residue), then its own
	// still-charging order is laid onto it the same way a LOAD restores one,
	// so the fixture reaches its dying-with-an-order state through a real
	// production path instead of a literal the constructor would strip.
	attacker := sim.Entity{ID: 1, X: 4, Y: 4, HP: -1, MaxHP: 40, Decay: sim.DecayFallen, Dwell: 3}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{attacker, target})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalActorActions([]sim.OriginalActorAction{{Entity: 1, HasTarget: true, Target: 2, Phase: sim.AttackCharging, Countdown: 0}}); err != nil {
		t.Fatal(err)
	}
	attackerRecord := sav.DocumentRecordData{Class: "Human", Raw: []sav.DocumentRawData{
		{Name: "U58", Bytes: []byte{0, 0, 0, 0}},
		{Name: "U54", Bytes: []byte{0xaa, 0xaa, 0xaa, 0xaa}},
	}}
	targetRecord := sav.DocumentRecordData{Class: "Human",
		Values: []sav.DocumentValueData{{Name: "Identity", Value: 33}},
		Raw:    []sav.DocumentRawData{{Name: "U58", Bytes: []byte{0, 0, 0, 0}}},
	}
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{attackerRecord, targetRecord}},
		Actors: []SnapshotSAVActor{
			{EntityID: 1, ObjectIndex: 1},
			{EntityID: 2, ObjectIndex: 2},
		},
	}
	if err := projectSavedActorActions(state, w); err != nil {
		t.Fatal(err)
	}
	r := &state.Document.Objects[0]
	if u54, err := savedMotionRaw(r, "U54", 4); err != nil || u54[0] != 0xaa {
		t.Fatalf("dying pursuer's own U54 was overwritten: %v %v", u54, err)
	}
	phase, err := savedMotionRaw(r, "U58", 4)
	if err != nil || binary.LittleEndian.Uint32(phase) != 5 {
		t.Fatalf("dying pursuer's charging phase was not written: %v %v", phase, err)
	}
	if key, err := savedStructureValue(r, "U5C"); err != nil || key != 33 {
		t.Fatalf("dying pursuer's own attack target key was not written: %v %v", key, err)
	}
	if countdown, err := savedStructureValue(r, "U6C"); err != nil || countdown != 0 {
		t.Fatalf("dying pursuer's own countdown was not written: %v %v", countdown, err)
	}
}

// The loader previously read a not-normal-stage record's Stage byte and, for
// any nonzero value (which a dying pursuer's own record carries), dropped its
// entire order block rather than reading it: Order::Serialize round-trips its
// fixed block through one raw archive read/write pair regardless of stage, so
// the bytes are present and correct on disk even though the pursuer never
// reaches the order machine to keep them current tick to tick. This proves
// the loader now restores them the same way an alive actor's would.
func TestImportSavedActorActionsRestoresADyingPursuersFrozenOrder(t *testing.T) {
	target := sim.Entity{ID: 2, X: 5, Y: 5, HP: 40, MaxHP: 40}
	attacker := sim.Entity{ID: 1, X: 4, Y: 4, HP: -1, MaxHP: 40, Decay: sim.DecayFallen, Dwell: 3}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{attacker, target})
	if err != nil {
		t.Fatal(err)
	}
	order := make([]byte, 148)
	attackerRecord := sav.DocumentRecordData{Class: "Human",
		Values: []sav.DocumentValueData{{Name: "Identity", Value: 11}, {Name: "Stage", Value: 1},
			{Name: "U5C", Value: 22}, {Name: "U6C", Value: 0}, {Name: "U40", Value: 0}, {Name: "U48", Value: 0}, {Name: "U136", Value: 0}},
		Raw: []sav.DocumentRawData{
			{Name: "U54", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U58", Bytes: []byte{5, 0, 0, 0}},
			{Name: "U50", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U158", Bytes: order},
		},
	}
	targetRecord := sav.DocumentRecordData{Class: "Human",
		Values: []sav.DocumentValueData{{Name: "Identity", Value: 22}, {Name: "Stage", Value: 0},
			{Name: "U5C", Value: 0}, {Name: "U6C", Value: 0}, {Name: "U40", Value: 0}, {Name: "U48", Value: 0}, {Name: "U136", Value: 0}},
		Raw: []sav.DocumentRawData{
			{Name: "U54", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U58", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U50", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U158", Bytes: order},
		},
	}
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{attackerRecord, targetRecord}},
		Actors: []SnapshotSAVActor{
			{EntityID: 1, ObjectIndex: 1},
			{EntityID: 2, ObjectIndex: 2},
		},
	}
	ms := &Mission{World: w}
	if err := importSavedActorActions(ms, state); err != nil {
		t.Fatal(err)
	}
	var got sim.Entity
	for _, e := range ms.World.Entities() {
		if e.ID == 1 {
			got = e
		}
	}
	if !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Fatalf("dying pursuer's own attack target was not restored: %+v", got)
	}
	if got.AttackPhase != sim.AttackCharging {
		t.Fatalf("dying pursuer's own attack phase was not restored: %v", got.AttackPhase)
	}
	if got.AttackCountdown != 0 {
		t.Fatalf("dying pursuer's own countdown was not restored: %v", got.AttackCountdown)
	}
}
