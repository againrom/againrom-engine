package game

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"image"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The forest fixture's words and object codes: land everywhere, a water
// block, a road row, a 3x2 structure, standing objects of a kind that is
// never ticked and of a kind that is.
const (
	forestLand   = 0x14
	forestWater  = 0x211
	forestRoad   = 0x311
	forestOther  = 50 // a standing kind the mix never names
	forestOakish = 3  // the first ticked kind, also standing on the map
)

// forestCodes are the classes the fixture's object list offers, after the
// clear row; the test ticks the first three.
var forestCodes = []int{forestOakish, 7, 9, forestOther}

// forestMap is a 40x40 map with water in columns 14..16 of rows 12..28, a
// road along row 24, the structure at cell (20,20), forestOther on every
// fifth cell of row 18 and forestOakish on every third cell of row 22.
func forestMap(t *testing.T) []byte {
	t.Helper()
	tiles := make([]byte, 2*40*40)
	overlay := make([]byte, 40*40)
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			w := uint16(forestLand)
			switch {
			case x >= 14 && x <= 16 && y >= 12 && y <= 28:
				w = forestWater
			case y == 24:
				w = forestRoad
			}
			binary.LittleEndian.PutUint16(tiles[2*(y*40+x):], w)
		}
	}
	for x := 0; x < 40; x += 5 {
		overlay[18*40+x] = forestOther
	}
	for x := 1; x < 40; x += 3 {
		overlay[22*40+x] = forestOakish
	}
	return synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Type1Payload: tiles, Overlay: overlay, Type7Payload: make([]byte, 12),
		Objects: []synth.ALMObject{{X: 20*256 + 128, Y: 20*256 + 128, Kind: 5, Field12: 7}}})
}

// forestEditor opens raw in an editor whose object list offers forestCodes
// and whose table resolves the structure.
func forestEditor(t *testing.T, raw []byte) (*ui.MapEditor, *MapInspector) {
	t.Helper()
	table := authoringTable()
	building := func(w, h int32) dbEntry { return dbEntry{name: "b", params: []int32{w, h, 0, 0, -1, -1, 0, 0}} }
	table.Buildings = dbCollection{{}, building(1, 1), building(1, 1), building(1, 1), building(1, 1), building(3, 2)}
	set := &terrain.StructureSet{}
	set.Classes[5] = &terrain.StructureClass{ID: 5, Name: "s", TileWidth: 3, TileHeight: 2, FullHeight: 2}
	x := &MapInspector{tiles: &terrain.Tileset{}, table: table, structures: set, objectRows: []ui.BrushObject{{Code: 0}}}
	for _, c := range forestCodes {
		x.objectRows = append(x.objectRows, ui.BrushObject{Code: c})
	}
	x.statics = forestStatics(forestCodes...)
	e := x.Editor()
	openSelectionMap(t, e, "forest.alm", raw)
	return e, x
}

// forestStatics is a static set whose listed classes each have a one-pixel
// opaque frame.
func forestStatics(codes ...int) *terrain.StaticSet {
	set := &terrain.StaticSet{}
	for _, c := range codes {
		f := &terrain.StaticFrame{Width: 1, Height: 1, Pixels: []terrain.StaticPixel{{Index: 1, Opaque: true}}}
		set.Classes[c] = &terrain.StaticClass{Frame: f}
	}
	return set
}

// forestDriver drives the editor the way the pointer and the keys do.
type forestDriver struct {
	t   *testing.T
	e   *ui.MapEditor
	c   *editorClicker
	now time.Time
}

func newForestDriver(t *testing.T, raw []byte) (*forestDriver, *MapInspector) {
	e, x := forestEditor(t, raw)
	d := &forestDriver{t: t, e: e, c: &editorClicker{t: t, now: time.Unix(1, 0)}, now: time.Unix(1, 0)}
	d.key("objects")
	return d, x
}

func (d *forestDriver) key(k string) {
	d.t.Helper()
	if err := d.e.Key(k); err != nil {
		d.t.Fatalf("key %s: %v", k, err)
	}
}

