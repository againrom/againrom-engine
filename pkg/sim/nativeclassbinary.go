package sim

import (
	"encoding/binary"
	"fmt"
)

const nativeClassFormVersion byte = 107
const nativeClassRecordLen = 5

func (w *World) appendNativeClasses(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.NativeClass.Present {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if !e.NativeClass.Present {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		flag := byte(0)
		if e.NativeClass.Fighter {
			flag = 1
		}
		b = append(b, flag)
	}
	return endWrapper(b, ws, nativeClassFormVersion)
}

func (w *World) unmarshalNativeClasses(data []byte, proof formProof) error {
	if len(data) < headerLen {
		return fmt.Errorf("byte form truncated")
	}
	fail := func() error { return fmt.Errorf("malformed native class section") }
	env, ok := readWrapper(data, nativeClassFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+nativeClassRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeClassRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	var prior EntityID
	for n := uint64(0); n < count; n++ {
		o := start + 4 + nativeClassRecordLen*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[o:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || n > 0 && id <= prior || data[o+4] > 1 || next.entities[i].ActorLoad.Source.Class != 0 {
			return fail()
		}
		prior = id
		next.entities[i].NativeClass = NativeClass{Present: true, Fighter: data[o+4] != 0}
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
