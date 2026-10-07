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

	// The first playable row is not drawn; every row below it is unchanged.
	for ty := hiddenTopRows; ty < cliffH; ty++ {
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
	// Column 0 is margin; row 0 is the first playable row, which is not
	// drawn, so (1,1) is the playable cell.
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

func TestTheOuterMarginIsBlack(t *testing.T) {
	if mapBorderDim != 0 {
		t.Fatalf("mapBorderDim = %v, want 0: the original's map edge is black", mapBorderDim)
	}

	// Column 0 is margin, row 0 the undrawn first playable row and (1,1)
	// playable, all lit: a failure here is dimBorder no longer running, not
	// the constant.
	g := withBlock(grid(2, 2, slotWord(0), slotWord(0), slotWord(0), slotWord(0)), marginPlane(2, 2, [2]int{0, 0}, [2]int{0, 1}))
	g.Altitudes = make([]uint8, 4)
	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}
	if got, want := v.cornerScales(0, 0), ([4]float32{0, 0, 0, 0}); got != want {
		t.Fatalf("lit margin cornerScales = %v, want all-0", got)
	}
	if got := v.cornerScales(1, 1); got == ([4]float32{0, 0, 0, 0}) {
		t.Fatal("the playable cell went black too: the margin test is vacuous")
	}
	if got := v.cornerScales(1, 0); got != ([4]float32{0, 0, 0, 0}) {
		t.Fatalf("the first playable row cornerScales = %v, want all-0: it is not drawn", got)
	}
}

// TestTheFirstLowerMarginRowIsATexturedImpassableApron pins the distinction
// visible in the owner's ROM1/Againrom comparison. The block plane marks both
// lower rows, so simulation still rejects both; presentation keeps the first
// row's terrain scales and blacks the second.
func TestTheFirstLowerMarginRowIsATexturedImpassableApron(t *testing.T) {
	const w, h = 1, 3
	g := withBlock(grid(w, h, slotWord(0), slotWord(0), slotWord(0)),
		marginPlane(w, h, [2]int{0, 1}, [2]int{0, 2}))
	g.Altitudes = make([]uint8, w*h)
	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	if !v.grid.BorderCell(0, 1) || !v.grid.BorderCell(0, 2) {
		t.Fatal("fixture must keep the apron and outer guard impassable")
	}
	if got, want := v.cornerScales(0, 1), baseScales(v, w, h, 0, 1); got != want {
		t.Fatalf("first lower margin row scales = %v, want textured %v", got, want)
	}
	if got, want := v.cornerScales(0, 2), ([4]float32{}); got != want {
		t.Fatalf("second lower margin row scales = %v, want black %v", got, want)
	}

	playX, playY := v.pathCellCentre(image.Pt(0, 0))
	if col, row, inside := v.groundCellAt(playX, playY); !inside || col != 0 || row != 0 {
		t.Fatalf("playable row pick = (%d,%d,%v), want (0,0,true)", col, row, inside)
	}
	lipX, lipY := v.pathCellCentre(image.Pt(0, 1))
	if col, row, inside := v.groundCellAt(lipX, lipY); inside || col != 0 || row != 0 {
		t.Fatalf("textured impassable lip pick = (%d,%d,%v), want refusal", col, row, inside)
	}
	if col, row := v.dropCellAt(int(lipX), int(lipY)); col != dropOffMapCell || row != dropOffMapCell {
		t.Fatalf("drop on textured impassable lip = (%d,%d), want off-map sentinel", col, row)
	}
}

