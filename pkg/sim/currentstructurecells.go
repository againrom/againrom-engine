package sim

import (
	"fmt"
	"slices"
)

// A transported actor cell need not have a native structure baseline. Its
// ordinary baseline bytes anchor that absence; an edit establishes real state.
type CurrentAbsentStructureCell struct {
	Cell         uint16
	Cost, Static uint8
}

func (w *World) RestoreAbsentStructureCells(rows []CurrentAbsentStructureCell) error {
	if len(rows) == 0 {
		return nil
	}
	if !w.hasSavedStructures {
		return fmt.Errorf("sim: absent structure cells lack their carrier")
	}
	seen, remove := map[uint16]bool{}, map[uint16]bool{}
	current := make(map[uint16]SavedStructureCell, len(w.savedStructureCells))
	for _, cell := range w.savedStructureCells {
		current[cell.Cell] = cell
	}
	for _, row := range rows {
		c, exists := current[row.Cell]
		if !exists || seen[row.Cell] {
			return fmt.Errorf("sim: absent structure cell binding is missing or repeated")
		}
		seen[row.Cell] = true
		if !c.HasStructure && c.BaselineCost == row.Cost && c.BaselineStatic == row.Static {
			remove[row.Cell] = true
		}
	}
	w.savedStructureCells = slices.DeleteFunc(slices.Clone(w.savedStructureCells), func(c SavedStructureCell) bool { return remove[c.Cell] })
	return nil
}
