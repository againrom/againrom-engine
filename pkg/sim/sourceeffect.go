package sim

// Source attachments retain nominal operands. They are not delta buckets
// adjusted to make a native clamped arithmetic inverse. Each attach/removal
// reaches its own state0 dispatcher and derive, in an isolated candidate.
func (w *World) attachSourceEffect(ti int, caster EntityID, rule SpellRule, kind EffectKind, magnitude int32, duration uint16, mode EffectMode) bool {
	if rule.ID == 8 && mode&EffectContinuous != 0 {
		return w.attachEffect(w.entities[ti].ID, caster, rule, kind, magnitude, duration, mode)
	}
	if !w.sourceMutationReady(ti) {
		return false
	}
	n := w.sourceMutationCopy(ti)
	id := n.entities[ti].ID
	remove := func(index int) bool {
		e := n.attached[index]
		if e.Mode&EffectContinuous == 0 {
			if _, ok := n.sourceEffect(ti, e.Kind, -e.Magnitude); !ok {
				return false
			}
		}
		n.attached = append(n.attached[:index], n.attached[index+1:]...)
		return true
	}
	if rule.ID == 23 || rule.ID == 27 {
		opposite := uint16(23)
		if rule.ID == 23 {
			opposite = 27
		}
		if index, ok := effectIndex(n.attached, id, opposite); ok {
			if !remove(index) {
				return false
			}
			*w = n
			w.clearFelled(ti)
			return true
		}
	}
	if index, ok := effectIndex(n.attached, id, rule.ID); ok {
		if mode&EffectContinuous != 0 {
			n.attached[index].Remaining = duration
			*w = n
			return true
		}
		if !remove(index) {
			return false
		}
	}
	if _, ok := n.sourceEffect(ti, kind, magnitude); !ok {
		return false
	}
	index, _ := effectIndex(n.attached, id, rule.ID)
	n.attached = append(n.attached, attachedEffect{})
	copy(n.attached[index+1:], n.attached[index:])
	n.attached[index] = attachedEffect{Target: id, Caster: caster, HasCaster: indexOfEntity(n.entities, caster) >= 0, Spell: rule.ID, Kind: kind, Mode: mode, Magnitude: magnitude, Remaining: duration}
	*w = n
	w.clearFelled(ti)
	w.markSpellEffect(ti, rule.ID)
	w.entities[ti].SpellFX = duration
	return true
}
