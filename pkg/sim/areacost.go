package sim

import (
	"slices"
	"sort"
)

// Area layers and the mutable movement-cost byte (MAGIC-AREACOST-046,
// MOVE-084, MOVE-085, MOVE-086).
//
// A recompute of a cell shifts its baseline cost left by two once per occupied
// layer slot, in eight bits. The step-duration reader divides a layered cell's
// stored byte by four on every read and stores the quotient back, so a second
// read with no recompute in between returns a quarter of the first.
//
// A world without a saved cell plane keeps that stored byte here, only for the
// cells that hold an applied layer. The entry's mask is the layer set the
// recompute saw. The byte itself is the one value a read changes; a recompute
// puts it back to the mask's fresh value.

// areaCostEntry is one layered cell's stored cost byte and the layer slots it
// was last recomputed for.
type areaCostEntry struct {
	Key  uint16
	Mask uint8
	Byte uint8
}

// freshAreaCost is the byte a recompute leaves: the baseline shifted left by
// two for every layer slot in mask, each shift truncated to eight bits.
func freshAreaCost(base, mask uint8) uint8 {
	for ; mask != 0; mask &= mask - 1 {
		base <<= 2
	}
	return base
}

// areaLayerMask is the set of layer slots the live clouds occupy at key, one
// bit per areaLayerSpells entry.
func (w *World) areaLayerMask(key uint16) uint8 {
	var mask uint8
	for _, e := range w.effects {
		if e.Mode != areaModeCloud || !containsKey(e.Cells, key) {
			continue
		}
		for layer, spell := range areaLayerSpells {
			if spell == e.Spell {
				mask |= 1 << layer
			}
		}
	}
	return mask
}

func (w *World) areaCostAt(key uint16) (int, bool) {
	at := sort.Search(len(w.areaCosts), func(i int) bool { return w.areaCosts[i].Key >= key })
	return at, at < len(w.areaCosts) && w.areaCosts[at].Key == key
}

// areaStored is the model's stored byte and layer mask for a cell of a world
// that keeps no saved cell plane. ok is false for a cell with no applied layer,
// whose byte is its baseline.
//
// A world that has run its first area-effect pass holds every applied layer as
// an entry. A world that has not (a new world, or one just loaded) has no
// recompute behind it yet, so a cell takes the byte a recompute would leave
// unless a loaded entry names it.
func (w *World) areaStored(key uint16, base uint8) (stored, mask uint8, ok bool) {
	if w.savedCellPlanes != nil || len(w.areaCosts) == 0 && (w.areaCostLive || len(w.effects) == 0) {
		return 0, 0, false
	}
	if at, found := w.areaCostAt(key); found {
		e := w.areaCosts[at]
		if w.areaCostLive || e.Mask == w.areaLayerMask(key) {
			return e.Byte, e.Mask, true
		}
	}
	if w.areaCostLive {
		return 0, 0, false
	}
	if mask = w.areaLayerMask(key); mask != 0 {
		return freshAreaCost(base, mask), mask, true
	}
	return 0, 0, false
}

// setAreaCost stores one entry. The slice is replaced, never edited in place,
// because worlds copied by value share it.
func (w *World) setAreaCost(key uint16, mask, b uint8) {
	at, found := w.areaCostAt(key)
	out := slices.Clone(w.areaCosts)
	switch {
	case found && mask == 0:
		out = slices.Delete(out, at, at+1)
	case found:
		out[at] = areaCostEntry{Key: key, Mask: mask, Byte: b}
	case mask != 0:
		out = slices.Insert(out, at, areaCostEntry{Key: key, Mask: mask, Byte: b})
	default:
		return
	}
	w.areaCosts = out
}

// layeredPlaneCell reports whether a saved-plane cell carries the record the
// step-duration reader divides by: static bit 5 set and a non-zero layer count.
// A cloud painted natively on such a world is a layer the count byte does not
// hold, so it counts too.
func (w *World) layeredPlaneCell(key uint16) bool {
	if win := w.costWindow; win != nil {
		return win.layered[key]
	}
	return w.layeredPlaneCellNow(key)
}

