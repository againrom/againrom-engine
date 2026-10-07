package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestDyingActorSAVKeepsCurrentMoverSpeed(t *testing.T) {
	const actorID sim.EntityID = 41
	actor := sim.Entity{
		ID: actorID, X: 4, Y: 5, HP: 30, MaxHP: 30,
		ActorLoad: sim.ActorLoad{Present: true, Source: sim.SourceActor{Class: 1, MoverSpeed: 19}},
	}
	world, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	frozen := sim.SavedActorMotion{
		Entity:   actorID,
		Position: sim.SavedActorPosition{Cell: 0x0504, PackedCell: 0x0504, FineX: 128, FineY: 128},
	}
	frozen.Mover[10], frozen.Mover[31] = 18, 0xa7
	if err := world.ImportOriginalActorMotions([]sim.SavedActorMotion{frozen}, nil, nil); err != nil {
		t.Fatal(err)
	}
	before, _, _, present := world.SavedActorMotions()
	if !present || len(before) != 1 || !before[0].Current {
		t.Fatalf("fixture did not import a current crossing: %+v, present=%v", before, present)
	}
	if err := world.HeadlessKill(actorID); err != nil {
		t.Fatal(err)
	}
	current, ok := world.Entity(actorID)
	if !ok || current.HP > 0 || current.Decay == sim.DecayNone || current.SourceNow().MoverSpeed != 19 {
		t.Fatalf("ordinary death did not leave a current dying actor with speed 19: %+v, held=%v", current, ok)
	}
	motions, _, _, present := world.SavedActorMotions()
	if !present || len(motions) != 1 || motions[0].Current || motions[0].Issue == "" {
		t.Fatalf("ordinary death did not freeze the imported crossing: %+v, present=%v", motions, present)
	}
	position := make([]byte, 12)
	binary.LittleEndian.PutUint16(position, frozen.Position.Cell)
	binary.LittleEndian.PutUint16(position[2:], frozen.Position.PackedCell)
	position[4], position[5] = frozen.Position.FineX, frozen.Position.FineY
	projectedMover := frozen.Mover
	projectedMover[10] = current.SourceNow().MoverSpeed
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{{
			Class:  "Unit",
			Values: []sav.DocumentValueData{{Name: "Identity", Value: 0x4100}},
			Counts: []sav.DocumentCountData{{Name: "U15C"}, {Name: "U178"}},
			Raw: []sav.DocumentRawData{
				{Name: "Block12", Bytes: position},
				{Name: "U154", Bytes: projectedMover[:]},
				{Name: "U158", Bytes: make([]byte, 148)},
				{Name: "U15C"}, {Name: "U178"},
				{Name: "U54", Bytes: make([]byte, 4)},
				{Name: "U58", Bytes: make([]byte, 4)},
			},
		}}},
		Actors: []SnapshotSAVActor{{EntityID: actorID, ObjectIndex: 1}},
	}
	if err := projectMotion(state, world); err != nil {
		t.Fatal(err)
	}
	written, err := savedMotionRaw(&state.Document.Objects[0], "U154", 180)
	if err != nil {
		t.Fatal(err)
	}
	if written[31] != frozen.Mover[31] {
		t.Fatalf("frozen movement residue changed: got %#x, want %#x", written[31], frozen.Mover[31])
	}
	if written[10] != current.SourceNow().MoverSpeed {
		t.Fatalf("dying actor SAV speed = %d, current actor = %d, stale crossing = %d", written[10], current.SourceNow().MoverSpeed, frozen.Mover[10])
	}
}
