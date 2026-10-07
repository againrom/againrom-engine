package sav

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPlayerControl1175PreservesFullWord(t *testing.T) {
	for _, control := range []uint32{2, 0x100, 0x102, 0xffffffff} {
		t.Run(fmt.Sprintf("%08x", control), func(t *testing.T) {
			f := open(t, standard())
			if err := Lookup("Player").set(f.Body, f.Players[1].Fields, "Participant", control); err != nil {
				t.Fatal(err)
			}
			original := append([]byte(nil), f.Body...)
			for n := 0; n < 2; n++ {
				var err error
				f, err = Open(f.Marshal())
				if err != nil {
					t.Fatalf("SAVE/LOAD %d rejected full Player control %#x: %v", n, control, err)
				}
				if len(f.Players) != 2 || f.Players[0].Participant != 0 || f.Players[1].Participant != control {
					t.Fatalf("Player controls changed: %+v", f.Players)
				}
				if !bytes.Equal(f.Body, original) {
					t.Fatal("full-width control round trip changed another body byte")
				}
			}
			if err := Lookup("Player").set(f.Body, f.Players[1].Fields, "Outcome", 3); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(f.Marshal()); err == nil {
				t.Fatal("accepting control width also admitted an invalid Outcome")
			}
		})
	}
}
