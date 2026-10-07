package sim

// actorCastBusy reports whether a book cast owns this actor right now: its
// wind-up record still stands, or its decoded recovery interval has not run
// out. It is the narrow half of the story's action guard.
//
// TWO PREDICATES AND NOT ONE. actorActionBusy (spell.go) asks "may a spell
// begin", and answers no for a loaded attack cycle as well; this one asks "does
// a spell already own the actor", and is what the AI's own retarget consults.
// The player's attack order does not (attachAttack). Collapsing the two is what
// made a fighting unit refuse a retarget click. A loaded attack blocks ordinary
// spell admission; a valid manual book command stores pending operands until
// physical progress ends (AI-356).
//
// Movement consults neither. A movement order from the player, from a script or
// from the AI is always taken; the mover pauses only for the wind-up itself and
// may walk through retained retry or recovery. A destination attached during
// wind-up survives the release and then advances under DIV-028.
func (w *World) actorCastBusy(i int) bool {
	if p := w.entities[i].PendingOrder.Kind; p == PendingActorCast || p == PendingCellCast {
		return true
	}
	if _, ok := w.scrollIndex(w.entities[i].ID); ok {
		return true
	}
	if w.entities[i].CastWait != 0 {
		return true
	}
	k, pending := w.bookCastIndex(w.entities[i].ID)
	return pending && w.bookCasts[k].Phase != bookApproach
}

// scrollCastPending reports whether a scroll cast owns this actor, from its
// reservation to its release. It is the half of actorCastBusy that a player's
// attack order still yields to.
func (w *World) scrollCastPending(i int) bool {
	_, ok := w.scrollIndex(w.entities[i].ID)
	return ok
}

// clearActorCast ends both canonical representations of a book action: an
// admitted cast record and the recovery interval left after a one-shot or
// moving retained release. Death and headless relocation share this seam so
// neither can leave a delayed application or recovery residue on the actor.
func (w *World) clearActorCast(i int) {
	w.cancelScroll(i)
	if cast, ok := w.bookCastIndex(w.entities[i].ID); ok {
		w.bookCasts = append(w.bookCasts[:cast], w.bookCasts[cast+1:]...)
	}
	w.entities[i].CastWait = 0
}

// unionOfClaims is the set the autocast sweep skips: the casts that released
// this tick together with the actors an explicit order claimed. Neither map is
// copied when the other is empty, which is the ordinary tick.
func unionOfClaims(released, commanded map[EntityID]bool) map[EntityID]bool {
	if len(commanded) == 0 {
		return released
	}
	if len(released) == 0 {
		return commanded
	}
	both := make(map[EntityID]bool, len(released)+len(commanded))
	for id := range released {
		both[id] = true
	}
	for id := range commanded {
		both[id] = true
	}
	return both
}
