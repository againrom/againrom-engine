package sim

import (
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
// does (adoptMovers). The claims form carries the walk's claim.
func (w *World) formMover(r moverRecord, e Entity) moverRecord {
	r.Claim = walkClaim{}
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
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, r := range rows {
		b = binary.LittleEndian.AppendUint32(b, uint32(r.Entity))
		b = r.appendHeld(b)
	}
	return endWrapper(b, ws, moverFormVersion)
}

func (w *World) unmarshalMovers(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed mover records") }
	env, ok := readWrapper(data, moverFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+moverFormEntryLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+moverFormEntryLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
