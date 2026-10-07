package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestCurrentActorAttachmentPreservesModeAndCounter(t *testing.T) {
	for mode := sim.EffectMode(1); mode <= 7; mode++ {
		for _, remaining := range []uint16{0, 1, 9601} {
			current := sim.ActiveEffect{Target: 7, Caster: 2, HasCaster: true, Spell: 12, Kind: sim.EffectScanRange, Mode: mode, Magnitude: -3, Remaining: remaining}
			record, err := newEffectAttachment(current)
			if err != nil {
				t.Fatal(err)
			}
			value, err := savedEffectRecord(&record)
			if err != nil {
				t.Fatal(err)
			}
			back, err := originalActorEffect(value, current.Target)
			want := current
			want.Caster, want.HasCaster = 0, false
			if err != nil || back != want || !current.HasCaster {
				t.Fatal("current fields or absent serialized caster changed", back, err)
			}
		}
	}
}
