package sim

import (
	"encoding/binary"
	"fmt"
)

const nativeTrainingFormVersion byte = 105
const nativeTrainingRecordLen = 4 + 4*skillSlots

func (w *World) appendNativeTraining(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.NativeTraining.Present {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if !e.NativeTraining.Present {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		for _, level := range e.NativeTraining.Levels {
			b = binary.LittleEndian.AppendUint32(b, uint32(level))
		}
	}
	return endWrapper(b, ws, nativeTrainingFormVersion)
}

func (w *World) unmarshalNativeTraining(data []byte, proof formProof) error {
	if len(data) < headerLen {
		return fmt.Errorf("byte form truncated")
	}
	fail := func() error { return fmt.Errorf("malformed native training section") }
	env, ok := readWrapper(data, nativeTrainingFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+nativeTrainingRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeTrainingRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	var prior EntityID
	for n := uint64(0); n < count; n++ {
		o := start + 4 + nativeTrainingRecordLen*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[o:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || n > 0 && id <= prior {
			return fail()
		}
		prior = id
		training := NativeTraining{Present: true}
		for j := range training.Levels {
			training.Levels[j] = int32(binary.LittleEndian.Uint32(data[o+4+4*j:]))
		}
		next.entities[i].NativeTraining = training
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
