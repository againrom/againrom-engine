package sim

import (
	"encoding/binary"
	"fmt"
)

const structureBlockingFormVersion byte = 100

func HasStructureBlockingForm(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	version := data[0]
	for version != structureBlockingFormVersion {
		fw, ok := formWrapperOf(version)
		if !ok {
			return false
		}
		base, inner, ok := fw.inner(data)
		if !ok || base >= version {
			return false
		}
		version, data = base, inner
	}
	return true
}

func (w *World) appendStructureBlocking(b []byte) []byte {
	count := 0
	for _, st := range w.structures {
		if st.Blocking != st.Attach {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, st := range w.structures {
		if st.Blocking != st.Attach {
			b = binary.LittleEndian.AppendUint32(b, uint32(st.ID))
			b = binary.LittleEndian.AppendUint32(b, st.Blocking)
		}
	}
	return endWrapper(b, ws, structureBlockingFormVersion)
}

func (w *World) unmarshalStructureBlocking(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed structure blocking section") }
	env, ok := readWrapper(data, structureBlockingFormVersion)
	payload := env.section()
	span := uint64(len(payload))
	if !ok || span < 12 {
		return fail()
	}
	count := uint64(binary.LittleEndian.Uint32(payload))
	if count == 0 || span != 4+8*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.structures)) {
		return fail()
	}
	var prior StructureID
	for i := uint64(0); i < count; i++ {
		o := 4 + 8*int(i)
		id := StructureID(binary.LittleEndian.Uint32(payload[o:]))
		mask := binary.LittleEndian.Uint32(payload[o+4:])
		if i > 0 && id <= prior {
			return fail()
		}
		prior = id
		at := indexOfStructure(next.structures, id)
		if at < 0 || mask == next.structures[at].Attach {
			return fail()
		}
		next.structures[at].Blocking = mask
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
