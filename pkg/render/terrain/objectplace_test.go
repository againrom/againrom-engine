package terrain_test

// Tests for the object layer's gate, its animated subset and the per-counter
// pass (0031 AC-3, AC-5, AC-9's builder half, SC-3, SC-4, SC-5).
//
// SEPARATE CONTEXT, as every other test file in this package states of itself.
// The fixtures are hand-built grids and classes: no game install, no window, no
// image. Every expected anchor and top-left below is worked out in its own
// comment from the class canvas and the drawn frame's size, never transcribed
// from the pass.

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// objectAnimFrames is the fixture sheet: three frames differing in BOTH
// dimensions, so a re-anchor that got one axis right and the other wrong cannot
// average out (SC-5).
//
//	0: 2x2    1: 6x10    2: 4x4
func objectAnimFrames() []*terrain.StaticFrame {
	frame := func(w, h int) *terrain.StaticFrame {
		return &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h)}
	}
	return []*terrain.StaticFrame{frame(2, 2), frame(6, 10), frame(4, 4)}
}

// objectAnimClass is the fixture class: canvas 32x32, centre (16, 24), Index 0,
// and the two-step timeline [0 1] — so its cycle reaches frame 0 at even
// counters and frame 1 at odd ones, on the cell the grids below place it at.
//
//	anchor(2x2)  = (16 - 16 + 1, 24 - 16 + 1) = ( 1,  9)
//	anchor(6x10) = (16 - 16 + 3, 24 - 16 + 5) = ( 3, 13)
func objectAnimClass() *terrain.StaticClass {
	frames := objectAnimFrames()
	return &terrain.StaticClass{
		Width: 32, Height: 32, CenterX: 16, CenterY: 24,
		Frame:    frames[0],
		Frames:   frames,
		Index:    0,
		Timeline: []int{0, 1},
	}
}

// objectStillClass is a class with a sheet and no cycle at all: its timeline is
// empty, so its period is 0 and no gate can open it. Its Index is 2, which is
// what makes the switch-off arm visible — frame 0 is a different frame.
func objectStillClass() *terrain.StaticClass {
	frames := objectAnimFrames()
	return &terrain.StaticClass{
		Width: 32, Height: 32, CenterX: 16, CenterY: 24,
		Frame:  frames[2],
		Frames: frames,
		Index:  2,
	}
}

func objectAnimSet() *terrain.StaticSet {
	var set terrain.StaticSet
	set.Classes[1] = objectAnimClass()
	set.Classes[2] = objectStillClass()
	return &set
}

// objectTiles builds a w*h tile layer with the given words written at the given
// cell indices and zero everywhere else.
func objectTiles(w, h int, at map[int]uint16) []uint16 {
	t := make([]uint16, w*h)
	for i, v := range at {
		t[i] = v
	}
	return t
}

