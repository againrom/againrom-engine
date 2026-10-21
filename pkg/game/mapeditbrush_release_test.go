package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// openBrushEditor opens key in a 1280x800 editor window with the Map panel
// closed.
func openBrushEditor(t *testing.T, x *MapInspector, key string) *ui.MapEditor {
	t.Helper()
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(key); err != nil {
		t.Fatal(err)
	}
	if e.PanelOpen() {
		_ = e.Key("escape")
	}
	return e
}

// shownMap is the map the window's document holds.
func shownMap(t *testing.T, e *ui.MapEditor) *alm.Map {
	t.Helper()
	m, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// pressRect presses and releases the primary button in the middle of r.
func pressRect(t *testing.T, e *ui.MapEditor, r image.Rectangle, ok bool, what string) {
	t.Helper()
	if !ok {
		t.Fatalf("%s is not shown", what)
	}
	p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(2, 0))
}

// setBrush sets the terrain brush's row to class, its size and its shape
// (0 Circle, 1 Square) through the list's rows and buttons.
func setBrush(t *testing.T, e *ui.MapEditor, class uint8, size, shape int) {
	t.Helper()
	row := -1
	for i, p := range mapedit.Palette {
		if p.Class == class {
			row = i
		}
	}
	for range len(mapedit.Palette) {
		_ = e.Key("up")
	}
	for range row {
		_ = e.Key("down")
	}
	for range mapedit.MaxRadius {
		r, ok := e.ListButtonRect(0, 0)
		pressRect(t, e, r, ok, "the size's minus")
	}
	for range size {
		r, ok := e.ListButtonRect(0, 1)
		pressRect(t, e, r, ok, "the size's plus")
	}
	r, ok := e.ListButtonRect(1, shape)
	pressRect(t, e, r, ok, "a shape button")
}

// toggleBorder presses the View menu's border row.
func toggleBorder(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	n, ok := e.MenuRow("view", "border")
	if !ok {
		t.Fatal("the View menu has no border row")
	}
	r, ok := e.BarRect("view")
	pressRect(t, e, r, ok, "the View entry")
	r, ok = e.MenuRowRect(n)
	pressRect(t, e, r, ok, "the border row")
}

