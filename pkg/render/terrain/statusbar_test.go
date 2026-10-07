package terrain_test

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// The installed classes the wide-unit cases below are written from: the
// dragon's and the troll's canvas centre, TileSize and selection box as
// units.reg holds them, the human mage's for the one-cell rule, and a one-cell
// class whose box is wider than its cell.
var (
	sbDragon = &terrain.UnitClass{Width: 160, Height: 160, CenterX: 80, CenterY: 80, TileSize: 3,
		Selection: image.Rect(32, 32, 128, 128)}
	sbTroll = &terrain.UnitClass{Width: 128, Height: 128, CenterX: 64, CenterY: 80, TileSize: 2,
		Selection: image.Rect(34, 30, 94, 100)}
	sbMage = &terrain.UnitClass{Width: 128, Height: 128, CenterX: 64, CenterY: 78, TileSize: 1,
		Selection: image.Rect(48, 48, 80, 90)}
	sbLance = &terrain.UnitClass{Width: 128, Height: 128, CenterX: 64, CenterY: 78, TileSize: 1,
		Selection: image.Rect(40, 40, 88, 88)}
)

// TestStatusBarRectGeometry writes each expected rectangle out from the
// contract rather than from statusbar.go. A unit drawn without a class, or of
// a one-cell class, has a 32 by 4 bar centred on its whole footprint, its top
// edge 16 rows (health) or 12 rows (mana) above the footprint's top edge.
// Cell (1,1) on a 4 by 4 map has its footprint at [32,64) x [32,64) and its
// centre, where a body's class centre lands, at (48,48). A class wider than
// one cell spans its selection box placed about that centre, health 2 rows
// above the box's top edge and mana directly under it.
func TestStatusBarRectGeometry(t *testing.T) {
	const cols, rows = 4, 4
	unboxed := &terrain.UnitClass{Width: 128, Height: 128, CenterX: 64, CenterY: 64, TileSize: 2,
		Selection: image.Rectangle{Min: image.Pt(-1, -1), Max: image.Pt(-1, -1)}}
	for _, tc := range []struct {
		name     string
		kind     terrain.StatusBarKind
		col, row int
		side     int
		class    *terrain.UnitClass
		want     image.Rectangle
		ok       bool
	}{
		{"health over one cell", terrain.HealthBar, 1, 1, 1, nil, image.Rect(32, 16, 64, 20), true},
		{"mana directly under health", terrain.ManaBar, 1, 1, 1, nil, image.Rect(32, 20, 64, 24), true},
		{"a side below one is one", terrain.HealthBar, 1, 1, 0, nil, image.Rect(32, 16, 64, 20), true},
		{"no class, two cells wide, centres on both", terrain.HealthBar, 1, 1, 2, nil, image.Rect(48, 16, 80, 20), true},
		{"no class, three cells wide, centres on all three", terrain.ManaBar, 1, 1, 3, nil, image.Rect(64, 20, 96, 24), true},
		{"a one-cell class keeps the cell rule", terrain.HealthBar, 1, 1, 1, sbMage, image.Rect(32, 16, 64, 20), true},
		{"a one-cell class with a wider box keeps the cell rule", terrain.ManaBar, 1, 1, 1, sbLance, image.Rect(32, 20, 64, 24), true},
		{"the dragon's health spans its box", terrain.HealthBar, 1, 1, 3, sbDragon, image.Rect(0, -2, 96, 2), true},
		{"the dragon's mana under its health", terrain.ManaBar, 1, 1, 3, sbDragon, image.Rect(0, 2, 96, 6), true},
		{"the troll's health spans its box", terrain.HealthBar, 1, 1, 2, sbTroll, image.Rect(18, -4, 78, 0), true},
		{"a wide class without a box keeps the cell rule", terrain.HealthBar, 1, 1, 2, unboxed, image.Rect(48, 16, 80, 20), true},
		{"row zero stands above the map", terrain.HealthBar, 0, 0, 1, nil, image.Rect(0, -16, 32, -12), true},
		{"a column off the map", terrain.HealthBar, 4, 1, 1, nil, image.Rectangle{}, false},
		{"a row off the map", terrain.ManaBar, 1, 4, 3, sbDragon, image.Rectangle{}, false},
		{"a negative cell", terrain.HealthBar, -1, 0, 1, nil, image.Rectangle{}, false},
		{"a kind past mana", terrain.StatusBarKind(2), 1, 1, 3, sbDragon, image.Rectangle{}, false},
		{"a kind before health", terrain.StatusBarKind(-1), 1, 1, 1, nil, image.Rectangle{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := terrain.StatusBarRect(tc.kind, tc.col, tc.row, cols, rows, tc.side, tc.class)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("StatusBarRect = %v, %v; want %v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestStatusBarsOfEveryInstalledWideClass pins the bars of each class wider
// than one cell, its centre, TileSize and selection box as units.reg holds
// them on both roots: the width and the health bar's place from the body's
// anchor, the class centre's pixel on the anchor cell's centre, as DIV-1481
// lists them. Mana stands 4 rows under health.
func TestStatusBarsOfEveryInstalledWideClass(t *testing.T) {
	const col, row = 5, 5
	anchor := image.Pt(col*32+16, row*32+16)
	for _, tc := range []struct {
		name             string
		centre           image.Point
		tile             int
		box              image.Rectangle
		left, above, wid int
	}{
		{"Catapult 1", image.Pt(64, 64), 2, image.Rect(32, 32, 96, 96), 32, 34, 64},
		{"Catapult 2", image.Pt(64, 64), 2, image.Rect(32, 32, 96, 96), 32, 34, 64},
		{"Ogre", image.Pt(64, 78), 2, image.Rect(34, 30, 94, 100), 30, 50, 60},
		{"Fat troll", image.Pt(64, 80), 2, image.Rect(34, 30, 94, 100), 30, 52, 60},
		{"Dragon", image.Pt(80, 80), 3, image.Rect(32, 32, 128, 128), 48, 50, 96},
		{"Death Star", image.Pt(64, 84), 2, image.Rect(32, 32, 96, 96), 32, 54, 64},
	} {
		c := &terrain.UnitClass{CenterX: tc.centre.X, CenterY: tc.centre.Y, TileSize: tc.tile, Selection: tc.box}
		health := image.Rect(anchor.X-tc.left, anchor.Y-tc.above, anchor.X-tc.left+tc.wid, anchor.Y-tc.above+4)
		for kind, want := range [...]image.Rectangle{terrain.HealthBar: health, terrain.ManaBar: health.Add(image.Pt(0, 4))} {
			if got, ok := terrain.StatusBarRect(terrain.StatusBarKind(kind), col, row, 16, 16, tc.tile, c); !ok || got != want {
				t.Errorf("%s, kind %d: StatusBarRect = %v, %v; want %v", tc.name, kind, got, ok, want)
			}
		}
	}
}

// TestStatusBarFillTruncatesAndClamps: the interior, the bar's width less its
// two 4-column caps, fills interior*value/maximum columns, truncated; it never
// shrinks as the value rises, is empty at or below zero and full at or above
// the maximum. A value below zero is a fallen unit's ordinary health, and
// nothing stops a value above its maximum.
func TestStatusBarFillTruncatesAndClamps(t *testing.T) {
	for _, tc := range []struct{ width, value, maximum, fill int }{
		{32, 60, 60, 24}, {32, 59, 60, 23}, {32, 45, 60, 18}, {32, 30, 60, 12}, {32, 3, 60, 1}, {32, 2, 60, 0},
		{32, 10, 40, 6}, {32, 0, 60, 0}, {32, -9, 60, 0}, {32, 61, 60, 24},
		{96, 60, 60, 88}, {96, 59, 60, 86}, {96, 45, 60, 66}, {96, 1, 88, 1}, {96, 1, 89, 0}, {96, 61, 60, 88},
		{60, 45, 60, 39}, {64, 45, 60, 42}, {6, 5, 10, 0},
	} {
		if fill, ok := terrain.StatusBarFill(tc.width, tc.value, tc.maximum); !ok || fill != tc.fill {
			t.Errorf("StatusBarFill(%d, %d, %d) = %d, %v; want %d, true", tc.width, tc.value, tc.maximum, fill, ok, tc.fill)
		}
	}
	for _, maximum := range []int{0, -5} {
		if fill, ok := terrain.StatusBarFill(32, 3, maximum); ok || fill != 0 {
			t.Errorf("StatusBarFill(32, 3, %d) = %d, %v; a unit without the pool has no bar", maximum, fill, ok)
		}
	}
	for _, width := range []int{32, 60, 64, 96} {
		for _, maximum := range []int{1, 7, 40, 1000} {
			last := 0
			for value := -2*maximum - 5; value <= 2*maximum+5; value++ {
				fill, _ := terrain.StatusBarFill(width, value, maximum)
				switch {
				case fill < last:
					t.Fatalf("width %d, maximum %d: value %d fills %d, value %d filled %d", width, maximum, value, fill, value-1, last)
				case value <= 0 && fill != 0:
					t.Fatalf("width %d, maximum %d: value %d fills %d, want 0", width, maximum, value, fill)
				case value >= maximum && fill != width-8:
					t.Fatalf("width %d, maximum %d: value %d fills %d, want %d", width, maximum, value, fill, width-8)
				}
				last = fill
			}
		}
	}
}

// TestStatusBarRunsPaintEachPixelOnce paints bars of a one-cell unit's width
// and of two wide classes' run by run and checks the structure the picture
// needs: no pixel is painted twice, both end caps are one picture repeated at
// the bar's own two ends rather than mirrored, the interior between them fills
// from the left and leaves its unfilled columns unpainted, a clamped fill
// paints what the clamp says, and a faded bar changes only its interior's
// colours. A bar too narrow for its two caps paints nothing.
func TestStatusBarRunsPaintEachPixelOnce(t *testing.T) {
	for _, width := range []int{32, 60, 96} {
		bar := image.Rect(100, 50, 100+width, 54)
		interior := width - 8
		paint := func(kind terrain.StatusBarKind, fill int, faded bool) map[image.Point]color.RGBA {
			t.Helper()
			px := map[image.Point]color.RGBA{}
			for _, run := range terrain.AppendStatusBarRuns(nil, kind, bar, fill, faded) {
				if run.Rect.Dy() != 1 || run.Rect.Empty() || !run.Rect.In(bar) {
					t.Fatalf("width %d, fill %d: the run %v is not one row inside the bar %v", width, fill, run.Rect, bar)
				}
				if run.Color.A == 0 {
					t.Fatalf("width %d, fill %d: the run %v paints a transparent colour", width, fill, run.Rect)
				}
				for y := run.Rect.Min.Y; y < run.Rect.Max.Y; y++ {
					for x := run.Rect.Min.X; x < run.Rect.Max.X; x++ {
						p := image.Pt(x, y)
						if _, twice := px[p]; twice {
							t.Fatalf("width %d, fill %d: pixel %v is painted twice", width, fill, p.Sub(bar.Min))
						}
						px[p] = run.Color
					}
				}
			}
			return px
		}

		for _, kind := range []terrain.StatusBarKind{terrain.HealthBar, terrain.ManaBar} {
			full := paint(kind, interior, false)
			faded := paint(kind, interior, true)
			for y := range 4 {
				for x := range 4 {
					left, right := bar.Min.Add(image.Pt(x, y)), bar.Min.Add(image.Pt(width-4+x, y))
					l, lok := full[left]
					r, rok := full[right]
					if lok != rok || l != r {
						t.Errorf("width %d, kind %d: cap pixel (%d,%d) is %v at the left and %v at the right", width, kind, x, y, l, r)
					}
					if f, fok := faded[left]; fok != lok || f != l || (lok && l.A != 0xff) {
						t.Errorf("width %d, kind %d: cap pixel (%d,%d) is %v opaque and %v faded; a cap stays opaque", width, kind, x, y, l, f)
					}
				}
				row := full[bar.Min.Add(image.Pt(4, y))]
				for x := 4; x < width-4; x++ {
					p := bar.Min.Add(image.Pt(x, y))
					if full[p] != row || row.A != 0xff {
						t.Errorf("width %d, kind %d: interior (%d,%d) is %v, want the row's opaque %v", width, kind, x, y, full[p], row)
					}
					if want := terrain.StatusBarFaded(row); faded[p] != want {
						t.Errorf("width %d, kind %d: faded interior (%d,%d) is %v, want %v", width, kind, x, y, faded[p], want)
					}
				}
			}

			for _, tc := range []struct{ fill, painted int }{
				{0, 0}, {-3, 0}, {1, 1}, {interior / 2, interior / 2}, {interior - 1, interior - 1}, {interior + 6, interior},
			} {
				px := paint(kind, tc.fill, false)
				for y := range 4 {
					for x := 4; x < width-4; x++ {
						p := bar.Min.Add(image.Pt(x, y))
						c, ok := px[p]
						if want := x < 4+tc.painted; ok != want || (ok && c != full[p]) {
							t.Errorf("width %d, kind %d, fill %d: interior (%d,%d) painted=%v %v, want painted=%v",
								width, kind, tc.fill, x, y, ok, c, want)
						}
					}
				}
			}
		}
	}

	bar := image.Rect(100, 50, 132, 54)
	prefix := []terrain.StatusBarRun{{Rect: image.Rect(0, 0, 1, 1), Color: color.RGBA{A: 0xff}}}
	if got := terrain.AppendStatusBarRuns(prefix, terrain.StatusBarKind(2), bar, 24, false); len(got) != 1 {
		t.Errorf("an unknown kind appended %d runs", len(got)-1)
	}
	if got := terrain.AppendStatusBarRuns(prefix, terrain.HealthBar, image.Rect(100, 50, 107, 54), 0, false); len(got) != 1 {
		t.Errorf("a bar 7 wide appended %d runs; it cannot hold both caps", len(got)-1)
	}
	if got := terrain.AppendStatusBarRuns(prefix, terrain.HealthBar, bar, 24, false); got[0] != prefix[0] {
		t.Errorf("appending changed the run already in the slice: %v", got[0])
	}
}

// TestStatusBarFadedIsHalfOpacityPremultiplied: the faded colour carries alpha
// 128, no channel exceeds it, and blended source-over the health row's
// brightest green over the owner's sampled ground #313421 reads #189a10. The
// original's own blend reads #189610 there (DIV-1459).
func TestStatusBarFadedIsHalfOpacityPremultiplied(t *testing.T) {
	green := color.RGBA{G: 0xff, A: 0xff}
	f := terrain.StatusBarFaded(green)
	if f != (color.RGBA{G: 128, A: 128}) {
		t.Fatalf("StatusBarFaded(%v) = %v, want {0 128 0 128}", green, f)
	}
	for _, c := range []color.RGBA{{G: 0x82, A: 0xff}, {B: 0xc6, A: 0xff}, {R: 0xff, G: 0xff, B: 0xff, A: 0xff}} {
		if f := terrain.StatusBarFaded(c); f.A != 128 || f.R > f.A || f.G > f.A || f.B > f.A {
			t.Errorf("StatusBarFaded(%v) = %v is not a premultiplied half", c, f)
		}
	}
	ground := color.RGBA{R: 0x31, G: 0x34, B: 0x21, A: 0xff}
	over := func(s, d uint8) uint8 { return uint8(uint32(s) + (uint32(d)*(255-uint32(f.A))+127)/255) }
	got := color.RGBA{R: over(f.R, ground.R), G: over(f.G, ground.G), B: over(f.B, ground.B), A: 0xff}
	if want := (color.RGBA{R: 0x18, G: 0x9a, B: 0x10, A: 0xff}); got != want {
		t.Errorf("the faded green over %v reads %v, want %v", ground, got, want)
	}
}

// TestStatusBarsReadAsTheirOwnUnit: a unit's two bars stand wholly above its
// own footprint and clear of every glyph its own cell carries, health directly
// over mana on the same columns. They fall inside the cell to the north, and a
// unit standing there has its own pair 24 rows higher, more than either pair's
// 8 rows. The bars of units side by side abut and never overlap. A wide unit
// drawn without a class has its pair centred over its whole footprint at the
// height a one-cell unit's takes; a wide class's pair is centred on the cell
// centre its body is anchored at, spans its selection box, and stands with
// health 2 rows above the box's top edge.
func TestStatusBarsReadAsTheirOwnUnit(t *testing.T) {
	const cols, rows = 6, 6
	pair := func(col, row, side int, c *terrain.UnitClass) (health, mana image.Rectangle) {
		t.Helper()
		var ok1, ok2 bool
		health, ok1 = terrain.StatusBarRect(terrain.HealthBar, col, row, cols, rows, side, c)
		mana, ok2 = terrain.StatusBarRect(terrain.ManaBar, col, row, cols, rows, side, c)
		if !ok1 || !ok2 {
			t.Fatalf("cell (%d,%d) side %d: no bar", col, row, side)
		}
		return health, mana
	}
	for _, cell := range []image.Point{{0, 1}, {3, 2}, {5, 5}, {2, 3}} {
		health, mana := pair(cell.X, cell.Y, 1, nil)
		if health.Max.Y != mana.Min.Y || health.Min.X != mana.Min.X || health.Max.X != mana.Max.X {
			t.Errorf("cell %v: mana %v is not directly under health %v", cell, mana, health)
		}
		own := image.Rect(cell.X*32, cell.Y*32, cell.X*32+32, cell.Y*32+32)
		north := own.Sub(image.Pt(0, 32))
		both := health.Union(mana)
		if both.Max.Y > own.Min.Y || !both.In(north) {
			t.Errorf("cell %v: the bars %v are not inside the northern cell %v", cell, both, north)
		}
		for _, glyphs := range [][]image.Rectangle{
			terrain.ObjectMarkerRects(cell.X, cell.Y, cols, rows, 32),
			terrain.UnitMarkerRects(cell.X, cell.Y, cols, rows, 32),
			terrain.StaticMarkerRects(cell.X, cell.Y, cols, rows, 32),
			terrain.EntityMarkerRects(cell.X, cell.Y, cols, rows, 32),
			terrain.SelectionMarkerRects(cell.X, cell.Y, cols, rows, 32),
		} {
			for _, r := range glyphs {
				if !both.Intersect(r).Empty() {
					t.Errorf("cell %v: the bars %v overlap the own cell's glyph piece %v", cell, both, r)
				}
			}
		}
		nh, nm := pair(cell.X, cell.Y-1, 1, nil)
		if gap := both.Min.Y - nh.Union(nm).Max.Y; gap != 24 || gap <= both.Dy() {
			t.Errorf("cell %v: the northern unit's bars end %d rows above these %d-row bars, want 24",
				cell, gap, both.Dy())
		}
		if cell.X+1 < cols {
			eh, em := pair(cell.X+1, cell.Y, 1, nil)
			if eh.Min.X != health.Max.X || !eh.Intersect(health).Empty() || !em.Intersect(mana).Empty() {
				t.Errorf("cell %v: the eastern unit's bars %v and %v do not abut %v and %v", cell, eh, em, health, mana)
			}
		}
	}
	for _, side := range []int{2, 3} {
		health, mana := pair(1, 1, side, nil)
		foot := image.Rect(32, 32, 32+side*32, 32+side*32)
		both := health.Union(mana)
		if both.Min.X-foot.Min.X != foot.Max.X-both.Max.X || foot.Min.Y-both.Min.Y != 16 || both.Dy() != 8 {
			t.Errorf("side %d without a class: the bars %v are not centred 16 rows above the footprint %v", side, both, foot)
		}
	}
	for _, c := range []*terrain.UnitClass{sbDragon, sbTroll} {
		health, mana := pair(2, 2, c.TileSize, c)
		both := health.Union(mana)
		anchor := image.Pt(2*32+16, 2*32+16)
		boxTop := anchor.Y + c.Selection.Min.Y - c.CenterY
		if health.Max.Y != mana.Min.Y || health.Min.X != mana.Min.X || health.Max.X != mana.Max.X {
			t.Errorf("class %+v: mana %v is not directly under health %v", c.Selection, mana, health)
		}
		if both.Min.X+both.Max.X != 2*anchor.X || both.Dx() != c.Selection.Dx() || both.Min.Y != boxTop-2 || both.Dy() != 8 {
			t.Errorf("class %+v: the bars %v are not the box's width centred on %v with health 2 rows above the box's top %d",
				c.Selection, both, anchor, boxTop)
		}
	}
}
