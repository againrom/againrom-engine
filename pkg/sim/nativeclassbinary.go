package sim

import (
	"bytes"
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
	base, start := b[0], len(b)
	b[0] = nativeClassFormVersion
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
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'C', 'L', 'S', '1')
}

func (w *World) unmarshalNativeClasses(data []byte) error {
	if len(data) < headerLen {
		return fmt.Errorf("byte form truncated")
	}
	fail := func() error { return fmt.Errorf("malformed native class section") }
	if len(data) < headerLen+9+4+nativeClassRecordLen || !bytes.Equal(data[len(data)-4:], []byte("CLS1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= nativeClassFormVersion || span < 4+nativeClassRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+nativeClassRecordLen*count {
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
		o := start + 4 + nativeClassRecordLen*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[o:]))
		i := indexOfEntity(next.entities, id)
		if i < 0 || n > 0 && id <= prior || data[o+4] > 1 || next.entities[i].ActorLoad.Source.Class != 0 {
			return fail()
		}
		prior = id
		next.entities[i].NativeClass = NativeClass{Present: true, Fighter: data[o+4] != 0}
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
