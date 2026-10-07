package sim

import (
	"bytes"
	"strings"
	"testing"
)

func TestSecondaryDamageSelectsExactlyOneProtectionSchool(t *testing.T) {
	strike := func(selector uint8) int32 {
		t.Helper()
		a := cbFighter(1, 0, 0, 0, 10)
		a.DamageBase = 0
		a.SecondaryDamage = SecondaryDamage{Base: 20, Selector: selector}
		v := cbEnt(2, 1, 0)
		v.Protection = [5]int32{100, 0, 0, 0, 0}
		w := cbWorld(t, 17, a, v)
		Step(w, []Command{cbOrder(1, 2)})
		return cbAt(t, w, 2).HP
	}
	if hp := strike(0); hp != 100 {
		t.Fatalf("Fire secondary damage through 100%% Fire protection left HP %d, want 100", hp)
	}
	if hp := strike(1); hp != 80 {
		t.Fatalf("Water secondary damage read a different school or was omitted: HP %d, want 80", hp)
	}
}

// TestSecondaryDamageUsesThePhysicalPairGateOnAMiss is HERO-DMG2-029's
// branch at the shared production consumer. Seed 17's second draw is an
// ordinary miss (roll -22), never the auto-hit band. The first case therefore
// reaches the secondary draw only because its physical base/spread pair is
// empty. Restoring resolveBlow's former unconditional miss return changes its
// HP from 80 to 100 and its draw count from three to two.
func TestSecondaryDamageUsesThePhysicalPairGateOnAMiss(t *testing.T) {
	cases := []struct {
		name              string
		physicalBase      int32
		secondary         SecondaryDamage
		toHit             int32
		wantHP, wantDraws int
	}{
		{"secondary only survives miss", 0, SecondaryDamage{Base: 20, Selector: 0}, 0, 80, 3},
		{"physical and secondary miss together", 7, SecondaryDamage{Base: 20, Selector: 0}, 0, 100, 2},
		{"physical only misses", 7, SecondaryDamage{}, 0, 100, 2},
		{"physical and secondary hit unchanged", 7, SecondaryDamage{Base: 20, Selector: 0}, 2000, 73, 3},
		{"empty pair without secondary stays zero", 0, SecondaryDamage{}, 0, 100, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := cbEnt(1, 0, 0)
			a.DamageBase = tc.physicalBase
			a.ToHit = tc.toHit
			a.SecondaryDamage = tc.secondary
			v := cbEnt(2, 1, 0)
			v.Defence = 1000
			w := cbWorld(t, 17, a, v)
			before := w.rng.state

			w.resolveBlow(0, 1, nil)

			if got := int(cbAt(t, w, 2).HP); got != tc.wantHP {
				t.Fatalf("target HP = %d, want %d", got, tc.wantHP)
			}
			if got := cbDraws(t, before, w.rng.state); got != tc.wantDraws {
				t.Fatalf("resolver draws = %d, want %d", got, tc.wantDraws)
			}
		})
	}
}

func TestSecondaryDamageRoundTripsHashesAndRejectsMalformedSelectors(t *testing.T) {
	entity := Entity{ID: 1, X: 1, Y: 1,
		SecondaryDamage: SecondaryDamage{Base: 11, Spread: 3, Selector: 4}}
	w := rcWorld(t, entity)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := back.Entities()[0].SecondaryDamage; got != entity.SecondaryDamage {
		t.Fatalf("round-trip secondary damage = %+v, want %+v", got, entity.SecondaryDamage)
	}
	again, err := back.MarshalBinary()
	if err != nil || !bytes.Equal(form, again) || back.Hash() != w.Hash() {
		t.Fatalf("round trip changed form/hash: err=%v bytes=%v hash=%#x/%#x", err,
			bytes.Equal(form, again), back.Hash(), w.Hash())
	}

	otherEntity := entity
	otherEntity.SecondaryDamage.Selector = 3
	other := rcWorld(t, otherEntity)
	if other.Hash() == w.Hash() {
		t.Fatal("worlds differing only in the selected protection school hash alike")
	}

	// With one empty entity, the canonical item-state section is: its two
	// counts, one carried count, twelve empty item heads, one source tag,
	// twelve regeneration/rotation bytes, then base, spread and selector.
	itemStateAt := len(form) - entityIDFloorLen - spellDeliverySpanLen - 65 - w.scriptSectionLen() - relationLen - w.originalDeadSectionLen() - w.instanceWeightSectionLen() - w.actorLoadSectionLen() - w.scrollSectionLen() - w.itemStateSectionLen()
	selectorAt := itemStateAt + 2*itemStateCountLen + itemStateCountLen + EquipSlots*itemHeadLen + 1 + 3*4 + 2
	malformed := append([]byte(nil), form...)
	malformed[selectorAt] = 5
	if err := back.UnmarshalBinary(malformed); err == nil || !strings.Contains(err.Error(), "secondary-damage selector 5") {
		t.Fatalf("malformed selector error = %v", err)
	}
	coexisting := append([]byte(nil), form...)
	coexisting[selectorAt+1] = 1
	if err := back.UnmarshalBinary(coexisting); err == nil || !strings.Contains(err.Error(), "secondary-damage reserved byte") {
		t.Fatalf("coexisting legacy component error = %v", err)
	}

	bad := entity
	bad.SecondaryDamage.Selector = 5
	if _, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{bad}); err == nil || !strings.Contains(err.Error(), "secondary-damage selector 5") {
		t.Fatalf("constructor malformed selector error = %v", err)
	}
}

func TestSetCombatReplacesTheWholeSecondaryDamageTripleAtomically(t *testing.T) {
	e := rcEnt(1, 1, 1, 1, 2, 3, 4, 5, 6, 7, false, 3)
	e.SecondaryDamage = SecondaryDamage{Base: 5, Spread: 7, Selector: 0}
	w := rcWorld(t, e)
	want := SecondaryDamage{Base: 9, Spread: 2, Selector: 1}
	if !w.SetCombat(1, CombatBlock{SecondaryDamage: want}) {
		t.Fatal("SetCombat refused a valid Water triple")
	}
	if got := rcAt(t, w, 1).SecondaryDamage; got != want {
		t.Fatalf("SetCombat secondary damage = %+v, want %+v", got, want)
	}

	copyOfEntities := w.Entities()
	copyOfEntities[0].SecondaryDamage.Selector = 4
	if got := rcAt(t, w, 1).SecondaryDamage; got != want {
		t.Fatalf("Entities copy mutated canonical triple to %+v", got)
	}

	before, hashBefore := rcAt(t, w, 1), w.Hash()
	if w.SetCombat(1, CombatBlock{DamageBase: 999,
		SecondaryDamage: SecondaryDamage{Base: 1, Spread: 2, Selector: 5}}) {
		t.Fatal("SetCombat accepted secondary-damage selector 5")
	}
	if got := rcAt(t, w, 1); got != before || w.Hash() != hashBefore {
		t.Fatalf("refused SetCombat mutated state: before=%+v after=%+v hashes=%#x/%#x",
			before, got, hashBefore, w.Hash())
	}
	if w.SetDerived(1, DerivedBlock{MaxHP: 999, Speed: 999,
		Combat: CombatBlock{SecondaryDamage: SecondaryDamage{Selector: 5}}}) {
		t.Fatal("SetDerived accepted secondary-damage selector 5")
	}
	if got := rcAt(t, w, 1); got != before || w.Hash() != hashBefore {
		t.Fatalf("refused SetDerived mutated state: before=%+v after=%+v hashes=%#x/%#x",
			before, got, hashBefore, w.Hash())
	}
}
