package sim

import (
	"fmt"
	"slices"
)

// CurrentTerminalMotion restores a removed actor from SAV and may
// retain frozen fine bytes distinct from its body.
type CurrentTerminalMotion struct {
	Motion   SavedActorMotion
	Key      uint32
	Slots    []CurrentTerminalMotionSlot
	Detached bool
}

type CurrentTerminalMotionSlot struct {
	Cell           uint16
	Key            uint32
	Air            bool
	Motion, Record bool
}

func (w *World) restoreCurrentTerminalMotions(rows []CurrentTerminalMotion) error {
	if len(rows) == 0 {
		return nil
	}
	if w.savedMotion == nil {
		return fmt.Errorf("sim: terminal motion has no carrier")
	}
	w.savedMotion = cloneActorMotions(w.savedMotion)
	w.savedCellRecords = slices.Clone(w.savedCellRecords)
	seen := map[EntityID]bool{}
	for _, row := range rows {
		m := row.Motion
		if seen[m.Entity] || indexOfEntity(w.entities, m.Entity) >= 0 || w.motionFor(m.Entity) != nil || m.Current || m.Active || m.Issue == "" || (row.Key == 0) != row.Detached {
			return fmt.Errorf("sim: invalid terminal motion identity or mode")
		}
		found, retained := false, false
		for _, body := range w.originalDead {
			if body.ID == m.Entity {
				retained = true
				found = body.Source.Identity == row.Key && body.terminal.Stage != 0
			}
		}
		if row.Detached && retained || !row.Detached && !found {
			return fmt.Errorf("sim: terminal motion has no exact removed actor")
		}
		seen[m.Entity] = true
		m.StaticRoute, m.DynamicRoute = slices.Clone(m.StaticRoute), slices.Clone(m.DynamicRoute)
		w.savedMotion.Motions = append(w.savedMotion.Motions, m)
		slices.SortFunc(w.savedMotion.Motions, func(a, b SavedActorMotion) int {
			if a.Entity < b.Entity {
				return -1
			}
			if a.Entity > b.Entity {
				return 1
			}
			return 0
		})
		slots := map[[2]uint16]bool{}
		for _, slot := range row.Slots {
			layer := uint16(0)
			if slot.Air {
				layer = 1
			}
			site := [2]uint16{slot.Cell, layer}
			if slots[site] || !slot.Motion && !slot.Record || slot.Key == 0 || !row.Detached && slot.Key != row.Key {
				return fmt.Errorf("sim: invalid terminal motion cell site")
			}
			slots[site] = true
			if slot.Motion {
				cell := w.motionCell(slot.Cell)
				if cell == nil {
					return fmt.Errorf("sim: terminal motion cell is absent")
				}
				target := motionSlot(cell, int(layer))
				if target.Key != slot.Key || target.Bound && target.Entity != m.Entity {
					return fmt.Errorf("sim: terminal motion cell key differs")
				}
				*target = SavedActorSlot{Key: slot.Key, Entity: m.Entity, Bound: true}
			}
			if slot.Record {
				found := false
				for i := range w.savedCellRecords {
					cell := &w.savedCellRecords[i]
					if cell.Cell != slot.Cell {
						continue
					}
					target := &cell.Ground
					if slot.Air {
						target = &cell.Air
					}
					if target.Key != slot.Key || target.Bound && target.Entity != m.Entity {
						return fmt.Errorf("sim: terminal record cell key differs")
					}
					*target = SavedCellActorSlot{Key: slot.Key, Entity: m.Entity, Bound: true}
					found = true
				}
				if !found {
					return fmt.Errorf("sim: terminal record cell is absent")
				}
			}
		}
	}
	return w.savedMotionFault()
}
