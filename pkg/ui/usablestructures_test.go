package ui

import (
	"fmt"
	"image"
	"testing"
	"time"

	"againrom/pkg/sim"
)

func TestUsableStructureHoverReadoutAndOrder(t *testing.T) {
	for _, tc := range []struct {
		class, current, maximum int
	}{{15, 2, 3}, {16, 0, 3}, {28, 0, 1}, {29, 1, 1}, {34, 0, 0}, {35, 0, 0}, {39, 0, 0}, {40, 0, 0}, {66, 0, 0}} {
		t.Run(fmt.Sprint(tc.class), func(t *testing.T) {
			a, v := inspectionFixture(t, image.Pt(1024, 768))
			v.entities[0].PlayerCharacter = true
			c := v.structureInfo[7]
			c.ID, c.Usable, c.Indestructible = int32(tc.class), true, tc.class != 28 && tc.class != 29
			v.SetStructures([]MapStructure{{ID: 7, Kind: uint16(tc.class), Size: image.Pt(1, 1), Health: uint16(tc.current), MaxHealth: uint16(tc.maximum), Cell: image.Pt(6, 6), UseAdmission: structureAdmission(uint16(tc.class), image.Pt(1, 1))}})
			ref := InspectionSubject{InspectionStructure, 7}
			inspectionHover(t, a, v, ref)
			if cursor, ok := v.missionHoverCursor(); !ok || cursor != "town" {
				t.Fatalf("cursor=%q,%v", cursor, ok)
			}
			_, _, lines, err := a.HeadlessStructureReadout()
			current, maximum := tc.current, tc.maximum
			if maximum == 0 {
				current, maximum = 1000, 1000
			}
			if err != nil || lines[2] != fmt.Sprintf("%d/%d", current, maximum) {
				t.Fatalf("readout=%q err=%v", lines, err)
			}
			var calls int
			v.SetStructureUseSink(func(actor, structure uint32) {
				calls++
				if actor != 1 || structure != 7 {
					t.Fatalf("use=%d,%d", actor, structure)
				}
			})
			x, y, err := v.InspectionPoint(ref)
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 || v.structureStates[0].MaxHealth != uint16(tc.maximum) {
				t.Fatalf("calls=%d state=%+v", calls, v.structureStates)
			}
		})
	}
}

func structureAdmission(kind uint16, size image.Point) sim.Structure {
	return sim.Structure{Kind: kind, Width: uint8(size.X), Height: uint8(size.Y)}
}

func checkStructureAdmission(t *testing.T, kind uint16) {
	t.Helper()
	for _, size := range []image.Point{{1, 1}, {0, 1}, {1, 0}} {
		t.Run(fmt.Sprintf("%d/%dx%d", kind, size.X, size.Y), func(t *testing.T) {
			a, v := inspectionFixture(t, image.Pt(1024, 768))
			ref := InspectionSubject{InspectionStructure, 7}
			x, y, err := v.InspectionPoint(ref)
			if err != nil {
				t.Fatal(err)
			}
			v.entities[0].PlayerCharacter = true
			c := v.structureInfo[7]
			c.ID, c.Usable, c.Indestructible = int32(kind), true, true
			admission := structureAdmission(kind, size)
			v.SetStructures([]MapStructure{{ID: 7, Kind: kind, Size: size, Cell: image.Pt(6, 6), UseAdmission: admission}})
			want := admission.Usable()
			if got := v.usableStructure(ref); got != want {
				t.Fatalf("UI use=%v simulation=%v", got, want)
			}
			if _, got := v.inspectionPanel(ref); got != want {
				t.Fatalf("UI readout=%v simulation=%v", got, want)
			}
			if err := a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			if _, got := v.inspectionAt(v.cursorX, v.cursorY); got != want {
				t.Fatalf("UI hit=%v simulation=%v", got, want)
			}
			cursor, ok := v.missionHoverCursor()
			if !ok || (cursor == "town") != want {
				t.Fatalf("cursor=%q,%v simulation=%v", cursor, ok, want)
			}
			uses, voices := 0, 0
			v.SetStructureUseSink(func(_, _ uint32) { uses++ })
			v.commandAcknowledgment = func(g VoiceGesture, _ []uint32, _ time.Time) {
				if g == VoiceTown {
					voices++
				}
			}
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			wantCalls := 0
			if want {
				wantCalls = 1
			}
			if uses != wantCalls || voices != wantCalls {
				t.Fatalf("uses=%d voices=%d simulation=%v", uses, voices, want)
			}
		})
	}
}

