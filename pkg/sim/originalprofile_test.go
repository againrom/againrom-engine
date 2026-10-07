package sim

import (
	"reflect"
	"testing"
)

func TestOriginalProfile1107ImportIsAtomicAndNotARebuild(t *testing.T) {
	a, v := cbEnt(1, 1, 1), cbEnt(2, 2, 1)
	a.KnownSpells = 2
	a.Book = Spellbook{State: BookPresent}
	a.Book.Slots[0] = BookSpell{Range: 17, ManaCost: 37, Defensive: 2}
	a.AttackPhase = AttackCharging
	a.AttackCharge = 5
	a.AttackCountdown = 1
	a.AttackTarget = 2
	a.HasAttackTarget = true
	w := cbWorld(t, 17, a, v)
	before := w.Hash()
	rng := w.rng.state
	original := w.Entities()[0]
	p := OriginalActorProfile{ID: 1, ToHit: 2000, DamageBase: 40, XPSlot: 1, SecondBase: 20, SecondaryDamage: SecondaryDamage{Base: 8},
		Defence: -7, Absorption: -5, Protection: [5]int16{-1, 2, -3, 4, -5}, Resistance: [5]uint8{10, 128, 200, 254, 255},
		Reaction: -32768, Mind: 32767, Spirit: -2, HealthPeriod: -32768, ManaPeriod: -1, HealthRegeneration: -32768, ManaRegeneration: -101, HealthHundredths: 255, ManaHundredths: 199}
	for _, bad := range []OriginalActorProfile{{ID: 99}, {ID: 2, XPSlot: 6}, {ID: 2, SecondaryDamage: SecondaryDamage{Selector: 5}}, p} {
		if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{p, bad}); err == nil || w.Hash() != before {
			t.Fatal("non-atomic import", bad, err)
		}
	}
	if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{p}); err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if w.rng.state != rng || w.Tick() != 0 || e.HP != original.HP || e.MaxHP != original.MaxHP || e.Mana != original.Mana || e.MaxMana != original.MaxMana ||
		e.Speed != original.Speed || e.AttackPhase != original.AttackPhase || e.AttackCountdown != original.AttackCountdown || e.Book != original.Book || e.Skill != original.Skill || e.SkillXP != original.SkillXP {
		t.Fatal("import rebuilt unrelated state")
	}
	if e.ToHit != 2000 || e.DamageBase != 40 || e.SecondBase != 20 || e.Defence != -7 || e.Absorption != -5 || e.Protection != ([5]int32{-1, 2, -3, 4, -5}) || e.Resistance != p.Resistance || e.CurrentProfileBasis != ProfileOriginalCurrent || e.HealthHundredths != 255 {
		t.Fatal("current fields changed", e)
	}
	if e.HealthRegenPeriod != -32768 || e.ManaRegenPeriod != -1 || e.HealthRegeneration != -32768 || e.ManaRegeneration != -101 || e.ManaHundredths != 199 {
		t.Fatal("signed regeneration group normalized", e)
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil || back.Hash() != w.Hash() {
		t.Fatal("source bytes did not roundtrip", err)
	}
	// Explicit retirement retains the transported raw remainder until a real
	// consumer updates it. It must not make an ordinary native save unreadable.
	if !w.SetCombat(1, CombatBlock{SecondBase: 20, Reach: 1, AttackCharge: 5}) {
		t.Fatal("native mutation refused")
	}
	if w.Entities()[0].CurrentProfileBasis != ProfileNativeRetired || w.Entities()[0].HealthHundredths != 255 {
		t.Fatal("retirement clamped")
	}
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil || back.Hash() != w.Hash() {
		t.Fatal("retired save refused", err)
	}
}

func TestOriginalProfile1107ActualBlowUsesCurrentThreeComponents(t *testing.T) {
	a, v := cbEnt(1, 1, 1), cbEnt(2, 2, 1)
	a.ToHit = 0
	a.DamageBase = 1
	v.HP = 100
	v.MaxHP = 100
	w := cbWorld(t, 17, a, v)
	if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{
		{ID: 1, ToHit: 2000, DamageBase: 40, XPSlot: 1, SecondBase: 20, SecondaryDamage: SecondaryDamage{Base: 8}},
		{ID: 2, Absorption: 10, Protection: [5]int16{50, 25}, Resistance: [5]uint8{50}},
	}); err != nil {
		t.Fatal(err)
	}
	// 40-10, reduced by 50%=15; second 20 reduced by 25%=15;
	// elemental 8 reduced by 50%=4. Literal answer: 100-34=66.
	report := StepReported(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	for i := 0; i < 32 && len(report.Damages) == 0; i++ {
		report = StepReported(w, nil)
	}
	if len(report.Damages) == 0 {
		t.Fatal("new attack did not deal damage")
	}
	if got := cbAt(t, w, 2).HP; got != 66 {
		t.Fatalf("current blow HP=%d want66", got)
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		a, b := StepReported(w, nil), StepReported(&back, nil)
		if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
			t.Fatal("blow continuation", i)
		}
	}
}

func TestOriginalProfile1107AdmittedRegenLiteralVectors(t *testing.T) {
	for _, tc := range []struct {
		modifier int16
		hp, mana int32
		fraction uint8
	}{{0, 12, 11, 0}, {50, 13, 11, 50}, {-100, 10, 10, 0}} {
		w := cbWorld(t, 17, Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 100, Mana: 10, MaxMana: 100})
		if err := w.ImportOriginalActorProfiles([]OriginalActorProfile{{ID: 1, HealthPeriod: 100, ManaPeriod: 100, HealthRegeneration: tc.modifier, ManaRegeneration: tc.modifier}}); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 13; i++ {
			Step(w, nil)
		}
		e := w.Entities()[0]
		if e.HP != tc.hp || e.Mana != tc.mana || e.ManaHundredths != tc.fraction {
			t.Fatalf("modifier%d: %d/%d remainder%d", tc.modifier, e.HP, e.Mana, e.ManaHundredths)
		}
	}
}
