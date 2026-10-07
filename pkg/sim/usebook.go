package sim

// readBook applies one deterministic use of a carried one-spell book. The
// container index is resolved at Step time, like equip and drop. Every refusal
// is before mutation: the reader must be alive and a mage, the element must
// exist, and its complete instance must be a readable BookSpell.
func (w *World) readBook(i, cindex int) {
	if !w.sourceMutationReady(i) {
		return
	}
	if !w.entities[i].Alive() || !isMage(w.entities[i]) || cindex < 0 || cindex >= len(w.carried[i]) {
		return
	}
	item := w.carried[i][cindex].Instance()
	spell, ok := item.BookSpell()
	if !ok {
		return
	}
	live := w
	if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	LearnBookSpell(&w.entities[i], spell, w.spells)
	w.refreshSavedBookRoots(i)
	if !w.consumeCarriedUnit(i, cindex) {
		return
	}
	if w != live {
		*live = *w
	}
}
