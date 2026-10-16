package sim

import (
	"encoding/binary"
	"fmt"
)

const bodyReferenceFormVersion byte = 120
const bodyReferenceRecordLen = 16

// BodyReferenceRecord binds a body's loaded order words to its actor.
type BodyReferenceRecord struct {
	ID         EntityID
	References [3]uint32
}

// bodyReferences are the words the byte form carries: they end when the
// actor lives again.
func bodyReferences(e Entity) [3]uint32 {
	if e.Alive() || e.Dying() {
		return [3]uint32{}
	}
	return e.BodyReferences
}

// deadReferences are a retired body's dead record References. U44, the
// book selection, and U40, the kill credit, are not body words.
func deadReferences(e Entity) [5]uint32 {
	b := bodyReferences(e)
	return [5]uint32{b[0], b[1], 0, b[2]}
}

func bodyReferenceFault(e Entity) error {
	if e.BodyReferences != ([3]uint32{}) && (e.Alive() || e.Dying()) {
		return fmt.Errorf("sim: body references on living actor %d", e.ID)
	}
	return nil
}

// RestoreBodyReferences sets the loaded order words of existing bodies.
func (w *World) RestoreBodyReferences(rows []BodyReferenceRecord) error {
	next := make([]Entity, len(w.entities))
	copy(next, w.entities)
	for _, row := range rows {
		i := indexOfEntity(next, row.ID)
		if i < 0 {
			return fmt.Errorf("sim: body references name no actor")
		}
		next[i].BodyReferences = row.References
		if err := bodyReferenceFault(next[i]); err != nil {
			return err
		}
	}
	w.entities = next
	return nil
}

// releaseBodyReferences drops the words of an actor that lives again.
func (w *World) releaseBodyReferences() {
	for i := range w.entities {
		e := &w.entities[i]
		e.BodyReferences = bodyReferences(*e)
	}
}

func (w *World) appendBodyReferences(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if bodyReferences(e) != ([3]uint32{}) {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		references := bodyReferences(e)
		if references == ([3]uint32{}) {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		for _, v := range references {
			b = binary.LittleEndian.AppendUint32(b, v)
		}
	}
	return endWrapper(b, ws, bodyReferenceFormVersion)
}

func (w *World) unmarshalBodyReferences(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed body references") }
	env, ok := readWrapper(data, bodyReferenceFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+bodyReferenceRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+bodyReferenceRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) {
		return fail()
	}
	last := -1
	for n := 0; n < int(count); n++ {
		at := start + 4 + bodyReferenceRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i <= last {
			return fail()
		}
		e := &next.entities[i]
		for k := range e.BodyReferences {
			e.BodyReferences[k] = binary.LittleEndian.Uint32(data[at+4+4*k:])
		}
		if e.BodyReferences == ([3]uint32{}) || bodyReferenceFault(*e) != nil {
			return fail()
		}
		last = i
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
