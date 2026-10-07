package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestSourceEffect1110ExpiryAndContinuousFaultKeepTimerAndActor(t *testing.T) {
	for _, continuous := range []bool{false, true} {
		w := sourceMutationWorld(t, PlainItem(0xe01))
		kind, mode, magnitude, duration := EffectSpeed, EffectDuration, int32(7), uint16(1)
		if continuous {
			kind, mode, magnitude, duration = EffectHealth, EffectContinuous, -1, 8
		}
		if !w.attachEffect(1, 1, SpellRule{ID: 13}, kind, magnitude, duration, mode) {
			t.Fatal("fixture attachment")
		}
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
			if continuous && s.Stats[8] == 48 || !continuous && binary.LittleEndian.Uint16(s.Modifier[4:]) == 0 {
				return s, fmt.Errorf("expiry/repeat source failure")
			}
			return s, nil
		})
		before := w.Hash()
		w.stepAttachedEffects()
		if w.Hash() != before || len(w.attached) != 1 || w.attached[0].Remaining != duration {
			t.Fatal("failed callback spent timer or state")
		}
	}
}

func TestSourceEffect1110UsesModifiersAndRetainsNominalNotNativeClamp(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	w.entities[0].Absorption = 99
	// The bounded identity callback isolates dispatch input from arithmetic.
	if !w.attachEffect(1, 1, SpellRule{ID: 7}, EffectAbsorption, 7, 2, EffectDuration) {
		t.Fatal("attach")
	}
	if binary.LittleEndian.Uint16(w.entities[0].ActorLoad.Source.Modifier[44:]) != 7 || w.entities[0].Absorption != 99 || w.attached[0].Magnitude != 7 {
		t.Fatal("source effect used native live addition")
	}
	w.entities[0].ActorLoad.Source.Modifier[44] = 3
	if !w.removeAttachedAt(0) || binary.LittleEndian.Uint16(w.entities[0].ActorLoad.Source.Modifier[44:]) != 0 {
		t.Fatal("source absorption removal did not floor its modifier")
	}
	for _, k := range []EffectKind{EffectSpeed, EffectScanRange, EffectProtectionFire, EffectProtectionWater, EffectProtectionAir, EffectProtectionEarth, EffectBless, EffectCurse, EffectInvisible} {
		before := w.entities[0].SourceNow()
		calls := 0
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { calls++; return s, nil })
		if _, ok := w.applyEffectDelta(0, k, 3); !ok || calls != 1 {
			t.Fatal("generic effect did not reach source derive", k, calls)
		}
		if k == EffectSpeed && w.entities[0].ActorLoad.Source.Modifier[4] != before.Modifier[4]+3 {
			t.Fatal("source speed modifier missing")
		}
	}
}
