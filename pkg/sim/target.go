package sim

const controlSpiritSpellID = 25

// spellFinishesBody is the ordinary damage population allowed to name a body
// in the 0 through -9 finishing band. Heal has its own restorative admission
// below; buffs and every other support row remain forbidden on a body.
func spellFinishesBody(rule SpellRule) bool {
	return rule.Damaging || rule.ID == 11 ||
		rule.EffectKind == EffectHealth && rule.EffectMagnitude < 0
}

// spellTargetable is the one target-state gate shared by spell admission,
// selection and application. Control Spirit's more specific bones-only gate is
// pointEffectRefusal; this predicate only preserves its explicit corpse route.
func spellTargetable(target Entity, rule SpellRule) bool {
	if rule.ID == controlSpiritSpellID {
		return true
	}
	if !target.OrdinaryTargetable() {
		return false
	}
	if rule.Restorative {
		return target.restorativeTargetable()
	}
	return target.Alive() || spellFinishesBody(rule)
}

// spellIDTargetable applies spellTargetable to a stored pending id. An unknown
// row is retained while its target is alive for backward compatibility, but it
// cannot claim a body as a support-or-damage guess. The explicit Control Spirit
// id remains retained even when its table is supplied later by a caller.
func (w *World) spellIDTargetable(target Entity, spell uint32) bool {
	if spell == controlSpiritSpellID {
		return true
	}
	rule, ok := w.findSpell(spell)
	if !ok {
		return target.Alive()
	}
	return spellTargetable(target, rule)
}

// clearInvalidTargetReferences removes every action that can no longer name
// target. Attack, pursuit, finishing damage and restorative Heal survive
// through -9; support does not. At -10 every ordinary reference is cleared.
// Control Spirit retains its separate corpse route.
func (w *World) clearInvalidTargetReferences(target EntityID) {
	ti := indexOfEntity(w.entities, target)
	if ti < 0 {
		return
	}
	targetState := w.entities[ti]
	for i := range w.entities {
		if !targetState.OrdinaryTargetable() && w.entities[i].HasPendingAttackTarget && w.entities[i].PendingAttackTargetKind == AttackTargetUnit && w.entities[i].PendingAttackTarget == target {
			w.entities[i].clearPendingAttack()
		}
		if !w.entities[i].HasAttackTarget || w.entities[i].AttackTargetKind != AttackTargetUnit || w.entities[i].AttackTarget != target {
			continue
		}
		if targetState.OrdinaryTargetable() {
			continue
		}
		w.entities[i].clearActiveAttack()
		w.clearOrder(i)
	}

	books := w.bookCasts[:0]
	for _, cast := range w.bookCasts {
		if !cast.AtCell && cast.Target == target && !w.spellIDTargetable(targetState, uint32(cast.Spell)) {
			continue
		}
		books = append(books, cast)
	}
	w.bookCasts = books

	scripts := w.casts[:0]
	for _, cast := range w.casts {
		if cast.AtUnit && cast.Target == target && !w.spellIDTargetable(targetState, uint32(cast.Spell)) {
			continue
		}
		scripts = append(scripts, cast)
	}
	w.casts = scripts
}

// normaliseTargetReferences accepts old forms whose references were legal when
// written, then removes every action that cannot run under the current target
// state. It deliberately preserves attack finishing and Control Spirit.
func (w *World) normaliseTargetReferences() {
	for i := range w.entities {
		if !w.entities[i].Alive() {
			w.clearInvalidTargetReferences(w.entities[i].ID)
		}
	}
}
