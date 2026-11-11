package sim

// Source attachments retain nominal operands. They are not delta buckets
// adjusted to make a native clamped arithmetic inverse. Each attach/removal
// reaches its own state0 dispatcher and derive, in an isolated candidate.
func (w *World) attachSourceEffect(ti int, caster EntityID, rule SpellRule, kind EffectKind, magnitude int32, duration uint16, mode EffectMode) bool {
	if areaRow(rule.arm()).poison && mode&EffectContinuous != 0 {
		return w.attachEffect(w.entities[ti].ID, caster, rule, kind, magnitude, duration, mode)
	}
	if !w.sourceMutationReady(ti) {
		return false
	}
	n := w.sourceMutationCopy(ti)
	unapply := func(index int) bool {
		e := n.attached[index]
		if e.Mode&EffectContinuous == 0 {
			if _, ok := n.sourceEffectOnCandidate(ti, e.Kind, -e.Magnitude); !ok {
				return false
			}
		}
		return true
	}
	remove := func(index int) bool {
		if !unapply(index) {
			return false
		}
		n.attached = append(n.attached[:index], n.attached[index+1:]...)
		return true
	}
	return n.attachEffectTransaction(ti, caster, rule, kind, magnitude, duration, mode, effectAttachmentAdapter{
		unapply: unapply,
		remove:  remove,
		apply: func(index int) bool {
			// attachEffect's renewal on the source operands (R2-ENGINE-986).
			e := n.attached[index]
			_, ok := n.sourceEffectOnCandidate(ti, e.Kind, e.Magnitude)
			return ok
		},
		applyBeforeInsert: func(kind EffectKind, magnitude int32) bool {
			_, ok := n.sourceEffectOnCandidate(ti, kind, magnitude)
			return ok
		},
		commit: func(clearFelled bool) *World {
			*w = n
			if clearFelled {
				w.clearFelled(ti)
			}
			return w
		},
	})
}
