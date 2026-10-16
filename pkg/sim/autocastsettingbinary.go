package sim

import (
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
	ws := beginWrapper(b)
	return endWrapper(append(b, rows...), ws, autoCastFormVersion)
}

func (w *World) unmarshalAutoCastSettings(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed automatic-casting section") }
	env, ok := readWrapper(data, autoCastFormVersion)
	count := len(env.section()) / 2
	if !ok || count == 0 || count > relationSlots {
		return fail()
	}
	start := env.start
	var settings [relationSlots]AutoCastSetting
	prior := -1
	for n := 0; n < count; n++ {
		slot, s := int(data[start+2*n]), AutoCastSetting(data[start+2*n+1])
		if slot >= relationSlots || slot <= prior || s.Level() == 0 || s&^(autoCastLevels|autoCastTargets) != 0 {
			return fail()
		}
		settings[slot], prior = s, slot
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.autoCastSetting = settings
	*w = next
	return nil
}