func TestUsableStructureAdmissionMatchesSimulation(t *testing.T) {
	if (sim.Structure{Kind: 70, Width: 1, Height: 1}).Usable() {
		t.Fatal("class 70 has no admitted use")
	}
	for _, kind := range []uint16{15, 16, 28, 29, 34, 35, 39, 40, 66, 70} {
		checkStructureAdmission(t, kind)
	}
}

func TestZeroMaximumStructureCtrlClickUsesGroundOrder(t *testing.T) {
	for _, kind := range []uint16{34, 35, 39, 40, 66} {
		for _, health := range []uint16{0, 1} {
			t.Run(fmt.Sprintf("%d/%d", kind, health), func(t *testing.T) {
				a, v := inspectionFixture(t, image.Pt(1024, 768))
				x, y, err := v.InspectionPoint(InspectionSubject{InspectionStructure, 7})
				if err != nil {
					t.Fatal(err)
				}
				v.entities[0].PlayerCharacter = true
				c := v.structureInfo[7]
				c.ID, c.Usable, c.Indestructible = int32(kind), true, true
				v.SetStructures([]MapStructure{{ID: 7, Kind: kind, Health: health, Cell: image.Pt(6, 6), Size: image.Pt(1, 1), UseAdmission: structureAdmission(kind, image.Pt(1, 1))}})
				attacks, moves := 0, 0
				a.flow.attack = func(_, _, _ uint32, _, _ int, _ bool) { attacks++ }
				a.flow.order = func(actor uint32, col, row int) { moves++ }
				for _, edge := range []bool{true, false} {
					in := atFrame(x, y)
					in.Viewer.Ctrl, in.AttackHeld = true, true
					in.PrimaryPressed, in.PrimaryReleased, in.Viewer.PrimaryDown = edge, !edge, edge
					a.step(in, atAt)
				}
				if attacks != 0 || moves != 1 {
					t.Fatalf("Ctrl zero maximum: attack sink=%d ground move=%d", attacks, moves)
				}
				if cursor, ok := v.missionHoverCursor(); !ok || cursor != "attack" {
					t.Fatalf("held attack cursor=%q,%v", cursor, ok)
				}
			})
		}
	}
}

func TestUnadmittedStructureKeepsPooledPhysicalHover(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.entities[0].PlayerCharacter = true
	c := v.structureInfo[7]
	c.ID, c.Usable, c.Indestructible = 70, true, true
	v.SetStructures([]MapStructure{{ID: 7, Kind: 70, Health: 1, MaxHealth: 1, Cell: image.Pt(6, 6), Size: image.Pt(1, 1), UseAdmission: structureAdmission(70, image.Pt(1, 1))}})
	ref := InspectionSubject{InspectionStructure, 7}
	inspectionHover(t, a, v, ref)
	if v.usableStructure(ref) {
		t.Fatal("class 70 admitted use")
	}
	if cursor, ok := v.missionHoverCursor(); !ok || cursor != "select" {
		t.Fatalf("pooled physical cursor=%q,%v", cursor, ok)
	}
	_, _, lines, err := a.HeadlessStructureReadout()
	if err != nil || len(lines) != 3 || lines[2] != "1/1" {
		t.Fatalf("pooled physical readout=%q err=%v", lines, err)
	}
	uses, moves := 0, 0
	v.SetStructureUseSink(func(_, _ uint32) { uses++ })
	a.flow.order = func(_ uint32, _, _ int) { moves++ }
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if uses != 0 || moves != 0 || len(v.sel) != 0 || v.selStructure.Kind != InspectionNone {
		t.Fatalf("pooled physical click uses=%d moves=%d units=%v structure=%v", uses, moves, v.sel, v.selStructure)
	}
}
