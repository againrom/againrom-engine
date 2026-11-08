package sim

import (
	"fmt"
	"sort"
)

// DeclareCellTails binds installed cell payloads without enacting an entry.
// UNIT-M10ENTRY-055: the initial writer refuses dynamic-plane bit 0. The
// derived block plane is this build's counterpart. Last write wins per cell.
// Decode uses the persisted records directly, never this initialization gate.
func (w *World) DeclareCellTails(tails []CellTail) error {
	for _, t := range tails {
		if t.X < 0 || t.X > 255 || t.Y < 0 || t.Y > 255 {
			return fmt.Errorf("sim: cell-tail position (%d,%d) exceeds byte coordinates", t.X, t.Y)
		}
		if _, ok := w.cellIndex(t.X, t.Y); !ok {
			return fmt.Errorf("sim: cell-tail position (%d,%d) is outside the world", t.X, t.Y)
		}
	}
	for _, t := range tails {
		i, _ := w.cellIndex(t.X, t.Y)
		if w.grid[i]&blockGround == 0 {
			w.writeCellTail(tailKey(t.X, t.Y), t.Bytes)
		}
	}
	return nil
}

// requestFootprintCasts is an attachment attempt, never a route-search probe.
// A stride runs it on its crossing tick, a refused step on the attempt; it
// visits every destination footprint cell, including overlap with the actor's
// prior footprint (MOVE-196). Standing does not
// call it. UNIT-M10ENTRY-055 admits ground and ghost, not air; UNIT-M10LIFE-057
// supplies the two disabled spell bytes and no local cooldown or once flag.
// External caller frequency is Unknown; DIV-540 records our integration seam.
func (w *World) requestFootprintCasts(i int, x, y int32) {
	e := w.entities[i]
	n := footprintSide(e.TokenSize)
	for dy := int32(0); dy < n; dy++ {
		for dx := int32(0); dx < n; dx++ {
			w.requestCellCast(i, x+dx, y+dy)
		}
	}
}

// requestCellCast is one cell's share of an entry: a ground or ghost actor
// entering a cell whose record names a trigger spell queues that cast, before
// the cell's slot is tested. An air actor's arm has no such call.
func (w *World) requestCellCast(i int, cx, cy int32) {
	e := w.entities[i]
	if e.Domain != DomainGround && e.Domain != DomainGhost || len(w.cellTails) == 0 {
		return
	}
	if cx < 0 || cx > 255 || cy < 0 || cy > 255 {
		return
	}
	key := tailKey(cx, cy)
	k := sort.Search(len(w.cellTails), func(k int) bool { return w.cellTails[k].Key >= key })
	if k == len(w.cellTails) || w.cellTails[k].Key != key {
		return
	}
	b := w.cellTails[k].Bytes
	if w.game.cellEventRelocates && b[0] == cellEventRelocate {
		w.casts = append(w.casts, scriptCast{FromX: uint8(cx), FromY: uint8(cy), ToX: b[4], ToY: b[5],
			Spell: b[0], Power: uint16(b[1]), Target: e.ID, AtUnit: true})
		return
	}
	if b[0] == 0 || b[0] == 26 {
		return
	}
	if _, ok := w.findSpell(uint32(b[0])); !ok {
		return
	}
	// Unlike instant 21, zero power is literal here (UNIT-M10CAST-056).
	w.casts = append(w.casts, scriptCast{FromX: b[2], FromY: b[3],
		Spell: b[0], Power: uint16(b[1]), Target: e.ID, AtUnit: true})
}

// cellEventRelocate is the second game's cell event that moves the unit on it
// to the event's target cell (R2-ENGINE-673, R2-ENGINE-557).
const cellEventRelocate = 0x17

// relocatesOnCellEvent says whether a pending record is that event. A unit
// cast an instant authors never holds a target cell, so the two cannot meet.
func (w *World) relocatesOnCellEvent(c scriptCast) bool {
	return w.game.cellEventRelocates && c.AtUnit && c.Spell == cellEventRelocate && c.ToX|c.ToY != 0
}

// relocateOnCellEvent moves the unit that entered the event's cell, held in
// the record's source. A unit refused that cell never stood on it and stays
// (DIV-4527 still sends it here); a refused target leaves the unit in place
// (R2-ENGINE-705).
func (w *World) relocateOnCellEvent(c scriptCast, obs *castObs) {
	i := indexOfEntity(w.entities, c.Target)
	if i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap {
		return
	}
	e, n := w.entities[i], footprintSide(w.entities[i].TokenSize)
	if cx, cy := int32(c.FromX), int32(c.FromY); cx < e.X || cy < e.Y || cx >= e.X+n || cy >= e.Y+n {
		return
	}
	w.relocateActor(i, int32(c.ToX), int32(c.ToY), obs)
}

// relocateActor puts the actor at index i on another cell at once, ending its
// walk and its target. It reports false, and changes nothing, when the cell
// refuses the actor.
func (w *World) relocateActor(i int, x, y int32, obs *castObs) bool {
	fromX, fromY := w.entities[i].X, w.entities[i].Y
	if !w.attachFootprint(i, x, y) {
		return false
	}
	e := &w.entities[i]
	e.X, e.Y = x, y
	w.invalidateActorMotion(e.ID, "native teleport supersedes original movement")
	e.clearStride()
	e.HasTarget = false
	w.routes[i] = nil
	obs.recordRelocation(*e, fromX, fromY)
	return true
}

// attachFootprint is the mutating placement counterpart of placementOpen.
// Terrain rejects before attachment; an occupied ground slot rejects after
// the cast request. No caller may use this to ask a hypothetical fit question.
func (w *World) attachFootprint(i int, x, y int32) bool {
	if !w.terrainOpenFootprint(w.entities[i], x, y) {
		return false
	}
	w.requestFootprintCasts(i, x, y)
	return w.placementOpen(i, x, y)
}
