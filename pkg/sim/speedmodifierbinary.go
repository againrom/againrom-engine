package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const speedModifierFormVersion byte = 119
const speedModifierRecordLen = 8

// speedModifierFault refuses a speed modifier on an entity whose speed is not
// a native Human's base plus modifier.
func speedModifierFault(e Entity) error {
	if e.SpeedModifier != 0 && !e.nativeHumanoid() {
		return fmt.Errorf("sim: speed modifier %d on entity %d (humanoid %t, source class %d) that is not a native Humanoid", e.SpeedModifier, e.ID, e.Humanoid, e.ActorLoad.Source.Class)
	}
	return nil
}

// nativeHumanoid reports a Humanoid whose speed is its own base plus
// SpeedModifier. A source-backed Human's modifier lives in its source record.
func (e *Entity) nativeHumanoid() bool {
	return e.Humanoid && e.ActorLoad.Source.Class == 0
}

func (w *World) appendSpeedModifiers(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.SpeedModifier != 0 {
			count++
		}
	}
	if count == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = speedModifierFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.SpeedModifier != 0 {
			b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
			b = binary.LittleEndian.AppendUint32(b, uint32(e.SpeedModifier))
		}
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'S', 'P', 'M', '1')
}

func (w *World) unmarshalSpeedModifiers(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed speed modifier") }
	if len(data) < headerLen+13+speedModifierRecordLen || !bytes.Equal(data[len(data)-4:], []byte("SPM1")) {
		return fail()
	}
	version := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if version >= speedModifierFormVersion || span < 4+speedModifierRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+speedModifierRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = version
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) {
		return fail()
	}
	last := -1
	for n := 0; n < int(count); n++ {
		at := start + 4 + speedModifierRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i <= last {
			return fail()
		}
		e := &next.entities[i]
		e.SpeedModifier = int32(binary.LittleEndian.Uint32(data[at+4:]))
		if e.SpeedModifier == 0 || speedModifierFault(*e) != nil {
			return fail()
		}
		last = i
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
