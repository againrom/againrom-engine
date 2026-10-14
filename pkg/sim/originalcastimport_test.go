package sim

import "testing"

// A charge an original document holds with a weapon borrow is a release for
// a mage and stays a blow for a fighter, whose weapon spell is a rider.
func TestOriginalWeaponCastChargeIsAReleaseOnlyForAMage(t *testing.T) {
	for _, tc := range []struct {
		name string
		mana int32
		want AttackPhase
	}{{"mage", 10, AttackCasting}, {"fighter", 0, AttackCharging}} {
		t.Run(tc.name, func(t *testing.T) {
			w := pursuitEscapeWorld(t)
			w.spells = []SpellRule{{ID: 20, TargetsUnit: true, MaxRange: 10}}
			w.entities[0].WeaponSpell, w.entities[0].MaxMana, w.entities[0].Mana = 20, tc.mana, tc.mana
			if err := w.ImportOriginalActorActions([]OriginalActorAction{{Entity: 1, HasTarget: true, Target: 2, Phase: AttackCasting}}); err != nil {
				t.Fatal(err)
			}
			if got := w.entities[0].AttackPhase; got != tc.want {
				t.Fatalf("imported phase %d, want %d", got, tc.want)
			}
		})
	}
}

// A book cast's recovery countdown is the caster's CastWait (MAGIC-CADENCE-126).
func TestOriginalBookRecoveryImportsItsCountdown(t *testing.T) {
	w := pursuitEscapeWorld(t)
	if err := w.ImportOriginalActorActions([]OriginalActorAction{{Entity: 1, CastWait: 9}}); err != nil {
		t.Fatal(err)
	}
	if got := w.entities[0].CastWait; got != 9 {
		t.Fatalf("CastWait %d, want 9", got)
	}
}
