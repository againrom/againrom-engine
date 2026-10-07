package sim

import "testing"

func TestSourceDeriveRefreshesFrozenMoverRate(t *testing.T) {
	actor := Entity{
		ID: 41, X: 4, Y: 5, HP: 30, MaxHP: 30,
		ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 1, MoverSpeed: 18}},
	}
	w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	motion := SavedActorMotion{
		Entity:   actor.ID,
		Position: SavedActorPosition{Cell: 0x0504, PackedCell: 0x0504, FineX: 128, FineY: 128},
	}
	motion.Mover[10], motion.Mover[31] = 18, 0xa7
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.HeadlessKill(actor.ID); err != nil {
		t.Fatal(err)
	}
	before, _, _, present := w.SavedActorMotions()
	if !present || len(before) != 1 {
		t.Fatalf("death lost imported crossing: count=%d present=%v", len(before), present)
	}
	if before[0].Current || before[0].Issue != "native death supersedes original movement" {
		t.Fatalf("death did not freeze the imported crossing: current=%v issue=%q present=%v", before[0].Current, before[0].Issue, present)
	}
	current, ok := w.Entity(actor.ID)
	if !ok {
		t.Fatal("dying actor left World")
	}
	derived := current.SourceNow()
	derived.MoverSpeed = 19
	w.publishSource(0, derived)
	after, _, _, _ := w.SavedActorMotions()
	if after[0].Mover[10] != 19 || after[0].Mover[31] != 0xa7 || after[0].Current || after[0].Issue != before[0].Issue {
		t.Fatalf("derived speed failed to refresh the frozen mover: speed=%d residue=%#x current=%v issue=%q", after[0].Mover[10], after[0].Mover[31], after[0].Current, after[0].Issue)
	}
}
