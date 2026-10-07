package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Removed actors keep no running action. A retained corpse owns ordinary
// mover values; only a truly missing actor needs detached frozen values.
type currentTerminalMotion struct {
	Entity     sim.EntityID
	Issue      string
	Slots      []sim.CurrentTerminalMotionSlot
	FrozenFine *currentFrozenFine     `json:",omitempty"`
	Detached   *currentDetachedMotion `json:",omitempty"`
}

type currentFrozenFine struct {
	X, Y uint8
}

type currentDetachedMotion struct {
	Position                  sim.SavedActorPosition
	Mover                     [180]byte
	StaticRoute, DynamicRoute []uint16
	ActorAction               uint32
}

func captureCurrentTerminalMotions(w *sim.World, a *currentActionData) error {
	live := map[sim.EntityID]bool{}
	for _, e := range w.Entities() {
		live[e.ID] = true
	}
	bodies := map[sim.EntityID]sim.DeadActorState{}
	for _, dead := range w.OriginalDeadActors() {
		bodies[dead.ID] = dead.Current
	}
	motions, cells, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if live[m.Entity] {
			continue
		}
		if m.Current || m.Active || m.Issue == "" {
			return fmt.Errorf("removed current motion has a running mode")
		}
		row := currentTerminalMotion{Entity: m.Entity, Issue: m.Issue}
		found := false
		for _, b := range a.Bindings {
			if b.ID != m.Entity || b.Structure {
				continue
			}
			found = true
			if b.Missing {
				row.Detached = &currentDetachedMotion{m.Position, m.Mover, slices.Clone(m.StaticRoute), slices.Clone(m.DynamicRoute), m.ActorAction}
			} else if body, ok := bodies[m.Entity]; ok && (m.Position.FineX != body.FineX || m.Position.FineY != body.FineY) {
				row.FrozenFine = &currentFrozenFine{X: m.Position.FineX, Y: m.Position.FineY}
			}
		}
		if !found {
			return fmt.Errorf("removed motion lacks its current identity binding")
		}
		sites := map[[2]uint16]sim.CurrentTerminalMotionSlot{}
		for _, c := range cells {
			for layer, v := range []sim.SavedActorSlot{c.Ground, c.Air} {
				if v.Bound && v.Entity == m.Entity {
					key := [2]uint16{c.Cell, uint16(layer)}
					sites[key] = sim.CurrentTerminalMotionSlot{Cell: c.Cell, Key: v.Key, Air: layer == 1, Motion: true}
				}
			}
		}
		for _, c := range w.SavedCellRecords() {
			for layer, v := range []sim.SavedCellActorSlot{c.Ground, c.Air} {
				if v.Bound && v.Entity == m.Entity {
					key := [2]uint16{c.Cell, uint16(layer)}
					slot := sites[key]
					slot.Cell, slot.Key, slot.Air, slot.Record = c.Cell, v.Key, layer == 1, true
					sites[key] = slot
				}
			}
		}
		for _, slot := range sites {
			row.Slots = append(row.Slots, slot)
		}
		slices.SortFunc(row.Slots, func(x, y sim.CurrentTerminalMotionSlot) int {
			if x.Cell != y.Cell {
				return int(x.Cell) - int(y.Cell)
			}
			if x.Air == y.Air {
				return 0
			}
			if x.Air {
				return 1
			}
			return -1
		})
		a.TerminalMotions = append(a.TerminalMotions, row)
	}
	return nil
}

func readCurrentTerminalMotions(ms *Mission, a *currentActionData, ids map[sim.EntityID]sim.EntityID) ([]sim.CurrentTerminalMotion, error) {
	if len(a.TerminalMotions) > 32767 {
		return nil, fmt.Errorf("current terminal motion population exceeds bounds")
	}
	doc := ms.savedDocument.Document
	live := map[sim.EntityID]bool{}
	for _, e := range ms.World.Entities() {
		live[e.ID] = true
	}
	cells := map[uint16]sav.DocumentCellData{}
	for _, cell := range doc.World.Cells {
		cells[cell.Cell] = cell
	}
	seen := map[sim.EntityID]bool{}
	var out []sim.CurrentTerminalMotion
	for _, row := range a.TerminalMotions {
		id, present := ids[row.Entity]
		if !present || seen[row.Entity] || row.Issue == "" || len(row.Issue) > 256 || len(row.Slots) > 131072 {
			return nil, fmt.Errorf("current terminal motion identity or issue is invalid")
		}
		seen[row.Entity] = true
		var object uint16
		missing := false
		for _, b := range a.Bindings {
			if b.ID == row.Entity && !b.Structure {
				object, missing = b.Object, b.Missing
			}
		}
		if (row.Detached != nil) != missing || missing && (object != 0 || row.FrozenFine != nil) || !missing && (object == 0 || int(object) > len(doc.Objects)) {
			return nil, fmt.Errorf("current terminal motion has conflicting ordinary ownership")
		}
		motion := sim.SavedActorMotion{Entity: id, Issue: row.Issue}
		key, stage := uint32(0), uint32(5)
		if d := row.Detached; d != nil {
			motion.Position, motion.Mover, motion.StaticRoute, motion.DynamicRoute, motion.ActorAction = d.Position, d.Mover, slices.Clone(d.StaticRoute), slices.Clone(d.DynamicRoute), d.ActorAction
		} else {
			r := &doc.Objects[object-1]
			var err error
			key, err = savedStructureValue(r, "Identity")
			if err != nil {
				return nil, err
			}
			stage, err = savedStructureValue(r, "Stage")
			if err != nil {
				return nil, err
			}
			motion, err = readSavedActorMotion(r, id)
			if err != nil {
				return nil, err
			}
			if row.FrozenFine != nil {
				motion.Position.FineX, motion.Position.FineY = row.FrozenFine.X, row.FrozenFine.Y
			}
			motion.Issue = row.Issue
		}
		seenSlots := map[[2]uint16]bool{}
		var slots []sim.CurrentTerminalMotionSlot
		for _, slot := range row.Slots {
			layer := uint16(0)
			if slot.Air {
				layer = 1
			}
			site := [2]uint16{slot.Cell, layer}
			if seenSlots[site] || !slot.Motion && !slot.Record || slot.Key == 0 {
				return nil, fmt.Errorf("current terminal cell site is repeated or empty")
			}
			seenSlots[site] = true
			cell, present := cells[slot.Cell]
			value := cell.GroundActor
			if slot.Air {
				value = cell.AirActor
			}
			if !missing {
				slot.Key = key
			}
			if present && value == slot.Key && slot.Key != 0 {
				slots = append(slots, slot)
			}
		}
		// An ordinary stage edit can restore a live body. Its admitted current
		// mode supersedes the prior removed-body presence marker.
		if !missing && (stage != 5 || live[id]) {
			continue
		}
		out = append(out, sim.CurrentTerminalMotion{Motion: motion, Key: key, Slots: slots, Detached: missing})
	}
	return out, nil
}