func TestWideCameraKeepsOneTexturedApronSoThePlayableEdgeIsNotClipped(t *testing.T) {
	const w, h = 64, 48
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h) // production displaced path, flat relief
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	v := newViewer(t, g)
	v.Layout(1920, 1080)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("mode = %v, want production displaced path", v.Mode())
	}
	if got := v.ViewportSize(); got != image.Pt(1206, 768) {
		t.Fatalf("wide viewport = %v, want (1206,768)", got)
	}
	if got, ok := v.playableCellRect(); !ok || got != image.Rect(8, 8, 56, 40) {
		t.Fatalf("playable cell rect = %v,%v, want (8,8)-(56,40),true", got, ok)
	}
	if got, ok := v.renderCellRect(); !ok || got != image.Rect(8, 9, 56, 41) {
		t.Fatalf("render cell rect = %v,%v, want the first row hidden and one lower lip (8,9)-(56,41),true", got, ok)
	}
	base := [4]float32{1, 1, 1, 1}
	if got := v.dimBorder(base, 8, 40); got != base {
		t.Fatalf("lower render lip scales = %v, want textured %v", got, base)
	}
	if got := v.dimBorder(base, 8, 41); got != ([4]float32{}) {
		t.Fatalf("row outside render lip scales = %v, want black", got)
	}
	if got := v.dimBorder(base, 7, 40); got != ([4]float32{}) {
		t.Fatalf("side margin beside render lip scales = %v, want black", got)
	}

	v.cam.Pan(-1e9, -1e9)
	if v.cam.X != 256 || v.cam.Y != 288 {
		t.Fatalf("near edge = (%v,%v), want the corner below the hidden first row (256,288)", v.cam.X, v.cam.Y)
	}
	v.cam.Pan(1e9, 1e9)
	if v.cam.X != 586 || v.cam.Y != 544 {
		t.Fatalf("far edge = (%v,%v), want one-row lower render lip at (586,544)", v.cam.X, v.cam.Y)
	}
	r := v.cam.VisibleTiles()
	if r.Col0 < 8 || r.Row0 < 8+hiddenTopRows || r.Col1 > w-8 {
		t.Fatalf("visible cells %+v reach a side or top black margin", r)
	}
	if r.Row1 != h-8+lowerRenderLipRows {
		t.Fatalf("visible bottom row = %d, want playable edge %d plus %d render-lip row",
			r.Row1, h-8, lowerRenderLipRows)
	}

	v.cam.SetZoom(0.5)
	if v.cam.X != -182 || v.cam.Y != 32 {
		t.Fatalf("zoomed-out origin = (%v,%v), want centred render span (-182,32)", v.cam.X, v.cam.Y)
	}
	left := (256 - v.cam.X) * v.cam.Zoom
	right := float64(v.cam.ViewW) - (1792-v.cam.X)*v.cam.Zoom
	top := (288 - v.cam.Y) * v.cam.Zoom
	bottom := float64(v.cam.ViewH) - (1312-v.cam.Y)*v.cam.Zoom
	if left != right || top != bottom || left != 219 || top != 128 {
		t.Fatalf("unavoidable slack left/right/top/bottom = %v/%v/%v/%v, want 219/219/128/128",
			left, right, top, bottom)
	}
}

func TestWideDisplacedCameraDoesNotCropADeepEdgeBecauseOfOffscreenRelief(t *testing.T) {
	const w, h = 64, 48
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h)
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	// A raised piece of the playable bottom edge is visible from the far left
	// but entirely outside a far-right widescreen view. The rejected global
	// intersection used it to crop the right-hand edge by 100 world pixels.
	for x := 8; x <= 16; x++ {
		g.Altitudes[(h-8)*w+x] = 100
	}

	v := newViewer(t, g)
	v.Layout(1920, 1080)
	v.cam.Pan(-1e9, 1e9)
	_, renderLeftBottom := v.proj.WorldCorner(8, h-8+lowerRenderLipRows)
	if got, want := v.cam.Y, float64(renderLeftBottom-v.cam.ViewH); got != want {
		t.Fatalf("left bottom = %v, want render-lip local edge %v", got, want)
	}

	v.cam.Pan(1e9, 1e9)
	_, renderRightBottom := v.proj.WorldCorner(w-8, h-8+lowerRenderLipRows)
	if got, want := v.cam.Y, float64(renderRightBottom-v.cam.ViewH); got != want {
		t.Fatalf("right bottom = %v, want render-lip local edge %v", got, want)
	}
	if got := v.minimapViewCells().Max.Y; got != h-8+lowerRenderLipRows {
		t.Fatalf("minimap viewport bottom row = %d, want playable edge %d plus %d render-lip row",
			got, h-8, lowerRenderLipRows)
	}
}

