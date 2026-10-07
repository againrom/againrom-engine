package sim

// DeclareStructures sets the world's whole structure list to fresh, on
// newWorld's own ascending-id order, and rebuilds the structure slots. It
// cannot refuse: there is nothing to compare fresh against.
func (w *World) DeclareStructures(fresh []Structure) {
	w.structures = append([]Structure(nil), fresh...)
	w.rebuildStructureSlots()
}

// RestoreStructureBlocking copies each started-mission structure's blocking
// mask onto a restored world that carries no saved structure registry, when
// both lists name the same structures at the same places. It reports whether
// it copied.
func (w *World) RestoreStructureBlocking(fresh []Structure) bool {
	if w == nil || w.hasSavedStructures || len(w.structures) != len(fresh) {
		return false
	}
	for i := range fresh {
		a, b := w.structures[i], fresh[i]
		if a.ID != b.ID || a.Kind != b.Kind || a.Col != b.Col || a.Row != b.Row ||
			a.Width != b.Width || a.Height != b.Height || a.Attach != b.Attach {
			return false
		}
	}
	for i := range fresh {
		w.structures[i].Blocking = fresh[i].Blocking
	}
	return true
}
