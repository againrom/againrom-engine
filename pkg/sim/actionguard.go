package sim

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
