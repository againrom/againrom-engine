package sim

import "testing"

// R3-A2 (round3-review.md): fix round 2 read one actor per cell-record layer
// (cellSlotOccupants, at most one ground-and-ghost index and one air index),
// on TERR-CELLREC-146's decoded cap. That cap is a consequence of ROM1's own
// movement refusing to register an actor's footprint over a cell another
// actor already holds (TERR-FOOTPRINT-147). At that review this build's
// occupancy plane was anchor-only, so two ground actors could genuinely cover
// one cell through runtime movement or Teleport.
// Reading one occupant per layer for that state made the second actor
// immune to every area effect on the shared cell.

// TestAreaEffectHitsEveryGroundActorOnAFootprintContestedCell reproduces the
// review's own measured configuration: a 2x2 actor anchored at (6,6), whose
// footprint covers (6,6) through (7,7), and a 1x1 actor anchored at (7,7) —
// inside that footprint in an already-overlapping synthetic state. An area
// effect covering (7,7) alone must reach both, at every mode
// applyAreaCells's own layer split separates.
//
// TO CONFIRM IT WITNESSES THE FIX, revert cellLayerOccupants to the old
// single-int cellSlotOccupants (ground, air int; first found wins, break on
// both set) and applyAreaCells's loop to `slots[:read]` over that pair, then
// rerun: the 1x1 actor inside the footprint reads exactly 100 (untouched).
func TestAreaEffectHitsEveryGroundActorOnAFootprintContestedCell(t *testing.T) {
	rule := SpellRule{ID: 1, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	for _, mode := range []struct {
		name string
		mode uint8
	}{
		{"blast", areaModeBlast},
		{"ring", areaModeRing},
		{"cloud", areaModeCloud},
	} {
		t.Run(mode.name, func(t *testing.T) {
			big := Entity{ID: 1, X: 6, Y: 6, HP: 100, MaxHP: 100, TokenSize: 2}
			small := Entity{ID: 2, X: 7, Y: 7, HP: 100, MaxHP: 100, TokenSize: 1}
			w := mustWorld(t, 0x1001, Bounds{Width: 40, Height: 40}, []Entity{big, small})

			w.applyAreaCells(cellEffect{Spell: 1, Power: 30, Mode: mode.mode}, rule, []uint16{cellKey(7, 7)})

			if got := w.entities[0].HP; got >= 100 {
				t.Errorf("the 2x2 footprint holder (entity 1) took no damage: HP=%d", got)
			}
			if got := w.entities[1].HP; got >= 100 {
				t.Errorf("the 1x1 actor inside the footprint (entity 2, at the same cell (7,7)) "+
					"took no damage: HP=%d, want it hit like entity 1", got)
			}
		})
	}
}

// TestFireballDamagesATargetStandingInsideAnotherActorsFootprint is the
// review's own R3-A2 measurement, reproduced: "2x2 actor 1 at (6,6); 1x1
// actor 2 at (7,7) — inside its footprint. cellSlotOccupants(7,7) = ground
// index 0, air index -1. Fire Ball at (7,7): 2x2 HP 100 -> -4, 1x1 HP 100 ->
// 100." Fire Ball (rule id 2) takes its own code path in applyAreaCells —
// one application per covered cell, divided by the target's own footprint
// area — so it is witnessed separately from the plain-rule table above.
//
// TO CONFIRM IT WITNESSES THE FIX, revert the same lines as the test above
// and rerun: entity 2 reads exactly 100.
func TestFireballDamagesATargetStandingInsideAnotherActorsFootprint(t *testing.T) {
	big := Entity{ID: 1, X: 6, Y: 6, HP: 100, MaxHP: 100, TokenSize: 2}
	small := Entity{ID: 2, X: 7, Y: 7, HP: 100, MaxHP: 100, TokenSize: 1}
	w := mustWorld(t, 0x1002, Bounds{Width: 40, Height: 40}, []Entity{big, small})

	rule := SpellRule{ID: 2, DamageMin: 10, DamageMax: 10, Damaging: true, TargetsUnit: true}
	w.applyAreaCells(cellEffect{Spell: 2, Power: 30, Mode: areaModeBlast}, rule, []uint16{cellKey(7, 7)})

	if got := w.entities[0].HP; got >= 100 {
		t.Errorf("the 2x2 footprint holder (entity 1) took no Fire Ball damage: HP=%d", got)
	}
	if got := w.entities[1].HP; got >= 100 {
		t.Errorf("the 1x1 target inside the footprint (entity 2) took no Fire Ball damage: HP=%d, "+
			"want it hit like the actor that used to win the slot race", got)
	}
}

// The same production path that used to create the overlap now refuses it
// before mana, recovery or position changes.
func TestTeleportDoesNotCreateAnOccupiedCellOverlap(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<26)
	other := Entity{ID: 2, X: 5, Y: 5, HP: 100, MaxHP: 100, TokenSize: 1}
	teleport := SpellRule{ID: 26, ManaCost: 20, School: 5, MaxRange: 16}
	w := hlWorld(t, 0x1003, Relations{}, []SpellRule{teleport}, caster, other)

	before := spAt(t, w, 1)
	spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 5, Y: 5, Spell: 26})
	after := spAt(t, w, 1)
	if after.X != before.X || after.Y != before.Y || after.Mana != before.Mana-20 || after.CastWait == 0 {
		t.Fatalf("occupied Teleport changed caster from (%d,%d), mana %d to (%d,%d), mana %d, recovery %d",
			before.X, before.Y, before.Mana, after.X, after.Y, after.Mana, after.CastWait)
	}
}
