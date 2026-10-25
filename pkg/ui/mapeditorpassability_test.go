package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"time"
)

// passProbe is a brush host whose terrains carry swatches and whose documents
// carry a passability plane: the cells of the first row are closed by the
// border, cell (30,30) by Water and the flag; a passability stroke sets or
// clears the flag arm on its first cell.
type passProbe struct {
	brushProbe
	swatch *image.RGBA
}

func (p *passProbe) BrushTerrains() []BrushTerrain {
	return []BrushTerrain{{Class: 2, Name: "Grass", Swatch: p.swatch}, {Class: 8, Name: "Mountain"}}
}

func (p *passProbe) Paint(s BrushStroke) (*InspectionDocument, error) {
	before := p.doc.Passability
	next, err := p.brushProbe.Paint(s)
	if err != nil || next == nil {
		return next, err
	}
	next.Passability = append([]PassArm(nil), before...)
	if s.Plane == BrushPlanePassability {
		i := s.Cells[0].Y*next.Width + s.Cells[0].X
		next.Passability[i] &^= PassFlag
		if s.Value == 1 {
			next.Passability[i] |= PassFlag
		}
	}
	return next, nil
}

func passEditor(t *testing.T) (*MapEditor, *passProbe) {
	t.Helper()
	e, b := brushEditor(t)
	swatch := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for i := range swatch.Pix {
		swatch.Pix[i] = 0xff
	}
	p := &passProbe{brushProbe: *b, swatch: swatch}
	arms := make([]PassArm, e.doc.Width*e.doc.Height)
	for x := 0; x < e.doc.Width; x++ {
		arms[x] = PassBorder
	}
	arms[30*e.doc.Width+30] = PassWater | PassFlag
	e.doc.Passability = arms
	e.doc.Edits = p
	p.doc = e.doc
	return e, p
}

func passClick(e *MapEditor, r image.Rectangle) {
	pt := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(2, 0))
}

// passMenu clicks the bar entry key, then row n of its menu.
func passMenu(t *testing.T, e *MapEditor, key string, n int) {
	t.Helper()
	r, ok := e.BarRect(key)
	if !ok {
		t.Fatalf("no %s entry", key)
	}
	passClick(e, r)
	r, ok = e.MenuRowRect(n)
	if !ok {
		t.Fatalf("the %s menu has no row %d", key, n)
	}
	passClick(e, r)
}

// The Brushes menu turns on each brush and turns the brush off; a row of the
// brush already on leaves it on. Its control, a press on a disabled entry,
// changes nothing.
func TestMapEditorBrushesMenuEntersEachMode(t *testing.T) {
	e, _ := passEditor(t)
	for n, want := range []brushMode{brushTerrain, brushHeight, brushObjects, brushPassability, brushOff} {
		passMenu(t, e, "brushes", n)
		if e.brush.mode != want || e.menu != "" {
			t.Fatalf("menu row %d: mode %d menu %q, want mode %d", n, e.brush.mode, e.menu, want)
		}
	}
	passMenu(t, e, "brushes", 1)
	passMenu(t, e, "brushes", 1)
	if e.brush.mode != brushHeight {
		t.Fatalf("choosing the brush on turned it to %d", e.brush.mode)
	}
	items := e.menuItems("brushes")
	if len(items) != 5 || !strings.HasPrefix(items[1].label, "[x] ") || !strings.HasPrefix(items[0].label, "[ ] ") {
		t.Fatalf("menu rows %+v", items)
	}

	ro, _ := brushEditor(t)
	ro.doc.Edits = nil // a document that cannot be edited offers no brush
	for _, b := range ro.barEntries() {
		if b.key == "brushes" && !b.disabled {
			t.Fatal("the Brushes entry is live on a read-only document")
		}
	}
	r, _ := ro.BarRect("brushes")
	passClick(ro, r)
	if ro.menu != "" || ro.brush.mode != brushOff {
		t.Fatal("a disabled Brushes entry opened its menu")
	}
}

// T, H, B and Q each switch their brush on and off again.
func TestMapEditorBrushKeysStillSwitch(t *testing.T) {
	e, _ := passEditor(t)
	for key, mode := range map[string]brushMode{"terrain": brushTerrain, "height": brushHeight, "objects": brushObjects, "passability": brushPassability} {
		if err := e.Key(key); err != nil || e.brush.mode != mode || !e.brushActive() {
			t.Fatalf("%s: mode %d", key, e.brush.mode)
		}
		if err := e.Key(key); err != nil || e.brush.mode != brushOff {
			t.Fatalf("%s twice: mode %d", key, e.brush.mode)
		}
	}
}

// The passability brush's rows send 1 and 0 on the passability plane; the
// window shows the plane of the document each stroke returns.
func TestMapEditorPassabilityBrushStrokes(t *testing.T) {
	e, p := passEditor(t)
	if err := e.Key("passability"); err != nil {
		t.Fatal(err)
	}
	if rows := e.brushRows(); len(rows) != 2 || rows[0] != "Block" || rows[1] != "Unblock" {
		t.Fatalf("rows %v", rows)
	}
	e.ShowCell(image.Pt(20, 20))
	drag(e, image.Pt(20, 20))
	_ = e.Key("down")
	drag(e, image.Pt(30, 30))
	if len(p.strokes) != 2 || p.strokes[0].Plane != BrushPlanePassability || p.strokes[0].Value != 1 || p.strokes[1].Plane != BrushPlanePassability || p.strokes[1].Value != 0 {
		t.Fatalf("strokes %+v", p.strokes)
	}
	if e.passArmAt(image.Pt(20, 20)) != PassFlag {
		t.Fatal("the blocked cell is not shown closed")
	}
	if e.passArmAt(image.Pt(30, 30)) != PassWater {
		t.Fatal("unblocked water is not shown closed by Water alone")
	}
}

