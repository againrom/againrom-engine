package sim

import (
	"fmt"
	"slices"
)

// ImportOriginalAttachedEffects restores already-applied actor attachments.
// MAGIC-CONSUME-144: LOAD restores the remaining counter and modified actor;
// it neither attaches again nor derives. Existing native serialization and
// the normal timer own all subsequent state. The caster was not serialized.
func (w *World) ImportOriginalAttachedEffects(effects []ActiveEffect) error {
	if len(effects) > len(w.entities)*29 || len(w.attached) != 0 {
		return fmt.Errorf("sim: invalid original attachment population")
	}
	next := make([]attachedEffect, len(effects))
	for i, effect := range effects {
		ti := indexOfEntity(w.entities, effect.Target)
		if ti < 0 || w.entities[ti].SourceBinding.Class == 0 || effect.HasCaster || effect.Caster != 0 || effect.Spell > 28 || effect.Kind < EffectHealth || effect.Kind > EffectManaRegeneration || effect.Mode == 0 || effect.Mode > 7 || effect.Magnitude != int32(int16(effect.Magnitude)) {
			return fmt.Errorf("sim: invalid original attachment %d", i)
		}
		if err := w.entities[ti].SourceBinding.Validate(w.entities[ti]); err != nil {
			return err
		}
		if effect.Spell == 0 && (effect.Mode > 2 || effect.Kind != EffectHealthRegeneration && effect.Kind != EffectManaRegeneration && effect.Kind != EffectAbsorption) {
			return fmt.Errorf("sim: unsupported original potion attachment %d", i)
		}
		next[i] = attachedEffect(effect)
	}
	slices.SortFunc(next, func(a, b attachedEffect) int {
		if a.Target < b.Target {
			return -1
		}
		if a.Target > b.Target {
			return 1
		}
		return int(a.Spell) - int(b.Spell)
	})
	for i := 1; i < len(next); i++ {
		if next[i].Target == next[i-1].Target && next[i].Spell == next[i-1].Spell {
			return fmt.Errorf("sim: duplicate original attachment target/id")
		}
	}
	w.attached = next
	return nil
}
