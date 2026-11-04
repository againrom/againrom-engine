package ui

import (
	"image"
	"testing"
)

// An indestructible unusable structure keeps the move cursor.
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

func TestStructureHoverKeepsPooledCardsAndEmptyRectangleUses(t *testing.T) {
	for _, usable := range []bool{false, true} {
		a, v := inspectionFixture(t, image.Pt(1024, 768))
		c := v.structureInfo[7]
		c.Indestructible, c.Usable = true, usable
		if usable {
			c.SelectionSet = true
			v.entities[0].PlayerCharacter = true
			v.SetStructures([]MapStructure{{ID: 7, Kind: 39, Size: image.Pt(1, 1), Cell: image.Pt(6, 6), UseAdmission: structureAdmission(39, image.Pt(1, 1))}})
		}
		ref := InspectionSubject{InspectionStructure, 7}
		x, y, err := v.InspectionPoint(ref)
		if err != nil {
			t.Fatalf("usable=%v: %v", usable, err)
		}
		if err := a.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		if card, ok := v.InspectionPanel(); !ok || card.Kind != ref.Kind || card.ID != ref.ID {
			t.Fatalf("usable=%v: card=%+v/%v", usable, card, ok)
		}
		want := "move"
		if usable {
			want = "town"
		}
		if cursor, ok := v.missionHoverCursor(); !ok || cursor != want {
			t.Fatalf("usable=%v: cursor=%q/%v, want %q", usable, cursor, ok, want)
		}
		if usable {
			calls := 0
			v.SetStructureUseSink(func(actor, structure uint32) {
				if actor != 1 || structure != 7 {
					t.Fatalf("use=%d,%d", actor, structure)
				}
				calls++
			})
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 {
				t.Fatalf("use calls=%d", calls)
			}
		}
	}
}

func TestKnownEmptyUnusableStructureDoesNotGainAFallback(t *testing.T) {
	_, v := inspectionFixture(t, image.Pt(1024, 768))
	v.structureInfo[7].SelectionSet = true
	if _, _, err := v.InspectionPoint(InspectionSubject{InspectionStructure, 7}); err == nil {
		t.Fatal("known empty unusable rectangle gained a fallback")
	}
}
