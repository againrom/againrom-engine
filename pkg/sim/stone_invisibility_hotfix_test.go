package sim

import "testing"

func TestStoneCurseFreezesMovementAttackAndAnAdmittedCastUntilExpiry(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, HasTarget: true, TargetX: 8, TargetY: 2,
			Transit: 2, TransitTotal: 3},
		{ID: 2, X: 4, Y: 4, HP: 100, MaxHP: 100, HasAttackTarget: true, AttackTarget: 3,
			AttackPhase: AttackCharging, AttackCountdown: 1, DamageBase: 9, DamageSpread: 0,
			AlwaysHits: true, Reach: 1, AttackCharge: 2},
		{ID: 3, X: 5, Y: 4, HP: 100, MaxHP: 100},
		{ID: 4, X: 7, Y: 7, HP: 100, MaxHP: 100, Mind: 10, Mana: 50, MaxMana: 50,
			KnownSpells: 1 << 1, AttackCharge: 8},
	}
	rule := SpellRule{ID: 1, ManaCost: 2, MaxRange: 10, TargetsUnit: true, Damaging: true,
		DamageMin: 3, DamageMax: 3}
	w, err := NewSpelledWorld(0x570e, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents, nil,
		[]SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	w.bookCasts = []bookCast{{Caster: 4, Target: 3, Spell: 1, X: 5, Y: 4,
		Remaining: 2, Phase: bookCharging}}
	for _, id := range []EntityID{1, 2, 4} {
		w.attached = append(w.attached, attachedEffect{Target: id, Spell: 20, Kind: EffectAbsorption,
			Mode: EffectDuration, Remaining: 2})
	}

	Step(w, nil)
	if got := w.entities[0]; got.Transit != 2 || got.X != 2 || got.Y != 2 {
		t.Fatalf("cursed mover advanced to (%d,%d) transit %d, want its exact frozen state", got.X, got.Y, got.Transit)
	}
	if got := w.entities[1]; got.AttackCountdown != 1 || w.entities[2].HP != 100 {
		t.Fatalf("cursed attacker countdown=%d victim hp=%d, want 1 and 100", got.AttackCountdown, w.entities[2].HP)
	}
	if got := w.bookCasts[0].Remaining; got != 2 {
		t.Fatalf("cursed caster's admitted wind-up advanced to %d, want frozen 2", got)
	}

	// Remaining 1 expires at the head of this tick. Each action then resumes
	// from the exact counters it held rather than from a reset action.
	Step(w, nil)
	if got := w.entities[0].Transit; got != 1 {
		t.Fatalf("released mover transit=%d, want resumed 1", got)
	}
	if w.entities[2].HP != 91 {
		t.Fatalf("released attacker left victim at %d hp, want its held 9-damage blow", w.entities[2].HP)
	}
	if got := w.bookCasts[0].Remaining; got != 1 {
		t.Fatalf("released caster wind-up=%d, want resumed 1", got)
	}
}

func TestStoneCurseRejectsFreshPlayerAndGroupActions(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, Group: 7, Mind: 10, Mana: 50,
			MaxMana: 50, KnownSpells: 1 << 1},
		{ID: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, Owner: 2},
	}
	rule := SpellRule{ID: 1, ManaCost: 2, MaxRange: 10, TargetsUnit: true, Damaging: true,
		DamageMin: 3, DamageMax: 3}
	w, err := NewSpelledWorld(0x570f, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents, nil,
		[]SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	w.attached = []attachedEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption,
		Mode: EffectDuration, Remaining: 20}}
	Step(w, []Command{
		{Kind: KindMoveTo, Entity: 1, X: 9, Y: 9},
		{Kind: KindAttack, Entity: 1, X: 2},
		{Kind: KindCast, Entity: 1, X: 2, Y: 1},
		{Kind: KindGroupMoveTo, Entity: 1, X: 8, Y: 8, Group: 44},
	})
	e := w.entities[0]
	if e.HasTarget || e.HasAttackTarget || len(w.bookCasts) != 0 || e.Mana != 50 {
		t.Fatalf("cursed actor accepted an action: target=%v attack=%v casts=%d mana=%d",
			e.HasTarget, e.HasAttackTarget, len(w.bookCasts), e.Mana)
	}
}

func TestInvisibleTargetCannotBeAcquiredAndBreaksAnExistingAttack(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, ScanRange: 12, Reach: 1,
			DamageBase: 10, DamageSpread: 0, AlwaysHits: true},
		{ID: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, Owner: 2},
	}
	w, err := NewWorld(0x1a15, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	w.attached = []attachedEffect{{Target: 2, Spell: 15, Kind: EffectInvisible,
		Mode: EffectDuration, Remaining: 20}}
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	if w.entities[0].HasAttackTarget {
		t.Fatal("player attack acquired an invisible undetected target")
	}

	w.entities[0].HasAttackTarget = true
	w.entities[0].AttackTarget = 2
	w.entities[0].AttackPhase = AttackCharging
	w.entities[0].AttackCountdown = 1
	Step(w, nil)
	if w.entities[0].HasAttackTarget || w.entities[1].HP != 100 {
		t.Fatalf("existing attack survived invisibility: held=%v victim hp=%d",
			w.entities[0].HasAttackTarget, w.entities[1].HP)
	}
}

func TestSeeInvisibleKeepsTheTargetAttackable(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Owner: 1, ScanRange: 12, SeeInvisible: 2,
			Reach: 1, DamageBase: 10, DamageSpread: 0, AlwaysHits: true},
		{ID: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, Owner: 2},
	}
	w, err := NewWorld(0x1a16, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	w.attached = []attachedEffect{{Target: 2, Spell: 15, Kind: EffectInvisible,
		Mode: EffectDuration, Remaining: 20}}
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	if !w.entities[0].HasAttackTarget {
		t.Fatal("detector failed to acquire the invisible target inside its radius")
	}
}