// TestObjectCycleOpenGate is AC-3 and SC-3: every gate case on a case of its
// own, in both diagnostic states.
//
// The fixture grid is 3x3 and the OPEN case sets ONE of the two bits on each of
// two DIFFERENT neighbours, so the test can only pass if all four words are
// ORed: a gate reading the cell's own word alone, or any three of the four,
// sees a single bit and closes.
func TestObjectCycleOpenGate(t *testing.T) {
	const w, h = 3, 3
	const period = 2

	// The interior cell (0,0) and its three neighbours: indices 0, 1, 3, 4.
	both := terrain.Grid{Width: w, Height: h, Tiles: objectTiles(w, h, map[int]uint16{1: 0x8000, 3: 0x4000})}
	one := terrain.Grid{Width: w, Height: h, Tiles: objectTiles(w, h, map[int]uint16{1: 0x8000})}
	neither := terrain.Grid{Width: w, Height: h, Tiles: objectTiles(w, h, map[int]uint16{1: 0x3fff, 4: 0x2000})}
	// Every word set, so the far edge's refusal is about the EDGE and not about
	// the words there.
	all := terrain.Grid{Width: w, Height: h, Tiles: objectTiles(w, h, map[int]uint16{
		0: 0xc000, 1: 0xc000, 2: 0xc000,
		3: 0xc000, 4: 0xc000, 5: 0xc000,
		6: 0xc000, 7: 0xc000, 8: 0xc000,
	})}

	cases := []struct {
		label     string
		g         terrain.Grid
		col, row  int
		period    int
		wantTiles bool // under AnimGateTiles
		wantAll   bool // under AnimGateAll
	}{
		{"an interior cell whose four words OR to both bits", both, 0, 0, period, true, true},
		{"an interior cell with one of the two bits", one, 0, 0, period, false, true},
		{"an interior cell with neither bit", neither, 0, 0, period, false, true},
		{"the LAST COLUMN, both bits set", all, w - 1, 0, period, false, true},
		{"the LAST ROW, both bits set", all, 0, h - 1, period, false, true},
		{"the far corner, both bits set", all, w - 1, h - 1, period, false, true},
		{"a class of period 0, both bits set", all, 0, 0, 0, false, false},
		{"a negative period, both bits set", all, 0, 0, -3, false, false},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			if got := terrain.ObjectCycleOpen(c.g, c.col, c.row, c.period, terrain.AnimGateTiles); got != c.wantTiles {
				t.Errorf("AnimGateTiles: open = %t, want %t", got, c.wantTiles)
			}
			if got := terrain.ObjectCycleOpen(c.g, c.col, c.row, c.period, terrain.AnimGateAll); got != c.wantAll {
				t.Errorf("AnimGateAll: open = %t, want %t", got, c.wantAll)
			}
		})
	}
}

// TestObjectCycleOpenWithoutTileWords is AC-9's gate share: a grid whose tile
// layer is absent or the wrong length closes the decoded gate rather than
// reading past a slice, and the diagnostic still opens — it is specified to
// ignore the tile words entirely.
func TestObjectCycleOpenWithoutTileWords(t *testing.T) {
	for _, g := range []terrain.Grid{
		{Width: 3, Height: 3},
		{Width: 3, Height: 3, Tiles: make([]uint16, 4)},
		{Width: 3, Height: 3, Tiles: make([]uint16, 100)},
	} {
		if terrain.ObjectCycleOpen(g, 0, 0, 2, terrain.AnimGateTiles) {
			t.Errorf("%d tile words over a 3x3 grid: the decoded gate opened", len(g.Tiles))
		}
		if !terrain.ObjectCycleOpen(g, 0, 0, 2, terrain.AnimGateAll) {
			t.Errorf("%d tile words over a 3x3 grid: the diagnostic did not open", len(g.Tiles))
		}
	}
}

// objectAnimGrid is the map the pass tests place from: 2x2 cells so the ONE
// interior cell is (0,0), with the animated class there and the cycle-less one
// beside it.
//
//	row 0:  1  2
//	row 1:  2  0
func objectAnimGrid(tiles []uint16) terrain.Grid {
	return terrain.Grid{
		Width: 2, Height: 2, Tiles: tiles,
		Overlay: []uint8{
			1, 2,
			2, 0,
		},
	}
}

// objectOpenTiles sets both gate bits on the cell (0,0)'s own word, so the OR of
// the four corners carries them.
func objectOpenTiles() []uint16 { return objectTiles(2, 2, map[int]uint16{0: 0xc000}) }

