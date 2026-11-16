package sim

import "testing"

func TestPursuingLargeFlyerNeverOverlapsAnotherFlyer(t *testing.T) {
	attacker := cbEnt(1, 0, 2)
	attacker.Domain, attacker.TokenSize = DomainAir, 2
	attacker.AttackCharge, attacker.AttackRelax = 1, 0
	attacker.DamageBase, attacker.AlwaysHits = 10, true
	resting := cbEnt(2, 2, 2)
	resting.Domain = DomainAir
	victim := cbEnt(3, 3, 2)
	w := mustWorld(t, 1065, Bounds{Width: 12, Height: 8}, []Entity{attacker, resting, victim})

	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 3}})
	for k := 0; k < 40; k++ {
		got, held := spAt(t, w, 1), spAt(t, w, 2)
		if footprintsOverlap(got.X, got.Y, got.TokenSize, held.X, held.Y, held.TokenSize) {
			t.Fatalf("tick %d: the 2x2 flyer at (%d,%d) overlaps the resting flyer at (%d,%d)",
				k+1, got.X, got.Y, held.X, held.Y)
		}
		Step(w, nil)
	}
	if victim := spAt(t, w, 3); victim.HP >= 100 {
		t.Fatalf("pursuit never reached the victim: HP=%d", victim.HP)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary pursuit: %v", err)
	}
	wantHash := w.Hash()
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary pursuit: %v", err)
	}
	if gotHash := back.Hash(); gotHash != wantHash {
		t.Fatalf("pursuit hash after load = %016x, want %016x", gotHash, wantHash)
	}
}

func TestLargeFlyerPursuitMovesItsWholeFootprint(t *testing.T) {
	attacker := cbEnt(1, 1, 1)
	attacker.Domain, attacker.TokenSize = DomainAir, 2
	w := mustWorld(t, 1065, Bounds{Width: 8, Height: 8}, []Entity{attacker, cbEnt(2, 6, 1)})
	s := newRouteScratch(w)
	w.walkTo(s, 0, 6, 1)

	at := w.entities[0]
	for y := int32(0); y < 5; y++ {
		for x := int32(0); x < 5; x++ {
			i, _ := w.cellIndex(x, y)
			want := int32(0)
			if x >= at.X && x < at.X+2 && y >= at.Y && y < at.Y+2 {
				want = 1
			}
			if got := s.at(DomainAir.layer(), i); got != want {
				t.Fatalf("moving 2x2 flyer at (%d,%d) leaves occupancy %d at (%d,%d), want %d", at.X, at.Y, got, x, y, want)
			}
		}
	}
}
