package ui

import (
	"image"
	"testing"
)

func TestInspectionSkipsDecorationsButKeepsDamagedBuildings(t *testing.T) {
	for _, flat := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			hp, max uint16
			inspect bool
		}{
			{"decoration", 0, 0, false},
			{"no maximum", 12, 0, false},
			{"tower", 456, 789, true},
			{"ruin", 0, 789, true},
			{"negative health", 0xffff, 789, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				a, v := inspectionFixture(t, image.Pt(1024, 768))
				v.structureInfo[7].Flat = flat
				ref := InspectionSubject{Kind: InspectionStructure, ID: 7}
				x, y, err := v.InspectionPoint(ref)
				if err != nil {
					t.Fatal(err)
				}
				v.SetStructures([]MapStructure{{ID: 7, Health: tc.hp, MaxHealth: tc.max, Cell: image.Pt(6, 6)}})
				if err := a.HeadlessPointer("hover", x, y); err != nil {
					t.Fatal(err)
				}
				want := InspectionSubject{Kind: InspectionUnit, ID: 1}
				if tc.inspect {
					want = ref
				}
				if got, ok := v.Inspection(); !ok || got != want {
					t.Fatalf("flat=%v hover=%+v/%v want=%+v", flat, got, ok, want)
				}
				card, ok := v.InspectionPanel()
				pane := v.characterPaneView(true)
				if !ok || card.Kind != want.Kind || card.ID != want.ID || !pane.HasSubject || pane.Subject != card {
					t.Fatal("figure and health card did not follow the eligible subject")
				}
				if !tc.inspect {
					v.sel = nil
					if _, ok := v.InspectionPanel(); ok || v.characterPaneView(true).HasSubject {
						t.Fatal("decoration showed a card without a selected unit")
					}
				}
			})
		}
	}
}
