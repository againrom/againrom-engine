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
	got := spellRules(rows)
	for i, r := range got {
		want := uint16(8)
		if i == 7 {
			want = 128
		}
		if r.EffectDuration != want || r.EffectMagnitude != -2 || r.EffectMode != sim.EffectContinuous {
			t.Fatal("Poison counter/magnitude or unrelated row changed", i, r)
		}
	}
	rows[7].Effect.Mode = data.SpellEffectCharges
	if r := spellRules(rows)[7]; r.EffectDuration != 8 {
		t.Fatal("charges duration was shifted", r)
	}
}