func TestStaticPlacementsAnimatedSubset(t *testing.T) {
	set := objectAnimSet()

	t.Run("the decoded gate over a map that opens one cell", func(t *testing.T) {
		g := objectAnimGrid(objectOpenTiles())
		places, counts, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)

		if len(places) != 3 || counts.Placed != 3 {
			t.Fatalf("%d placements, census %+v, want 3", len(places), counts)
		}
		if want := []int{0}; !reflect.DeepEqual(animated, want) {
			t.Errorf("animated subset = %v, want %v — only cell (0,0) opens", animated, want)
		}
		if counts.Animated != len(animated) {
			t.Errorf("counts.Animated = %d, want %d", counts.Animated, len(animated))
		}
		// The subset indexes the list it was built beside.
		if got := places[animated[0]].Cell; got != image.Pt(0, 0) {
			t.Errorf("subset entry 0 points at cell %v, want (0,0)", got)
		}
	})

	t.Run("the decoded gate over a map that opens nothing", func(t *testing.T) {
		g := objectAnimGrid(make([]uint16, 4))
		places, counts, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)

		if len(animated) != 0 || counts.Animated != 0 {
			t.Errorf("animated = %v, counts.Animated = %d, want empty — no cell sets the bits",
				animated, counts.Animated)
		}
		if counts.Placed != 3 || len(places) != 3 {
			t.Errorf("%d placements, census %+v — the gate must move no placement", len(places), counts)
		}
	})

	t.Run("the diagnostic opens every non-zero period and no other", func(t *testing.T) {
		// No tile word is set at all, so nothing here comes from the map.
		g := objectAnimGrid(make([]uint16, 4))
		places, counts, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateAll)

		if want := []int{0}; !reflect.DeepEqual(animated, want) {
			t.Errorf("animated subset = %v, want %v — byte 2's class has no cycle to open", animated, want)
		}
		if counts.Animated != 1 {
			t.Errorf("counts.Animated = %d, want 1", counts.Animated)
		}
		// The diagnostic moves the SUBSET and nothing else: same placements,
		// same census otherwise, same order.
		plain, plainCounts, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
		if !reflect.DeepEqual(places, plain) {
			t.Errorf("the diagnostic changed the built list")
		}
		plainCounts.Animated = counts.Animated
		if plainCounts != counts {
			t.Errorf("the diagnostic changed the census beyond Animated: %+v vs %+v", counts, plainCounts)
		}
	})
}

func TestAnimateStaticsReanchors(t *testing.T) {
	set := objectAnimSet()
	g := objectAnimGrid(objectOpenTiles())
	places, _, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
	if len(animated) != 1 {
		t.Fatalf("animated subset = %v, want one entry", animated)
	}
	built := places[animated[0]]
	ground := built.Ground()

	// Cell (0,0), centre (16, 16). Timeline [0 1] over Index 0, so counter 0
	// selects frame 0 (2x2) and counter 1 selects frame 1 (6x10):
	//
	//	counter 0: anchor ( 1,  9)  topLeft (16 -  1, 16 -  9) = (15,  7)
	//	counter 1: anchor ( 3, 13)  topLeft (16 -  3, 16 - 13) = (13,  3)
	want := []struct {
		counter        uint32
		frameW, frameH int
		anchor         image.Point
		topLeft        image.Point
	}{
		{0, 2, 2, image.Pt(1, 9), image.Pt(15, 7)},
		{1, 6, 10, image.Pt(3, 13), image.Pt(13, 3)},
	}

	var buf []terrain.StaticPlacement
	var seen []image.Point
	for _, c := range want {
		out := terrain.AnimateStatics(buf, places, animated, c.counter, true)
		buf = out
		p := out[animated[0]]

		if p.Frame.Width != c.frameW || p.Frame.Height != c.frameH {
			t.Fatalf("counter %d: drew a %dx%d frame, want %dx%d",
				c.counter, p.Frame.Width, p.Frame.Height, c.frameW, c.frameH)
		}
		if p.Anchor != c.anchor {
			t.Errorf("counter %d: anchor %v, want %v", c.counter, p.Anchor, c.anchor)
		}
		if p.TopLeft != c.topLeft {
			t.Errorf("counter %d: top-left %v, want %v", c.counter, p.TopLeft, c.topLeft)
		}
		if p.Ground() != ground {
			t.Errorf("counter %d: ground %v, want %v (the build's)", c.counter, p.Ground(), ground)
		}
		wantRect := image.Rectangle{Min: c.topLeft, Max: c.topLeft.Add(image.Pt(c.frameW, c.frameH))}
		if p.Rect() != wantRect {
			t.Errorf("counter %d: rect %v, want %v", c.counter, p.Rect(), wantRect)
		}
		seen = append(seen, p.TopLeft)

		// The cells the subset does not name are the builder's own, untouched.
		for i := range out {
			if i == animated[0] {
				continue
			}
			if out[i] != places[i] {
				t.Errorf("counter %d: placement %d moved and is not in the subset", c.counter, i)
			}
		}
	}

	// The two top-lefts differ by exactly the halved size difference, in BOTH
	// axes: (6/2 - 2/2, 10/2 - 2/2) = (2, 4), subtracted because the anchor is
	// added to the frame's own half.
	if got, want := seen[0].Sub(seen[1]), image.Pt(2, 4); got != want {
		t.Errorf("top-lefts differ by %v, want %v — the halved size difference", got, want)
	}
}