// block is the centre of a (2r+1)-square of cells of class, at least 12
// cells inside the map.
func block(m *alm.Map, class uint8, r int) (image.Point, bool) {
	for y := 12 + r; y < m.Height-12-r; y++ {
		for x := 12 + r; x < m.Width-12-r; x++ {
			all := true
			for dy := -r; dy <= r && all; dy++ {
				for dx := -r; dx <= r && all; dx++ {
					all = mapload.TileClass(m.Tiles[(y+dy)*m.Width+x+dx]) == class
				}
			}
			if all {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}

// area is the centre of a (2r+1)-square of cells of class on the shown map:
// one the map has, or else one a Square stroke of that size paints at the
// map's centre first, which painted reports.
func area(t *testing.T, e *ui.MapEditor, class uint8, r int) (at image.Point, painted bool) {
	t.Helper()
	m := shownMap(t, e)
	if at, ok := block(m, class, r); ok {
		return at, false
	}
	at = image.Pt(m.Width/2, m.Height/2)
	setBrush(t, e, class, r, 1)
	e.ShowCell(at)
	pointerStroke(e, []image.Point{at})
	if _, ok := block(shownMap(t, e), class, r); !ok {
		t.Fatalf("a Square stroke of class %d, size %d left no pure area", class, r)
	}
	return at, true
}

// cornerAt is the class cell (x, y) of m shows at map point (px, py), one of
// its four corners.
func cornerAt(m *alm.Map, x, y, px, py int) uint8 {
	q, _ := installedCorners(m.Tiles[y*m.Width+x])
	return q[(px-x)+2*(py-y)]
}

// hard reports whether the four cells meeting at map point (px, py) of m
// show more than one class there, Flowers read as Grass.
func hard(m *alm.Map, px, py int) bool {
	grass := func(c uint8) uint8 {
		if c == mapedit.ClassFlowers {
			return mapedit.ClassGrass
		}
		return c
	}
	first := grass(cornerAt(m, px-1, py-1, px, py))
	for _, n := range [3][2]int{{px, py - 1}, {px - 1, py}, {px, py}} {
		if grass(cornerAt(m, n[0], n[1], px, py)) != first {
			return true
		}
	}
	return false
}

// hardEdges counts the cells a stroke rewrote and the corners of those cells
// where the cells meeting there show more than one class and did not before.
func hardEdges(before, after *alm.Map) (changed, hardCorners int) {
	w := after.Width
	for y := 8; y < after.Height-1; y++ {
		for x := 1; x < w-1; x++ {
			if after.Tiles[y*w+x] == before.Tiles[y*w+x] {
				continue
			}
			changed++
			for py := y; py <= y+1; py++ {
				for px := x; px <= x+1; px++ {
					if hard(after, px, py) && !hard(before, px, py) {
						hardCorners++
					}
				}
			}
		}
	}
	return changed, hardCorners
}

// Terrain strokes through the window on an installed map, of each listed
// class into an area of another (the map's own, or one painted first), at
// sizes 0, 1 and 3 and both shapes: no stroke leaves a corner where the cells
// meeting show two classes and did not before, and undo gives back the bytes.
func TestReleaseMapEditorStrokesLeaveNoSeam(t *testing.T) {
	x := installedInspector(t)
	e := openBrushEditor(t, x, "scenario/10.alm")
	_ = e.Key("terrain")
	original := e.Document().SourceBytes
	for _, pair := range [][2]uint8{
		{mapedit.ClassSand, mapedit.ClassLand}, {mapedit.ClassLand, mapedit.ClassStones}, {mapedit.ClassSand, mapedit.ClassMountain},
		{mapedit.ClassCracked, mapedit.ClassMountain}, {mapedit.ClassWater, mapedit.ClassGrass}, {mapedit.ClassFlowers, mapedit.ClassLand},
		{mapedit.ClassRoad, mapedit.ClassSavanna}, {mapedit.ClassMountain, mapedit.ClassWater},
	} {
		at, painted := area(t, e, pair[1], 6)
		m := shownMap(t, e)
		t.Logf("class %d into %d at %v, area painted first: %t", pair[0], pair[1], at, painted)
		for _, size := range []int{0, 1, 3} {
			for shape := range 2 {
				setBrush(t, e, pair[0], size, shape)
				e.ShowCell(at)
				before := e.Document().SourceBytes
				pointerStroke(e, []image.Point{at, at.Add(image.Pt(1, 1))})
				changed, corners := hardEdges(m, shownMap(t, e))
				if changed == 0 || corners != 0 {
					t.Errorf("class %d into %d, size %d, shape %d: %d cells rewritten, %d new hard corners", pair[0], pair[1], size, shape, changed, corners)
				}
				_ = e.Key("undo")
				if string(e.Document().SourceBytes) != string(before) {
					t.Fatalf("class %d into %d: undo did not restore the bytes", pair[0], pair[1])
				}
			}
		}
		if painted {
			_ = e.Key("undo")
		}
		if string(e.Document().SourceBytes) != string(original) {
			t.Fatalf("class %d into %d: undo did not restore the map", pair[0], pair[1])
		}
	}
}

// The border view dims exactly the cells the game's border rule closes, on
// an installed map of the root and on a map that is not square; hidden, it
// dims none.
func TestReleaseMapEditorBorderIsTheGameBorder(t *testing.T) {
	x := installedInspector(t)
	fresh, err := mapedit.NewMap(alm.NewMapSpec{Width: 48, Height: 32, Tile: 0x14, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wide.alm")
	if err := os.WriteFile(path, fresh.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	keys := installedMapKeys(x)
	for _, key := range []string{keys[0], keys[len(keys)-1], path} {
		e := openBrushEditor(t, x, key)
		m := shownMap(t, e)
		in, out := 0, 0
		for y := range m.Height {
			for xx := range m.Width {
				want := mapload.InBorderRing(m, xx, y)
				if got := e.BorderCell(image.Pt(xx, y)); got != want {
					t.Fatalf("%s (%d,%d): border view %t, game border %t", key, xx, y, got, want)
				}
				if want {
					in++
				} else {
					out++
				}
			}
		}
		if in == 0 || out == 0 {
			t.Fatalf("%s: %d border and %d playable cells", key, in, out)
		}
		toggleBorder(t, e)
		if e.BorderCell(image.Pt(0, 0)) {
			t.Fatalf("%s: the hidden border still dims", key)
		}
		t.Logf("%s %dx%d: %d border cells, %d playable", key, m.Width, m.Height, in, out)
	}
}

// TestReleaseMapEditorBrushFrames draws the owner's frames of strokes, the
// brush list, the two markers and the border, into AGAINROM_BRUSH_FRAMES when
// set; without it the frames are drawn and checked, not written.
func TestReleaseMapEditorBrushFrames(t *testing.T) {
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_BRUSH_FRAMES")
	shot := func(e *ui.MapEditor, name string) {
		t.Helper()
		writeEditorFrame(t, e, filepath.Join(dir, name+".png"), dir != "", image.Rectangle{})
	}
	e := openBrushEditor(t, x, "scenario/10.alm")
	e.ShowCell(image.Pt(14, 14))
	shot(e, "border-shown")
	toggleBorder(t, e)
	shot(e, "border-hidden")
	toggleBorder(t, e)

	_ = e.Key("terrain")
	land, _ := area(t, e, mapedit.ClassLand, 12)
	m := shownMap(t, e)
	for shape, name := range []string{"marker-circle", "marker-square"} {
		setBrush(t, e, mapedit.ClassSand, 3, shape)
		e.ShowCell(land)
		hoverStatus(e, land)
		shot(e, name)
	}
	stroke := func(name string, steps ...func()) {
		t.Helper()
		shot(e, name+"-before")
		for _, s := range steps {
			s()
		}
		shot(e, name+"-after")
		for range steps {
			_ = e.Key("undo")
		}
	}
	paint := func(class uint8, size, shape int, at image.Point) func() {
		return func() {
			setBrush(t, e, class, size, shape)
			pointerStroke(e, []image.Point{at})
		}
	}
	type scene struct {
		name        string
		fill, paint uint8
	}
	for _, s := range []scene{{"sand-into-land", mapedit.ClassLand, mapedit.ClassSand}, {"land-into-stones", mapedit.ClassStones, mapedit.ClassLand}, {"sand-into-mountain", mapedit.ClassMountain, mapedit.ClassSand}} {
		at, found := block(m, s.fill, 5)
		if !found {
			t.Logf("%s: no area of class %d; painted one first", s.name, s.fill)
			at = land
			setBrush(t, e, s.fill, 6, 1)
			pointerStroke(e, []image.Point{at})
		}
		e.ShowCell(at)
		stroke(s.name, paint(s.paint, 2, 0, at))
		if !found {
			_ = e.Key("undo")
		}
	}
	e.ShowCell(land)
	stroke("three-class-corner",
		paint(mapedit.ClassSand, 3, 1, land.Add(image.Pt(-4, -2))),
		paint(mapedit.ClassStones, 3, 1, land.Add(image.Pt(4, -2))),
		paint(mapedit.ClassMountain, 3, 1, land.Add(image.Pt(0, 5))))
	setBrush(t, e, mapedit.ClassSand, 3, 0)
	e.ShowCell(land)
	hoverStatus(e, land)
	shot(e, "brush-panel")

	e = openBrushEditor(t, x, "scenario/31.alm")
	_ = e.Key("terrain")
	setBrush(t, e, mapedit.ClassCracked, 0, 0)
	click := image.Pt(40, 40)
	e.ShowCell(click)
	shot(e, "cracked-click-before")
	pointerStroke(e, []image.Point{click})
	shot(e, "cracked-click-after")
	if dir != "" {
		t.Logf("frames in %s", dir)
	}
}
