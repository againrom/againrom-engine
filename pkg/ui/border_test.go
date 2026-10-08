package ui

// The engine margin's dim: how cornerScales answers for a cell the block
// plane marks (AC-3..AC-7).

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// borderBit is bit 1 of a block byte, spelled here so the fixtures say which
// bit they set rather than carrying a bare 2. The production constant it
// mirrors is unexported in another package; a fixture that reached for it would
// be asserting the reader against itself.
const borderBit uint8 = 0x02

// marginPlane is a w*h block plane whose bit 1 is set on the cells listed, and
// whose bit 0 is set on every cell — standing for a map on which everything is
// blocked to a ground mover, so a reader keying on bit 0, or on "the byte is
// nonzero", dims the whole map and fails every assertion below.
func marginPlane(w, h int, margin ...[2]int) []uint8 {
	p := make([]uint8, w*h)
	for i := range p {
		p[i] = 0x01
	}
	for _, c := range margin {
		p[c[1]*w+c[0]] |= borderBit
	}
	return p
}

// withBlock returns g carrying the block plane b.
func withBlock(g terrain.Grid, b []uint8) terrain.Grid {
	g.Block = b
	return g
}

// baseScales is what cornerScales answered before this story: ShadeScale of the
// cell's four corner levels, computed here from the viewer's own level grid so
// that the dim is measured against the lighting rather than against a literal.
func baseScales(v *Viewer, w, h, tx, ty int) [4]float32 {
	lv := terrain.CornerLevels(v.levels, w, h, tx, ty)
	return [4]float32{
		terrain.ShadeScale(int(lv[0])),
		terrain.ShadeScale(int(lv[1])),
		terrain.ShadeScale(int(lv[2])),
		terrain.ShadeScale(int(lv[3])),
	}
}

func TestCornerScalesBlacksTheLitMargin(t *testing.T) {
	// The whole left column is margin; the rest is playable.
	g := withBlock(cliffGrid(), marginPlane(cliffW, cliffH, [2]int{0, 0}, [2]int{0, 1}, [2]int{0, 2}, [2]int{0, 3}))

	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	relief := 0
	for ty := 0; ty < cliffH; ty++ {
		base := baseScales(v, cliffW, cliffH, 0, ty)
		want := [4]float32{
			base[0] * mapBorderDim,
			base[1] * mapBorderDim,
			base[2] * mapBorderDim,
			base[3] * mapBorderDim,
		}
		got := v.cornerScales(0, ty)
		if got != want {
			t.Fatalf("margin cell (0,%d) cornerScales = %v, want %v (undimmed %v)", ty, got, want, base)
		}
		for i, s := range got {
			if s != 0 {
				t.Fatalf("margin cell (0,%d) corner %d = %v, want 0: the map edge is black", ty, i, s)
			}
			if base[i] <= 0 {
				t.Fatalf("margin cell (0,%d) corner %d was already 0 before the dim: vacuous", ty, i)
			}
		}
		if base[0] != base[1] || base[0] != base[2] || base[0] != base[3] {
			relief++
		}
	}
	if relief == 0 {
		t.Fatal("no margin cell in the fixture had four unequal corners BEFORE the dim; the " +
			"fixture no longer proves the light ran at all")
	}
}

func TestCornerScalesLeavesPlayableCellsAlone(t *testing.T) {
	g := withBlock(cliffGrid(), marginPlane(cliffW, cliffH, [2]int{0, 0}, [2]int{0, 1}, [2]int{0, 2}, [2]int{0, 3}))

	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	// Every playable row keeps its shading.
	for ty := 0; ty < cliffH; ty++ {
		for tx := 1; tx < cliffW; tx++ {
			want := baseScales(v, cliffW, cliffH, tx, ty)
			if got := v.cornerScales(tx, ty); got != want {
				t.Fatalf("playable cell (%d,%d) cornerScales = %v, want %v — unchanged", tx, ty, got, want)
			}
		}
	}
}

