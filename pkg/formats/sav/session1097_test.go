package sav

import (
	"encoding/binary"
	"testing"
)

func TestSavedSession1097CountersUseIndependentWireOffsets(t *testing.T) {
	// SAV-SESS-031: six bytes after the five raw spans, then WIN, unknown,
	// LOSE. Literal offsets are independent of Counters and its setters.
	body := make([]byte, 4374)
	binary.LittleEndian.PutUint32(body[4362:], 0xfedcba98)
	binary.LittleEndian.PutUint32(body[4366:], 0x76543210)
	binary.LittleEndian.PutUint32(body[4370:], 3)
	f := &File{Body: body, World: &WorldHalf{SessionOff: 0}}
	state, err := f.SessionState()
	if err != nil || state.Won != 0xfedcba98 || state.Lost != 3 {
		t.Fatalf("session counters = %d/%d: %v", state.Won, state.Lost, err)
	}
	for _, off := range []int{-1, 1, 4374, 1 << 30} {
		f.World.SessionOff = off
		if _, err := f.SessionState(); err == nil {
			t.Fatalf("accepted session at %d without its complete extent", off)
		}
	}
}
