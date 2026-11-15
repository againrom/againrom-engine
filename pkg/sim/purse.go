package sim

import (
	"encoding/binary"
	"fmt"
)

// purseFormVersion carries every actor's non-zero Purse. A World with none
// writes no form, so every World before a map authors a unit's gold keeps its
// bytes.
const purseFormVersion byte = 137

// purseRecordLen is one record: the entity index and the gold.
const purseRecordLen = 8

func (w *World) appendPurses(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		if e.Purse != 0 {
			b = le.AppendUint32(b, uint32(i))
			b = le.AppendUint32(b, e.Purse)
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, purseFormVersion)
}

func (w *World) unmarshalPurses(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed purse section") }
	env, ok := readWrapper(data, purseFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%purseRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += purseRecordLen {
		index, gold := le.Uint32(s[at:]), le.Uint32(s[at+4:])
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || gold == 0 {
			return fail()
		}
		prior = int64(index)
		entities[index].Purse = gold
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
