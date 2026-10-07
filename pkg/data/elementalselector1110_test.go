package data

import "testing"

func TestHumanDerivedElementalSelectorMatchesResolverPermutation(t *testing.T) {
	for raw := uint8(1); raw <= 5; raw++ {
		h := trainingHuman()
		h.Attack.ElementalBase, h.Attack.ElementalSpread, h.Attack.ElementalKind = 8, 9, raw
		if err := h.ProjectionError(); err != nil {
			t.Fatalf("raw kind %d: unexpected ProjectionError %v", raw, err)
		}
		d := h.Derived(nil, 0)
		want := ElementalSelectorOrder[raw-1]
		if d.SecondaryDamage.Selector != want || d.Combat.SecondaryDamage.Selector != want {
			t.Errorf("raw kind %d: Selector = (%d,%d), want %d from the resolver permutation",
				raw, d.SecondaryDamage.Selector, d.Combat.SecondaryDamage.Selector, want)
		}
	}
	// Zero base/spread carries no elemental component: the selector stays 0
	// (Fire) regardless of the raw stored kind, on both accessors.
	h := trainingHuman()
	h.Attack.ElementalBase, h.Attack.ElementalSpread, h.Attack.ElementalKind = 0, 0, 3
	d := h.Derived(nil, 0)
	if d.SecondaryDamage.Selector != 0 || d.Combat.SecondaryDamage.Selector != 0 {
		t.Fatalf("zero base/spread kept a nonzero selector: %+v", d.SecondaryDamage)
	}
}

// ElementalSelectorOrder pins HERO-DMG2-029's exact permutation so a future
// edit cannot silently drift back toward raw-1 without failing here first.
func TestElementalSelectorOrderPinsResolverPermutation(t *testing.T) {
	want := [5]uint8{0, 3, 2, 1, 4}
	if ElementalSelectorOrder != want {
		t.Fatalf("ElementalSelectorOrder = %v, want %v (HERO-DMG2-029)", ElementalSelectorOrder, want)
	}
}
