package sim

// CellPlaneStart names what a World's cell planes are built from.
//
// Base is the installed terrain's constructor planes (TERR-PASS-049/050) under
// the installed map.reg cost table; nil where no install exists, and then the
// planes start from zero bytes with no cost table. Overlay is the map's
// Overlay plane, row-major over the World's bounds.
//
// Document marks a start from a loaded SAV: its Block records overlay Static
// and Dynamic on the base (SAV-LOAD-057, TERR-PASS-053) and nothing else does.
// Without it the World's own modelled fields fold over the base.
type CellPlaneStart struct {
	Base     *SavedCellPlanes
	Overlay  []byte
	Document bool
	Blocks   []SavedActorBlock
}

// CellPlanes is the one builder of a World's cell planes. Fresh mission entry
// and the SAV imports build through it; it writes nothing into the World.
//
// At mission start each in-bounds cell takes Static's low three bits from the
// native grid (passable bits 0..1, the scenery bit 3 as bit 2), with bit 2 set
// as well where the map's raw domain value is 2 or the Overlay plane marks the
// cell; Dynamic's low three bits follow Static; Cost and Height are the native
// planes' bytes and CostKnown is 1. Every other byte is the base's: the border
// stamp, the Overlay's Static 5 outside the low bits and the cost table.
func (w *World) CellPlanes(s CellPlaneStart) *SavedCellPlanes {
	planes := &SavedCellPlanes{}
	if s.Base != nil {
		*planes = *s.Base
	}
	if s.Document {
		for _, b := range s.Blocks {
			planes.Static[b.Cell], planes.Dynamic[b.Cell] = b.Static, b.Dyn
		}
		return planes
	}
	width := int(w.bounds.Width)
	for at, grid := range w.grid {
		key := uint16(at/width)<<8 | uint16(at%width)
		stat := grid&3 | (grid&8)>>1
		// A fresh grid still holds the map's raw domain value in bits 0..1;
		// domain 2 or an Overlay cell is the bit a loaded grid carries as bit 3.
		if grid&2 != 0 || (at < len(s.Overlay) && s.Overlay[at] != 0) {
			stat |= 4
		}
		planes.Static[key] = planes.Static[key]&^7 | stat
		planes.Dynamic[key] = planes.Dynamic[key]&^7 | planes.Static[key]&7
		planes.Cost[key], planes.Height[key], planes.CostKnown[key] = w.cost[at], w.height[at], 1
	}
	return planes
}
