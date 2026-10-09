package sim

import (
	"bytes"
	"testing"
)

func TestTurnStateFormAdmissionRetainsStructureBlocking(t *testing.T) {
	for _, tc := range []struct {
		name     string
		blocking uint32
		wantForm bool
	}{
		{"default mask", 1, false},
		{"independent mask", 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			structure := Structure{ID: 7, Col: 10, Row: 11, Width: 1, Height: 1, Attach: 1, Blocking: tc.blocking}
			w := sbWorld(t, []Structure{structure})
			w.entities[0] = turnActor(1, 1, 1, 0, 16)
			w.entities[0].NativeBasis.ScalarsPresent = true
			w.entities[0].NativeBasis.ScalarKnown = 1 << ScalarT0C
			w.entities[0].NativeBasis.Scalars[ScalarT0C] = 7
			base, err := w.MarshalBinary()
			if err != nil || base[0] != nativeScalarFormVersion || HasStructureBlockingForm(base) != tc.wantForm {
				t.Fatal("native scalar predecessor lost blocking presence", err)
			}
			w.entities[0].requestFacing(128)
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if form[0] != turnStateFormVersion || HasStructureBlockingForm(form) != tc.wantForm {
				t.Fatal("turn wrapper changed structure blocking presence")
			}
			if err := CheckSaveForm(form); err != nil {
				t.Fatal("current turn form refused at the save boundary", err)
			}
			var cold World
			if err := cold.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			if cold.Hash() != w.Hash() || cold.structures[0] != structure || cold.entities[0].TurnState != w.entities[0].TurnState {
				t.Fatal("cold turn form lost current turn or structure blocking")
			}
			bad := bytes.Clone(form)
			bad[len(bad)-1] = 'X'
			if CheckSaveForm(bad) == nil || HasStructureBlockingForm(bad) {
				t.Fatal("malformed turn wrapper accepted")
			}
		})
	}
}
