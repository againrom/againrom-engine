package sim

import (
	"bytes"
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
	base, start := b[0], len(b)
	b[0] = nativeTrainingFormVersion
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
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'T', 'R', 'N', '1')
}

func (w *World) unmarshalNativeTraining(data []byte) error {
	if len(data) < headerLen {
		return fmt.Errorf("byte form truncated")
	}
	fail := func() error { return fmt.Errorf("malformed native training section") }
	if len(data) < headerLen+9+4+nativeTrainingRecordLen || !bytes.Equal(data[len(data)-4:], []byte("TRN1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= nativeTrainingFormVersion || span < 4+nativeTrainingRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeTrainingRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
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
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
