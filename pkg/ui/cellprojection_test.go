package ui

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// editorCentres are the screen points the map editor draws on one cell, read
// from the drawn lines: the selected unit's marker cross, a structure
// footprint's reference cross and a square area's centre cross.
type editorCentres struct {
	marker, footprint, area [2]float64
}

// openCellEditor opens an editor over v with a unit, a reference to a 2x2
// structure footprint and a square area, all anchored at cell, and selects the
// unit.
func openCellEditor(t *testing.T, v *Viewer, w, h int, cell image.Point) *MapEditor {
	t.Helper()
	unit := InspectionRecord{Kind: "Unit", Spatial: true, Cell: cell, Size: image.Pt(1, 1),
		References: []InspectionReference{{Label: "footprint", Role: "condition", Targets: []InspectionTarget{{Cell: cell, Size: image.Pt(2, 2)}}}}}
	structure := InspectionRecord{Kind: "Structure", Index: 0, Spatial: true, Cell: cell, Size: image.Pt(2, 2)}
	script := &InspectionScriptModel{Areas: []InspectionArea{{Trigger: 0, Shape: AreaSquare, Rect: image.Rect(cell.X-2, cell.Y-2, cell.X+3, cell.Y+3), Centre: cell, Radius: 2}}}
	e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
		return &InspectionDocument{Viewer: v, Width: w, Height: h, Records: []InspectionRecord{unit, structure}, Script: script, SourceBytes: []byte("unchanged")}, nil
	})
	e.Layout(1280, 800)
	if err := e.Open("cell projection"); err != nil {
		t.Fatal(err)
	}
	if v.Mode() != ModeDisplaced {
		t.Fatal("the fixture must draw displaced terrain")
	}
	e.Select(0, true)
	return e
}

// drawnCentres reads the three crosses off the editor's own line lists.
func drawnCentres(t *testing.T, e *MapEditor) editorCentres {
	t.Helper()
	var c editorCentres
	found := 0
	for _, l := range e.markerLines() {
		if l.color == editorGold && l.y0 == l.y1 && l.x1-l.x0 == 10 {
			c.marker = [2]float64{float64(l.x0+l.x1) / 2, float64(l.y0)}
			found++
		}
	}
	refs := e.referenceMarkerLines()
	if len(refs) != 6 {
		t.Fatalf("reference lines %d, want an outline and a cross", len(refs))
	}
	cross := refs[4]
	c.footprint = [2]float64{float64(cross.x0+cross.x1) / 2, float64(cross.y0)}
	for _, l := range e.areaLines() {
		if l.x1-l.x0 == 8 && l.y1-l.y0 == 8 {
			c.area = [2]float64{float64(l.x0+l.x1) / 2, float64(l.y0+l.y1) / 2}
			found++
		}
	}
	if found != 2 {
		t.Fatalf("found %d marker and area crosses, want one of each", found)
	}
	return c
}

// samePoint is equality within the float32 rounding the editor's lines carry.
func samePoint(a, b [2]float64) bool {
	return math.Abs(a[0]-b[0]) < 0.01 && math.Abs(a[1]-b[1]) < 0.01
}

// midpointCentre is the editor's former displaced centre: the midpoint of the
// cell's top-left and bottom-right mesh corners. It is the loss control.
func midpointCentre(v *Viewer, cell image.Point) [2]float64 {
	left, top := v.proj.WorldCorner(cell.X, cell.Y)
	right, bottom := v.proj.WorldCorner(cell.X+1, cell.Y+1)
	x, y := v.cam.WorldToScreen(float64((left+right)/2), float64((top+bottom)/2))
	return [2]float64{x, y}
}

// checkCellCentresAgree is the same-height proof on one sloped cell: the three
// editor crosses stand on the game's path-line centre, and the former midpoint
// rule does not.
func checkCellCentresAgree(t *testing.T, v *Viewer, w, h int, cell image.Point) {
	t.Helper()
	e := openCellEditor(t, v, w, h, cell)
	px, py := v.pathCellCentre(cell)
	path := [2]float64{px, py}
	got := drawnCentres(t, e)
	for _, c := range []struct {
		name string
		at   [2]float64
	}{{"unit marker", got.marker}, {"footprint cross", got.footprint}, {"area centre", got.area}} {
		if !samePoint(c.at, path) {
			t.Errorf("%s at %v, the game's path centre at %v", c.name, c.at, path)
		}
	}
	old := midpointCentre(v, cell)
	if samePoint(old, path) || math.Abs(old[1]-path[1]) < 1 {
		t.Fatalf("loss control: the midpoint rule %v also meets the path centre %v; the cell does not discriminate", old, path)
	}
}

func TestEditorCrossesStandOnThePathCentreOfASlopedCell(t *testing.T) {
	const w, h = 64, 64
	alt := make([]uint8, w*h)
	cell := image.Pt(20, 20)
	// The anti-diagonal corners are raised: the anchor height is 30, and the
	// midpoint of the top-left and bottom-right corners stays at 0.
	alt[cell.Y*w+cell.X+1] = 60
	alt[(cell.Y+1)*w+cell.X] = 60
	v, err := NewViewer("cell projection", terrain.Grid{Width: w, Height: h, Tiles: make([]uint16, w*h), Altitudes: alt}, &terrain.Tileset{})
	if err != nil {
		t.Fatal(err)
	}
	if got := v.proj.AnchorHeight(cell.X, cell.Y); got != 30 {
		t.Fatalf("anchor height %d, want 30", got)
	}
	checkCellCentresAgree(t, v, w, h, cell)
}

// TestCellSpaceFlatAndDisplacedCorners pins the family's two modes against
// raw arithmetic: flat corners on the 32-pixel lattice, displaced corners at
// r*32 - h - MinV, and the displaced centre at the four-corner mean.
func TestCellSpaceFlatAndDisplacedCorners(t *testing.T) {
	const w, h = 4, 4
	alt := make([]uint8, w*h)
	alt[1*w+1], alt[1*w+2], alt[2*w+1], alt[2*w+2] = 10, 0xfe, 7, 3 // 0xfe is -2
	p := terrain.Project(alt, w, h)
	flat, disp := cellSpace{}, cellSpace{proj: &p}
	if x, y := flat.corner(3, 2); x != 96 || y != 64 {
		t.Fatalf("flat corner %v,%v", x, y)
	}
	if x, y := flat.centre(image.Pt(1, 1)); x != 48 || y != 48 {
		t.Fatalf("flat centre %v,%v", x, y)
	}
	if x, y := disp.corner(2, 1); x != 64 || y != float64(32+2-p.MinV) {
		t.Fatalf("displaced corner %v,%v (MinV %d)", x, y, p.MinV)
	}
	// (10 - 2 + 7 + 3) / 4 = 4
	if x, y := disp.centre(image.Pt(1, 1)); x != 48 || y != float64(48-4-p.MinV) {
		t.Fatalf("displaced centre %v,%v", x, y)
	}
	q := disp.quad(1, 1)
	for i, d := range [4][2]int{{1, 1}, {2, 1}, {2, 2}, {1, 2}} {
		x, y := disp.corner(d[0], d[1])
		if q[i] != [2]float64{x, y} {
			t.Fatalf("quad corner %d is %v, want %v,%v", i, q[i], x, y)
		}
	}
}
