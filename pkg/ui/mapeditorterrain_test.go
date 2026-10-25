package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// terrainHost is a full host that also takes the whole-map commands and the
// area copy. Its height plane is alt; RandomHeights sets every height to the
// seed modulo 128, so a test can name what the numbers then show.
type terrainHost struct {
	modesFull
	alt []uint8
}

func (p *terrainHost) RandomTerrain(seed uint32) (*InspectionDocument, error) {
	return p.next("RandomTerrain %d", seed)
}

func (p *terrainHost) RandomHeights(seed uint32) (*InspectionDocument, error) {
	for i := range p.alt {
		p.alt[i] = uint8(seed % 128)
	}
	return p.next("RandomHeights %d", seed)
}

// areaClip is the terrain host's area copy.
type areaClip struct {
	refs []RecordRef
	box  image.Rectangle
}

func (c areaClip) Len() int { return len(c.refs) + 1 }

func (p *terrainHost) CopyArea(refs []RecordRef, box image.Rectangle) (InspectionClip, string, error) {
	p.calls.add("CopyArea %v %v", refs, box)
	return areaClip{refs, box}, "area copied", nil
}

const terrainSide = 50

// terrainEditor opens a 50 x 50 map whose height at cell (x, y) is (3x+y)
// modulo 128 in a 1280 x 800 window, and returns the sources it was asked to
// open.
func terrainEditor(t *testing.T) (*MapEditor, *terrainHost, *[]string) {
	t.Helper()
	h := &terrainHost{alt: make([]uint8, terrainSide*terrainSide)}
	for i := range h.alt {
		h.alt[i] = uint8((3*(i%terrainSide) + i/terrainSide) % 128)
	}
	h.calls, h.self = &modesCalls{}, h
	h.doc = func(edits InspectionEdits) *InspectionDocument {
		g := terrain.Grid{Width: terrainSide, Height: terrainSide, Tiles: make([]uint16, terrainSide*terrainSide), Altitudes: slices.Clone(h.alt)}
		v, err := NewViewer("terrain", g, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		d := &InspectionDocument{Viewer: v, Source: "scenario/10.alm", Width: terrainSide, Height: terrainSide, SourceBytes: []byte("map"), Edits: edits}
		d.Records = []InspectionRecord{
			{Kind: "Unit", Index: 0, Section: 6, Cell: image.Pt(10, 10), Spatial: true, Size: image.Pt(1, 1)},
			{Kind: "Unit", Index: 1, Section: 6, Cell: image.Pt(30, 30), Spatial: true, Size: image.Pt(1, 1)},
		}
		return d
	}
	var opened []string
	open := func(source string) (*InspectionDocument, error) {
		opened = append(opened, source)
		return h.doc(h), nil
	}
	e := NewMapEditor(panelFont(), nil, open)
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	return e, h, &opened
}

// Ctrl+E and Ctrl+Shift+R, and their Tools rows, ask for a seed holding the
// one after the last used, 1 at first; Enter runs the command with the typed
// seed and the status line names it; a seed that is not a whole number from
// 0 to 4294967295 runs nothing and says why.
func TestRandomCommandsTakeTheTypedSeed(t *testing.T) {
	e, h, _ := terrainEditor(t)
	run := func(key, typed string) {
		t.Helper()
		if err := e.Key(key); err != nil {
			t.Fatal(err)
		}
		if e.prompt.kind != promptSeed {
			t.Fatalf("%s opened prompt %d", key, e.prompt.kind)
		}
		if typed != "" {
			_ = e.Key("clear")
			e.TextInput(typed)
		}
		if err := e.Key("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if err := e.Key("randomterrain"); err != nil || e.prompt.path != "1" || e.promptTitle() != e.word("editor.prompt.randomterrain") {
		t.Fatalf("the first seed prompt holds %q titled %q (%v)", e.prompt.path, e.promptTitle(), err)
	}
	_ = e.Key("enter")
	run("randomterrain", "4294967295")
	run("randomheights", "")
	run("randomheights", "77")
	want := []string{"RandomTerrain 1", "RandomTerrain 4294967295", "RandomHeights 0", "RandomHeights 77"}
	if !slices.Equal(h.calls.lines, want) {
		t.Fatalf("calls %q, want %q", h.calls.lines, want)
	}
	if s := e.statusText(); !strings.Contains(s, fmt.Sprintf(e.word("editor.random.heightsdone"), 77)) {
		t.Fatalf("status %q does not name seed 77", s)
	}
	for _, bad := range []string{"-1", "4294967296", "seed"} {
		run("randomterrain", bad)
		if !strings.Contains(e.message, bad) {
			t.Fatalf("seed %q: message %q", bad, e.message)
		}
		e.message = ""
	}
	if len(h.calls.lines) != len(want) {
		t.Fatalf("a refused seed ran a command: %q", h.calls.lines)
	}
	var keys []string
	for _, it := range e.menuItems("tools") {
		keys = append(keys, it.key)
	}
	if !slices.Contains(keys, "randomterrain") || !slices.Contains(keys, "randomheights") {
		t.Fatalf("the Tools menu holds %q", keys)
	}
	e.menuAct("randomheights")
	if e.prompt.kind != promptSeed || e.prompt.scope != "randomheights" || e.prompt.path != "78" {
		t.Fatalf("the Tools row opened prompt %d %q %q", e.prompt.kind, e.prompt.scope, e.prompt.path)
	}
}

// boxDrag drags a plain selection box on the map from cell a to cell b.
func boxDrag(e *MapEditor, a, b image.Point) {
	now := time.Unix(1, 0)
	pa, pb := e.CellPoint(a), e.CellPoint(b)
	e.Pointer(Input{CursorX: pa.X, CursorY: pa.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: pb.X, CursorY: pb.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: pb.X, CursorY: pb.Y}, now)
}

// Ctrl+C after a selection box copies the box's terrain with its records,
// even when the box holds none, and the box stays outlined until the
// selection changes; with no box Ctrl+C copies the records alone.
func TestBoxCopyTakesTheTerrainUnderTheBox(t *testing.T) {
	e, h, _ := terrainEditor(t)
	e.ShowCell(image.Pt(10, 10))
	boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
	if e.selection.box != image.Rect(8, 8, 13, 12) || len(e.marqueeLines()) == 0 {
		t.Fatalf("the box is %v with %d outline lines", e.selection.box, len(e.marqueeLines()))
	}
	_ = e.Key("copy")
	boxDrag(e, image.Pt(14, 14), image.Pt(15, 15))
	_ = e.Key("copy")
	want := []string{"CopyArea [{Unit 0}] (8,8)-(13,12)", "CopyArea [] (14,14)-(16,16)"}
	if !slices.Equal(h.calls.lines, want) {
		t.Fatalf("calls %q, want %q", h.calls.lines, want)
	}
	e.hoverCell, e.hoverOK = image.Pt(40, 40), true
	_ = e.Key("paste")
	if last := h.calls.lines[len(h.calls.lines)-1]; !strings.HasPrefix(last, "PasteRecords") {
		t.Fatalf("an area clip with no records did not paste: %q", h.calls.lines)
	}
	e.clearSelection()
	if !e.selection.box.Empty() || len(e.marqueeLines()) != 0 {
		t.Fatal("clearing the selection kept the box")
	}
	e.setSelection([]RecordRef{{"Unit", 1}})
	_ = e.Key("copy")
	if last := h.calls.lines[len(h.calls.lines)-1]; last != "CopyRecords [{Unit 1}]" {
		t.Fatalf("a copy with no box called %q", last)
	}
}

// heightAt is the number the cell corner at window point p shows: the
// stored height of the vertex whose corner is p, minus 64.
func heightAt(e *MapEditor, alt []uint8, p image.Point) (string, bool) {
	ox, oy := e.doc.Viewer.place.Origin()
	for r := range terrainSide {
		for c := range terrainSide {
			x, y := e.corner(image.Pt(c, r))
			if image.Pt(int(float64(x)+ox)+1, int(float64(y)+oy)+1) == p {
				return strconv.Itoa(int(alt[r*terrainSide+c]) - 64), true
			}
		}
	}
	return "", false
}

// The height numbers are off by default; V turns them on, and each stands at
// its vertex's corner showing the stored height minus 64; zoomed out until a
// cell is narrower than a number, none is drawn; after a height edit they
// show the new heights; V turns them off.
func TestHeightNumbersShowTheVertexHeightMinus64(t *testing.T) {
	e, h, _ := terrainEditor(t)
	if rects, _ := e.HeightLabels(); len(rects) != 0 {
		t.Fatalf("%d numbers with the switch off", len(rects))
	}
	_ = e.Key("heightlabels")
	e.doc.Viewer.cam.Zoom = 2
	e.doc.Viewer.cam.Clamp()
	rects, texts := e.HeightLabels()
	if len(rects) < 50 {
		t.Fatalf("%d numbers at zoom 2", len(rects))
	}
	seen := map[string]bool{}
	for i, r := range rects {
		want, ok := heightAt(e, h.alt, r.Min)
		if !ok || texts[i] != want {
			t.Fatalf("number %d at %v shows %q; the vertex there shows %q (found %v)", i, r.Min, texts[i], want, ok)
		}
		seen[texts[i]] = true
	}
	if !seen["-64"] && !seen["-1"] && len(seen) < 10 {
		t.Fatalf("the numbers show %d values", len(seen))
	}
	e.doc.Viewer.cam.Zoom = 0.25
	rects, texts = e.HeightLabels()
	if len(rects) == 0 {
		t.Fatal("no numbers zoomed out")
	}
	k := vertexSpacing(t, e, rects)
	if k < 2 || k&(k-1) != 0 {
		t.Fatalf("zoomed out the numbers stand %d vertices apart, want a power of two above 1", k)
	}
	for i, r := range rects {
		if want, ok := heightAt(e, h.alt, r.Min); !ok || texts[i] != want {
			t.Fatalf("zoomed out, number %d shows %q; its vertex shows %q", i, texts[i], want)
		}
	}
	e.doc.Viewer.cam.Zoom = 2
	_ = e.Key("randomheights")
	_ = e.Key("clear")
	e.TextInput("90")
	_ = e.Key("enter")
	e.doc.Viewer.cam.Zoom = 2
	e.doc.Viewer.cam.Clamp()
	_, texts = e.HeightLabels()
	if len(texts) == 0 {
		t.Fatal("no numbers after the height edit")
	}
	for _, s := range texts {
		if s != "26" {
			t.Fatalf("after setting every height to 90 a number shows %q", s)
		}
	}
	e.menuAct("heightlabels")
	if rects, _ := e.HeightLabels(); len(rects) != 0 {
		t.Fatal("the View row did not turn the numbers off")
	}
}

// The new-map question holds the starting height 64; Enter and the mode
// buttons open the new map at the typed height, the default leaving the
// source as it was; a height outside 0..126 keeps the question open and
// says why.
func TestNewMapQuestionTakesTheStartingHeight(t *testing.T) {
	e, _, opened := terrainEditor(t)
	ask := func(typed string) {
		t.Helper()
		e.askNewMode("new/80x80")
		if e.prompt.kind != promptNewMode || e.prompt.path != "64" {
			t.Fatalf("the question holds %q", e.prompt.path)
		}
		if typed != "" {
			_ = e.Key("clear")
			e.TextInput(typed)
		}
	}
	ask("")
	_ = e.Key("enter")
	ask("0")
	_ = e.newModeOpen(true)
	ask("126")
	_ = e.Key("enter")
	want := []string{"scenario/10.alm", "new/80x80", "new/80x80/altitude0/multiplayer", "new/80x80/altitude126"}
	if !slices.Equal(*opened, want) {
		t.Fatalf("opened %q, want %q", *opened, want)
	}
	for _, bad := range []string{"127", "-1", "high"} {
		ask(bad)
		_ = e.Key("enter")
		if e.prompt.kind != promptNewMode || !strings.Contains(e.message, bad) {
			t.Fatalf("height %q: prompt %d message %q", bad, e.prompt.kind, e.message)
		}
		s, _ := e.promptSpec()
		if !slices.Contains(s.Notes, e.message) {
			t.Fatalf("height %q: the question does not show %q", bad, e.message)
		}
		e.cancelPopup()
		e.message = ""
	}
	if len(*opened) != len(want) {
		t.Fatalf("a refused height opened a map: %q", *opened)
	}
	ask("40")
	s, _ := e.promptSpec()
	if len(s.Rows) != 1 || s.Rows[0].Value != "40" || s.Rows[0].Label != e.word("editor.newmap.height") {
		t.Fatalf("the question's rows %+v", s.Rows)
	}
}

// A record picked by a click or from the records list after a selection box
// is the selection now, not the box: Ctrl+C copies that record as at base,
// with no terrain; a box that is still the selection keeps copying its terrain.
func TestRecordPickedAfterABoxCopiesAsRecords(t *testing.T) {
	pick := map[string]func(e *MapEditor){
		"click": func(e *MapEditor) {
			e.ShowCell(image.Pt(30, 30))
			now := time.Unix(2, 0)
			p := e.CellPoint(image.Pt(30, 30))
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, now)
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, now)
		},
		"list": func(e *MapEditor) {
			if !e.Select(1, true) {
				t.Fatal("the list pick selected nothing")
			}
		},
	}
	for name, do := range pick {
		t.Run(name, func(t *testing.T) {
			e, h, _ := terrainEditor(t)
			e.ShowCell(image.Pt(10, 10))
			boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
			if e.selection.box.Empty() {
				t.Fatal("the box drag made no box")
			}
			do(e)
			if !e.selection.box.Empty() || len(e.marqueeLines()) != 0 {
				t.Fatalf("the box %v outlived the pick", e.selection.box)
			}
			_ = e.Key("copy")
			if last := h.calls.lines[len(h.calls.lines)-1]; last != "CopyRecords [{Unit 1}]" {
				t.Fatalf("copy after the pick called %q", last)
			}
		})
	}
}

// vertexSpacing is the largest power of two that divides the column and the
// row of every vertex a number in rects stands at, and fails when the numbers
// do not stand on every such vertex of the drawn area's rows and columns.
func vertexSpacing(t *testing.T, e *MapEditor, rects []image.Rectangle) int {
	t.Helper()
	ox, oy := e.doc.Viewer.place.Origin()
	at := map[image.Point]bool{}
	for _, r := range rects {
		at[r.Min] = true
	}
	k := 1 << 12
	for r := range terrainSide {
		for c := range terrainSide {
			x, y := e.corner(image.Pt(c, r))
			if !at[image.Pt(int(float64(x)+ox)+1, int(float64(y)+oy)+1)] {
				continue
			}
			for c%k != 0 || r%k != 0 {
				k /= 2
			}
		}
	}
	return k
}
