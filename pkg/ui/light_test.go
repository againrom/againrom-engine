package ui

// The viewer's lighting state: Lit(), SetUnshaded, the pure per-tile corner
// scales cornerScales computes, and the two draw loops' pure vertex builders
// -- withScales and flatTileVertices -- that carry those scales onto screen
// geometry (SC-2, SC-3, SC-6, SC-7, SC-8, SC-12).
//
// Nothing here opens a window: cornerScales, Lit, SetUnshaded, withScales,
// flatTileVertices and tileVertices are all reachable over a plain Viewer
// built by NewViewer, or, for withScales, over a bare struct literal with no
// Viewer at all — an ebiten.Vertex is a Go value, so every assertion below
// reads struct fields (SC-8).

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// litTileset is a Tileset with exactly one populated slot — g=0, b=0 (slot
// 0) — holding one solid sub-cell. A tile word that resolves elsewhere
// finds no image and stays a placeholder; a word resolving to slot 0 has
// one.
func litTileset() *terrain.Tileset {
	pal := color.Palette{color.RGBA{R: 1, G: 1, B: 1, A: 0xff}}
	img := image.NewPaletted(image.Rect(0, 0, terrain.CellSize, terrain.CellSize), pal)
	var ts terrain.Tileset
	ts.Slots[0] = &terrain.Strip{SubCells: []*image.Paletted{img}}
	return &ts
}

// slotWord builds a type1 tile word naming strip group g, blend column 0, sub
// 0 — g*4 matches TileRef.Slot's g*blendColumns+b for b=0, so word 0 names
// slot 0 (populated by litTileset) and word (1<<6) names slot 4 (absent).
func slotWord(g int) uint16 { return uint16(g << 6) }

func TestCornerScalesUnlitIsAllOnes(t *testing.T) {
	unlit := [4]float32{1, 1, 1, 1}

	t.Run("no altitude grid at all", func(t *testing.T) {
		v, err := NewViewer("m", grid(cliffW, cliffH), litTileset())
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		if v.Lit() {
			t.Fatal("fixture must be unlit: no altitude grid")
		}
		if got := v.cornerScales(1, 1); got != unlit {
			t.Fatalf("cornerScales = %v, want all-1", got)
		}
	})

	t.Run("unshaded diagnostic on over a sloped map", func(t *testing.T) {
		v, err := NewViewer("m", cliffGrid(), litTileset())
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.SetUnshaded(true)
		if v.Lit() {
			t.Fatal("SetUnshaded(true) left the viewer lit")
		}
		if got := v.cornerScales(1, 1); got != unlit {
			t.Fatalf("cornerScales = %v, want all-1 (unshaded)", got)
		}
	})
}

func TestSetUnshadedTogglesLitWithoutRebuilding(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("a valid altitude grid must start lit")
	}
	v.SetUnshaded(true)
	if v.Lit() {
		t.Fatal("SetUnshaded(true) did not turn lighting off")
	}
	v.SetUnshaded(false)
	if !v.Lit() {
		t.Fatal("SetUnshaded(false) did not turn lighting back on")
	}
}

func TestCornerScalesMatchesShadeScaleOfCornerLevels(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	distinct := 0
	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			got := v.cornerScales(tx, ty)
			lv := terrain.CornerLevels(v.levels, cliffW, cliffH, tx, ty)
			want := [4]float32{
				terrain.ShadeScale(int(lv[0])),
				terrain.ShadeScale(int(lv[1])),
				terrain.ShadeScale(int(lv[2])),
				terrain.ShadeScale(int(lv[3])),
			}
			if got != want {
				t.Fatalf("tile (%d,%d) cornerScales = %v, want %v (levels %v)", tx, ty, got, want, lv)
			}
			if lv[0] != lv[1] || lv[0] != lv[2] || lv[0] != lv[3] {
				distinct++
			}
		}
	}
	if distinct == 0 {
		t.Fatal("no tile in the fixture had four unequal corner levels; a transposed corner " +
			"or a single scale reused for all four could not have been detected")
	}
}

