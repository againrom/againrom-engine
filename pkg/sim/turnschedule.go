package sim

func (w *World) beginTurnSubTick() {
	clear(w.turnSteps)
	w.turnStepScope = true
}

func (w *World) endTurnSubTick() {
	clear(w.turnSteps)
	w.turnStepScope = false
}

func (w *World) turnAlreadyStepped(id EntityID) bool {
	_, stepped := w.turnSteps[id]
	return w.turnStepScope && stepped
}

func (w *World) markTurnStepped(id EntityID) {
	if !w.turnStepScope {
		return
	}
	if w.turnSteps == nil {
		w.turnSteps = make(map[EntityID]struct{})
	}
	w.turnSteps[id] = struct{}{}
}