// button presses button k of brush popup row row.
func (d *forestDriver) button(row, k int, what string) {
	d.t.Helper()
	r, ok := d.e.ListButtonRect(row, k)
	d.c.click(d.e, r, ok, what)
}

// forest switches the object brush to Forest, ticks the first n kinds of
// the grid and sets a square brush of radius r.
func (d *forestDriver) forest(n, r int) {
	d.t.Helper()
	d.button(2, 1, "the Forest button")
	for i := 0; i < n; i++ {
		rect, ok := d.e.PlaceGridCellRect(i)
		d.c.click(d.e, rect, ok, "a forest grid cell")
	}
	d.button(1, 1, "the Square button")
	for i := 0; i < 20; i++ {
		d.key("radiusdown")
	}
	for i := 0; i < r; i++ {
		d.key("radiusup")
	}
}

// stroke drags over cells with the left button, or the right one to thin.
// The view centres on the first cell; a cell under the brush popup would
// press the popup instead, so the test refuses it.
func (d *forestDriver) stroke(right bool, cells ...image.Point) {
	d.t.Helper()
	d.e.ShowCell(cells[0])
	pop, _ := d.e.PopupRect()
	for _, c := range cells {
		p := d.e.CellPoint(c)
		if p.In(pop) {
			d.t.Fatalf("cell %v is at %v, under the popup %v", c, p, pop)
		}
		d.e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: !right, SecondaryDown: right}, d.now)
	}
	p := d.e.CellPoint(cells[len(cells)-1])
	d.e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, d.now)
	d.now = d.now.Add(time.Second)
}

func (d *forestDriver) bytes() []byte { return bytes.Clone(d.e.Document().SourceBytes) }

func forestPlanes(t *testing.T, raw []byte) (tiles []uint16, overlay []byte) {
	t.Helper()
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m.Tiles, m.Overlay
}

// forestFenced is the fixture's structure footprint through the game's own
// reader.
func forestFenced(t *testing.T, raw []byte, x *MapInspector) []bool {
	t.Helper()
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	fenced := mapload.StructureCells(m, x.table)
	if fenced == nil {
		t.Fatal("the fixture's structure attaches no cell")
	}
	return fenced
}

// forestOpen is the test's own reading of a cell a forest may plant on.
func forestOpen(i int, tiles []uint16, overlay []byte, fenced []bool) bool {
	return overlay[i] == 0 && tiles[i] != forestWater && tiles[i] != forestRoad && !fenced[i]
}

// checkPlanting checks one planting stroke from before to after: every
// changed cell lies in footprint, was open and took a ticked kind; it
// returns how many open footprint cells there were and how many of each
// kind were planted.
func checkPlanting(t *testing.T, before, after []byte, fenced []bool, footprint []image.Point, ticked []int) (open int, planted map[int]int) {
	t.Helper()
	tiles, was := forestPlanes(t, before)
	_, now := forestPlanes(t, after)
	in := map[int]bool{}
	for _, c := range footprint {
		i := c.Y*40 + c.X
		in[i] = true
		if forestOpen(i, tiles, was, fenced) {
			open++
		}
	}
	planted = map[int]int{}
	kinds := map[int]bool{}
	for _, k := range ticked {
		kinds[k] = true
	}
	for i := range now {
		if now[i] == was[i] {
			continue
		}
		switch {
		case !in[i]:
			t.Fatalf("cell %d outside the brush changed", i)
		case was[i] != 0:
			t.Fatalf("standing object %d at cell (%d,%d) became %d", was[i], i%40, i/40, now[i])
		case tiles[i] == forestWater:
			t.Fatalf("a tree was planted on water at (%d,%d)", i%40, i/40)
		case tiles[i] == forestRoad:
			t.Fatalf("a tree was planted on the road at (%d,%d)", i%40, i/40)
		case fenced[i]:
			t.Fatalf("a tree was planted on the structure at (%d,%d)", i%40, i/40)
		case !kinds[int(now[i])]:
			t.Fatalf("cell (%d,%d) took %d, which is not ticked", i%40, i/40, now[i])
		}
		planted[int(now[i])]++
	}
	return open, planted
}