func TestAnimateStaticsClosedGateIsThePreStoryList(t *testing.T) {
	set := objectAnimSet()
	g := objectAnimGrid(make([]uint16, 4))

	places, counts, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
	if counts.Animated != 0 || len(animated) != 0 {
		t.Fatalf("counts.Animated = %d, subset %v — no cell of this map sets the gate bits",
			counts.Animated, animated)
	}
	before := append([]terrain.StaticPlacement(nil), places...)

	var buf []terrain.StaticPlacement
	for _, counter := range []uint32{0, 1, 2, 3, 27, 28, 1000, ^uint32(0)} {
		out := terrain.AnimateStatics(buf, places, animated, counter, true)
		if len(out) != len(places) {
			t.Fatalf("counter %d: %d placements, want %d", counter, len(out), len(places))
		}
		if &out[0] != &places[0] {
			t.Errorf("counter %d: the pass copied a list nothing could change", counter)
		}
		for i := range out {
			if out[i] != before[i] {
				t.Errorf("counter %d: placement %d = %+v, want the built %+v", counter, i, out[i], before[i])
			}
		}
	}
	if buf != nil {
		t.Errorf("the pass allocated into the caller's buffer with an empty subset")
	}
}

func TestAnimateVisibleStaticsReadsTheGateOnEveryDraw(t *testing.T) {
	set := objectAnimSet()
	g := objectAnimGrid(make([]uint16, 4))
	places, _, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateAll)
	if len(animated) != 1 {
		t.Fatalf("cycle-capable subset = %v, want one entry", animated)
	}
	at := animated[0]
	if got := places[at].Frame; got != places[at].Class.Frames[0] {
		t.Fatalf("built frame = %p, want class Index frame %p", got, places[at].Class.Frames[0])
	}

	visible := false
	gate := func(col, row int) bool {
		return visible && col == 0 && row == 0
	}
	closed := terrain.AnimateVisibleStatics(nil, places, animated, 1, true, gate)
	if !sameStaticSlice(closed, places) {
		t.Fatal("a closed live gate copied or changed the built list")
	}

	visible = true
	open := terrain.AnimateVisibleStatics(nil, places, animated, 1, true, gate)
	if sameStaticSlice(open, places) {
		t.Fatal("opening the live gate did not produce the animated draw list")
	}
	if got, want := open[at].Frame, places[at].Class.Frames[1]; got != want {
		t.Errorf("open live gate drew frame %p, want counter-1 frame %p", got, want)
	}

	visible = false
	closedAgain := terrain.AnimateVisibleStatics(open, places, animated, 1, true, gate)
	if !sameStaticSlice(closedAgain, places) {
		t.Fatal("closing the live gate did not return to the unchanged built list")
	}
}

func sameStaticSlice(a, b []terrain.StaticPlacement) bool {
	return len(a) == len(b) && (len(a) == 0 || &a[0] == &b[0])
}