// TestWideDisplacedCameraReachesLowGroundBesideAHillInTheSameView is the owner's
// mission-81 report: the lower edge of the visible area stopped short of the
// playable surface, so the unit standing near the bottom row could not be
// reached at all. Shipped map 81 raises its render lip by 119 world pixels — 3.7
// cells — across the columns a wide view holds together with the flat right-hand
// edge, and the bound that took the SHALLOWEST of those columns hid every lower
// cell beside the hill.
//
// The relief is put on the lip row itself because that is the row the bound
// samples; a hill one row above moves no bound and would let the fixture pass
// against either reading.
func TestWideDisplacedCameraReachesLowGroundBesideAHillInTheSameView(t *testing.T) {
	const w, h = 64, 48
	const relief = 100
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h)
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	for x := 24; x <= 32; x++ {
		g.Altitudes[(h-8+lowerRenderLipRows)*w+x] = relief
	}

	v := newViewer(t, g)
	v.Layout(1920, 1080)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("mode = %v, want production displaced path", v.Mode())
	}

	_, flatBottom := v.proj.WorldCorner(w-8, h-8+lowerRenderLipRows)
	_, raisedBottom := v.proj.WorldCorner(24, h-8+lowerRenderLipRows)
	if got := flatBottom - raisedBottom; got != relief {
		t.Fatalf("fixture relief = %d px, want %d", got, relief)
	}

	v.cam.Pan(1e9, 1e9)
	if got, want := v.cam.Y, float64(flatBottom-v.cam.ViewH); got != want {
		t.Fatalf("bottom = %v, want the flat lip edge %v, not the raised %v",
			got, want, float64(raisedBottom-v.cam.ViewH))
	}
	r := v.cam.VisibleTiles()
	if r.Row1 != h-8+lowerRenderLipRows {
		t.Fatalf("visible bottom row = %d, want playable edge %d plus %d render-lip row",
			r.Row1, h-8, lowerRenderLipRows)
	}
	if r.Col1 < w-8 {
		t.Fatalf("visible columns %+v do not reach the right playable edge %d", r, w-8)
	}
}

// At the upper scroll limit the first playable row is above the view: the top
// screen row picks the row below it.
func TestUpperScrollLimitHidesTheFirstPlayableRow(t *testing.T) {
	const w, h = 64, 48
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h)
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	v := newViewer(t, g)
	v.Layout(1920, 1080)
	v.cam.Pan(-1e9, -1e9)
	if want := float64((8 + hiddenTopRows) * 32); v.cam.Y != want {
		t.Fatalf("upper scroll limit = %v, want %v", v.cam.Y, want)
	}
	if col, row, inside := v.groundCellAt(40, 1); !inside || row != 8+hiddenTopRows || col < 8 {
		t.Fatalf("top screen row picks (%d,%d,%v), want row %d", col, row, inside, 8+hiddenTopRows)
	}
	if got := v.minimapViewCells().Min.Y; got != 8+hiddenTopRows {
		t.Fatalf("minimap viewport top row = %d, want %d", got, 8+hiddenTopRows)
	}
}

func TestTopRowShownKeepsTheFirstPlayableRow(t *testing.T) {
	const w, h = 64, 48
	g := grid(w, h)
	g.Altitudes = make([]uint8, w*h)
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < 8 || x >= w-8 || y < 8 || y >= h-8 {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	v := newViewer(t, g)
	v.Layout(1920, 1080)
	v.SetTopRowShown(true)
	v.cam.Pan(-1e9, -1e9)
	if want := float64(8 * 32); v.cam.Y != want {
		t.Fatalf("upper scroll limit = %v, want %v", v.cam.Y, want)
	}
	if got := v.minimapViewCells().Min.Y; got != 8 {
		t.Fatalf("minimap viewport top row = %d, want 8", got)
	}
	if sc := v.dimBorder([4]float32{1, 1, 1, 1}, 20, 8); sc == ([4]float32{}) {
		t.Fatal("the kept first playable row still draws black")
	}
	v.SetTopRowShown(false)
	v.cam.Pan(-1e9, -1e9)
	if want := float64((8 + hiddenTopRows) * 32); v.cam.Y != want {
		t.Fatalf("hidden upper limit = %v, want %v", v.cam.Y, want)
	}
}