// forestSeed fixes the stroke source for the test.
func forestSeed(t *testing.T, seed uint64) {
	old := strokeSeed
	t.Cleanup(func() { strokeSeed = old })
	strokeSeed = func() uint64 { return seed }
}

// Each density preset and a typed per cent plant that share of the open
// cells under the brush, within five points on a fixed seed, with the three
// ticked kinds each near a third; nothing lands on water, the road, the
// structure or a standing object; one Undo restores the bytes.
func TestMapEditorForestPlantsTheShareOnOpenCellsOnly(t *testing.T) {
	forestSeed(t, 11)
	raw := forestMap(t)
	d, x := newForestDriver(t, raw)
	fenced := forestFenced(t, raw, x)
	d.forest(3, 16)
	ticked := forestCodes[:3]
	centre := image.Pt(20, 20)
	footprint := objectEdits(t, d.e).BrushCells(ui.BrushStroke{Radius: 16, Shape: ui.BrushSquare, Cells: []image.Point{centre}})
	for _, tc := range []struct {
		name  string
		share int
		set   func()
	}{
		{"sparse", 10, func() { d.button(4, 0, "Sparse") }},
		{"medium", 25, func() { d.button(4, 1, "Medium") }},
		{"dense", 50, func() { d.button(4, 2, "Dense") }},
		{"solid", 100, func() { d.button(4, 3, "Solid") }},
		{"typed 37", 37, func() {
			d.button(3, 0, "the per cent button")
			d.key("clear")
			d.e.TextInput("37")
			d.key("enter")
		}},
	} {
		tc.set()
		before := d.bytes()
		d.stroke(false, centre)
		after := d.bytes()
		open, planted := checkPlanting(t, before, after, fenced, footprint, ticked)
		total := 0
		for _, n := range planted {
			total += n
		}
		got := 100 * float64(total) / float64(open)
		t.Logf("%s: %d of %d open cells planted (%.1f%%), kinds %v", tc.name, total, open, got, planted)
		if got < float64(tc.share)-5 || got > float64(tc.share)+5 {
			t.Errorf("%s: planted %.1f%% of the open cells, want %d%% within 5 points", tc.name, got, tc.share)
		}
		if tc.share == 100 && total != open {
			t.Errorf("solid: planted %d of %d open cells", total, open)
		}
		for _, k := range ticked {
			if share := float64(planted[k]) / float64(total); share < 1.0/3-0.1 || share > 1.0/3+0.1 {
				t.Errorf("%s: kind %d is %.2f of the plants, want a third within 0.1", tc.name, k, share)
			}
		}
		d.key("undo")
		if !bytes.Equal(d.bytes(), before) {
			t.Fatalf("%s: one Undo did not restore the bytes from before the stroke", tc.name)
		}
		if !bytes.Equal(before, raw) {
			t.Fatalf("%s: the stroke did not start from the opened map", tc.name)
		}
	}
}

// A dragged stroke is one edit: one Undo restores the bytes from before it,
// Redo the planted bytes.
func TestMapEditorForestDragIsOneUndoStep(t *testing.T) {
	forestSeed(t, 3)
	raw := forestMap(t)
	d, x := newForestDriver(t, raw)
	fenced := forestFenced(t, raw, x)
	d.forest(2, 2)
	d.button(4, 2, "Dense")
	path := []image.Point{{X: 24, Y: 16}, {X: 26, Y: 14}, {X: 30, Y: 20}, {X: 28, Y: 25}, {X: 32, Y: 26}}
	d.stroke(false, path...)
	planted := d.bytes()
	footprint := objectEdits(t, d.e).BrushCells(ui.BrushStroke{Radius: 2, Shape: ui.BrushSquare, Cells: dragCells(path)})
	if _, n := checkPlanting(t, raw, planted, fenced, footprint, forestCodes[:2]); len(n) == 0 {
		t.Fatal("the drag planted nothing")
	}
	d.key("undo")
	if !bytes.Equal(d.bytes(), raw) {
		t.Fatal("one Undo after a dragged forest stroke did not restore the bytes")
	}
	d.key("redo")
	if !bytes.Equal(d.bytes(), planted) {
		t.Fatal("Redo did not restore the planted bytes")
	}
}