func TestCornerScalesPlaceholderStaysUnlitWhileNeighboursAreLit(t *testing.T) {
	g := grid(2, 1, slotWord(0), slotWord(1)) // (0,0) -> slot 0 (present); (1,0) -> slot 4 (absent)
	g.Altitudes = make([]uint8, 2)            // flat: every vertex levels to 46

	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	if got, want := v.cornerScales(1, 0), ([4]float32{1, 1, 1, 1}); got != want {
		t.Fatalf("placeholder cell cornerScales = %v, want all-1", got)
	}

	wantLit := terrain.ShadeScale(46) // flat daytime ground (spec, DefaultDaytime)
	if wantLit == 1 {
		t.Fatal("fixture drift: flat ground no longer levels to a non-identity multiplier")
	}
	for i, s := range v.cornerScales(0, 0) {
		if s != wantLit {
			t.Fatalf("lit neighbour corner %d = %v, want %v — the placeholder must not have "+
				"suppressed its own neighbour's lighting", i, s, wantLit)
		}
	}
}

func TestLitAndModeAreIndependentPredicates(t *testing.T) {
	valid := altGrid(4, 4, make([]uint8, 16)...) // flat, but a valid altitude layer
	invalid := grid(4, 4)                        // no altitude layer at all

	for _, tc := range []struct {
		name  string
		g     terrain.Grid
		valid bool
	}{
		{"valid altitude grid", valid, true},
		{"no altitude grid", invalid, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("m", tc.g, litTileset())
			if err != nil {
				t.Fatalf("NewViewer: %v", err)
			}
			if (v.levels != nil) != tc.valid {
				t.Fatalf("levels != nil is %v, want %v", v.levels != nil, tc.valid)
			}

			for _, unshaded := range []bool{false, true} {
				v.SetUnshaded(unshaded)
				for _, flat := range []bool{false, true} {
					v.SetFlat(flat)

					wantLit := tc.valid && !unshaded
					if got := v.Lit(); got != wantLit {
						t.Fatalf("unshaded=%v flat=%v: Lit() = %v, want %v", unshaded, flat, got, wantLit)
					}

					wantMode := ModeFlat
					if tc.valid && !flat {
						wantMode = ModeDisplaced
					}
					if got := v.Mode(); got != wantMode {
						t.Fatalf("unshaded=%v flat=%v: Mode() = %v, want %v", unshaded, flat, got, wantMode)
					}
				}
			}
		})
	}
}

func TestCornerScalesPassLeavesAltitudesUntouched(t *testing.T) {
	alts := append([]uint8(nil), bigCliffAlts...)
	before := append([]uint8(nil), alts...)

	g := grid(bigCliffW, bigCliffH)
	g.Altitudes = alts
	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	for i := range before {
		if alts[i] != before[i] {
			t.Fatalf("altitude %d changed from %d to %d right after construction", i, before[i], alts[i])
		}
	}

	for ty := 0; ty < bigCliffH; ty++ {
		for tx := 0; tx < bigCliffW; tx++ {
			_ = v.cornerScales(tx, ty)
		}
	}

	for i := range before {
		if alts[i] != before[i] {
			t.Fatalf("altitude %d changed from %d to %d after a full pass over every tile's corner scales",
				i, before[i], alts[i])
		}
	}
}

