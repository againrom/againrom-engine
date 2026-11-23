package sim

import (
	"reflect"
	"sort"
	"testing"
)

// TestASpellPayloadWithoutACasterSkipsBlessCurseAndAttribution pins HERO-205
// and MAGIC-454 at the first game's spell resolver. Without a caster the
// spread draw of the empty pair, the 0..200 roll and the payload draw run,
// and neither Bless nor Curse is rolled; the attribution writer leaves the
// target's credit alone. With a caster its own Bless and Curse are found by
// the bearer and the spell alone, whoever cast them.
func TestASpellPayloadWithoutACasterSkipsBlessCurseAndAttribution(t *testing.T) {
	widths := func(ci int) []int32 {
		t.Helper()
		w := cbWorld(t, 1, cbEnt(1, 0, 0), cbEnt(2, 1, 0), cbEnt(3, 2, 0))
		for _, spell := range []uint16{w.blessingSpell(), w.cursingSpell()} {
			w.attached = append(w.attached, attachedEffect{Target: 1, Spell: spell, Mode: EffectDuration,
				Magnitude: 50, Remaining: 100, Caster: 3, HasCaster: true})
		}
		sort.Slice(w.attached, func(i, j int) bool { return w.attached[i].Spell < w.attached[j].Spell })
		script := &scriptedDraws{answers: map[int32][]int32{100: {80, 80}, 200: {100}, 4: {2}}}
		w.rng.scripted = script
		if got := w.directDamage(ci, 1, 5, 4, 2); got != 7 {
			t.Errorf("caster index %d: damage = %d, want 7", ci, got)
		}
		return script.widths
	}
	if got, want := widths(-1), []int32{0, 200, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("no caster: draw widths = %v, want %v", got, want)
	}
	if got, want := widths(0), []int32{0, 100, 100, 200, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("caster with Bless and Curse: draw widths = %v, want %v", got, want)
	}

	w := cbWorld(t, 1, cbEnt(1, 0, 0), cbEnt(2, 1, 0))
	w.entities[1].KillCreditSource, w.entities[1].HasKillCredit = 9, true
	w.resolveDamageAttribution(-1, 1, 2)
	if e := w.entities[1]; e.KillCreditSource != 9 || !e.HasKillCredit {
		t.Errorf("no caster: credit = %d/%v, want 9/true unchanged", e.KillCreditSource, e.HasKillCredit)
	}
}
