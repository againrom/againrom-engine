package sav

// CellTrigger is the supported six-byte projection of a saved cell payload.
// SAV-CELLLOAD-111: operation, power, source x/y, relocation x/y at +2c..+31.
// Neither operation zero nor 26 makes this record absent.
type CellTrigger struct {
	Cell  uint16
	Bytes [6]byte
}

// CellTriggers returns detached values in archive order. SAV-CELLLOAD-109
// restores records sequentially, so duplicate keys must retain that order.
// No key or operation value is filtered; the remaining payload is not modeled.
func (f *File) CellTriggers() ([]CellTrigger, bool, error) {
	// Body is editable; cached World offsets may describe an earlier layout.
	// The exact walk bounds every count and the whole document before any
	// projected record is returned, including short/extended WriteCount starts.
	doc, present, err := f.exactDocument()
	if err != nil || !present {
		return nil, present, err
	}
	w := doc.world
	out := make([]CellTrigger, w.CellRecCount)
	for i := range out {
		off := w.CellRecDataOff + i*cellRecLen
		out[i].Cell = uint16(f.Body[off]) | uint16(f.Body[off+1])<<8
		copy(out[i].Bytes[:], f.Body[off+2+0x2c:off+2+0x32])
	}
	return out, true, nil
}
