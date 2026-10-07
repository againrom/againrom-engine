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

func click1093(e *MapEditor, x, y int) {
	e.Pointer(Input{CursorX: x, CursorY: y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: x, CursorY: y}, time.Time{})
}

func TestMapEditor1093TriggerFilterLateLinkAndContextAt640(t *testing.T) {
	e := editor1092(t)
	e.Layout(640, 480)
	r := InspectionRecord{Kind: "Trigger", Section: 7, Index: 0, Label: "Authored trigger"}
	for i := 0; i < 50; i++ {
		r.Lines = append(r.Lines, InspectionLine{Text: fmt.Sprintf("authored line %02d", i), Role: "condition"})
	}
	r.Lines = append(r.Lines, InspectionLine{Text: "late target link", Role: "action", Reference: 1})
	r.References = []InspectionReference{{Label: "target", Role: "action", Targets: []InspectionTarget{{Cell: image.Pt(35, 20), Size: image.Pt(1, 1)}}}}
	e.doc.Records = append(e.doc.Records, r)
	click1093(e, 320+250, 160) // actual eighth filter
	if e.filter != 7 || len(e.rows()) != 1 {
		t.Fatal("eighth filter did not expose trigger")
	}
	click1093(e, 340, editorListTop+4)
	selected := e.selected
	if selected != 40 {
		t.Fatalf("selected %d", selected)
	}
	beforeCamera := *e.doc.Viewer.cam
	for i := 0; i < 100; i++ {
		e.Pointer(Input{CursorX: 620, CursorY: 420, WheelY: -0.25}, time.Time{})
		e.PanelImage(480)
	}
	if !reflect.DeepEqual(beforeCamera, *e.doc.Viewer.cam) {
		t.Fatal("detail wheel touched camera")
	}
	rows := e.detailLayout(480)
	last := rows[len(rows)-1]
	if last.Reference != 1 || last.Text != "> late target link" {
		t.Fatalf("late link unreachable %+v", rows)
	}
	// Independent font drawing checks the exact pixels uploaded by Draw.
	want := image.NewRGBA(image.Rect(0, 0, 320, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 320; x++ {
			want.SetRGBA(x, y, editorPanel)
		}
	}
	e.font.Draw(want, "> late target link", 10, 0, editorGold)
	pic := e.PanelImage(480)
	for y := 0; y < 16; y++ {
		if !bytes.Equal(pic.Pix[pic.PixOffset(0, last.Rect.Min.Y+y):pic.PixOffset(0, last.Rect.Min.Y+y)+1280], want.Pix[y*1280:(y+1)*1280]) {
			t.Fatal("link text absent from actual rail pixels")
		}
	}
	beforeOffset, beforeList, beforeWheel := e.detailOffset, e.offset, e.detailWheel
	click1093(e, 340, last.Rect.Min.Y+2)
	if e.focusedReference != 1 || e.selected != selected || e.filter != 7 || e.detailOffset != beforeOffset || e.offset != beforeList || e.detailWheel != beforeWheel {
		t.Fatal("link lost selected trigger/scroll/filter")
	}
	if reflect.DeepEqual(beforeCamera, *e.doc.Viewer.cam) || len(e.referenceMarkerLines()) != 6 {
		t.Fatal("link failed to focus/mark map target")
	}
	wx, wy := e.cellCenter(image.Pt(35, 20))
	sx, sy := e.doc.Viewer.cam.WorldToScreen(wx, wy)
	if sx < 0 || sx >= 320 || sy < 0 || sy >= 480 {
		t.Fatal("focused target outside canvas")
	}
	if !bytes.Equal(e.doc.SourceBytes, []byte("untouched")) || e.doc.Viewer.AnimationCounter() != 0 {
		t.Fatal("inspector mutated source or ticked")
	}
	// Failed open retains all trigger context, not just the map pointer.
	if err := e.Open("bad"); err == nil || e.selected != selected || e.focusedReference != 1 || e.detailOffset != beforeOffset {
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
	e.pathMode = true
	e.focusedReference = 0
	for _, row := range e.detailLayout(480) {
		if row.Reference == 2 {
			click1093(e, 340, row.Rect.Min.Y+2)
		}
	}
	if e.focusedReference != 0 {
		t.Fatal("click leaked through path overlay")
	}
}

func TestMapEditor1093WrappedContinuationClickAfterResize(t *testing.T) {
	e := editor1092(t)
	r := &e.doc.Records[0]
	r.Lines = []InspectionLine{{Text: strings.Repeat("authored target reference ", 40), Role: "condition", Reference: 1}}
	r.References = []InspectionReference{{Label: "wrapped", Role: "condition", Targets: []InspectionTarget{{Cell: image.Pt(31, 27), Size: image.Pt(1, 1)}}}}
	e.Select(0, false)
	if e.font.Advance(r.Lines[0].Text) <= 300 {
		t.Fatal("fixture must wrap at the actual rail width")
	}
	e.detailOffset = 3 // the first visible line is a link continuation, not its start
	for _, size := range []image.Point{image.Pt(800, 600), image.Pt(640, 480)} {
		e.Layout(size.X, size.Y)
		e.PanelImage(size.Y)
		rows := e.detailLayout(size.Y)
		if len(rows) < 2 || rows[0].Reference != 1 || strings.HasPrefix(rows[0].Text, "> ") {
			t.Fatalf("not a wrapped continuation: %+v", rows)
		}
		before := e.detailOffset
		e.focusedReference = 0
		click1093(e, size.X-300, rows[0].Rect.Min.Y+2)
		if e.focusedReference != 1 || e.detailOffset != before || e.selected != 0 {
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
			e.detailOffset, e.detailWheel, e.listWheel = 2, 0.25, 0.5
			if !e.FocusReference(1) {
				t.Fatal("focus rejected")
			}
			check := func(x, y float64) {
				sx, sy := v.cam.WorldToScreen(x, y)
				if sx < 0 || sx >= 320 || sy < 0 || sy >= 480 {
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
					if p[0] < 0 || p[0] >= 320 || p[1] < 0 || p[1] >= 480 {
						t.Fatalf("actual marker outside focused canvas: %v", p)
					}
				}
			}
			if e.selected != 0 || e.filter != 7 || e.offset != 0 || e.detailOffset != 2 || e.detailWheel != 0.25 || e.listWheel != 0.5 || !bytes.Equal(e.doc.SourceBytes, []byte("unchanged")) {
				t.Fatal("focus changed inspection context/source")
			}
			e.Fit()
			if v.cam.Zoom != 320.0/(w*32) {
				t.Fatal("focus changed complete-map Fit")
			}
		})
	}
}
