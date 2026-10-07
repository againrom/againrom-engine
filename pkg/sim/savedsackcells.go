package sim

import (
	"encoding/binary"
	"sort"
)

// registerSavedSackCell applies SAV-SACKENTRY-590 to an already validated
// opaque Sack identity. The caller owns identity binding and the Sack's actual
// collection lifecycle. False with no issue is the original local refusal.
func (w *World) registerSavedSackCell(cell uint16, sackKey uint32) (bool, string) {
	if w == nil || w.savedCellPlanes == nil {
		return false, "Sack cell registration lacks saved plane authority"
	}
	if sackKey == 0 {
		return false, "Sack cell registration requires a nonzero opaque Sack key"
	}
	if w.savedCellPlanes.Dynamic[cell]&1 != 0 {
		return false, ""
	}
	if c := w.motionCell(cell); c != nil {
		if binary.LittleEndian.Uint32(c.Payload[16:]) != 0 {
			return false, ""
		}
		// Existing-record reuse changes only the Sack slot. Its saved
		// baselines, current planes and other occupants are not recomputed.
		binary.LittleEndian.PutUint32(c.Payload[16:], sackKey)
		w.syncCurrentCellSack(cell, sackKey)
		return true, ""
	}
	// createSavedCell refuses unknown current cost before changing the World.
	// Its new zero52 payload has no Building key needing external authority.
	if issue := w.createSavedCell(cell); issue != "" {
		return false, issue
	}
	binary.LittleEndian.PutUint32(w.motionCell(cell).Payload[16:], sackKey)
	if issue := w.recomputeSavedCell(cell); issue != "" {
		return false, issue
	}
	w.syncCurrentCellSack(cell, sackKey)
	w.refreshSavedPlaneBlocks()
	return true, ""
}

// removeSavedSackCell applies SAV-SACKREMOVE-591/SAV-SACKPLANES-592. It
// deliberately accepts an empty or different Sack slot: there is no supplied
// identity comparison. Allocator/callback effects are not implemented here.
func (w *World) removeSavedSackCell(cell uint16) (bool, string) {
	if w == nil || w.savedCellPlanes == nil {
		return false, "Sack cell removal lacks saved plane authority"
	}
	c := w.motionCell(cell)
	if c == nil {
		return false, ""
	}
	// Resolve the current native tail and every recompute dependency without
	// publishing even the slot clear. The Sack slot is not a recompute input.
	next := *c
	next.Payload = w.savedCellPayload(next)
	if issue := w.savedCellRecomputeIssue(next); issue != "" {
		return false, issue
	}
	binary.LittleEndian.PutUint32(next.Payload[16:], 0)
	*c = next
	if issue := w.recomputeSavedCell(cell); issue != "" {
		return false, issue
	}
	w.syncCurrentCellSack(cell, 0)
	if motionCellDeletionEligible(c.Payload) && !w.nativeAreaCellPresent(cell) {
		p := w.savedCellPlanes
		carry := p.Static[cell] & 0x10
		p.Cost[cell], p.CostKnown[cell], p.Static[cell] = c.Payload[0], 1, c.Payload[1]|carry
		// The actor-detach helper restores Dynamic too. Sack deletion does
		// not: its recomputed bit5/layer result survives the node removal.
		p.Dynamic[cell] |= carry
		at := sort.Search(len(w.savedMotion.Cells), func(i int) bool { return w.savedMotion.Cells[i].Cell >= cell })
		w.savedMotion.Cells = append(w.savedMotion.Cells[:at], w.savedMotion.Cells[at+1:]...)
		w.removeCurrentCellRecord(cell)
		if at := sort.Search(len(w.cellTails), func(i int) bool { return w.cellTails[i].Key >= cell }); at < len(w.cellTails) && w.cellTails[at].Key == cell {
			w.cellTails = append(w.cellTails[:at], w.cellTails[at+1:]...)
		}
		if w.hasSavedStructures {
			at := sort.Search(len(w.savedStructureCells), func(i int) bool { return w.savedStructureCells[i].Cell >= cell })
			if at < len(w.savedStructureCells) && w.savedStructureCells[at].Cell == cell {
				w.savedStructureCells = append(w.savedStructureCells[:at], w.savedStructureCells[at+1:]...)
			}
			w.rebuildStructureSlots()
		}
		w.syncNativeSavedPlaneCell(cell)
	}
	w.refreshSavedPlaneBlocks()
	return true, ""
}

// savedSackAtCell is SAV-SACKCALLER-593's local accessor, not a collection
// lookup. Missing plane authority cannot establish a visible Sack key.
func (w *World) savedSackAtCell(cell uint16) uint32 {
	if w == nil || w.savedCellPlanes == nil || w.savedCellPlanes.Static[cell]&0x20 == 0 {
		return 0
	}
	if c := w.motionCell(cell); c != nil {
		return binary.LittleEndian.Uint32(c.Payload[16:])
	}
	return 0
}