// layeredPlaneCellNow is layeredPlaneCell over the cell's present records,
// whatever a step's window holds back.
func (w *World) layeredPlaneCellNow(key uint16) bool {
	p := w.savedCellPlanes
	if p == nil || p.CostKnown[key] == 0 || p.Static[key]&0x20 == 0 {
		return false
	}
	c := w.motionCell(key)
	return c != nil && (c.Payload[2] != 0 || w.areaLayerMask(key) != 0)
}

// searchCostAt is the cost byte a route search prices a step into c with. It is
// the plane byte as it stands, multiplied or decayed, and is never written.
func (w *World) searchCostAt(i int, c cell) uint8 {
	if w.savedCellPlanes != nil {
		if key, ok := savedPlaneKey(c.x, c.y); ok && w.layeredPlaneCell(key) {
			return w.savedCellPlanes.Cost[key]
		}
		return w.cost[i]
	}
	if key, ok := savedPlaneKey(c.x, c.y); ok {
		if b, _, layered := w.areaStored(key, w.cost[i]); layered {
			return b
		}
	}
	return w.cost[i]
}

// readCost is the step-duration reader's view of cell c. A layered cell answers
// its stored byte divided by four. With commit set the quotient is stored back,
// which is what the original reader does on every call; without it the answer is
// what the next call would return.
func (w *World) readCost(c cell, commit bool) uint8 {
	key, ok := savedPlaneKey(c.x, c.y)
	if !ok {
		return w.costAt(c)
	}
	if w.savedCellPlanes != nil {
		if !w.layeredPlaneCell(key) {
			return w.costAt(c)
		}
		q := w.savedCellPlanes.Cost[key] >> 2
		if commit {
			w.savedCellPlanes.Cost[key] = q
		}
		return q
	}
	i, in := w.cellIndex(c.x, c.y)
	if !in {
		return 0
	}
	stored, mask, layered := w.areaStored(key, w.cost[i])
	if !layered {
		return w.cost[i]
	}
	q := stored >> 2
	if commit {
		w.setAreaCost(key, mask, q)
	}
	return q
}

// recomputeAreaCosts is the cell recompute at a crossing: the cell goes back to
// the byte its applied layers give. A saved-plane cell takes the shared cell
// recompute.
func (w *World) recomputeAreaCosts(cells ...cell) {
	touched := false
	for _, c := range cells {
		key, ok := savedPlaneKey(c.x, c.y)
		if !ok {
			continue
		}
		if w.savedCellPlanes != nil {
			if w.layeredPlaneCell(key) {
				w.recomputeSavedCell(key)
				touched = true
			}
			continue
		}
		i, in := w.cellIndex(c.x, c.y)
		if !in {
			continue
		}
		if at, found := w.areaCostAt(key); found {
			e := w.areaCosts[at]
			w.setAreaCost(key, e.Mask, freshAreaCost(w.cost[i], e.Mask))
		}
	}
	if touched {
		w.refreshSavedPlaneBlocks()
	}
}

