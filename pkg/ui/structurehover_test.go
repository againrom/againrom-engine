package ui

import (
	"image"
	"testing"
)

// A structure class that is indestructible and not usable is no hover hit.
func TestHoverCursorSkipsIndestructibleUnusableStructure(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	ref := InspectionSubject{InspectionStructure, 7}
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		indestructible, usable bool
		want                   string
	}{
		{false, false, "select"},
		{false, true, "select"},
		{true, true, "select"},
		{true, false, "move"},
	} {
		c := v.structureInfo[7]
		c.Indestructible, c.Usable = tc.indestructible, tc.usable
		v.structureStates[0].Kind = 1
		if tc.usable {
			v.structureStates[0].Kind = 15
		}
		v.structureStates[0].Size = image.Pt(1, 1)
		v.structureStates[0].UseAdmission = structureAdmission(v.structureStates[0].Kind, image.Pt(1, 1))
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		if name, ok := v.missionHoverCursor(); !ok || name != tc.want {
			t.Errorf("indestructible=%v usable=%v: hover cursor %q,%v, want %q",
				tc.indestructible, tc.usable, name, ok, tc.want)
		}
	}
}