func TestLevelsFollowsTheSameGuardAsProjection(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    terrain.Grid
	}{
		{"one entry long", altGrid(2, 2, 1, 2, 3, 4, 5)},
		{"a whole extra row", altGrid(2, 2, 1, 2, 3, 4, 5, 6)},
		{"one entry short", altGrid(2, 2, 1, 2, 3)},
		{"no altitude layer at all", grid(2, 2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newViewer(t, tc.g)
			if v.proj != nil {
				t.Fatal("fixture drift: this altitude length must already be invalid for proj")
			}
			if v.levels != nil {
				t.Fatalf("levels != nil but proj == nil; DD-1 requires the two to decide " +
					"this grid's validity identically")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// T3: withScales and flatTileVertices, and the geometry half of AC-3/AC-4.
// ---------------------------------------------------------------------------

func TestWithScalesAppliesEachScaleToItsOwnCornerAndLeavesGeometryAlone(t *testing.T) {
	in := [4]ebiten.Vertex{
		{DstX: 1, DstY: 2, SrcX: 3, SrcY: 4, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 5, DstY: 6, SrcX: 7, SrcY: 8, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 9, DstY: 10, SrcX: 11, SrcY: 12, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: 13, DstY: 14, SrcX: 15, SrcY: 16, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	sc := [4]float32{2, 3, 5, 7} // four distinct primes: no pair could be confused for another

	got := withScales(in, sc)

	names := [4]string{"TL", "TR", "BL", "BR"}
	for i := range got {
		if got[i].DstX != in[i].DstX || got[i].DstY != in[i].DstY ||
			got[i].SrcX != in[i].SrcX || got[i].SrcY != in[i].SrcY {
			t.Fatalf("%s geometry = (Dst %v,%v Src %v,%v), want it unchanged from (%v,%v %v,%v)",
				names[i], got[i].DstX, got[i].DstY, got[i].SrcX, got[i].SrcY,
				in[i].DstX, in[i].DstY, in[i].SrcX, in[i].SrcY)
		}
		if got[i].ColorA != in[i].ColorA {
			t.Fatalf("%s ColorA = %v, want %v unchanged (alpha is untouched, FR-2)", names[i], got[i].ColorA, in[i].ColorA)
		}
		if got[i].ColorA == 0 {
			t.Fatalf("%s ColorA is 0 -- ebiten draws this fully transparent", names[i])
		}
		if got[i].ColorR != sc[i] || got[i].ColorG != sc[i] || got[i].ColorB != sc[i] {
			t.Fatalf("%s colour = (%v,%v,%v), want (%v,%v,%v) -- its OWN scale, not another corner's",
				names[i], got[i].ColorR, got[i].ColorG, got[i].ColorB, sc[i], sc[i], sc[i])
		}
	}
}

func TestFlatTileVerticesIsTheUndisplacedLatticeWithUnitColour(t *testing.T) {
	v := newViewer(t, grid(cliffW, cliffH)) // no altitude layer: flat mode
	cam := v.Camera()

	corners := [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	names := [4]string{"TL", "TR", "BL", "BR"}
	b := SubCellBounds()
	wantSrc := [4][2]float32{
		{float32(b.Min.X), float32(b.Min.Y)},
		{float32(b.Max.X), float32(b.Min.Y)},
		{float32(b.Min.X), float32(b.Max.Y)},
		{float32(b.Max.X), float32(b.Max.Y)},
	}

	distinct := 0
	for _, state := range cameraStates() {
		state(v)
		for row := 0; row < cliffH; row++ {
			for col := 0; col < cliffW; col++ {
				got := flatTileVertices(cam, col, row)

				wantTLx, wantTLy := v.flatTileScreen(col, row)
				if got[0].DstX != float32(wantTLx) || got[0].DstY != float32(wantTLy) {
					t.Fatalf("tile (%d,%d) TL = (%v,%v), want flatTileScreen's (%v,%v)",
						col, row, got[0].DstX, got[0].DstY, wantTLx, wantTLy)
				}

				for i, c := range corners {
					wx := float64((col + c[0]) * terrain.CellSize)
					wy := float64((row + c[1]) * terrain.CellSize)
					sx, sy := cam.WorldToScreen(wx, wy)
					if got[i].DstX != float32(sx) || got[i].DstY != float32(sy) {
						t.Fatalf("tile (%d,%d) %s = (%v,%v), want the lattice corner (%v,%v)",
							col, row, names[i], got[i].DstX, got[i].DstY, float32(sx), float32(sy))
					}
					if got[i].SrcX != wantSrc[i][0] || got[i].SrcY != wantSrc[i][1] {
						t.Fatalf("tile (%d,%d) %s src = (%v,%v), want (%v,%v)",
							col, row, names[i], got[i].SrcX, got[i].SrcY, wantSrc[i][0], wantSrc[i][1])
					}
					if got[i].ColorR != 1 || got[i].ColorG != 1 || got[i].ColorB != 1 || got[i].ColorA != 1 {
						t.Fatalf("tile (%d,%d) %s colour = (%v,%v,%v,%v), want 1,1,1,1",
							col, row, names[i], got[i].ColorR, got[i].ColorG, got[i].ColorB, got[i].ColorA)
					}
				}

				seen := map[[2]float32]bool{}
				for i := range got {
					seen[[2]float32{got[i].DstX, got[i].DstY}] = true
				}
				if len(seen) == 4 {
					distinct++
				}
			}
		}
	}
	if distinct == 0 {
		t.Fatal("no tested tile had four distinct corners, so a transposed corner could have passed")
	}
}

// TestFlatCornersRetainFlatTileVerticesGeometryAfterScaling - AC-4 (geometry
// half): composing withScales onto flatTileVertices, exactly as drawFlat now
// does, leaves the Dst/Src fields byte-identical to flatTileVertices' own
// output -- so the flat destination positions are the un-displaced lattice
// through the camera, unchanged by lighting.
func TestFlatCornersRetainFlatTileVerticesGeometryAfterScaling(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset()) // sloped and lit, but drawFlat ignores proj
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFlat(true) // request flat mode; the map is still lit (AC-5)
	if v.Mode() != ModeFlat || !v.Lit() {
		t.Fatalf("setup: Mode()=%v Lit()=%v, want flat and lit", v.Mode(), v.Lit())
	}

	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			raw := flatTileVertices(v.Camera(), tx, ty)
			scaled := withScales(raw, v.cornerScales(tx, ty))
			for i := range raw {
				if scaled[i].DstX != raw[i].DstX || scaled[i].DstY != raw[i].DstY ||
					scaled[i].SrcX != raw[i].SrcX || scaled[i].SrcY != raw[i].SrcY {
					t.Fatalf("tile (%d,%d) corner %d geometry changed by scaling: %v -> %v",
						tx, ty, i, raw[i], scaled[i])
				}
				if scaled[i].ColorA != 1 {
					t.Fatalf("tile (%d,%d) corner %d alpha = %v, want 1 (opaque)", tx, ty, i, scaled[i].ColorA)
				}
			}
		}
	}
}

func TestDisplacedCornersRetainTileVerticesGeometryAfterScaling(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if v.Mode() != ModeDisplaced || !v.Lit() {
		t.Fatalf("setup: Mode()=%v Lit()=%v, want displaced and lit", v.Mode(), v.Lit())
	}

	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			raw := tileVertices(v.Camera(), v.proj, tx, ty)
			scaled := withScales(raw, v.cornerScales(tx, ty))
			for i := range raw {
				if scaled[i].DstX != raw[i].DstX || scaled[i].DstY != raw[i].DstY ||
					scaled[i].SrcX != raw[i].SrcX || scaled[i].SrcY != raw[i].SrcY {
					t.Fatalf("tile (%d,%d) corner %d geometry changed by scaling: %v -> %v",
						tx, ty, i, raw[i], scaled[i])
				}
				if scaled[i].ColorA != 1 {
					t.Fatalf("tile (%d,%d) corner %d alpha = %v, want 1 (opaque)", tx, ty, i, scaled[i].ColorA)
				}
			}
		}
	}
}

func TestUnshadedDrawIsUnitColourAtThePreChangeFlatLattice(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetUnshaded(true)
	if v.Lit() {
		t.Fatal("SetUnshaded(true) left the viewer lit")
	}

	for ty := 0; ty < cliffH; ty++ {
		for tx := 0; tx < cliffW; tx++ {
			verts := withScales(flatTileVertices(v.Camera(), tx, ty), v.cornerScales(tx, ty))
			wantX, wantY := v.flatTileScreen(tx, ty)
			if verts[0].DstX != float32(wantX) || verts[0].DstY != float32(wantY) {
				t.Fatalf("tile (%d,%d) TL = (%v,%v), want flatTileScreen's (%v,%v)",
					tx, ty, verts[0].DstX, verts[0].DstY, wantX, wantY)
			}
			for i, c := range verts {
				if c.ColorR != 1 || c.ColorG != 1 || c.ColorB != 1 || c.ColorA != 1 {
					t.Fatalf("tile (%d,%d) corner %d colour = (%v,%v,%v,%v), want 1,1,1,1 (unshaded)",
						tx, ty, i, c.ColorR, c.ColorG, c.ColorB, c.ColorA)
				}
			}
		}
	}
}

func TestCornerScalesEqualLevelsYieldEqualMultipliers(t *testing.T) {
	g := altGrid(4, 4, make([]uint8, 16)...) // flat: every vertex levels to 46
	v, err := NewViewer("m", g, litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}

	want := terrain.ShadeScale(46)
	if want == 1 {
		t.Fatal("fixture drift: flat ground no longer levels to a non-identity multiplier")
	}
	for ty := 0; ty < 4; ty++ {
		for tx := 0; tx < 4; tx++ {
			got := v.cornerScales(tx, ty)
			if got != ([4]float32{want, want, want, want}) {
				t.Fatalf("tile (%d,%d) cornerScales = %v, want all four equal to %v", tx, ty, got, want)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// T3b: the draw call becomes observable (SC-13).
//
// T3's own suite above tests withScales, flatTileVertices and cornerScales as
// pure functions, recomposing them inside each test body -- which is exactly
// why deleting withScales from BOTH drawFlat and drawDisplaced left every
// package green, measured before this task began. recordingTarget closes that
// gap: it satisfies triangleTarget with a plain Go struct (no window, no
// graphics context -- SC-8) and lets a test drive drawFlat/drawDisplaced
// themselves and read back exactly what each call to DrawTriangles received.
// ---------------------------------------------------------------------------

// recordedTriangles is one call the recording target observed, with its
// vertex and index slices copied out so a later DrawTriangles call reusing
// the same backing array cannot retroactively change what an earlier one is
// compared against.
type recordedTriangles struct {
	verts   []ebiten.Vertex
	indices []uint16
}

// recordingTarget satisfies triangleTarget by recording every call rather
// than only the last, or only the first: SC-13 requires every visible tile's
// submission to be checked, not one tile standing in for the whole loop.
type recordingTarget struct {
	calls []recordedTriangles
}

func (r *recordingTarget) DrawTriangles(vertices []ebiten.Vertex, indices []uint16, img *ebiten.Image, options *ebiten.DrawTrianglesOptions) {
	r.calls = append(r.calls, recordedTriangles{
		verts:   append([]ebiten.Vertex(nil), vertices...),
		indices: append([]uint16(nil), indices...),
	})
}

// TestDrawDisplacedSubmitsWithScaledTileVerticesThroughSharedIndices - SC-13:
// drawDisplaced driven against a recordingTarget over a sloped, lit map.
//
// The expected tile order and set come from forEachDisplacedTile itself
// (displacedOrder, mode_test.go's own helper), not a re-derivation, so this
// checks the loop's OWN reachable set rather than asserting a second copy of
// it agrees. A call-count mismatch fails on its own -- one tile standing in
// for the whole visible set, or a dropped tile, cannot pass. Each recorded
// call's vertices must equal withScales of that SAME tile's tileVertices
// output at that tile's own cornerScales (not a neighbour's), and its indices
// must equal quadIndices itself, by value -- a length-only comparison would
// not catch a different split. Dropping withScales from the loop, or
// swapping which tile's scale reaches which tile's geometry, each lands on a
// visibly wrong recorded value.
func TestDrawDisplacedSubmitsWithScaledTileVerticesThroughSharedIndices(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if v.Mode() != ModeDisplaced || !v.Lit() {
		t.Fatalf("setup: Mode()=%v Lit()=%v, want displaced and lit", v.Mode(), v.Lit())
	}

	want := displacedOrder(v)
	if len(want) == 0 {
		t.Fatal("fixture draws nothing; the loop is not exercised")
	}

	rec := &recordingTarget{}
	v.drawDisplaced(rec)

	if len(rec.calls) != len(want) {
		t.Fatalf("drawDisplaced submitted %d DrawTriangles calls, want %d — one per tile "+
			"forEachDisplacedTile visits, in its own order", len(rec.calls), len(want))
	}

	distinct := 0
	for i, tile := range want {
		tx, ty := tile[0], tile[1]
		call := rec.calls[i]

		sc := v.cornerScales(tx, ty)
		wantVerts := terrainTextureVertices(withScales(tileVertices(v.Camera(), v.proj, tx, ty), sc))
		if len(call.verts) != 4 {
			t.Fatalf("tile (%d,%d): submitted %d vertices, want 4", tx, ty, len(call.verts))
		}
		for c := range wantVerts {
			if call.verts[c] != wantVerts[c] {
				t.Fatalf("tile (%d,%d) corner %d: submitted %+v, want terrainTextureVertices(withScales(tileVertices(...), "+
					"cornerScales(...)) = %+v", tx, ty, c, call.verts[c], wantVerts[c])
			}
		}

		if len(call.indices) != len(quadIndices) {
			t.Fatalf("tile (%d,%d): submitted %d indices, want %d", tx, ty, len(call.indices), len(quadIndices))
		}
		var gotIdx [6]uint16
		copy(gotIdx[:], call.indices)
		if gotIdx != quadIndices {
			t.Fatalf("tile (%d,%d): submitted indices %v, want quadIndices itself %v", tx, ty, gotIdx, quadIndices)
		}

		if sc[0] != sc[1] || sc[0] != sc[2] || sc[0] != sc[3] {
			distinct++
		}
	}
	if distinct == 0 {
		t.Fatal("no visited tile had four unequal corner scales; dropping withScales could not have been caught here")
	}
}

func TestDrawFlatSubmitsWithScaledFlatVerticesThroughSharedIndices(t *testing.T) {
	v, err := NewViewer("m", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.SetFlat(true) // request flat mode; the map is still lit (AC-5)
	if v.Mode() != ModeFlat || !v.Lit() {
		t.Fatalf("setup: Mode()=%v Lit()=%v, want flat and lit", v.Mode(), v.Lit())
	}

	r := v.Camera().VisibleTiles()
	var want [][2]int
	for row := r.Row0; row < r.Row1; row++ {
		for col := r.Col0; col < r.Col1; col++ {
			want = append(want, [2]int{col, row})
		}
	}
	if len(want) == 0 {
		t.Fatal("fixture draws nothing; the loop is not exercised")
	}

	rec := &recordingTarget{}
	v.drawFlat(rec)

	if len(rec.calls) != len(want) {
		t.Fatalf("drawFlat submitted %d DrawTriangles calls, want %d — one per visible tile",
			len(rec.calls), len(want))
	}

	distinct := 0
	for i, tile := range want {
		tx, ty := tile[0], tile[1]
		call := rec.calls[i]

		sc := v.cornerScales(tx, ty)
		wantVerts := terrainTextureVertices(withScales(flatTileVertices(v.Camera(), tx, ty), sc))
		if len(call.verts) != 4 {
			t.Fatalf("tile (%d,%d): submitted %d vertices, want 4", tx, ty, len(call.verts))
		}
		for c := range wantVerts {
			if call.verts[c] != wantVerts[c] {
				t.Fatalf("tile (%d,%d) corner %d: submitted %+v, want terrainTextureVertices(withScales(flatTileVertices(...), "+
					"cornerScales(...)) = %+v", tx, ty, c, call.verts[c], wantVerts[c])
			}
		}

		if len(call.indices) != len(quadIndices) {
			t.Fatalf("tile (%d,%d): submitted %d indices, want %d", tx, ty, len(call.indices), len(quadIndices))
		}
		var gotIdx [6]uint16
		copy(gotIdx[:], call.indices)
		if gotIdx != quadIndices {
			t.Fatalf("tile (%d,%d): submitted indices %v, want quadIndices itself %v", tx, ty, gotIdx, quadIndices)
		}

		if sc[0] != sc[1] || sc[0] != sc[2] || sc[0] != sc[3] {
			distinct++
		}
	}
	if distinct == 0 {
		t.Fatal("no visible tile had four unequal corner scales; dropping withScales could not have been caught here")
	}
}
