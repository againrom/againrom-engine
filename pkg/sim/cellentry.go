package sim

import (
	"fmt"
	"sort"

	"againrom/pkg/scriptprofile"
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
	b, ok := w.cellTailOf(cx, cy)
	if !ok {
		return
	}
	if cellEventBodies[w.game.cellEvent].entry(w, e, cx, cy, b) {
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

// cellTailOf is the record of cell (cx, cy), when the cell holds one.
func (w *World) cellTailOf(cx, cy int32) ([cellTailLen]byte, bool) {
	if cx < 0 || cx > 255 || cy < 0 || cy > 255 {
		return [cellTailLen]byte{}, false
	}
	key := tailKey(cx, cy)
	k := sort.Search(len(w.cellTails), func(k int) bool { return w.cellTails[k].Key >= key })
	if k == len(w.cellTails) || w.cellTails[k].Key != key {
		return [cellTailLen]byte{}, false
	}
	return w.cellTails[k].Bytes, true
}

// stepCentred is the end of entity i's step, centred on its cell: a cell
// event that waits for the centre acts there (R2-ENGINE-1778).
func (w *World) stepCentred(i int) { cellEventBodies[w.game.cellEvent].centred(w, i) }

// cellMoveEvent is the cell event that moves the unit on it to the event's
// target cell: the game's script vocabulary names it, 23 in the second game
// (R2-ENGINE-1778) and 26 in the first (MOVE-129); the editor names it a
// teleport.
func (w *World) cellMoveEvent() uint8 { return w.game.cellMoveEvent() }

func (g gameProfile) cellMoveEvent() uint8 {
	return uint8(scriptprofile.VocabularyOf(g.scriptLimits().Ops).CellMove)
}

// relocatesOnCellEvent says whether a pending record is that event. A unit
// cast an instant authors never holds a target cell, so the two cannot meet.
func (w *World) relocatesOnCellEvent(c scriptCast) bool {
	return cellEventBodies[w.game.cellEvent].relocates(w, c)
}

// cellEventRule selects whether the cell move event moves the entering unit.
type cellEventRule uint8

const (
	cellEventIgnored cellEventRule = iota
	cellEventRelocating
)

// cellEventBody holds what a cellEventRule does with the event: take it on
// entry, queue it at a step's centre, recognise its pending record, and
// repair an older save's tail.
type cellEventBody struct {
	entry     func(w *World, e Entity, cx, cy int32, b [cellTailLen]byte) bool
	centred   func(w *World, i int)
	relocates func(w *World, c scriptCast) bool
	repair    func(w *World)
}

// cellEventBodies holds the body each cellEventRule runs.
var cellEventBodies = [...]cellEventBody{
	cellEventIgnored: {
		entry:     func(*World, Entity, int32, int32, [cellTailLen]byte) bool { return false },
		centred:   func(*World, int) {},
		relocates: func(*World, scriptCast) bool { return false },
		repair:    func(*World) {},
	},
	// Entering the cell casts nothing; the step that ends centred on the
	// unit's own cell passes the unit to the move with the target, tail
	// bytes 4 and 5 (R2-ENGINE-1778, MOVE-129). Air units stay outside both,
	// as on entry.
	cellEventRelocating: {
		entry: func(w *World, e Entity, cx, cy int32, b [cellTailLen]byte) bool {
			return b[0] == w.cellMoveEvent()
		},
		centred: func(w *World, i int) {
			e := w.entities[i]
			if e.Domain != DomainGround && e.Domain != DomainGhost {
				return
			}
			b, ok := w.cellTailOf(e.X, e.Y)
			if !ok || b[0] != w.cellMoveEvent() {
				return
			}
			// Where the profile's cellMoveSkipsOrigin holds, a target (0,0)
			// queues nothing; elsewhere the record is queued and relocates
			// refuses it.
			if w.game.cellMoveSkipsOrigin && b[4]|b[5] == 0 {
				return
			}
			w.casts = append(w.casts, scriptCast{FromX: uint8(e.X), FromY: uint8(e.Y), ToX: b[4], ToY: b[5],
				Spell: b[0], Power: uint16(b[1]), Target: e.ID, AtUnit: true})
		},
		relocates: func(w *World, c scriptCast) bool {
			return c.AtUnit && c.Spell == w.cellMoveEvent() && c.ToX|c.ToY != 0
		},
		repair: (*World).repairRelocatingCellEvents,
	},
}

// relocateOnCellEvent moves the unit whose step ended on the event's cell,
// held in the record's source. A unit off that cell when the record runs
// stays; a refused target leaves the unit in place (R2-ENGINE-705, MOVE-129).
// Where the profile's cellMoveStands holds, a moved unit then stands at its
// new cell with state 0xc and no order (MOVE-129, AI-535, DIV-5176).
func (w *World) relocateOnCellEvent(c scriptCast, obs *castObs) {
	i := indexOfEntity(w.entities, c.Target)
	if i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap {
		return
	}
	e, n := w.entities[i], footprintSide(w.entities[i].TokenSize)
	if cx, cy := int32(c.FromX), int32(c.FromY); cx < e.X || cy < e.Y || cx >= e.X+n || cy >= e.Y+n {
		return
	}
	w.finishCellMove(i, int32(c.ToX), int32(c.ToY), obs)
}

// finishCellMove is the trap teleport's one move: relocate the actor and,
// where the profile's cellMoveStands holds, stand it at its new cell.
func (w *World) finishCellMove(i int, x, y int32, obs *castObs) bool {
	if !w.relocateActor(i, x, y, obs) {
		return false
	}
	if w.game.cellMoveStands {
		w.standScriptedUnit(i, actorStateAcquire)
	}
	return true
}

// teleportCellOccupants moves every living on-map unit covering (srcX, srcY)
// to (dstX, dstY), or the nearest free cell when the target is taken, through
// finishCellMove (DIV-5200).
func (w *World) teleportCellOccupants(srcX, srcY, dstX, dstY int32, obs *castObs) {
	if _, ok := w.cellIndex(srcX, srcY); !ok {
		return
	}
	if _, ok := w.cellIndex(dstX, dstY); !ok {
		return
	}
	ground, air := w.cellLayerOccupants(srcX, srcY)
	for _, layer := range [][]int{ground, air} {
		for _, i := range layer {
			e := w.entities[i]
			if !e.Alive() || e.OffMap {
				continue
			}
			tx, ty := dstX, dstY
			if !w.placeFree(i, tx, ty) {
				var ok bool
				if tx, ty, ok = nearestFreeCell(dstX, dstY)(w, i); !ok {
					continue
				}
			}
			if tx == e.X && ty == e.Y {
				continue
			}
			w.finishCellMove(i, tx, ty, obs)
		}
	}
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
	e.dropWalk()
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
