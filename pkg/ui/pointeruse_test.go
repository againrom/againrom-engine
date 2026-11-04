package ui

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

// A second-game Pointer with an action slot answers the hover with the use
// cursor and a card at HEALTH 1000/1000, and a click sends one structure use
// (R2-ENGINE-605, R2-ENGINE-606). Without a slot it answers neither.
func TestPointerWithASlotAnswersHoverClickAndCard(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.structureInfo[7].ID = 0x4a
	v.entities[0].PlayerCharacter = true
	pointer := sim.Structure{Kind: 0x4a, UseAmount: 24, Width: 1, Height: 1}
	v.SetStructures([]MapStructure{{ID: 7, Kind: 0x4a, Size: image.Pt(1, 1), Cell: image.Pt(6, 6), UseAdmission: pointer}})
	var calls [][2]uint32
	v.SetStructureUseSink(func(e, s uint32) { calls = append(calls, [2]uint32{e, s}) })
	ref := InspectionSubject{InspectionStructure, 7}
	inspectionHover(t, a, v, ref)
	if name, ok := v.missionHoverCursor(); !ok || name != "town" {
		t.Fatal("Pointer cursor", name, ok)
	}
	card, ok := v.inspectionPanel(ref)
	if !ok || card.HP != 1000 || card.MaxHP != 1000 || card.Kind != InspectionStructure {
		t.Fatalf("Pointer card %+v ok %v", card, ok)
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
	if len(calls) != 1 || calls[0] != [2]uint32{1, 7} {
		t.Fatal("one Pointer use", calls)
	}
	pointer.UseAmount = 0
	v.structureStates[0].UseAdmission = pointer
	if name, ok := v.missionHoverCursor(); ok && name == "town" {
		t.Fatal("a Pointer without a slot admitted use")
	}
	if _, ok := v.inspectionPanel(ref); ok {
		t.Fatal("a Pointer without a slot showed a card")
	}
}

// A usable structure on explored ground out of current sight is picked only
// where the game's mission screen data admits it.
func TestUsableStructureInExploredGroundPicksByGameData(t *testing.T) {
	for _, admit := range []bool{false, true} {
		a, v := inspectionFixture(t, image.Pt(1024, 768))
		v.entities[0].PlayerCharacter = true
		v.SetMissionScreen(MissionScreen{ColumnW: MissionPanelW, PickUsableInExploredGround: admit})
		v.SetStructures([]MapStructure{{ID: 7, Kind: 0x4a, Size: image.Pt(1, 1), Cell: image.Pt(6, 6),
			UseAdmission: sim.Structure{Kind: 0x4a, UseAmount: 24, Width: 1, Height: 1}}})
		ref := InspectionSubject{InspectionStructure, 7}
		x, y, err := v.InspectionPoint(ref)
		if err != nil {
			t.Fatal(err)
		}
		fog := make([]byte, 1024)
		for i := range fog {
			fog[i] = FogVisible
		}
		fog[6*32+6] = FogExplored
		v.SetFog(fog, 32, 32)
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		_ = a.HeadlessStep()
		got, hit := v.hoverAt(x, y)
		if hit != admit || hit && got != ref {
			t.Fatalf("admit %v: pick %+v hit %v", admit, got, hit)
		}
	}
}
