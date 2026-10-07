package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// Independent test-only peel of the successor's bounded trailing section.
// Its expected form82 bytes come only from the frozen predecessor above.
func beforeCellStateForm1115(t *testing.T, raw []byte) []byte {
	t.Helper()
	raw = beforeActionClockForm1146(t, raw)
	if len(raw) > 0 && (raw[0] == 84 || raw[0] == 85) {
		raw = beforeObjects1115Form(t, raw)
	}
	if len(raw) < 38 || raw[0] != 82 && raw[0] != 83 {
		t.Fatal("unexpected cell compatibility form", len(raw))
	}
	end := len(raw)
	if raw[0] == 83 {
		span := uint64(binary.LittleEndian.Uint32(raw[end-4:]))
		if span > uint64(end-38) {
			t.Fatal("invalid cell compatibility suffix span", span, end)
		}
		end -= 4 + int(span)
	}
	out := bytes.Clone(raw[:end])
	out[0] = 82
	return out
}

func beforeCellStateAbsent1115(t *testing.T, w *sim.World) {
	t.Helper()
	// Reflection keeps this witness executable on the exact predecessor. Once
	// the new getter exists, its typed state must also remain genuinely absent.
	if reader := reflect.ValueOf(w).MethodByName("SavedCellPlanes"); reader.IsValid() {
		values := reader.Call(nil)
		if len(values) != 2 || values[0].Kind() != reflect.Pointer || !values[0].IsNil() || values[1].Bool() {
			t.Fatal("old native LOAD fabricated typed cell-lifecycle authority")
		}
	}
	// The raw zero span proves absence independently of a production getter.
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	end := len(raw)
	if raw[0] == 84 {
		// This helper also checks newly imported originals without map.reg.
		// Their object coverage may be present; only planes must be absent.
		span := uint64(binary.LittleEndian.Uint32(raw[end-4:]))
		if span > uint64(end-38) {
			t.Fatal("invalid object suffix before absent plane check")
		}
		end -= 4 + int(span)
	}
	if (raw[0] == 83 || raw[0] == 84) && binary.LittleEndian.Uint32(raw[end-4:]) != 0 {
		t.Fatal("old native LOAD fabricated cell-lifecycle authority")
	}
}
