package sim

import (
	"encoding/binary"
	"fmt"
)

// beyondBookFormVersion carries the instances of the spell ids past the
// entity record's book slots. A World whose books hold none writes no form,
// so every World before a second-game book keeps its bytes.
const beyondBookFormVersion byte = 140

// beyondBookRecordLen is one record: the entity index, then the instance of
// each spell id past the slots as range, Defensive and mana word.
const beyondBookRecordLen = 4 + 4*(BookIDs-BookWireSlots)

func (w *World) appendBeyondBooks(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		if e.Book.Beyond == ([BookIDs - BookWireSlots]BookSpell{}) {
			continue
		}
		b = le.AppendUint32(b, uint32(i))
		for _, s := range e.Book.Beyond {
			b = append(b, s.Range, s.Defensive)
			b = le.AppendUint16(b, s.ManaCost)
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, beyondBookFormVersion)
}

func (w *World) unmarshalBeyondBooks(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed beyond-slot book section") }
	env, ok := readWrapper(data, beyondBookFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%beyondBookRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += beyondBookRecordLen {
		index := le.Uint32(s[at:])
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) {
			return fail()
		}
		prior = int64(index)
		e := &entities[index]
		for i := range e.Book.Beyond {
			o := at + 4 + 4*i
			e.Book.Beyond[i] = BookSpell{s[o], s[o+1], le.Uint16(s[o+2:])}
		}
		if e.Book.Beyond == ([BookIDs - BookWireSlots]BookSpell{}) || e.Book.Validate(e.KnownSpells) != nil {
			return fail()
		}
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