func TestAnimateStaticsSwitchOffDrawsFrameZero(t *testing.T) {
	set := objectAnimSet()
	g := objectAnimGrid(make([]uint16, 4)) // the gate opens nowhere
	places, _, animated := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)

	grounds := make([]image.Point, len(places))
	for i, p := range places {
		grounds[i] = p.Ground()
	}
	frameZero := objectAnimFrames()[0]

	// Frame 0 is 2x2 and both fixture classes share the canvas 32x32 centred
	// (16, 24), so every placement re-anchors to (16 - 16 + 1, 24 - 16 + 1).
	// Ground() alone cannot see a dropped re-anchor — it is TopLeft + Anchor and
	// both stay the build's — so the anchor is asserted as a value.
	wantAnchor := image.Pt(1, 9)

	var buf []terrain.StaticPlacement
	for _, counter := range []uint32{0, 1, 7, ^uint32(0)} {
		out := terrain.AnimateStatics(buf, places, animated, counter, false)
		buf = out
		for i, p := range out {
			if p.Frame.Width != frameZero.Width || p.Frame.Height != frameZero.Height {
				t.Errorf("counter %d: placement %d drew a %dx%d frame, want frame 0's %dx%d",
					counter, i, p.Frame.Width, p.Frame.Height, frameZero.Width, frameZero.Height)
			}
			if p.Frame != p.Class.Frames[0] {
				t.Errorf("counter %d: placement %d did not draw its own sheet's frame 0", counter, i)
			}
			if p.Ground() != grounds[i] {
				t.Errorf("counter %d: placement %d ground %v, want %v", counter, i, p.Ground(), grounds[i])
			}
			if p.Anchor != wantAnchor {
				t.Errorf("counter %d: placement %d anchor %v, want %v — frame 0's own half, not the Index frame's",
					counter, i, p.Anchor, wantAnchor)
			}
			if p.TopLeft != grounds[i].Sub(wantAnchor) {
				t.Errorf("counter %d: placement %d top-left %v, want %v",
					counter, i, p.TopLeft, grounds[i].Sub(wantAnchor))
			}
		}
		// The built list is not written through: the switch changes what is
		// painted, never what was placed.
		for i, p := range places {
			if p.Frame != set.Classes[g.Overlay[p.Cell.Y*g.Width+p.Cell.X]].Frame {
				t.Fatalf("counter %d: the built list was mutated at %d", counter, i)
			}
		}
	}
}

