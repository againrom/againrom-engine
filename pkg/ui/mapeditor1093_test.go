package ui

import (
	"bytes"
	"fmt"
	"image"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// objectRowOf is the spec row whose text is label, and its link.
func objectRowOf(t *testing.T, e *MapEditor, label string) (int, int) {
	t.Helper()
	s, links, ok := e.objectSpec()
	if !ok {
		t.Fatal("no object popup")
	}
	for n, r := range s.Rows {
		if r.Label == label {
			return n, links[n]
		}
	}
	t.Fatalf("no object row %q", label)
	return 0, 0
}

func TestMapEditor1093TriggerListLateLinkOpensItsTargetAt640(t *testing.T) {
	e := editor1092(t)
	e.Layout(640, 480)
	r := InspectionRecord{Kind: "Trigger", Section: 7, Index: 0, Label: "Authored trigger"}
	for i := 0; i < 50; i++ {
		r.Lines = append(r.Lines, InspectionLine{Text: fmt.Sprintf("authored line %02d", i), Role: "condition"})
	}
	r.Lines = append(r.Lines, InspectionLine{Text: "late target link", Role: "action", Reference: 1})
	r.References = []InspectionReference{{Label: "target", Role: "action", Targets: []InspectionTarget{{Cell: image.Pt(35, 20), Size: image.Pt(1, 1)}}}}
	e.doc.Records = append(e.doc.Records, r)
	chromeClick(e, barPoint(t, e, "scripts"))
	if e.filter != 7 || len(e.rows()) != 1 {
		t.Fatal("the Scripts entry did not list the trigger")
	}
	chromeClick(e, listRowPoint(t, e, 1))
	selected := e.selected
	if selected != 40 {
		t.Fatalf("selected %d", selected)
	}
	beforeCamera := *e.doc.Viewer.cam
	_, l := objectLayout(t, e)
	at := center(l.Rows[0].Rect)
	for i := 0; i < 400; i++ {
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -0.25}, time.Time{})
	}
	if !reflect.DeepEqual(beforeCamera, *e.doc.Viewer.cam) {
		t.Fatal("detail wheel touched camera")
	}
	n, link := objectRowOf(t, e, "> late target link")
	if link != 1 {
		t.Fatalf("late link row %d carries link %d", n, link)
	}
	row := objectRowRect(t, e, n)
	if !row.Rect.In(image.Rect(0, 0, 640, 480)) {
		t.Fatal("late link outside the window")
	}
	chromeClick(e, center(row.Rect))
	if e.selected != 34 || e.doc.Records[e.selected].Cell != image.Pt(35, 20) {
		t.Fatalf("the link did not select its target: %d", e.selected)
	}
	if _, _, ok := e.objectSpec(); !ok {
		t.Fatal("the link did not open its target's popup")
	}
	wx, wy := e.cellCenter(image.Pt(35, 20))
	sx, sy := e.doc.Viewer.cam.WorldToScreen(wx, wy)
	if reflect.DeepEqual(beforeCamera, *e.doc.Viewer.cam) || sx < 0 || sx >= 640 || sy < 0 || sy >= float64(e.doc.Viewer.cam.ViewH) {
		t.Fatal("focused target outside canvas")
	}
	if !bytes.Equal(e.doc.SourceBytes, []byte("untouched")) || e.doc.Viewer.AnimationCounter() != 0 {
		t.Fatal("inspector mutated source or ticked")
	}
	// A failed open retains the selection and its popup.
	if err := e.Open("bad"); err == nil || e.selected != 34 {
		t.Fatal("failed open destroyed context")
	}
}

func TestMapEditor1093AmbiguousLinksModalAndBoundedGeometry(t *testing.T) {
	e := editor1092(t)
	e.Layout(640, 480)
	r := &e.doc.Records[0]
	r.References = []InspectionReference{{Label: "unresolved"}, {Role: "condition", Targets: []InspectionTarget{{Cell: image.Pt(-100, -100), Size: image.Pt(100000, 100000)}}}}
	r.Lines = []InspectionLine{{Text: "not a link", Role: "warning"}, {Text: "large bounded reference", Role: "condition", Reference: 2}}
	e.Select(0, false)
	if e.FocusReference(0) || e.FocusReference(1) || e.FocusReference(3) {
		t.Fatal("invalid/unresolved link moved view")
	}
	if !e.FocusReference(2) || len(e.referenceMarkerLines()) != 6 {
		t.Fatal("large region must use six bounded primitives")
	}
	n, _ := objectRowOf(t, e, "> large bounded reference")
	at := center(objectRowRect(t, e, n).Rect)
	e.prompt.kind = promptOpen
	e.focusedReference = 0
	chromeClick(e, at)
	if e.focusedReference != 0 {
		t.Fatal("click leaked through the prompt")
	}
	e.prompt.kind = promptNone
	chromeClick(e, at)
	if e.focusedReference != 2 || e.selected != 0 {
		t.Fatal("a many-cell link selected a record or lost its focus")
	}
}

