package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestPoison1164DurationProducerKeepsOtherModesAndRows(t *testing.T) {
	rows := make([]data.Spell, 12)
	for i := range rows {
		rows[i].Effect = data.SpellEffect{Kind: data.SpellEffectHealth, Mode: data.SpellEffectContinuous, Magnitude: -2, Duration: 8}
	}
	got := spellRules(rows, false)
	for i, r := range got {
		want := uint16(128)
		if r.EffectDuration != want || r.EffectMagnitude != -2 || r.EffectMode != sim.EffectContinuous {
			t.Fatal("Poison counter/magnitude or unrelated row changed", i, r)
		}
	}
	rows[7].Effect.Mode = data.SpellEffectCharges
	if r := spellRules(rows, false)[7]; r.EffectDuration != 128 {
		t.Fatal("charges counter lost the numeric word shift", r)
	}
}
