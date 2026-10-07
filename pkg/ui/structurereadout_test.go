package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

// readoutExpected draws the readout's three lines independently of the
// production composer: a flat (8,8,8) shadow one pixel down and right, then the
// ink, at relative heights 0, 16 and 26.
func readoutExpected(f *text.Font, lines [3]string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 40))
	for i, ys := range [3]int{0, 16, 26} {
		ink := color.RGBA{R: 185, G: 159, B: 73, A: 255}
		if i == 2 {
			ink = color.RGBA{R: 107, G: 154, B: 120, A: 255}
		}
		f.DrawFlat(img, lines[i], 1, ys+1, color.RGBA{R: 8, G: 8, B: 8, A: 255})
		f.Draw(img, lines[i], 0, ys, ink)
	}
	return img
}

func readoutSame(t *testing.T, got *image.RGBA, want *image.RGBA) bool {
	t.Helper()
	for y := 0; y < got.Bounds().Dy(); y++ {
		for x := 0; x < got.Bounds().Dx(); x++ {
			if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// Widget 8 draws a structure's building name, the word Health and current
// over maximum health, at the column's right edge minus 88 (MENU-071). The
// hovered structure wins; with none hovered the single selected one is read.
func TestStructureReadoutIsTheHoveredThenTheSelectedStructure(t *testing.T) {
	a, v := inspectionFixture(t, image.Pt(1024, 768))
	v.sel = nil
	ref := InspectionSubject{InspectionStructure, 7}

	if _, _, ok := v.structureReadoutPresent(); ok {
		t.Fatal("a readout is drawn with nothing hovered or selected")
	}
	inspectionHover(t, a, v, ref)
	pic, at, ok := v.structureReadoutPresent()
	if !ok {
		t.Fatal("no readout over a hovered structure")
	}
	if want := image.Pt(1024-88, 480+28); at != want {
		t.Errorf("readout at %v, want %v", at, want)
	}
	lines := [3]string{"Gate", v.words.PanelCaptions[19], "456/789"}
	if v.structKey.lines != lines {
		t.Errorf("lines %q, want %q", v.structKey.lines, lines)
	}
	if !readoutSame(t, pic, readoutExpected(v.cardFont(), lines)) {
		t.Error("the picture differs from the independently drawn lines")
	}

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
