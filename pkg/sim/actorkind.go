package sim

import (
	"encoding/binary"
	"fmt"
)

// actorKindFormVersion carries every actor's non-zero kind word and main
// skill. The first game's actors hold neither, so its Worlds write no form
// and keep their bytes.
const actorKindFormVersion byte = 139

// actorKindRecordLen is one record: the entity index, the kind word and the
// main skill.
const actorKindRecordLen = 10

func (w *World) appendActorKinds(b []byte) []byte {
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, e := range w.entities {
		if e.KindWord != 0 || e.MainSkill != 0 {
			b = le.AppendUint32(b, uint32(i))
			b = le.AppendUint16(b, e.KindWord)
			b = le.AppendUint32(b, uint32(e.MainSkill))
		}
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, actorKindFormVersion)
}

func (w *World) unmarshalActorKinds(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed actor kind section") }
	env, ok := readWrapper(data, actorKindFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%actorKindRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += actorKindRecordLen {
		index, kind, main := le.Uint32(s[at:]), le.Uint16(s[at+4:]), int32(le.Uint32(s[at+6:]))
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || kind == 0 && main == 0 {
			return fail()
		}
		prior = int64(index)
		entities[index].KindWord, entities[index].MainSkill = kind, main
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
