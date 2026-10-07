package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestTrailerLocatorMarkerAlignmentAndStructuralSetter(t *testing.T) {
	for _, noWorld := range []bool{false, true} {
		for _, name := range []string{"t.alm", "tt.alm"} {
			for _, marker := range []uint32{0xbadface1, 0x12345678} {
				t.Run(fmt.Sprintf("city=%v/name=%s/marker=%x", noWorld, name, marker), func(t *testing.T) {
					s := standard()
					s.noWorld, s.mapName = noWorld, name
					body := s.body()
					// A literal marker in the independent fixture writer defines
					// the suffix. No decoded offset constructs the expectation.
					at := bytes.LastIndex(body, []byte{0xe1, 0xac, 0xdf, 0xba})
					if at < 0 {
						t.Fatal("fixture marker absent")
					}
					body = binary.LittleEndian.AppendUint32(body[:at], marker)
					if marker == 0xbadface1 {
						body = binary.LittleEndian.AppendUint32(body, 0x55667788)
					}
					off := len(body)
					for i := range 100 {
						body = binary.LittleEndian.AppendUint32(body, 0xa1b2c300+uint32(i))
					}
					if len(body)&1 != 0 {
						body = append(body, 0xa7)
					}
					f := open(t, s)
					f.Body = body
					f, err := Open(f.Marshal())
					if err != nil {
						t.Fatal(err)
					}
					check := func(wantOff int) {
						t.Helper()
						if f.TrailerOff != wantOff {
							t.Fatalf("TrailerOff %d, want writer offset %d", f.TrailerOff, wantOff)
						}
						for i := range 100 {
							if binary.LittleEndian.Uint32(f.Body[f.TrailerOff+4*i:]) != 0xa1b2c300+uint32(i) {
								t.Fatal("located a different trailer dword", i)
							}
						}
					}
					check(off)
					if err := f.SetMapName("x" + name); err != nil {
						t.Fatal(err)
					}
					check(off + 1)
				})
			}
		}
	}
}
