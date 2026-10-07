package ui

import (
	"againrom/pkg/render/terrain"
	"image"
	"testing"
)

func TestSavedStructureRoster1114RebuildsArtHoverAndPortrait(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	inspectionHover(t, a, v, InspectionSubject{Kind: InspectionStructure, ID: 7})
	old := v.structureSet.Classes[1]
	copyClass := *old
	copyClass.Name = "Saved tower"
	copyClass.ID = 2
	v.structureSet.Classes[2] = &copyClass
	records := []terrain.StructureRecord{{ID: 7, X: 10 << 8, Y: 9 << 8, Key: 2}, {ID: 15, X: 12 << 8, Y: 9 << 8, Key: 1}}
	v.SetStructureRecords(records)
	v.SetStructures([]MapStructure{{ID: 7, Health: 77, MaxHealth: 130, Cell: image.Pt(10, 9)},
		{ID: 15, Health: 0xffff, MaxHealth: 0, Cell: image.Pt(12, 9)}})
	records[0].X = 0 // input is detached from the live geometry cache
	if len(v.grid.Structures) != 2 || v.grid.Structures[0].X != 10<<8 {
		t.Fatal("roster alias or missing source-only art")
	}
	for _, tc := range []struct {
		id      uint32
		name    string
		hp, max int
	}{{7, "Saved tower", 77, 130}, {15, "Gate", -1, 0}} {
		if tc.max == 0 {
			if _, _, err := v.InspectionPoint(InspectionSubject{Kind: InspectionStructure, ID: tc.id}); err == nil {
				t.Fatal("saved decoration without a health pool became inspectable")
			}
			continue
		}
		inspectionHover(t, a, v, InspectionSubject{Kind: InspectionStructure, ID: tc.id})
		panel, ok := v.InspectionPanel()
		if !ok || panel.Name != tc.name || panel.HP != tc.hp || panel.MaxHP != tc.max {
			t.Fatalf("saved card %+v", panel)
		}
		if v.characterPaneView(true).Figure == nil {
			t.Fatal("saved kind portrait absent")
		}
	}
	if entries, _ := v.Structures(); entries != 2 {
		t.Fatalf("art entries %d", entries)
	}
	v.SetStructureRecords(nil)
	v.SetStructures(nil)
	if entries, _ := v.Structures(); entries != 0 || len(v.structureInfo) != 0 {
		t.Fatal("empty saved roster left ALM art")
	}
	if _, _, err := v.InspectionPoint(InspectionSubject{Kind: InspectionStructure, ID: 7}); err == nil {
		t.Fatal("removed structure remains inspectable")
	}
}