func TestMapEditor1093WrappedContinuationClickAfterResize(t *testing.T) {
	e := editor1092(t)
	r := &e.doc.Records[0]
	r.Lines = []InspectionLine{{Text: strings.Repeat("authored target reference ", 400), Role: "condition", Reference: 1}}
	r.References = []InspectionReference{{Label: "wrapped", Role: "condition", Targets: []InspectionTarget{{Cell: image.Pt(31, 27), Size: image.Pt(1, 1)}}}}
	e.Select(0, false)
	if e.font.Advance(r.Lines[0].Text) <= 420 {
		t.Fatal("fixture must wrap at the popup width")
	}
	for _, size := range []image.Point{image.Pt(800, 600), image.Pt(640, 480)} {
		e.Layout(size.X, size.Y)
		e.objPlaced = false
		s, links, _ := e.objectSpec()
		first := -1
		for n := range s.Rows {
			if links[n] == 1 && first < 0 {
				first = n
			}
		}
		e.objectTop = first + 1 // the first shown row is a link continuation
		s, links, _ = e.objectSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		top := l.Rows[0]
		if links[top.Index] != 1 || strings.HasPrefix(s.Rows[top.Index].Label, "> ") {
			t.Fatalf("not a wrapped continuation: %+v", s.Rows[top.Index])
		}
		before := e.objectTop
		e.focusedReference = 0
		chromeClick(e, center(top.Rect))
		if e.focusedReference != 1 || e.objectTop != before || e.selected != 0 {
			t.Fatal("continuation click lost link identity or scroll after resize")
		}
	}
}

func TestMapEditor1093FocusFitsProjectedFootprintsAndAnchors(t *testing.T) {
	const w, h = 128, 128
	alt := make([]uint8, w*h)
	// Non-diagonal group members and an interior anchor-cell corner have
	// different authored heights from the union's two diagonal extrema.
	for _, p := range []image.Point{{10, 12}, {11, 13}, {11, 26}, {12, 27}, {19, 20}} {
		alt[p.Y*w+p.X] = 127
	}
	alt[20*w+20] = 128 // signed -128 at a multi-cell footprint's anchor cell
	for _, tc := range []struct {
		name    string
		targets []InspectionTarget
	}{
		{"group", []InspectionTarget{{Cell: image.Pt(10, 12), Size: image.Pt(1, 1)}, {Cell: image.Pt(10, 26), Size: image.Pt(1, 1)}, {Cell: image.Pt(11, 12), Size: image.Pt(1, 1)}, {Cell: image.Pt(11, 26), Size: image.Pt(1, 1)}}},
		{"multi-cell", []InspectionTarget{{Cell: image.Pt(19, 20), Size: image.Pt(6, 9)}}},
		{"whole-map", []InspectionTarget{{Cell: image.Pt(0, 0), Size: image.Pt(100000, 100000)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("projected reference", terrain.Grid{Width: w, Height: h, Tiles: make([]uint16, w*h), Altitudes: alt}, &terrain.Tileset{})
			if err != nil {
				t.Fatal(err)
			}
			r := InspectionRecord{Kind: "Trigger", References: []InspectionReference{{Role: "condition", Targets: tc.targets}}}
			for i := 0; i < 40; i++ {
				r.Lines = append(r.Lines, InspectionLine{Text: fmt.Sprintf("raw detail %d", i)})
			}
			e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
				return &InspectionDocument{Viewer: v, Width: w, Height: h, Records: []InspectionRecord{r}, SourceBytes: []byte("unchanged")}, nil
			})
			e.Layout(640, 480)
			if err := e.Open("raw heights"); err != nil {
				t.Fatal(err)
			}
			if v.Mode() != ModeDisplaced {
				t.Fatal("fixture must exercise projected terrain")
			}
			_ = e.Key("triggers")
			e.Select(0, false)
			e.objectTop, e.objectWheel, e.listWheel = 2, 0.25, 0.5
			if !e.FocusReference(1) {
				t.Fatal("focus rejected")
			}
			check := func(x, y float64) {
				sx, sy := v.cam.WorldToScreen(x, y)
				if sx < 0 || sx >= float64(v.cam.ViewW) || sy < 0 || sy >= float64(v.cam.ViewH) {
					t.Errorf("raw projected point %g,%g outside: %g,%g", x, y, sx, sy)
				}
			}
			// Expected world points use only raw height bytes and fixed 32px
			// cells. The untouched top row makes the raw world origin Y=0.
			for _, target := range tc.targets {
				box := image.Rectangle{Min: target.Cell, Max: target.Cell.Add(target.Size)}.Intersect(image.Rect(0, 0, w, h))
				for _, p := range []image.Point{box.Min, {X: box.Max.X, Y: box.Min.Y}, box.Max, {X: box.Min.X, Y: box.Max.Y}} {
					height := int(int8(alt[min(p.Y, h-1)*w+min(p.X, w-1)]))
					check(float64(p.X*32), float64(p.Y*32-height))
				}
				p := target.Cell
				hh := (int(int8(alt[p.Y*w+p.X])) + int(int8(alt[p.Y*w+p.X+1])) + int(int8(alt[(p.Y+1)*w+p.X])) + int(int8(alt[(p.Y+1)*w+p.X+1]))) / 4
				check(float64(p.X*32+16), float64(p.Y*32+16-hh))
			}
			lines := e.referenceMarkerLines()
			if len(lines) != 6*len(tc.targets) {
				t.Fatal("reference geometry expanded per cell")
			}
			for _, line := range lines {
				for _, p := range [][2]float32{{line.x0, line.y0}, {line.x1, line.y1}} {
					if p[0] < 0 || p[0] >= float32(v.cam.ViewW) || p[1] < 0 || p[1] >= float32(v.cam.ViewH) {
						t.Fatalf("actual marker outside focused canvas: %v", p)
					}
				}
			}
			if e.selected != 0 || e.filter != 7 || e.objectTop != 2 || e.objectWheel != 0.25 || e.listWheel != 0.5 || !bytes.Equal(e.doc.SourceBytes, []byte("unchanged")) {
				t.Fatal("focus changed inspection context/source")
			}
			e.Fit()
			if v.cam.Zoom != float64(min(v.cam.ViewW, v.cam.ViewH))/(w*32) {
				t.Fatal("focus changed complete-map Fit")
			}
		})
	}
}