// The view is off by default; on, it fills every drawn cell white or red by
// the document's plane, and the status line names the closing arms. The
// passability brush shows it too.
func TestMapEditorPassabilityViewFillsAndNames(t *testing.T) {
	e, _ := passEditor(t)
	if e.view.passability || len(e.passabilityQuads()) != 0 {
		t.Fatal("the passability view is on by default")
	}
	items := e.menuItems("view")
	row := -1
	for i, it := range items {
		if it.key == "passview" {
			row = i
		}
	}
	if row < 0 {
		t.Fatal("the View menu has no Passability row")
	}
	passMenu(t, e, "view", row)
	if !e.view.passability {
		t.Fatal("the View row did not turn the view on")
	}
	e.ShowCell(image.Pt(30, 30))
	quads := e.passabilityQuads()
	if len(quads) == 0 {
		t.Fatal("no cell is filled")
	}
	red, white := 0, 0
	for _, q := range quads {
		switch q.fill {
		case passClosedFill:
			red++
		case passOpenFill:
			white++
		default:
			t.Fatalf("fill %v", q.fill)
		}
	}
	if red == 0 || white == 0 {
		t.Fatalf("%d red and %d white cells", red, white)
	}
	pt := e.CellPoint(image.Pt(30, 30))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(3, 0))
	if s := e.statusText(); !strings.Contains(s, "Blocked: Water, flag") {
		t.Fatalf("status %q", s)
	}
	pt = e.CellPoint(image.Pt(31, 30))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(4, 0))
	if s := e.statusText(); !strings.Contains(s, "Passable") {
		t.Fatalf("status %q", s)
	}

	v := e.doc.Viewer
	grey := color.RGBA{0x40, 0x40, 0x40, 0xff}
	canvas := image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH))
	for i := 0; i < len(canvas.Pix); i += 4 {
		copy(canvas.Pix[i:], []byte{grey.R, grey.G, grey.B, grey.A})
	}
	e.paintPassability(canvas)
	cx, cy := v.cellScreenCentre(image.Pt(30, 30))
	ox, oy := v.cellScreenCentre(image.Pt(31, 30))
	if got := canvas.RGBAAt(int(cx), int(cy)); got != blendOver(grey, passClosedFill) {
		t.Fatalf("the closed cell reads %v", got)
	}
	if got := canvas.RGBAAt(int(ox), int(oy)); got != blendOver(grey, passOpenFill) {
		t.Fatalf("the open cell reads %v", got)
	}
	e.view.passability = false
	before := append([]byte(nil), canvas.Pix...)
	e.paintPassability(canvas)
	if string(before) != string(canvas.Pix) {
		t.Fatal("the view off still fills")
	}
	if s := e.statusText(); strings.Contains(s, "Passable") {
		t.Fatal("the status line names passability with the view off")
	}
	_ = e.Key("passability")
	if len(e.passabilityQuads()) == 0 {
		t.Fatal("the passability brush does not show the fills")
	}
}

// The height brush names the hovered cell's height; other modes do not.
func TestMapEditorHeightBrushNamesTheCellHeight(t *testing.T) {
	e, _ := passEditor(t)
	alt := make([]uint8, e.doc.Width*e.doc.Height)
	alt[20*e.doc.Width+20] = 37
	e.doc.Viewer.grid.Altitudes = alt
	e.ShowCell(image.Pt(20, 20))
	pt := e.CellPoint(image.Pt(20, 20))
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(3, 0))
	if s := e.statusText(); strings.Contains(s, "Height -27") {
		t.Fatal("the height shows with no height brush")
	}
	_ = e.Key("height")
	e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(4, 0))
	if s := e.statusText(); !strings.Contains(s, "Height -27") {
		t.Fatalf("status %q", s)
	}
}

// The terrain list's rows carry the host's swatches as the picker's icon
// form and the list box draws them at the row's left; a row with no swatch
// is text alone.
func TestMapEditorTerrainListShowsSwatches(t *testing.T) {
	e, p := passEditor(t)
	_ = e.Key("terrain")
	rows, _ := e.listRows("brush")
	if len(rows) != 2 || rows[0].Icon != p.swatch || rows[1].Icon != nil {
		t.Fatalf("rows %+v", rows)
	}
	_ = e.Key("objects")
	if rows, _ := e.listRows("brush"); rows[0].Icon != nil {
		t.Fatal("an objects row carries a swatch")
	}
	_ = e.Key("terrain")
	pic := e.ChromeImage()
	r, ok := e.ListRowRect(0)
	if !ok {
		t.Fatal("the first row is not shown")
	}
	at := image.Pt(r.Min.X+listTextX+r.Dy()/2, r.Min.Y+r.Dy()/2)
	if got := pic.RGBAAt(at.X, at.Y); got != (color.RGBA{0xff, 0xff, 0xff, 0xff}) {
		t.Fatalf("the swatch pixel reads %v", got)
	}
	r1, _ := e.ListRowRect(1)
	at1 := image.Pt(r1.Min.X+listTextX+r1.Dy()/2, r1.Min.Y+r1.Dy()/2)
	if got := pic.RGBAAt(at1.X, at1.Y); got == (color.RGBA{0xff, 0xff, 0xff, 0xff}) {
		t.Fatal("a row with no swatch draws one")
	}
}
