package sim

import (
	"slices"
)

// The area-effect tick runs after every actor tick (MOVE-084), and a layer's
// cost byte is written by the recompute inside it. A tick of this engine runs
// its effect step first, so on a world with a saved cell plane the byte a
// layer event would write is held until the actors have moved: during the
// tick the movers read each cell as the previous tick left it, and the held
// bytes are written once the actors are done. The block planes are not held.
//
// The window exists only inside one step and is never part of the world's
// form.
type savedCostWindow struct {
	// layered is the step-duration reader's layered-cell test as the tick
	// found it, for every cell that holds a record.
	layered map[uint16]bool
	// due is the byte the latest layer event wants at each cell.
	due map[uint16]uint8
	// hold counts the layer events in progress; only their cost writes are held.
	hold int
}

// openCostWindow starts the window for a step on a world with a saved cell plane.
func (w *World) openCostWindow() {
	w.costWindow = nil
	if w.savedCellPlanes == nil || w.savedMotion == nil {
		return
	}
	win := &savedCostWindow{layered: make(map[uint16]bool), due: make(map[uint16]uint8)}
	for _, c := range w.savedMotion.Cells {
		if w.layeredPlaneCellNow(c.Cell) {
			win.layered[c.Cell] = true
		}
	}
	w.costWindow = win
}

// holdLayerCosts marks the start of a layer event; its cost writes are held
// until the window settles. The returned function ends the event.
func (w *World) holdLayerCosts() func() {
	win := w.costWindow
	if win == nil {
		return func() {}
	}
	win.hold++
	return func() { win.hold-- }
}

// writeSavedCost is the one write of a saved plane's cost byte by a recompute.
func (w *World) writeSavedCost(key uint16, v uint8) {
	if win := w.costWindow; win != nil && win.hold > 0 {
		win.due[key] = v
		return
	}
	p := w.savedCellPlanes
	p.Cost[key], p.CostKnown[key] = v, 1
}

// settleCostWindow writes the held bytes, the area-effect tick's recompute
// after the actors, and closes the window.
func (w *World) settleCostWindow() {
	win := w.costWindow
	w.costWindow = nil
	if win == nil || w.savedCellPlanes == nil {
		return
	}
	keys := make([]uint16, 0, len(win.due))
	for key := range win.due {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		w.savedCellPlanes.Cost[key], w.savedCellPlanes.CostKnown[key] = win.due[key], 1
	}
}
