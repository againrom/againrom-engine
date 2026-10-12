package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const moverFormVersion byte = 122
const moverFormEntryLen = 4 + moverRecordLen

// moversForm lists the Mover records the decoder cannot build again: those
// that differ from the record the builder gives the actor from its current
// state. A World whose every record is as built writes no form.
func (w *World) moversForm() []moverRecord {
	var out []moverRecord
	for _, e := range w.entities {
		r := w.moverFor(e.ID)
		if r == nil || !w.moverExpected(e) {
			continue
		}
		if held := w.formMover(*r, e); held != w.builtMover(e, false) {
			out = append(out, held)
		}
	}
	return out
}

// formMover is the record as the byte form states it. A carried member that
// has made no crossing writes the step cost 0 either way while the cost of
// its cell is 0, so the form states it as crossed then, as a LOAD of its SAV
// does (adoptMovers).
func (w *World) formMover(r moverRecord, e Entity) moverRecord {
	if r.Joined && w.stepCostWord(e, moverCell(e)) == 0 {
		r.Joined = false
	}
	return r
}

func (w *World) appendMovers(b []byte) []byte {
	rows := w.moversForm()
	if len(rows) == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = moverFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, r := range rows {
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Entity))
		b = r.appendHeld(b)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'M', 'V', 'R', '1')
}

func (w *World) unmarshalMovers(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed mover records") }
	if len(data) < headerLen+13+moverFormEntryLen || !bytes.Equal(data[len(data)-4:], []byte("MVR1")) {
		return fail()
	}
	version := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if version >= moverFormVersion || span < 4+moverFormEntryLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+moverFormEntryLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = version
	next := *w
	if err := next.unmarshalForm(base); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) || next.SecondGame() {
		return fail()
	}
	next.constructMovers()
	last := -1
	for n := 0; n < int(count); n++ {
		at := start + 4 + moverFormEntryLen*n
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		i := indexOfEntity(next.entities, id)
		if i <= last || !next.moverExpected(next.entities[i]) {
			return fail()
		}
		r, ok := moverRecordFrom(id, data[at+4:])
		if !ok {
			return fail()
		}
		next.setMover(r)
		last = i
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
