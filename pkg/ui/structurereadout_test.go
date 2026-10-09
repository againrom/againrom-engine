package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/text"
)

// readoutExpected draws the centred lines on the whole card independently.
func readoutExpected(f *text.Font, lines [3]string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 160, 242))
	for i, ys := range [3]int{28, 44, 54} {
		w, _ := f.Measure(lines[i])
		x := 72 - w/2
		ink := color.RGBA{R: 185, G: 159, B: 73, A: 255}
		if i == 2 {
			ink = color.RGBA{R: 107, G: 154, B: 120, A: 255}
		}
		f.DrawFlat(img, lines[i], x+1, ys+1, color.RGBA{R: 8, G: 8, B: 8, A: 255})
		f.Draw(img, lines[i], x, ys, ink)
	}
	return img
}

func readoutSame(t *testing.T, got *image.RGBA, want *image.RGBA) bool {
	t.Helper()
	if got.Bounds() != want.Bounds() {
		return false
	}
	for y := 0; y < got.Bounds().Dy(); y++ {
		for x := 0; x < got.Bounds().Dx(); x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// Widget 8 centres each structure line inside the card (MENU-071).
func TestStructureReadoutIsTheHoveredThenTheSelectedStructure(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.sel = nil
	ref := InspectionSubject{InspectionStructure, 7}
	f := v.cardFont()
	f.Glyphs['H'-text.FirstChar].Advance += 3
	if w, _ := f.Measure(v.words.PanelCaptions[19]); w != 30 {
		t.Fatalf("caption width %d, want 30", w)
	}
	if w, _ := f.Measure("456/789"); w != 29 {
		t.Fatalf("ratio width %d, want 29", w)
	}

	if _, _, ok := v.structureReadoutPresent(); ok {
		t.Fatal("a readout is drawn with nothing hovered or selected")
	}
	inspectionHover(t, a, v, ref)
	lines := [3]string{"Gate", v.words.PanelCaptions[19], "456/789"}
	check := func(what string) {
		t.Helper()
		pic, at, ok := v.structureReadoutPresent()
		if !ok {
			t.Fatalf("%s: no structure readout", what)
		}
		width := 0
		for _, line := range lines {
			w, _ := v.cardFont().Measure(line)
			width = max(width, w)
		}
		if want := image.Pt(v.frameW-88-width/2, 526+28); at != want {
			t.Errorf("%s: readout at %v, want %v", what, at, want)
		}
		if v.structKey.lines != lines {
			t.Errorf("%s: lines %q, want %q", what, v.structKey.lines, lines)
		}
		got := image.NewRGBA(image.Rect(0, 0, 160, 242))
		rect := pic.Bounds().Add(at.Sub(image.Pt(v.frameW-160, 526)))
		if !rect.In(got.Bounds()) {
			t.Errorf("%s: readout %v lies outside the card", what, rect)
		}
		draw.Draw(got, rect, pic, pic.Bounds().Min, draw.Src)
		if !readoutSame(t, got, readoutExpected(v.cardFont(), lines)) {
			t.Errorf("%s: the card differs from the independently centred lines", what)
		}
	}
	check("hovered")
	v.selStructure = ref
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatal(err)
	}
	check("selected")
	a.Layout(1280, 768)
	check("selected at wider frame")

	// Hovering a unit draws no structure readout, though a structure is selected.
	v.selStructure = ref
	inspectionHover(t, a, v, InspectionSubject{InspectionUnit, 7})
	if _, _, ok := v.structureReadoutPresent(); ok {
		t.Error("a hovered unit drew a structure readout")
	}
}

// A plain click selects a destructible structure and a rectangle never does
// (MISSION-065).
func TestAPlainClickSelectsAStructureAndARectangleDoesNot(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	ref := InspectionSubject{InspectionStructure, 7}
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}

	// A rectangle round the structure selects nothing and leaves the unit.
	for _, act := range []struct {
		edge string
		x, y int
	}{{"press", x - 30, y - 30}, {"release", x + 30, y + 30}} {
		if err := a.HeadlessPointer(act.edge, act.x, act.y); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := a.HeadlessSelectedStructure(); ok {
		t.Error("a rectangle selected a structure")
	}

	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	got, ok := a.HeadlessSelectedStructure()
	if !ok || got != ref {
		t.Fatalf("selected structure %+v,%v after a plain click, want %+v", got, ok, ref)
	}
	if sel := a.HeadlessSelection(); len(sel) != 0 {
		t.Errorf("the unit selection %v survived a structure click", sel)
	}
	if v.selectionSummary()&selSummaryStructure == 0 {
		t.Error("the selection summary lacks bit 0x20 for a selected structure")
	}

	// With the pointer on plain ground the selected structure's readout shows.
	gx, gy, err := a.HeadlessGroundPoint()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", gx, gy); err != nil {
		t.Fatal(err)
	}
	if _, _, lines, err := a.HeadlessStructureReadout(); err != nil || lines[2] != "456/789" {
		t.Errorf("selected-structure readout %q, %v", lines, err)
	}

	// A right click clears it.
	for _, edge := range []string{"right-press", "right-release"} {
		if err := a.HeadlessPointer(edge, gx, gy); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := a.HeadlessSelectedStructure(); ok {
		t.Error("a right click left the structure selected")
	}
}

// An indestructible class is not selected by a click (MISSION-065).
func TestAPlainClickDoesNotSelectAnIndestructibleStructure(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.sel = nil
	c := v.structureInfo[7]
	c.Indestructible, c.Usable = true, true
	ref := InspectionSubject{InspectionStructure, 7}
	x, y, err := v.InspectionPoint(ref)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := a.HeadlessSelectedStructure(); ok {
		t.Errorf("an indestructible structure was selected: %+v", got)
	}
}
