package sim

import (
	"bytes"
	"fmt"
)

// autoCastFormVersion marks a form that appends the stored automatic-casting
// settings. A world whose Players all read the default stores none, so its
// bytes and digest are the ones the base form gives.
const autoCastFormVersion byte = 121

func (w *World) appendAutoCastSettings(b []byte) []byte {
	var rows []byte
	for slot, s := range w.autoCastSetting {
		if s.Level() != 0 {
			rows = append(rows, byte(slot), byte(s))
		}
	}
	if len(rows) == 0 {
		return b
	}
	base := b[0]
	b[0] = autoCastFormVersion
	b = append(b, rows...)
	return append(b, byte(len(rows)/2), base, 'A', 'C', 'S', '1')
}

func (w *World) unmarshalAutoCastSettings(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed automatic-casting section") }
	if len(data) < headerLen+8 || !bytes.Equal(data[len(data)-4:], []byte("ACS1")) {
		return fail()
	}
	baseVersion, count := data[len(data)-5], int(data[len(data)-6])
	if baseVersion >= autoCastFormVersion || count == 0 || count > relationSlots || len(data)-6-2*count < headerLen {
		return fail()
	}
	start := len(data) - 6 - 2*count
	var settings [relationSlots]AutoCastSetting
	prior := -1
	for n := 0; n < count; n++ {
		slot, s := int(data[start+2*n]), AutoCastSetting(data[start+2*n+1])
		if slot >= relationSlots || slot <= prior || s.Level() == 0 || s&^(autoCastLevels|autoCastTargets) != 0 {
			return fail()
		}
		settings[slot], prior = s, slot
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.unmarshalForm(base, proof); err != nil {
		return err
	}
	next.autoCastSetting = settings
	*w = next
	return nil
}
