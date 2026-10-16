package sim

import (
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
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.SpeedModifier != 0 {
			b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
			b = binary.LittleEndian.AppendUint32(b, uint32(e.SpeedModifier))
		}
	}
	return endWrapper(b, ws, speedModifierFormVersion)
}

func (w *World) unmarshalSpeedModifiers(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed speed modifier") }
	env, ok := readWrapper(data, speedModifierFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+speedModifierRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+speedModifierRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
