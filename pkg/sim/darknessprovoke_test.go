package sim

import "testing"

// darknessWorld is a first-game world with a caster (owner SelfSlot) at (x,y),
// two guards of owner 2 at (40,40) and (40,41), whose notice circle has radius
// 12, and a Darkness cloud row.
func darknessWorld(t *testing.T, x, y int32) (*World, SpellRule) {
	t.Helper()
	rule := SpellRule{ID: 17, Area: true, Distribution: distributionDiamond, Radius: 1, AreaDuration: 2,
		TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 200}
	caster := effectMage(1, x, y, 1<<17)
	first, second := engFighter(2, 2, 40, 40), engFighter(3, 2, 40, 41)
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical,
		Terrain{}, []Entity{caster, first, second}, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	wakeGroups(w)
	return w, rule
}

// A first-game Darkness hit runs the blow hook and writes no order: each
// victim remembers the caster's cell and turns hostile to it, and holds no
// victim, through the pulses and the group passes that follow, because the
// caster stands outside the group's circle (AI-707, MAGIC-LIGHTDARK-044).
func TestADarknessHitSendsNoVictimAtTheCaster(t *testing.T) {
	t.Parallel()
	w, rule := darknessWorld(t, 2, 2)
	if !w.landArea(rule, 0, 1, true, 2, 2, 40, 40, nil) {
		t.Fatal("fixture: the Darkness cloud did not land")
	}
	hit := false
	for k := 0; k < 64; k++ {
		Step(w, nil)
		for _, id := range []EntityID{2, 3} {
			e := laEnt(t, w, id)
			if e.HasAttackTarget {
				t.Fatalf("tick %d: Darkness victim %d holds victim %d", k, id, e.AttackTarget)
			}
			if n := w.attackNoticeAt(indexOfEntity(w.entities, id)); n.Cell == 2|2<<8 {
				hit = true
			}
		}
	}
	if !hit {
		t.Fatal("no victim remembered the caster's cell, so the cloud never hit")
	}
	if !w.relations.Hostile(2, SelfSlot) {
		t.Error("the hit did not turn the victims' owner hostile to the caster")
	}
}

// A Darkness pulse resumed from a SAV payload answers as the live pulse does:
// the victim remembers the caster's cell and holds no victim.
func TestALoadedDarknessPulseSendsNoVictimAtTheCaster(t *testing.T) {
	t.Parallel()
	live, rule := darknessWorld(t, 2, 2)
	if !live.landArea(rule, 0, 1, true, 2, 2, 40, 40, nil) {
		t.Fatal("fixture: the Darkness cloud did not land")
	}
	rows, err := live.NativeAreaSaveStates()
	if err != nil || len(rows) != 1 {
		t.Fatalf("fixture: area save states %d, %v", len(rows), err)
	}
	w, _ := darknessWorld(t, 2, 2)
	at := uint16(40 | 40<<8)
	w.applySavedAreaPayload(SavedAreaDriver{Key: at, Spell: rule.ID, Mode: areaModeCloud}, SavedSpellEffect{AE44: &rows[0].Payload}, []uint16{at}, 1)
	i := indexOfEntity(w.entities, 2)
	if e := w.entities[i]; e.HasAttackTarget {
		t.Errorf("the loaded pulse gave its victim victim %d", e.AttackTarget)
	}
	if n := w.attackNoticeAt(i); n.Cell != 2|2<<8 {
		t.Errorf("the victim remembers cell %#x, want the caster's (2,2)", n.Cell)
	}
}

// A caster inside the group's circle is taken by the group pass after the
// hit, not by the hit: on the tick the victims first remember the caster's
// cell none of them holds it, and a later pass gives it to them as an
// ordinary engage (AI-707).
func TestADarknessCasterInsideTheCircleIsTakenByTheNextGroupPass(t *testing.T) {
	t.Parallel()
	w, rule := darknessWorld(t, 30, 40)
	if !w.landArea(rule, 0, 1, true, 30, 40, 40, 40, nil) {
		t.Fatal("fixture: the Darkness cloud did not land")
	}
	hitAt, takenAt := -1, -1
	for k := 0; k < 96 && takenAt < 0; k++ {
		Step(w, nil)
		for _, id := range []EntityID{2, 3} {
			i := indexOfEntity(w.entities, id)
			if hitAt < 0 && w.attackNoticeAt(i).Cell == 30|40<<8 {
				hitAt = k
				for _, other := range []EntityID{2, 3} {
					if e := laEnt(t, w, other); e.HasAttackTarget {
						t.Fatalf("tick %d: victim %d holds victim %d on the tick of the hit", k, other, e.AttackTarget)
					}
				}
			}
			if e := w.entities[i]; e.HasAttackTarget && e.AttackTarget == 1 && takenAt < 0 {
				takenAt = k
			}
		}
	}
	if hitAt < 0 {
		t.Fatal("no victim remembered the caster's cell, so the cloud never hit")
	}
	if takenAt <= hitAt {
		t.Errorf("the caster was taken at tick %d, the hit came at tick %d; want a later group pass to take it", takenAt, hitAt)
	}
}
