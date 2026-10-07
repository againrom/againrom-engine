package terrain_test

import (
	"math"
	"testing"

	"againrom/pkg/render/terrain"
)

// identityLight forces every vertex to level 64 (the unattenuated identity row)
// regardless of heights: Range 0 removes the slope term, so
// L = (0>>1) + 0x20 + 0x20 = 64 and level = int(0.5*(64+64)) = 64.
var identityLight = terrain.Light{Theta: math.Pi / 4, Ambient: 0x20, Range: 0, SkyTint: [3]uint8{0, 0, 0}}

func TestCompositeLit(t *testing.T) {
	ts := terrain.LoadTileset(compositeSource())

	// (a) Identity: uniform level 64 must reproduce Composite exactly. The grid
	// mixes land, water, an impassable-dirt cell and one absent slot, so every
	// path (terrain, dirt overlay, placeholder) is exercised.
	g2 := terrain.Grid{Width: 2, Height: 2, Tiles: []uint16{
		tileWord(0, 0, 3),
		tileWord(8, 0, 5),
		tileWord(0, 0, 2) | terrain.ImpassableBit,
		tileWord(1, 0, 0),
	}}
	lit, err := terrain.CompositeLit(ts, g2, make([]uint8, 4), identityLight, 1)
	if err != nil {
		t.Fatalf("CompositeLit identity: %v", err)
	}
	plain, err := terrain.Composite(ts, g2, 1)
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if lit.Placeholders != 1 || plain.Placeholders != 1 {
		t.Fatalf("placeholders lit=%d plain=%d, want 1 each", lit.Placeholders, plain.Placeholders)
	}
	b := plain.Image.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			if lit.Image.RGBAAt(x, y) != plain.Image.RGBAAt(x, y) {
				t.Fatalf("identity mismatch at (%d,%d): lit %v plain %v",
					x, y, lit.Image.RGBAAt(x, y), plain.Image.RGBAAt(x, y))
			}
		}
	}

	// (b) Relief: a height ramp along x shades cells non-uniformly. Every cell is
	// the same land tile (sub 0), so the only variation is the shading. The cell's
	// top-left pixel is InterpRow at (0,0) == that vertex's level, so it must equal
	// ShadeRGBA(base, tint, level) with the tested LevelGrid as oracle. The last
	// column exercises the DD6 far-edge clamp (its +1 corner is clamped in-grid).
	const W, H = 4, 4
	heights := make([]uint8, W*H)
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			heights[y*W+x] = uint8(x * 40) // 0, 40, 80, 120 across each row
		}
	}
	tiles := make([]uint16, W*H)
	for i := range tiles {
		tiles[i] = tileWord(0, 0, 0) // land tile1-00 sub 0 everywhere
	}
	gr := terrain.Grid{Width: W, Height: H, Tiles: tiles}

	lit2, err := terrain.CompositeLit(ts, gr, heights, terrain.DefaultDaytime, 1)
	if err != nil {
		t.Fatalf("CompositeLit relief: %v", err)
	}
	if lit2.Placeholders != 0 {
		t.Fatalf("relief placeholders = %d, want 0 (every cell is a present tile)", lit2.Placeholders)
	}

	levels := terrain.LevelGrid(heights, W, H, terrain.DefaultDaytime)
	base := rampColor(fillOf(landBase, 0)) // the land tile's sub-0 colour
	distinct := map[uint8]bool{}
	for row := 0; row < H; row++ {
		for col := 0; col < W; col++ {
			lvl := levels[row*W+col]
			distinct[lvl] = true
			want := terrain.ShadeRGBA(base, [3]uint8{0, 0, 0}, int(lvl))
			got := lit2.Image.RGBAAt(col*terrain.CellSize, row*terrain.CellSize)
			if got != want {
				t.Fatalf("cell (%d,%d) top-left = %v, want %v (level %d)", col, row, got, want, lvl)
			}
		}
	}
	if len(distinct) < 2 {
		t.Fatalf("relief is uniform (%d distinct levels); the fixture must shade non-uniformly", len(distinct))
	}

	// (c) The shaded relief must genuinely differ from the unshaded render, or the
	// story's whole point (shading) is untested.
	unshaded, err := terrain.Composite(ts, gr, 1)
	if err != nil {
		t.Fatalf("Composite relief: %v", err)
	}
	same := true
	for y := 0; y < H*terrain.CellSize && same; y++ {
		for x := 0; x < W*terrain.CellSize; x++ {
			if lit2.Image.RGBAAt(x, y) != unshaded.Image.RGBAAt(x, y) {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("shaded relief render is identical to the unshaded one; shading was not applied")
	}

	// (d) Validation: every bad argument is rejected atomically with a nil render.
	good := terrain.Grid{Width: 2, Height: 1, Tiles: []uint16{0, 0}}
	for _, tc := range []struct {
		name    string
		ts      *terrain.Tileset
		g       terrain.Grid
		heights []uint8
		scale   int
	}{
		{"nil tileset", nil, good, make([]uint8, 2), 1},
		{"zero width", ts, terrain.Grid{Width: 0, Height: 1, Tiles: nil}, nil, 1},
		{"short tiles", ts, terrain.Grid{Width: 2, Height: 1, Tiles: []uint16{0}}, make([]uint8, 2), 1},
		{"short heights", ts, good, make([]uint8, 1), 1},
		{"long heights", ts, good, make([]uint8, 3), 1},
		{"zero scale", ts, good, make([]uint8, 2), 0},
	} {
		r, err := terrain.CompositeLit(tc.ts, tc.g, tc.heights, terrain.DefaultDaytime, tc.scale)
		if err == nil {
			t.Errorf("%s: expected an error, got a render", tc.name)
		}
		if r != nil {
			t.Errorf("%s: expected a nil render on error", tc.name)
		}
	}
}
