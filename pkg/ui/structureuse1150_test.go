package ui

import (
	"image"
	"testing"
)

func TestStructureUse1150RealPointerAndGates(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.structureInfo[7].ID, v.structureInfo[7].Usable = 28, true
	v.entities[0].PlayerCharacter = true
	v.SetStructures([]MapStructure{{ID: 7, Health: 0, MaxHealth: 1, Cell: image.Pt(6, 6)}})
	var calls [][2]uint32
	v.SetStructureUseSink(func(e, s uint32) { calls = append(calls, [2]uint32{e, s}) })
	ref := InspectionSubject{InspectionStructure, 7}
	inspectionHover(t, a, v, ref)
	if name, ok := v.missionHoverCursor(); !ok || name != "town" {
		t.Fatal("usable cursor", name, ok)
	}
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err = a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 1 || calls[0] != [2]uint32{1, 7} || len(v.sel) != 1 || v.sel[0] != 1 {
		t.Fatal("one click identity/selection", calls, v.sel)
	}
	for _, clear := range []func(){
		func() { v.entities[0].PlayerCharacter = false },
		func() { v.entities[0].PlayerCharacter = true; v.structureInfo[7].Usable = false },
		func() { v.structureInfo[7].Usable = true; v.structureInfo[7].ID = 39 },
	} {
		clear()
		if name, ok := v.missionHoverCursor(); ok && name == "town" {
			t.Fatal("unsupported actor/class admitted use")
		}
	}
}