// dragCells is the line of cells a drag through path visits, rounded to the
// nearest cell as the window joins them, so the footprint check covers the
// whole drag.
func dragCells(path []image.Point) []image.Point {
	out := []image.Point{path[0]}
	for _, c := range path[1:] {
		last := out[len(out)-1]
		dx, dy := c.X-last.X, c.Y-last.Y
		n := max(dx, -dx, dy, -dy)
		for i := 1; i <= n; i++ {
			out = append(out, image.Pt(last.X+(dx*i+cmp.Compare(dx, 0)*n/2)/n, last.Y+(dy*i+cmp.Compare(dy, 0)*n/2)/n))
		}
	}
	return out
}

// The right button thins: it removes about the share of the ticked kind
// standing under the brush and leaves every other object; one Undo restores
// the bytes.
func TestMapEditorForestRightButtonThinsTickedKindsOnly(t *testing.T) {
	forestSeed(t, 5)
	raw := forestMap(t)
	d, _ := newForestDriver(t, raw)
	d.forest(1, 16) // forestOakish alone
	d.button(4, 3, "Solid")
	centre := image.Pt(20, 12)
	d.stroke(false, centre)
	planted := d.bytes()
	d.button(4, 2, "Dense")
	d.stroke(true, centre)
	thinned := d.bytes()
	_, was := forestPlanes(t, planted)
	_, now := forestPlanes(t, thinned)
	footprint := map[int]bool{}
	for _, c := range objectEdits(t, d.e).BrushCells(ui.BrushStroke{Radius: 16, Shape: ui.BrushSquare, Cells: []image.Point{centre}}) {
		footprint[c.Y*40+c.X] = true
	}
	ticked, removed := 0, 0
	for i := range now {
		if footprint[i] && was[i] == forestOakish {
			ticked++
		}
		if now[i] == was[i] {
			continue
		}
		if now[i] != 0 || was[i] != forestOakish || !footprint[i] {
			t.Fatalf("thinning changed cell (%d,%d) from %d to %d", i%40, i/40, was[i], now[i])
		}
		removed++
	}
	got := 100 * float64(removed) / float64(ticked)
	t.Logf("thinning removed %d of %d ticked objects (%.1f%%)", removed, ticked, got)
	if got < 45 || got > 55 {
		t.Errorf("thinning removed %.1f%%, want 50%% within 5 points", got)
	}
	_, orig := forestPlanes(t, raw)
	for i := range orig {
		if orig[i] == forestOther && now[i] != forestOther {
			t.Fatalf("thinning removed the unticked object at (%d,%d)", i%40, i/40)
		}
	}
	d.key("undo")
	if !bytes.Equal(d.bytes(), planted) {
		t.Fatal("one Undo after thinning did not restore the bytes")
	}
}

// With no kind ticked a stroke writes nothing and says why; a typed per cent
// outside 1..100 keeps the share and says why.
func TestMapEditorForestRefusesNoKindAndABadShare(t *testing.T) {
	raw := forestMap(t)
	d, _ := newForestDriver(t, raw)
	d.forest(0, 2)
	d.stroke(false, image.Pt(24, 20))
	if !bytes.Equal(d.bytes(), raw) || d.e.Message() == "" {
		t.Fatalf("a stroke with no kind ticked wrote bytes or said nothing (%q)", d.e.Message())
	}
	d.button(3, 0, "the per cent button")
	d.key("clear")
	d.e.TextInput("0")
	d.key("enter")
	if d.e.Message() == "" {
		t.Fatal("a typed 0 per cent was taken without a word")
	}
}
