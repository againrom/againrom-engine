package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
)

// areaCostFormVersion marks a form that appends the cost bytes an area layer's
// reads have decayed. A world with no decayed cell writes no trailer, so its
// bytes and digest are the ones the base form gives. A cell that has not been
// read since its last recompute holds the byte its layers give and is not
// listed; a world loaded without the trailer recomputes every layered cell.
const areaCostFormVersion byte = 104

const areaCostRecordLen = 4

// decayedAreaCosts lists the entries whose byte differs from a recompute's.
func (w *World) decayedAreaCosts() []areaCostEntry {
	var out []areaCostEntry
	for _, e := range w.areaCosts {
		x, y := keyCell(e.Key)
		if i, ok := w.cellIndex(x, y); ok && e.Byte != freshAreaCost(w.cost[i], e.Mask) {
			out = append(out, e)
		}
	}
	return out
}

func (w *World) appendAreaCosts(b []byte) []byte {
	rows := w.decayedAreaCosts()
	if len(rows) == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = areaCostFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, e := range rows {
		b = binary.LittleEndian.AppendUint16(b, e.Key)
		b = append(b, e.Mask, e.Byte)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'A', 'C', 'P', '1')
}

func (w *World) unmarshalAreaCosts(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed area cost section") }
	if len(data) < headerLen+9+4+areaCostRecordLen || !bytes.Equal(data[len(data)-4:], []byte("ACP1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= areaCostFormVersion || span < 4+areaCostRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65536 || span != 4+areaCostRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	rows := make([]areaCostEntry, 0, count)
	for n := uint64(0); n < count; n++ {
		o := start + 4 + areaCostRecordLen*int(n)
		e := areaCostEntry{Key: binary.LittleEndian.Uint16(data[o:]), Mask: data[o+2], Byte: data[o+3]}
		x, y := keyCell(e.Key)
		if _, ok := next.cellIndex(x, y); !ok || e.Mask == 0 || e.Mask >= 1<<len(areaLayerSpells) || n > 0 && e.Key <= rows[n-1].Key {
			return fail()
		}
		rows = append(rows, e)
	}
	next.areaCosts, next.areaCostLive = rows, false
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) || !slices.Equal(next.decayedAreaCosts(), rows) {
		return fail()
	}
	*w = next
	return nil
}
