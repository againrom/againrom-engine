package sim

// damageObservation exists only during a reported step. Mutation candidates
// own independent copies, so a refused transaction publishes no messages.
type damageObservation struct {
	events []DamageEvent
}

func (o *damageObservation) loss(before, after Entity) {
	if o == nil || before.MaxHP <= 0 || !before.OrdinaryTargetable() || after.HP >= before.HP {
		return
	}
	o.events = append(o.events, DamageEvent{Target: before.ID, BeforeHP: before.HP, AfterHP: after.HP})
}

func (w *World) reportHealthLoss(before Entity, index int) {
	w.damageObservation.loss(before, w.entities[index])
}

func (w *World) reportHealthMessage(before Entity, index int) {
	if w.damageObservation == nil || before.MaxHP <= 0 {
		return
	}
	w.damageObservation.events = append(w.damageObservation.events, DamageEvent{Target: before.ID, BeforeHP: before.HP, AfterHP: w.entities[index].HP})
}

func (w *World) reportPhysicalBlow(before Entity, index int) {
	after := w.entities[index]
	if w.damageObservation == nil || before.MaxHP <= 0 || !before.OrdinaryTargetable() || before.Dead() && after.HP <= -10 {
		return
	}
	w.damageObservation.events = append(w.damageObservation.events, DamageEvent{Target: before.ID, BeforeHP: before.HP, AfterHP: after.HP})
}
