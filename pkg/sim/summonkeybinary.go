package sim

import (
	"encoding/binary"
	"fmt"
)

// summonKeyFormVersion carries the caster key of each summoned actor, which
// no entity record holds. A form without it holds no key, so every World this
// build wrote before it reads with every actor outside the summoned cap.
const summonKeyFormVersion byte = 131

// summonKeyPairLen is one keyed actor: the uint32 entity index and its key.
const summonKeyPairLen = 8

// appendSummonKeys writes one pair per keyed entity, indices ascending; a
// World with no keyed entity writes no form.
func (w *World) appendSummonKeys(b []byte) []byte {
	if !w.hasSummonKeys() {
		return b
	}
	ws := beginWrapper(b)
	for i := range w.entities {
		if key := w.entities[i].summonKey; key != 0 {
			b = binary.LittleEndian.AppendUint32(b, uint32(i))
			b = binary.LittleEndian.AppendUint32(b, uint32(key))
		}
	}
	return endWrapper(b, ws, summonKeyFormVersion)
}

func (w *World) hasSummonKeys() bool {
	for i := range w.entities {
		if w.entities[i].summonKey != 0 {
			return true
		}
	}
	return false
}

func (w *World) unmarshalSummonKeys(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed summoned key section") }
	env, ok := readWrapper(data, summonKeyFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%summonKeyPairLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	entities := append([]Entity(nil), next.entities...)
	prior := int64(-1)
	for at := 0; at < len(s); at += summonKeyPairLen {
		index, key := binary.LittleEndian.Uint32(s[at:]), EntityID(binary.LittleEndian.Uint32(s[at+4:]))
		if int64(index) <= prior || uint64(index) >= uint64(len(entities)) || key == 0 {
			return fail()
		}
		prior = int64(index)
		entities[index].summonKey = key
	}
	next.entities = entities
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
