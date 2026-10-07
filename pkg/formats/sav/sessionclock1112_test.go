package sav

import (
	"encoding/binary"
	"testing"
)

func TestSessionClock1112PreservesIndependentWorldHeadWords(t *testing.T) {
	for _, pair := range [][2]uint32{{9343, 584}, {12, 3}, {0xffffffff, 0x80000000}, {0, 0xffffffff}} {
		f := open(t, standard())
		// Literal offsets from SAV-HEAD-025, independent of the handoff's
		// field names. Reparse the head rather than directly assigning it.
		binary.LittleEndian.PutUint32(f.Body[0:4], pair[0])
		binary.LittleEndian.PutUint32(f.Body[4:8], pair[1])
		if err := f.readHead(); err != nil {
			t.Fatal(err)
		}
		s, err := f.SessionState()
		if err != nil {
			t.Fatal(err)
		}
		if s.SubTick != pair[0] || s.FullTick != pair[1] {
			t.Fatalf("world head%08x/%08x became%08x/%08x", pair[0], pair[1], s.SubTick, s.FullTick)
		}
		s.SubTick, s.FullTick = 17, 18
		if binary.LittleEndian.Uint32(f.Body[:4]) != pair[0] || binary.LittleEndian.Uint32(f.Body[4:8]) != pair[1] {
			t.Fatal("detached handoff changed the source head")
		}
	}
}
