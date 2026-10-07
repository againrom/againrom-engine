package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func originalAttachmentWorld(t *testing.T) *World {
	t.Helper()
	w := sourceMutationWorld(t, PlainItem(0xe01))
	w.entities[0].Humanoid = true
	w.entities[0].SourceBinding = SourceBinding{Class: 2, ArchiveIndex: 1}
	return w
}

func TestOriginalEffects1161ImportNoReplayAndExpiry(t *testing.T) {
	w := originalAttachmentWorld(t)
	// The saved actor already owns its modifier. Import must not invoke the
	// state0/derive callback, which is installed as a discriminating trap.
	binary.LittleEndian.PutUint16(w.entities[0].ActorLoad.Source.Modifier[10:], 100)
	w.entities[0].HealthRegeneration = 100
	before := w.entities[0]
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		t.Error("LOAD replayed derive")
		return s, fmt.Errorf("LOAD must not derive")
	})
	e := ActiveEffect{Target: 1, Spell: 0, Kind: EffectHealthRegeneration, Mode: EffectDuration, Magnitude: 100, Remaining: 2}
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{e}); err != nil {
		t.Fatal(err)
	}
	if w.entities[0] != before {
		t.Fatal("LOAD changed saved actor")
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
	w.stepAttachedEffects()
	if len(w.attached) != 1 || w.attached[0].Remaining != 1 || w.entities[0].HealthRegeneration != 100 || w.entities[0].SpellFX != 1 {
		t.Fatal("timer/marker did not continue")
	}
	w.stepAttachedEffects()
	if len(w.attached) != 0 || w.entities[0].HealthRegeneration != 0 || binary.LittleEndian.Uint16(w.entities[0].ActorLoad.Source.Modifier[10:]) != 0 {
		t.Fatal("nominal expiry did not reverse saved modifier")
	}
}

func TestOriginalEffects1161ContinuousFrozenAndActionMask(t *testing.T) {
	for _, remaining := range []uint16{9, 9601} {
		w := originalAttachmentWorld(t)
		e := ActiveEffect{Target: 1, Spell: 8, Kind: EffectHealth, Mode: EffectContinuous, Magnitude: -1, Remaining: remaining}
		if err := w.ImportOriginalAttachedEffects([]ActiveEffect{e}); err != nil {
			t.Fatal(err)
		}
		w.stepAttachedEffects()
		if remaining == 9601 {
			if w.attached[0].Remaining != 9601 || w.entities[0].HP != 50 {
				t.Fatal("frozen counter advanced")
			}
			continue
		}
		if w.attached[0].Remaining != 8 || w.entities[0].HP != 50 {
			t.Fatal("old9 counter must not pulse before reaching8")
		}
		for range 8 {
			w.stepAttachedEffects()
		}
		if len(w.attached) != 0 || w.entities[0].HP != 49 {
			t.Fatal("continuous expiry incorrectly unapplied")
		}
	}
	w := originalAttachmentWorld(t)
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{{Target: 1, Spell: 20, Kind: EffectSpeed, Mode: EffectDuration, Remaining: 1}}); err != nil {
		t.Fatal(err)
	}
	if !w.HasEffectSpell(1, 20) || !w.stoneCursed(0) {
		t.Fatal("restored effect did not engage existing action blocker")
	}
	w.stepAttachedEffects()
	if w.HasEffectSpell(1, 20) || w.stoneCursed(0) {
		t.Fatal("expired blocker remains")
	}
}

func TestOriginalEffects1161RejectsLossAndInvalidAdmissionAtomically(t *testing.T) {
	base := ActiveEffect{Target: 1, Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: 7, Remaining: 8}
	for name, edit := range map[string]func(*ActiveEffect){
		"target": func(e *ActiveEffect) { e.Target = 999 }, "caster": func(e *ActiveEffect) { e.HasCaster = true }, "id": func(e *ActiveEffect) { e.Spell = 29 }, "kind": func(e *ActiveEffect) { e.Kind = EffectNone }, "charges": func(e *ActiveEffect) { e.Mode = EffectCharges }, "operand": func(e *ActiveEffect) { e.Magnitude = 65536 }, "potion-kind": func(e *ActiveEffect) { e.Kind = EffectSpeed },
	} {
		t.Run(name, func(t *testing.T) {
			w := originalAttachmentWorld(t)
			e := base
			edit(&e)
			before := w.Hash()
			if err := w.ImportOriginalAttachedEffects([]ActiveEffect{e}); err == nil || w.Hash() != before {
				t.Fatal("invalid import accepted or partially changed World")
			}
		})
	}
	w := originalAttachmentWorld(t)
	before := w.Hash()
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{base, base}); err == nil || w.Hash() != before {
		t.Fatal("duplicate target/id accepted")
	}
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{base}); err != nil {
		t.Fatal(err)
	}
	with := w.Hash()
	w.attached = nil
	if with == w.Hash() {
		t.Fatal("dropping restored timer did not change hashed state")
	}
}

func TestOriginalEffects1161ExpiryPublishesCurrentTurnRate(t *testing.T) {
	w := originalAttachmentWorld(t)
	binary.LittleEndian.PutUint16(w.entities[0].ActorLoad.Source.Modifier[4:], 7)
	w.entities[0].ActorLoad.Source.MoverSpeed = 25
	w.entities[0].RotationSpeed = 25
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		s.MoverSpeed = 18 + uint8(binary.LittleEndian.Uint16(s.Modifier[4:]))
		return s, nil
	})
	if err := w.ImportOriginalAttachedEffects([]ActiveEffect{{Target: 1, Spell: 13, Kind: EffectSpeed, Mode: EffectDuration, Magnitude: 7, Remaining: 1}}); err != nil {
		t.Fatal(err)
	}
	if w.entities[0].RotationSpeed != 25 {
		t.Fatal("LOAD derived instead of restoring current turn rate")
	}
	w.stepAttachedEffects()
	if w.entities[0].ActorLoad.Source.MoverSpeed != 18 || w.entities[0].RotationSpeed != 18 {
		t.Fatal("effect expiry left live turn rate stale", w.entities[0].ActorLoad.Source.MoverSpeed, w.entities[0].RotationSpeed)
	}
}
