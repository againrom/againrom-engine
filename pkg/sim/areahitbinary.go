package sim

import (
	"encoding/binary"
	"fmt"
)

const areaHitFormVersion byte = 141
const areaHitRecordLen = 8

func (w *World) appendAreaHitCounts(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		if e.AreaHitCount != 0 {
			b = le.AppendUint32(b, uint32(i))
			b = le.AppendUint32(b, uint32(e.AreaHitCount))
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, areaHitFormVersion)
}

func (w *World) unmarshalAreaHitCounts(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed area hit count section") }
	env, ok := readWrapper(data, areaHitFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%areaHitRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += areaHitRecordLen {
		index, count := le.Uint32(s[at:]), int32(le.Uint32(s[at+4:]))
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || count == 0 {
			return fail()
		}
		prior = int64(index)
		entities[index].AreaHitCount = count
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
