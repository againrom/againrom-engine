package data

import "testing"

// TestElementKindsTwoAndFourReadWaterAndEarth is MAGIC-453: the resolver's
// table leads elemental kind 2 to the Water protection word and kind 4 to
// the Earth word. A first-game saved Human's raw kind and the class-12
// ranged weapon's kind 2 (ITEM-254) therefore select Water and Earth by
// column order. The second game's class-11 weapon (R2-ENGINE-479) keeps
// Fire.
func TestElementKindsTwoAndFourReadWaterAndEarth(t *testing.T) {
	for _, tc := range []struct{ raw, want uint8 }{{2, 1}, {4, 3}} {
		h := trainingHuman()
		h.Attack.ElementalBase, h.Attack.ElementalSpread, h.Attack.ElementalKind = 8, 9, tc.raw
		d := h.Derived(nil, 0)
		if d.Combat.SecondaryDamage.Selector != tc.want || d.SecondaryDamage.Selector != tc.want {
			t.Errorf("saved kind %d: Selector = (%d,%d), want Protection index %d",
				tc.raw, d.Combat.SecondaryDamage.Selector, d.SecondaryDamage.Selector, tc.want)
		}
	}
	w := Weapon{AttackType: 12, DamageBase: 4, DamageSpread: 2, Range: 6}
	if got := FoldWeapon(Combat{}, &w, 0).SecondaryDamage; got != (SecondaryDamage{Base: 4, Spread: 2, Selector: 1}) {
		t.Errorf("class-12 weapon: SecondaryDamage = %+v, want Water (Selector 1)", got)
	}
	w.AttackType = 11
	if got := FoldWeapon(Combat{}, &w, 0).SecondaryDamage; got != (SecondaryDamage{Base: 4, Spread: 2, Selector: 0}) {
		t.Errorf("class-11 weapon: SecondaryDamage = %+v, want Fire (Selector 0)", got)
	}
}
