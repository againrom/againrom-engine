package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBookSelectionSurvivesCompletionAndColdForm(t *testing.T) {
	w := nativeClassWorld(t, 10, 10)
	base, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	w.entities[0].AdmittedBookSpell = 6
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != bookSelectionFormVersion || CheckSaveForm(raw) != nil || HasStructureBlockingForm(raw) != HasStructureBlockingForm(base) {
		t.Fatal("current selection form", err)
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	peeled := bytes.Clone(raw[:start])
	peeled[0] = raw[len(raw)-5]
	if !bytes.Equal(peeled, base) {
		t.Fatal("selection changed historical base")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].AdmittedBookSpell != 6 || cold.Hash() != w.Hash() {
		t.Fatal("cold selection", err)
	}
	if err := cold.UnmarshalBinary(base); err != nil || cold.entities[0].AdmittedBookSpell != 0 {
		t.Fatal("absent old selection default", err)
	}
	if cold.Hash() == w.Hash() {
		t.Fatal("loss control did not detect selection removal")
	}
	before := cold.Hash()
	for _, edit := range []func([]byte){
		func(b []byte) { b[len(b)-1] = 'X' },
		func(b []byte) { b[len(b)-5] = bookSelectionFormVersion },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], 0xffffffff) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 65536) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 999) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[start+8:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[start+8:], 29) },
	} {
		bad := bytes.Clone(raw)
		edit(bad)
		if cold.UnmarshalBinary(bad) == nil || CheckSaveForm(bad) == nil || cold.Hash() != before {
			t.Fatal("malformed selection admitted or changed receiver")
		}
	}
}