// syncAreaCosts is the area-effect pass's recompute: after every actor has
// moved, each cell whose layer set differs from the one last applied is
// recomputed, and a cell with no layer left returns to its baseline. A cell
// whose layer set is unchanged keeps its stored byte.
func (w *World) syncAreaCosts() {
	if w.savedCellPlanes != nil {
		return
	}
	if len(w.effects) == 0 && len(w.areaCosts) == 0 {
		w.areaCostLive = true
		return
	}
	keys := make([]uint16, 0, len(w.areaCosts))
	for _, e := range w.areaCosts {
		keys = append(keys, e.Key)
	}
	for _, e := range w.effects {
		if e.Mode == areaModeCloud {
			keys = append(keys, e.Cells...)
		}
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	next := make([]areaCostEntry, 0, len(keys))
	for _, key := range keys {
		x, y := keyCell(key)
		i, in := w.cellIndex(x, y)
		mask := w.areaLayerMask(key)
		if !in || mask == 0 {
			continue
		}
		if at, found := w.areaCostAt(key); found && w.areaCosts[at].Mask == mask {
			next = append(next, w.areaCosts[at])
			continue
		}
		next = append(next, areaCostEntry{Key: key, Mask: mask, Byte: freshAreaCost(w.cost[i], mask)})
	}
	if !slices.Equal(next, w.areaCosts) {
		w.areaCosts = next
	}
	w.areaCostLive = true
}

// strideCell is the cell a stride's mover stands in after steps ticks of
// transit. The position is one sixteen-bit value per axis, the cell byte over
// the fraction byte, and a transit starts at fraction 0x80; each tick adds the
// axis's signed step, so a step down borrows from the cell byte as soon as the
// fraction goes below zero (MOVE-STEP-010).
func strideCell(s NativeStride, steps int32) cell {
	return cell{
		x: (s.FromX*256 + 128 + steps*int32(s.StepX)) >> 8,
		y: (s.FromY*256 + 128 + steps*int32(s.StepY)) >> 8,
	}
}

// RewriteImportedLayerCosts rewrites the cost byte of every layered cell of a
// saved plane from its record. An original-SAV import has no cost plane, so a
// layered cell would keep the byte terrain ingest gave it; the original's own
// LOAD does that and its first read returns a quarter (MOVE-086). The engine
// rewrites the cells at the import instead (DIV-1842). A native form carries the
// plane's cost, so a form's load never calls this.
func (w *World) RewriteImportedLayerCosts() {
	p := w.savedCellPlanes
	if p == nil || w.savedMotion == nil {
		return
	}
	for _, c := range w.savedMotion.Cells {
		if w.layeredPlaneCellNow(c.Cell) {
			p.Cost[c.Cell], p.CostKnown[c.Cell] = w.savedCellCost(c), 1
		}
	}
}

// ResetLoadedAreaCosts rebuilds the area cost entries of a native world after
// its form loads. The form lists only the cells whose byte a read decayed, and
// the first tick after LOAD runs no area pass, so every standing cloud's cell is
// listed here with the byte the form carries for a decayed cell and the byte its
// layers give for any other. A world with a saved cell plane keeps the plane its
// form restored.
func (w *World) ResetLoadedAreaCosts() {
	if w.savedCellPlanes != nil {
		return
	}
	var keys []uint16
	for _, e := range w.effects {
		if e.Mode == areaModeCloud {
			keys = append(keys, e.Cells...)
		}
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	rows := make([]areaCostEntry, 0, len(keys))
	for _, key := range keys {
		x, y := keyCell(key)
		i, in := w.cellIndex(x, y)
		mask := w.areaLayerMask(key)
		if !in || mask == 0 {
			continue
		}
		if at, found := w.areaCostAt(key); found && w.areaCosts[at].Mask == mask {
			rows = append(rows, w.areaCosts[at])
			continue
		}
		rows = append(rows, areaCostEntry{Key: key, Mask: mask, Byte: freshAreaCost(w.cost[i], mask)})
	}
	w.areaCosts, w.areaCostLive = rows, true
}

// recomputeCrossing is the release of the footprint at from and the occupation
// of the footprint at to, each cell recomputed (MOVE-084, MOVE-085).
func (w *World) recomputeCrossing(e Entity, from, to cell) {
	side := footprintSide(e.TokenSize)
	cells := make([]cell, 0, 2*side*side)
	for _, at := range [2]cell{from, to} {
		for dy := int32(0); dy < side; dy++ {
			for dx := int32(0); dx < side; dx++ {
				cells = append(cells, cell{x: at.x + dx, y: at.y + dy})
			}
		}
	}
	w.recomputeAreaCosts(cells...)
}

// strideCrossing reports the cells a transit leaves and enters on its tick-th
// tick after the one that started it, when the position's cell changes on that
// tick. The starting tick already adds the first step (MOVE-084), so tick t
// adds step t+1. A diagonal whose axes cross on different ticks passes through
// the cell between the two corners.
func strideCrossing(s NativeStride, tick int32) (from, to cell, crossed bool) {
	from, to = strideCell(s, tick), strideCell(s, tick+1)
	return from, to, from != to
}
