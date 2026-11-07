package sim

import "testing"

// claimRange is the second game's range draw as R2-ENGINE-959 states it: the
// core s = 214013 s + 2531011 yields (s >> 16) & 0x7fff, and the draw over
// 0..n is (core * (n + 1)) >> 15, with no core call for n = 0.
func claimRange(s *uint32, n int32) int32 {
	if n == 0 {
		return 0
	}
	*s = *s*214013 + 2531011
	return int32((int64(*s>>16&0x7fff) * int64(n+1)) >> 15)
}

// TestSecondSummonDrawsItsCreaturesFacingBetweenKindAndPlacement: one Summon
// cast on the original's stream rolls the kind 1..3, then Unit
// initialization draws the creature's facing 0x40 + 0..128 and copies it to
// the heading, then the placement tries draw y and x (R2-ENGINE-790,
// R2-ENGINE-959). The expected values come from the claim's arithmetic.
func TestSecondSummonDrawsItsCreaturesFacingBetweenKindAndPlacement(t *testing.T) {
	for _, state := range []uint32{0x1234567, 0x2468ace, 99} {
		w := summonWorld(t, 0x25, 0)
		w.rng = rng{state: uint64(state), original: true}
		s := state
		k := 1 + claimRange(&s, 2)
		facing := uint8(0x40 + claimRange(&s, 0x80))
		px, py := int32(-1), int32(-1)
		for try := 0; try < 20 && px < 0; try++ {
			cy := 16 - 3 + claimRange(&s, 6)
			cx := 16 - 3 + claimRange(&s, 6)
			if cx != 16 || cy != 16 {
				px, py = cx, cy
			}
		}
		if !w.ordinaryEffectPayload(0, 0, w.spells[0], 50) || len(w.entities) != 2 {
			t.Fatalf("state %#x: Summon did not land", state)
		}
		got := w.entities[1]
		var template SpellCreature
		for _, c := range summonCreatures() {
			if c.Name == SummonCreatureNames[k-1] {
				template = c
			}
		}
		if got.TypeID != template.Actor.TypeID || got.X != px || got.Y != py {
			t.Errorf("state %#x: built type %d at %d,%d, want type %d at %d,%d", state, got.TypeID, got.X, got.Y, template.Actor.TypeID, px, py)
		}
		if got.Facing != facing || got.DesiredFacing != facing {
			t.Errorf("state %#x: the creature faces %#x heading %#x, want %#x", state, got.Facing, got.DesiredFacing, facing)
		}
	}
}

// TestSecondAnimateDeadDrawsTheRaisedUnitsFacing: spell 22 builds its unit
// through Unit initialization (R2-ENGINE-781), which draws the facing, and
// places it with radius 0, which draws nothing (R2-ENGINE-782). On the
// original's stream the cast takes exactly that one draw; in seeded mode the
// facing lies in 0x40..0xC0 and equals the heading.
func TestSecondAnimateDeadDrawsTheRaisedUnitsFacing(t *testing.T) {
	for _, state := range []uint32{0x1234567, 0x2468ace} {
		w := animateWorld(t, animateCorpse(2, 2))
		w.rng = rng{state: uint64(state), original: true}
		s := state
		facing := uint8(0x40 + claimRange(&s, 0x80))
		if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 2 {
			t.Fatalf("state %#x: Animate Dead did not raise the corpse", state)
		}
		if got := w.entities[1]; got.Facing != facing || got.DesiredFacing != facing {
			t.Errorf("state %#x: the raised unit faces %#x heading %#x, want %#x", state, got.Facing, got.DesiredFacing, facing)
		}
		if w.RandomState() != uint64(s) {
			t.Errorf("state %#x: the cast left the stream at %#x, want %#x after one draw", state, w.RandomState(), s)
		}
	}
	w := animateWorld(t, animateCorpse(2, 2))
	if !w.ordinaryEffectPayload(0, 1, w.spells[0], 17) || len(w.entities) != 2 {
		t.Fatal("Animate Dead did not raise the corpse")
	}
	if got := w.entities[1]; got.Facing < 0x40 || got.Facing > 0xC0 || got.DesiredFacing != got.Facing {
		t.Errorf("seeded: the raised unit faces %#x heading %#x, want 0x40..0xC0 and equal", got.Facing, got.DesiredFacing)
	}
}
