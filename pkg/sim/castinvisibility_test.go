package sim

import "testing"

// A cast at another unit or at a cell leaves the caster's invisibility a
// remainder of one tick (MAGIC-CAST-003); a cast at oneself leaves it alone.
func TestCastAtAnotherTargetLeavesInvisibilityOneTick(t *testing.T) {
	for _, second := range []bool{false, true} {
		name := map[bool]string{false: "first", true: "second"}[second]
		mk := func(t *testing.T) (*World, SpellRule) {
			t.Helper()
			hide := SpellRule{ID: 15, Arm: InvisibilitySpellArm(), TargetsUnit: true, Defensive: true, Delivery: 1}
			bolt := SpellRule{ID: 1, ManaCost: 2, MaxRange: 10, TargetsUnit: true, Damaging: true,
				DamageMin: 3, DamageMax: 3, Delivery: 1}
			rows := []SpellRule{bolt, hide}
			if second {
				hide.ID = 12
				rows = []SpellRule{secondRow(bolt), secondRow(hide)}
			}
			caster := effectMage(1, 2, 2, 1<<rows[0].ID)
			caster.AttackCharge = 8
			w := spWorld(t, 1, rows, caster, spEnt(2, 4, 2))
			if second {
				w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
			}
			if !w.attachEffect(1, 1, rows[1], EffectInvisible, 1, 30, EffectDuration) || !w.invisible(0) {
				t.Fatal("invisibility did not attach")
			}
			return w, rows[0]
		}
		remaining := func(w *World) (uint16, bool) {
			for _, e := range w.attached {
				if e.Target == 1 && e.Spell == w.hidingSpell() {
					return e.Remaining, true
				}
			}
			return 0, false
		}
		admit := func(w *World, rule SpellRule, c bookCast) {
			c.Caster, c.Spell = 1, rule.ID
			w.bookCasts = []bookCast{c}
			w.admitBookPayment(0, rule)
		}
		t.Run(name+"/unit", func(t *testing.T) {
			w, bolt := mk(t)
			if !w.castSpell(0, 2, uint32(bolt.ID), nil) {
				t.Fatal("cast refused")
			}
			if r, ok := remaining(w); !ok || r != 1 {
				t.Fatalf("remaining=%d present=%v, want 1", r, ok)
			}
			w.stepAttachedEffects()
			if _, ok := remaining(w); ok {
				t.Fatal("invisibility survived the next tick")
			}
		})
		t.Run(name+"/admission-unit", func(t *testing.T) {
			w, bolt := mk(t)
			admit(w, bolt, bookCast{Target: 2})
			if r, ok := remaining(w); !ok || r != 1 {
				t.Fatalf("remaining=%d present=%v, want 1", r, ok)
			}
		})
		t.Run(name+"/cell", func(t *testing.T) {
			w, bolt := mk(t)
			admit(w, bolt, bookCast{Target: 1, AtCell: true, X: 5, Y: 5})
			if r, ok := remaining(w); !ok || r != 1 {
				t.Fatalf("remaining=%d present=%v, want 1", r, ok)
			}
			w.stepAttachedEffects()
			if _, ok := remaining(w); ok {
				t.Fatal("invisibility survived the next tick")
			}
		})
		t.Run(name+"/self", func(t *testing.T) {
			w, bolt := mk(t)
			admit(w, bolt, bookCast{Target: 1})
			if r, ok := remaining(w); !ok || r != 30 {
				t.Fatalf("remaining=%d present=%v, want unchanged 30", r, ok)
			}
		})
	}
}
