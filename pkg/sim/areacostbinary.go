package sim

import (
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
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, e := range rows {
		b = binary.LittleEndian.AppendUint16(b, e.Key)
		b = append(b, e.Mask, e.Byte)
	}
	return endWrapper(b, ws, areaCostFormVersion)
}

func (w *World) unmarshalAreaCosts(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed area cost section") }
	env, ok := readWrapper(data, areaCostFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+areaCostRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65536 || span != 4+areaCostRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) || !slices.Equal(next.decayedAreaCosts(), rows) {
		return fail()
	}
	*w = next
	return nil
}