// TestAnimateStaticsIsTotal is AC-9's builder-and-pass half: a nil bundle, a
// grid with no tile words, a class of period 0, placements at (0,0) and at the
// far corner, and the counter at its maximum. Nothing panics, no index leaves
// its slice, and the bundle-less build draws WHAT IT DRAWS TODAY — compared
// whole rather than by count.
func TestAnimateStaticsIsTotal(t *testing.T) {
	const maxCounter = ^uint32(0)

	t.Run("a nil bundle builds and draws nothing", func(t *testing.T) {
		g := objectAnimGrid(objectOpenTiles())
		for _, gate := range []bool{terrain.AnimGateTiles, terrain.AnimGateAll} {
			places, counts, animated := terrain.StaticPlacements(g, nil, nil, 0, gate)
			if len(places) != 0 || len(animated) != 0 || counts != (terrain.StaticCounts{}) {
				t.Fatalf("gate %t: %d placements, subset %v, census %+v — want the empty build",
					gate, len(places), animated, counts)
			}
			for _, animate := range []bool{true, false} {
				out := terrain.AnimateStatics(nil, places, animated, maxCounter, animate)
				if len(out) != 0 {
					t.Errorf("gate %t animate %t: the pass produced %d placements over a nil bundle",
						gate, animate, len(out))
				}
			}
		}
	})

	t.Run("cells at the origin and at the far corner, at the maximum counter", func(t *testing.T) {
		// Every cell placed and every tile word set, so the diagnostic and the
		// decoded gate both have something to answer over the whole grid,
		// including the two edges the decoded one closes.
		const w, h = 4, 4
		overlay := make([]uint8, w*h)
		for i := range overlay {
			overlay[i] = 1
		}
		tiles := make([]uint16, w*h)
		for i := range tiles {
			tiles[i] = 0xffff
		}
		g := terrain.Grid{Width: w, Height: h, Tiles: tiles, Overlay: overlay}

		for _, gate := range []bool{terrain.AnimGateTiles, terrain.AnimGateAll} {
			places, counts, animated := terrain.StaticPlacements(g, objectAnimSet(), nil, 0, gate)
			if counts.Placed != w*h {
				t.Fatalf("gate %t: %d placements, want %d", gate, counts.Placed, w*h)
			}
			if places[0].Cell != image.Pt(0, 0) || places[len(places)-1].Cell != image.Pt(w-1, h-1) {
				t.Fatalf("gate %t: the list runs %v..%v", gate, places[0].Cell, places[len(places)-1].Cell)
			}
			// Under the decoded gate the last column and last row are closed,
			// so the open cells are exactly the (w-1)*(h-1) interior ones.
			wantOpen := (w - 1) * (h - 1)
			if gate == terrain.AnimGateAll {
				wantOpen = w * h
			}
			if counts.Animated != wantOpen || len(animated) != wantOpen {
				t.Errorf("gate %t: %d open cycles, want %d", gate, counts.Animated, wantOpen)
			}

			var buf []terrain.StaticPlacement
			for _, animate := range []bool{true, false} {
				out := terrain.AnimateStatics(buf, places, animated, maxCounter, animate)
				buf = out
				for i, p := range out {
					if p.Frame == nil {
						t.Fatalf("gate %t animate %t: placement %d drew no frame", gate, animate, i)
					}
					if p.Rect() != (image.Rectangle{Min: p.TopLeft, Max: p.TopLeft.Add(image.Pt(p.Frame.Width, p.Frame.Height))}) {
						t.Errorf("gate %t animate %t: placement %d rect disagrees with its frame", gate, animate, i)
					}
				}
			}
		}
	})

	t.Run("a class of period 0 is never re-selected under the decoded gate", func(t *testing.T) {
		// Byte 2's class has an empty timeline, so no gate opens it and the
		// switch-on pass leaves it drawing its own Index at every counter.
		g := objectAnimGrid(objectOpenTiles())
		places, _, animated := terrain.StaticPlacements(g, objectAnimSet(), nil, 0, terrain.AnimGateAll)
		var buf []terrain.StaticPlacement
		for _, counter := range []uint32{0, 1, 5, maxCounter} {
			out := terrain.AnimateStatics(buf, places, animated, counter, true)
			buf = out
			for i, p := range out {
				if len(p.Class.Timeline) != 0 {
					continue
				}
				if p.Frame != p.Class.Frames[p.Class.Index] {
					t.Errorf("counter %d: placement %d has no cycle but drew a frame other than its Index", counter, i)
				}
			}
		}
	})

	t.Run("a subset that does not index its list is ignored", func(t *testing.T) {
		g := objectAnimGrid(objectOpenTiles())
		places, _, _ := terrain.StaticPlacements(g, objectAnimSet(), nil, 0, terrain.AnimGateTiles)
		out := terrain.AnimateStatics(nil, places, []int{-1, len(places), 0}, maxCounter, true)
		if len(out) != len(places) {
			t.Errorf("%d placements out of %d in", len(out), len(places))
		}
	})
}

func TestAnimateStaticsReusesTheBuffer(t *testing.T) {
	g := objectAnimGrid(objectOpenTiles())
	places, _, animated := terrain.StaticPlacements(g, objectAnimSet(), nil, 0, terrain.AnimGateTiles)

	buf := terrain.AnimateStatics(nil, places, animated, 0, true)
	if len(buf) != len(places) {
		t.Fatalf("%d placements out, %d in", len(buf), len(places))
	}
	first := &buf[0]
	for counter := uint32(1); counter < 8; counter++ {
		out := terrain.AnimateStatics(buf, places, animated, counter, true)
		if &out[0] != first {
			t.Fatalf("counter %d: the pass allocated a second buffer", counter)
		}
		buf = out
	}
	// The built list is never written through, whatever the buffer does.
	if &buf[0] == &places[0] {
		t.Error("the pass wrote into the list the builder produced")
	}
}
