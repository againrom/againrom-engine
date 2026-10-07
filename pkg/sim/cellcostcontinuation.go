package sim

import (
	"fmt"
	"slices"
)

type CurrentCellCost struct {
	Cell uint16
	Cost uint8
}

type CurrentCellPlaneResidue struct {
	Cell                  uint16
	Cost, Static, Dynamic *uint8
}

// Cell records keep their baseline cost for later detach. The current cost
// includes current Buildings and area layers and must not replace that baseline.
func (w *World) RestoreCurrentCellCosts(costs []CurrentCellCost, residue ...CurrentCellPlaneResidue) error {
	for i, c := range costs {
		if w.savedCellPlanes == nil || w.motionCell(c.Cell) == nil || i > 0 && costs[i-1].Cell >= c.Cell {
			return fmt.Errorf("sim: invalid current cell cost binding %04x (planes %t, motion %t, ordered %t)", c.Cell, w.savedCellPlanes != nil, w.motionCell(c.Cell) != nil, i == 0 || costs[i-1].Cell < c.Cell)
		}
	}
	for i, c := range residue {
		if w.savedCellPlanes == nil || i > 0 && residue[i-1].Cell >= c.Cell || c.Cost == nil && c.Static == nil && c.Dynamic == nil || c.Cost != nil && w.motionCell(c.Cell) != nil {
			return fmt.Errorf("sim: invalid omitted cell plane binding %04x", c.Cell)
		}
		dynamic := w.savedCellPlanes.Dynamic[c.Cell]
		if c.Dynamic != nil {
			dynamic = *c.Dynamic
		}
		if (c.Static != nil || c.Dynamic != nil) && c.Cell >= 0x807 && c.Cell <= 0xeded && dynamic > 15 {
			return fmt.Errorf("sim: omitted cell plane value has an ordinary Block at %04x", c.Cell)
		}
	}
	if len(costs)+len(residue) == 0 {
		return nil
	}
	next := *w
	planes := *w.savedCellPlanes
	next.savedCellPlanes = &planes
	next.grid = slices.Clone(w.grid)
	next.savedMotion = cloneActorMotions(w.savedMotion)
	for _, c := range costs {
		planes.Cost[c.Cell] = c.Cost
	}
	for _, c := range residue {
		if c.Cost != nil {
			planes.Cost[c.Cell] = *c.Cost
		}
		if c.Static != nil {
			planes.Static[c.Cell] = *c.Static
			next.syncNativeSavedPlaneCell(c.Cell)
		}
		if c.Dynamic != nil {
			planes.Dynamic[c.Cell] = *c.Dynamic
		}
	}
	next.refreshSavedPlaneBlocks()
	if err := next.savedCellPlaneStateFault(); err != nil {
		return err
	}
	*w = next
	return nil
}
