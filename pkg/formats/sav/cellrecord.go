package sav

import "fmt"

// Cell is the complete typed projection of one 54-byte cell-record table
// entry (SAV-CELLREC-032): the two-byte packed-cell key, plus every field
// SAV-CELLLOAD-109/110/111 name in the 52-byte payload behind it.
// CellTriggers and StructureCells remain this package's narrower,
// already-consumed projections of the same table; Cells is the complete one,
// used by the native writer and by an acceptance instrument that must
// account for every byte.
type Cell struct {
	Key uint16

	// BaselineCost (+0x00) and BaselineStatic (+0x01) are the captured
	// terrain baselines recomputation reads. LayerCount (+0x02) is the
	// occupied six-layer count, retained from the save until an ordinary
	// layer attach/detach recounts it. Residue0 (+0x03) is constructor-zero
	// with no known consumer in the enumerated population.
	BaselineCost, BaselineStatic, LayerCount, Residue0 uint8

	// Ground (+0x04, movement domain 1/2 actor), Air (+0x08, domain 3
	// actor), Building (+0x0c) and Sack (+0x10) are archive identity keys,
	// each later rebound to a live object on successful lookup
	// (SAV-CELLLOAD-110).
	Ground, Air, Building, Sack uint32

	// SpellEffects (+0x14..+0x28) are the six SpellEffect layer identity
	// keys. The same rebind rule names them (SAV-CELLLOAD-110); this
	// package only carries the raw keys.
	SpellEffects [6]uint32

	// Trigger (+0x2c..+0x31) is the six-byte trigger tail SAV-CELLLOAD-111
	// names; see CellTrigger for this package's own long-supported narrow
	// projection of the same bytes.
	Trigger [6]byte

	// Residue1 (+0x32..+0x33) is constructor-zero with no known consumer.
	Residue1 [2]byte
}

// Cells returns the complete cell-record table in archive order, one entry
// per record including repeated keys: SAV-CELLLOAD-109 restores records
// sequentially, so a caller that must reproduce last-write-wins duplicate
// handling needs that order, not a deduplicated map.
func (f *File) Cells() ([]Cell, bool, error) {
	// Body is editable; cached World offsets may describe an earlier layout,
	// on CellTriggers' own rule. The exact walk bounds every count and the
	// whole document before any projected record is returned.
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	w := doc.world
	out := make([]Cell, w.CellRecCount)
	for i := range out {
		off := w.CellRecDataOff + i*cellRecLen
		out[i] = decodeCell(f.Body[off : off+cellRecLen])
	}
	return out, true, nil
}

func decodeCell(rec []byte) Cell {
	var c Cell
	c.Key = u16(rec, 0)
	p := rec[2:]
	c.BaselineCost, c.BaselineStatic, c.LayerCount, c.Residue0 = p[0], p[1], p[2], p[3]
	c.Ground, c.Air, c.Building, c.Sack = u32(p, 4), u32(p, 8), u32(p, 12), u32(p, 16)
	for i := range c.SpellEffects {
		c.SpellEffects[i] = u32(p, 20+4*i)
	}
	copy(c.Trigger[:], p[0x2c:0x32])
	copy(c.Residue1[:], p[0x32:0x34])
	return c
}

func encodeCell(c Cell) [cellRecLen]byte {
	var rec [cellRecLen]byte
	put16(rec[:], 0, c.Key)
	p := rec[2:]
	p[0], p[1], p[2], p[3] = c.BaselineCost, c.BaselineStatic, c.LayerCount, c.Residue0
	put32(p, 4, c.Ground)
	put32(p, 8, c.Air)
	put32(p, 12, c.Building)
	put32(p, 16, c.Sack)
	for i, v := range c.SpellEffects {
		put32(p, 20+4*i, v)
	}
	copy(p[0x2c:0x32], c.Trigger[:])
	copy(p[0x32:0x34], c.Residue1[:])
	return rec
}

// SetCell writes the complete record at index i, key included: SAV-CELLLOAD-109
// does not add or remove records on LOAD, so a native writer's own record
// count and archive order already equal the loaded file's, and there is no
// separate immutable-key contract to enforce the way SetBlockRecord enforces
// one for the block-plane delta.
func (f *File) SetCell(i int, c Cell) error {
	w := f.World
	if w == nil {
		return fmt.Errorf("sav: this save has no world half")
	}
	if i < 0 || i >= w.CellRecCount {
		return fmt.Errorf("sav: cell record %d of %d", i, w.CellRecCount)
	}
	off := w.CellRecDataOff + i*cellRecLen
	enc := encodeCell(c)
	copy(f.Body[off:off+cellRecLen], enc[:])
	return nil
}
