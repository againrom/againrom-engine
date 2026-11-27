package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// structureOwnerFormVersion carries non-zero structure owners.
const structureOwnerFormVersion byte = 143

const structureOwnerRecordLen = 8

func (w *World) dropUnkeptStructureOwners(list []Structure) {
	if w.game.structureOwners {
		return
	}
	for i := range list {
		list[i].Owner = 0
	}
}

func (w *World) appendStructureOwners(b []byte) []byte {
	if !w.game.structureOwners {
		return b
	}
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for _, st := range w.structures {
		if st.Owner == 0 {
			continue
		}
		b = le.AppendUint32(b, uint32(st.ID))
		b = le.AppendUint32(b, st.Owner)
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, structureOwnerFormVersion)
}

func (w *World) unmarshalStructureOwners(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed structure-owner section") }
	env, ok := readWrapper(data, structureOwnerFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%structureOwnerRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	structs := slices.Clone(next.structures)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += structureOwnerRecordLen {
		id := StructureID(le.Uint32(s[at:]))
		owner := le.Uint32(s[at+4:])
		if int64(id) <= prior || owner == 0 {
			return fail()
		}
		idx := indexOfStructure(structs, id)
		if idx < 0 {
			return fail()
		}
		prior = int64(id)
		structs[idx].Owner = owner
	}
	next.structures = structs
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