func TestCornerScalesDimsTheUnlitMargin(t *testing.T) {
	dim := [4]float32{mapBorderDim, mapBorderDim, mapBorderDim, mapBorderDim}
	unlit := [4]float32{1, 1, 1, 1}
	// Column 0 is margin; column 1 is playable.
	plane := marginPlane(2, 2, [2]int{0, 0}, [2]int{0, 1})

	t.Run("no usable altitude grid", func(t *testing.T) {
		g := withBlock(grid(2, 2, slotWord(0), slotWord(0), slotWord(0), slotWord(0)), plane)
		v, err := NewViewer("m", g, litTileset())
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		if v.Lit() {
			t.Fatal("fixture must be unlit: no altitude grid")
		}
		if got := v.cornerScales(0, 0); got != dim {
			t.Fatalf("unlit margin cornerScales = %v, want %v", got, dim)
		}
		if got := v.cornerScales(1, 1); got != unlit {
			t.Fatalf("unlit playable cornerScales = %v, want all-1", got)
		}
	})

	t.Run("unshaded diagnostic on", func(t *testing.T) {
		g := withBlock(grid(2, 2, slotWord(0), slotWord(0), slotWord(0), slotWord(0)), plane)
		g.Altitudes = make([]uint8, 4)
		v, err := NewViewer("m", g, litTileset())
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		if !v.Lit() {
			t.Fatal("fixture must start lit")
		}
		v.SetUnshaded(true)
		if v.Lit() {
			t.Fatal("SetUnshaded(true) did not turn lighting off")
		}
		if got := v.cornerScales(0, 0); got != dim {
			t.Fatalf("unshaded margin cornerScales = %v, want %v", got, dim)
		}
		if got := v.cornerScales(1, 1); got != unlit {
			t.Fatalf("unshaded playable cornerScales = %v, want all-1", got)
		}
	})
}

func TestCornerScalesPlaceholderInTheMarginIsUndimmed(t *testing.T) {
	// (0,0) -> slot 0 (present); (1,0) -> slot 4 (absent). Both are margin.
	g := withBlock(grid(2, 1, slotWord(0), slotWord(1)), marginPlane(2, 1, [2]int{0, 0}, [2]int{1, 0}))
	g.Altitudes = make([]uint8, 2)

	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	if got, want := v.cornerScales(1, 0), ([4]float32{1, 1, 1, 1}); got != want {
		t.Fatalf("placeholder cell in the margin cornerScales = %v, want all-1", got)
	}

	wantDim := terrain.ShadeScale(46) * mapBorderDim // flat daytime ground, dimmed
	for i, s := range v.cornerScales(0, 0) {
		if s != wantDim {
			t.Fatalf("margin neighbour corner %d = %v, want %v — the placeholder must not have "+
				"suppressed the dim on its own neighbour", i, s, wantDim)
		}
	}
}

func TestCornerScalesWithoutAMarginPlaneIsThePreStoryPicture(t *testing.T) {
	planes := map[string][]uint8{
		"no plane at all":         nil,
		"a plane marking nothing": marginPlane(cliffW, cliffH),
	}

	for name, plane := range planes {
		t.Run(name, func(t *testing.T) {
			v, err := NewViewer("m", withBlock(cliffGrid(), plane), litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			if !v.Lit() {
				t.Fatal("fixture must be lit")
			}
			for ty := 0; ty < cliffH; ty++ {
				for tx := 0; tx < cliffW; tx++ {
					want := baseScales(v, cliffW, cliffH, tx, ty)
					if got := v.cornerScales(tx, ty); got != want {
						t.Fatalf("cell (%d,%d) cornerScales = %v, want %v", tx, ty, got, want)
					}
				}
			}
		})
	}
}

func TestFourLowerTerrainRowsKeepTheirTextureAndRefuseMovement(t *testing.T) {
	const w, h = 64, 64
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h)
	for i := range g.Altitudes {
		g.Altitudes[i] = 127
	}
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = 2
			}
		}
	}
	v := newViewer(t, g)
	v.cam.ViewW, v.cam.ViewH = 27*32, 24*32
	v.cam.Pan(0, 1e9)
	scales := [4]float32{1, 1, 1, 1}
	for y := h - 8; y < h-4; y++ {
		if !v.grid.BorderCell(20, y) || v.dimBorder(scales, 20, y) != scales {
			t.Fatalf("row %d lost texture or movement border", y)
		}
		x, sy := v.pathCellCentre(image.Pt(20, y))
		if x < 0 || sy < 0 || x >= float64(v.cam.ViewW) || sy >= float64(v.cam.ViewH) {
			t.Fatalf("row %d sample (%v,%v) is outside the viewport", y, x, sy)
		}
		if _, _, inside := v.groundCellAt(x, sy); inside {
			t.Fatalf("border row %d accepts ground input", y)
		}
		if col, row := v.dropCellAt(int(x), int(sy)); col != dropOffMapCell || row != dropOffMapCell {
			t.Fatalf("border row %d accepts drop (%d,%d)", y, col, row)
		}
		v.grid.Block[y*w+20] = 0
		col, row, inside := v.groundCellAt(x, sy)
		v.grid.Block[y*w+20] = borderBit
		if !inside || col != 20 || row != y {
			t.Fatalf("open row %d sample picks (%d,%d,%t)", y, col, row, inside)
		}
	}
	if v.dimBorder(scales, 20, h-4) != ([4]float32{}) {
		t.Fatal("terrain below four-row band is lit")
	}
}
