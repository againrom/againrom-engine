package sim

import (
	"bytes"
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
	b[0] = autoHealingFormVersion
	start := len(b)
	b = append(b, byte(count))
	for slot, p := range w.autoHealing {
		if p.Present {
			b = append(b, byte(slot))
			b = binary.LittleEndian.AppendUint32(b, p.Percent)
		}
	}
	b = binary.LittleEndian.AppendUint16(b, uint16(len(b)-start))
	return append(b, 'A', 'H', 'L', '1')
}

func (w *World) unmarshalAutoHealing(data []byte) error {
	malformed := func() error { return fmt.Errorf("malformed autohealing save section") }
	if len(data) < headerLen+12 {
		return fmt.Errorf("byte form truncated: autohealing header needs %d bytes", headerLen+12)
	}
	if !bytes.Equal(data[len(data)-4:], []byte("AHL1")) {
		return malformed()
	}
	span := int(binary.LittleEndian.Uint16(data[len(data)-6:]))
	if span < 6 || span > 1+relationSlots*5 || len(data)-6-span < headerLen {
		return malformed()
	}
	start := len(data) - 6 - span
	raw := data[start : len(data)-6]
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
	base := bytes.Clone(data[:start])
	base[0] = formatVersion
	next := *w
	if err := next.unmarshalBinary(base); err != nil {
		return err
	}
	next.autoHealing = policies
	*w = next
	return nil
}
