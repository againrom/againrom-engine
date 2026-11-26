package sim

import "testing"

// The first game's sink scales the kill award in binary64: 180 x (3/30.0 +
// 0.25) is just below 63 there and truncates to 62, where the integer form
// the second game keeps answers 63 (HERO-209).
func TestFirstGameKillAwardScalesMindInBinary64(t *testing.T) {
	killer := killCreditFighter(1, 1)
	killer.Mind = 3
	victim := killCreditVictim(2, -5)
	victim.XPValue = 360
	victim.KillCreditSource, victim.HasKillCredit = 1, true
	w := spWorld(t, 1, nil, killer, victim)
	before := w.entities[0].SkillXP[1]
	w.processKillCredit(1)
	if got := w.entities[0].SkillXP[1] - before; got != 62 {
		t.Errorf("first-game kill award at Mind 3 for 180 = %d, want 62", got)
	}
	if got := xpGain(180, 3); got != 63 {
		t.Errorf("integer form at Mind 3 for 180 = %d, want 63", got)
	}
}

// The integer binary64 form equals the hardware's float64 expression over
// every Mind word from -100 to 400 and every amount up to 3000.
func TestBinaryMindGainEqualsTheFloat64Expression(t *testing.T) {
	for mind := int32(-100); mind <= 400; mind++ {
		for amount := int64(0); amount <= 3000; amount++ {
			want := int64(float64(amount) * (float64(mind)/30.0 + 0.25))
			if got := binaryMindGain(amount, mind); got != want {
				t.Fatalf("binaryMindGain(%d, %d) = %d, want %d", amount, mind, got, want)
			}
		}
	}
}

// A first-game mage's kill whose byte still holds the damage kind its
// resolver wrote: the award reads the byte as a spell id, and spells 1..5
// are Fire rows, so damage kind 3 trains Fire and not Air (MAGIC-473).
func TestFirstGameKillReadsTheDamageKindAsASpellId(t *testing.T) {
	var spells []SpellRule
	for id := uint16(1); id <= 5; id++ {
		spells = append(spells, SpellRule{ID: id, School: 1})
	}
	w := spWorld(t, 1, spells, killCreditMage(1, 3), killCreditVictim(2, 10))
	w.resolveDamageAttribution(0, 1, 3)
	if got := w.entities[1].KillCreditSpell; got != 3 {
		t.Fatalf("resolver wrote kill byte %d, want damage kind 3", got)
	}
	before := w.entities[0].SkillXP
	w.processKillCredit(1)
	for slot := range before {
		if rose := w.entities[0].SkillXP[slot] > before[slot]; rose != (slot == 1) {
			t.Errorf("skill %d rose %v", slot, rose)
		}
	}
}
