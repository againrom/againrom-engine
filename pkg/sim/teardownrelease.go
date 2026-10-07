package sim

// releaseAtTeardown is the footprint release an actor's teardown makes, at once
// and at the actor's position: every cell of the footprint is recomputed, so a
// layered cell under a removed body returns to the byte its layers give and no
// cost contribution of the body stays in the plane (MOVE-089). The cell slots
// themselves are cleared by the terminal registry.
func (w *World) releaseAtTeardown(i int) {
	e := w.entities[i]
	if e.OffMap {
		return
	}
	side := footprintSide(e.TokenSize)
	cells := make([]cell, 0, side*side)
	for dy := int32(0); dy < side; dy++ {
		for dx := int32(0); dx < side; dx++ {
			cells = append(cells, cell{x: e.X + dx, y: e.Y + dy})
		}
	}
	w.recomputeAreaCosts(cells...)
}
