package sim

import (
	"encoding/binary"
	"fmt"
)

// Form96 is the opt-in global-healing extension. A world without a policy keeps
// its canonical form95 bytes and historical behavior. A present policy changes
// both the version and hash; it can never hide in an application-only setting.
const autoHealingFormVersion byte = 96

func (w *World) appendAutoHealing(b []byte) []byte {
	count := 0
	for _, p := range w.autoHealing {
		if p.Present {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = append(b, byte(count))
	for slot, p := range w.autoHealing {
		if p.Present {
			b = append(b, byte(slot))
			b = binary.LittleEndian.AppendUint32(b, p.Percent)
		}
	}
	return endWrapper(b, ws, autoHealingFormVersion)
}

func (w *World) unmarshalAutoHealing(data []byte, _ formProof) error {
	malformed := func() error { return fmt.Errorf("malformed autohealing save section") }
	if len(data) < headerLen+12 {
		return fmt.Errorf("byte form truncated: autohealing header needs %d bytes", headerLen+12)
	}
	env, ok := readWrapper(data, autoHealingFormVersion)
	raw := env.section()
	span := len(raw)
	if !ok || span < 6 || span > 1+relationSlots*5 {
		return malformed()
	}
	count := int(raw[0])
	if count == 0 || count > relationSlots || span != 1+count*5 {
		return malformed()
	}
	var policies [relationSlots]autoHealingPolicy
	prior := -1
	for i := 0; i < count; i++ {
		at := 1 + i*5
		slot := int(raw[at])
		if slot >= relationSlots || slot <= prior {
			return malformed()
		}
		policies[slot] = autoHealingPolicy{Present: true, Percent: binary.LittleEndian.Uint32(raw[at+1:])}
		prior = slot
	}
	next := *w
	if err := next.unmarshalBinary(env.base()); err != nil {
		return err
	}
	next.autoHealing = policies
	*w = next
	return nil
}
