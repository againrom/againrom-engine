package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestSavedMotionAttackOrderProjectionKeepsCrossingProgress(t *testing.T) {
	actor := sim.Entity{ID: 1, Owner: 1, X: 15, Y: 16, HP: 100, MaxHP: 100}
	target := sim.Entity{ID: 2, Owner: 2, X: 17, Y: 16, HP: 100, MaxHP: 100}
	world, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{actor, target})
	if err != nil {
		t.Fatal(err)
	}
	order := sim.SavedActorOrder{Entity: actor.ID, State: 0xb}
	order.Raw[8], order.Raw[9] = 1, 3
	group := sim.SavedGroup{ID: 1, Selector: 1, Members: []sim.SavedGroupMember{{Archive: 1, Entity: actor.ID, Bound: true}}}
	if err := world.ImportSavedGroups([]sim.SavedGroup{group}, []sim.SavedActorOrder{order}); err != nil {
		t.Fatal(err)
	}
	motion := sim.SavedActorMotion{Entity: actor.ID, Position: sim.SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 42, FineY: 128}, ActorAction: 1}
	binary.LittleEndian.PutUint16(motion.Mover[0xaa:], 16)
	binary.LittleEndian.PutUint16(motion.Mover[0xac:], 10)
	binary.LittleEndian.PutUint16(motion.Mover[0xae:], 2)
	motion.Mover[0xb0] = 32
	if err := world.ImportOriginalActorMotions([]sim.SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := world.ImportOriginalActorActions([]sim.OriginalActorAction{{Entity: actor.ID, HasTarget: true, Target: target.ID}}); err != nil {
		t.Fatal(err)
	}
	if !world.ActorMotionActive(actor.ID) {
		t.Fatal("original fine crossing did not remain active")
	}
	actorOrder := make([]byte, 148)
	copy(actorOrder, order.Raw[:])
	position := make([]byte, 12)
	binary.LittleEndian.PutUint16(position, motion.Position.Cell)
	binary.LittleEndian.PutUint16(position[2:], motion.Position.PackedCell)
	position[4], position[5] = motion.Position.FineX, motion.Position.FineY
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			{Class: "Human", Values: []sav.DocumentValueData{{Name: "Identity", Value: 11}}, Counts: []sav.DocumentCountData{{Name: "U15C"}, {Name: "U178"}}, Raw: []sav.DocumentRawData{
				{Name: "Block12", Bytes: position}, {Name: "U154", Bytes: motion.Mover[:]}, {Name: "U158", Bytes: actorOrder},
				{Name: "U15C"}, {Name: "U178"}, {Name: "U54", Bytes: []byte{1, 0, 0, 0}}, {Name: "U58", Bytes: make([]byte, 4)},
			}},
			{Class: "Human", Values: []sav.DocumentValueData{{Name: "Identity", Value: 22}}, Raw: []sav.DocumentRawData{
				{Name: "U54", Bytes: make([]byte, 4)}, {Name: "U58", Bytes: make([]byte, 4)},
			}},
		}},
		Actors: []SnapshotSAVActor{{EntityID: actor.ID, ObjectIndex: 1}, {EntityID: target.ID, ObjectIndex: 2}},
	}
	if err := projectSavedActorActions(state, world); err != nil {
		t.Fatal(err)
	}
	written := state.Document.Objects[0]
	progress, err := savedMotionRaw(&written, "U158", 148)
	if err != nil {
		t.Fatal(err)
	}
	if progress[9] != order.Raw[9] {
		t.Fatalf("active crossing order progress written %d, World has %d", progress[9], order.Raw[9])
	}
	action, err := savedMotionRaw(&written, "U54", 4)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(action); got != motion.ActorAction {
		t.Fatalf("active crossing action written %d, World has %d", got, motion.ActorAction)
	}
	writtenMotion, err := readSavedActorMotion(&written, actor.ID)
	if err != nil {
		t.Fatal(err)
	}
	cold, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil, []sim.Entity{actor, target})
	if err != nil {
		t.Fatal(err)
	}
	copy(order.Raw[:], progress)
	if err := cold.ImportSavedGroups([]sim.SavedGroup{group}, []sim.SavedActorOrder{order}); err != nil {
		t.Fatal(err)
	}
	if err := cold.ImportOriginalActorMotions([]sim.SavedActorMotion{writtenMotion}, nil, nil); err != nil {
		t.Fatalf("cold reader rejected written crossing: %v", err)
	}
}
