package sim

import (
	"fmt"
	"sort"
)

// SavedCellActorSlot is an archive identity key for one of a saved cell
// record's two movement-domain actor slots, together with this build's own
// live rebind of it where the key names a current actor.
//
// SAV-CELLLOAD-110 names one rebind rule for all ten of a cell record's
// identity slots: a nonzero key is looked up in the archive identity map,
// and success replaces it with a live address, while zero and a failed
// lookup leave the saved value unchanged. Ground and Air are the two slots
// this build has a live actor registry for at LOAD; Bound reports whether
// that lookup found a current actor. This type deliberately does not reuse
// SavedActorSlot (savedmotion.go): the two are independent readers of the
// same rebind RULE for two different subsystems, not one shared mechanism,
// and conflating them would make this story's own resolution a special case
// of the unrelated motion-continuation overlay that type belongs to.
type SavedCellActorSlot struct {
	Key    uint32
	Entity EntityID
	Bound  bool
}

// SavedCellRecord is one archived cell's residual state: the fields no
// earlier story already carries in its own typed form (SAV-CELLLOAD-111).
// BaselineCost/BaselineStatic live in SavedStructureCell and the six trigger
// bytes in cellTail; this holds the layer count, both residue spans, the two
// movement-domain actor keys and their rebind, and the Sack/SpellEffect
// layer keys — carried opaquely because this build has no live identity
// registry for either class yet (docs/DIVERGENCES.md).
type SavedCellRecord struct {
	Cell         uint16
	LayerCount   uint8
	Residue0     uint8
	Residue1     [2]byte
	Ground, Air  SavedCellActorSlot
	Sack         uint32
	SpellEffects [6]uint32
}

// SavedCellRecords returns this world's carried cell-record residue, in
// ascending cell-key order. It is a fresh slice: mutating the result cannot
// reach the world it came from.
func (w *World) SavedCellRecords() []SavedCellRecord {
	out := make([]SavedCellRecord, len(w.savedCellRecords))
	copy(out, w.savedCellRecords)
	return out
}

// SetSavedCellRecords replaces the carried records outright, with no
// validation. Form85 gives the field a wire position
// (carriedresumebinary.go), so both staging round trips now carry it through
// UnmarshalBinary like every other field and neither calls this any more; it
// remains for a caller that wants to set the field directly without a
// byte-form round trip.
func (w *World) SetSavedCellRecords(records []SavedCellRecord) {
	w.savedCellRecords = append([]SavedCellRecord(nil), records...)
}

// Physical slot writers update the carried view at the same mutation point.
// They do not create an absent residue carrier or change the other layer.
func (w *World) syncCurrentCellActor(key uint16, layer int, slot SavedActorSlot) {
	i := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
	if i == len(w.savedCellRecords) || w.savedCellRecords[i].Cell != key {
		return
	}
	value := SavedCellActorSlot{Key: slot.Key, Entity: slot.Entity, Bound: slot.Bound}
	if layer == 0 {
		w.savedCellRecords[i].Ground = value
	} else {
		w.savedCellRecords[i].Air = value
	}
}

func (w *World) syncCurrentCellSack(key uint16, identity uint32) {
	i := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
	if i < len(w.savedCellRecords) && w.savedCellRecords[i].Cell == key {
		w.savedCellRecords[i].Sack = identity
	}
}

func (w *World) removeCurrentCellRecord(key uint16) {
	i := sort.Search(len(w.savedCellRecords), func(i int) bool { return w.savedCellRecords[i].Cell >= key })
	if i < len(w.savedCellRecords) && w.savedCellRecords[i].Cell == key {
		w.savedCellRecords = append(w.savedCellRecords[:i], w.savedCellRecords[i+1:]...)
	}
}

// ImportOriginalCellRecords overlays the saved projection after map
// construction, the same overlay point ImportOriginalCellTails uses: this is
// an independent reader of the same source table SAV-CELLLOAD-109 describes,
// not a second writer of live gameplay state, so it does not gate on or
// interact with cellTails or savedStructureCells. The caller has already
// applied SAV-CELLLOAD-109's own last-write-wins duplicate-key rule; this
// only sorts and validates the result is then strictly ascending.
func (w *World) ImportOriginalCellRecords(records []SavedCellRecord) error {
	out := append([]SavedCellRecord(nil), records...)
	sort.Slice(out, func(i, j int) bool { return out[i].Cell < out[j].Cell })
	for i := 1; i < len(out); i++ {
		if out[i].Cell == out[i-1].Cell {
			return fmt.Errorf("sim: saved cell records have duplicate key %04x", out[i].Cell)
		}
	}
	w.savedCellRecords = out
	return nil
}
