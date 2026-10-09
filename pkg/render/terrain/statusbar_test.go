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
// a class without valid selection geometry, has a 32 by 4 fallback bar, its top
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
		{"a one-cell class uses its selection width", terrain.ManaBar, 1, 1, 1, sbLance, image.Rect(24, 20, 72, 24), true},
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

func TestStatusBarFillTruncatesWithPoolMinimum(t *testing.T) {
	for _, tc := range []struct {
		kind                        terrain.StatusBarKind
		width, value, maximum, want int
	}{
		{terrain.HealthBar, 32, 0, 100, 0}, {terrain.HealthBar, 32, 1, 100, 1},
		{terrain.HealthBar, 32, -1, 100, 1}, {terrain.HealthBar, 32, -100, 100, -24},
		{terrain.ManaBar, 32, 0, 100, 1}, {terrain.ManaBar, 32, 1, 100, 1},
		{terrain.HealthBar, 25, 1, 3, 5}, {terrain.ManaBar, 25, 150, 100, 25},
		{terrain.HealthBar, 32, 150, 100, 36}, {terrain.HealthBar, 96, 1, 89, 1},
		{terrain.HealthBar, 6, 1, 100, 0},
	} {
		if got, ok := terrain.StatusBarFill(tc.kind, tc.width, tc.value, tc.maximum); !ok || got != tc.want {
			t.Errorf("kind %d width %d pool %d/%d N=%d,%v want %d,true", tc.kind, tc.width, tc.value, tc.maximum, got, ok, tc.want)
		}
	}
	for _, kind := range []terrain.StatusBarKind{terrain.HealthBar, terrain.ManaBar} {
		for _, maximum := range []int{0, -5} {
			if fill, ok := terrain.StatusBarFill(kind, 32, 3, maximum); ok || fill != 0 {
				t.Errorf("nonpositive maximum %d gives N=%d,%v", maximum, fill, ok)
			}
		}
	}
}

func TestStatusBarRunsPaintRowsAndRemainder(t *testing.T) {
	grey := [4]color.RGBA{{65, 64, 65, 255}, {131, 129, 131, 255}, {98, 97, 98, 255}, {65, 64, 65, 255}}
	for _, width := range []int{32, 60, 96} {
		bar := image.Rect(100, 50, 100+width, 54)
		for _, kind := range []terrain.StatusBarKind{terrain.HealthBar, terrain.ManaBar} {
			for _, faded := range []bool{false, true} {
				for _, value := range []int{0, 1, 24, 25, 49, 50, 100, 150} {
					fill := (width - 8) * value / 100
					if fill == 0 && (kind == terrain.ManaBar || value != 0) {
						fill = 1
					}
					pixels := map[image.Point]terrain.StatusBarRun{}
					for _, run := range terrain.AppendStatusBarRuns(nil, kind, bar, value, 100, faded) {
						if run.Rect.Dy() != 1 || run.Rect.Empty() {
							t.Fatalf("invalid row %v", run.Rect)
						}
						for x := run.Rect.Min.X; x < run.Rect.Max.X; x++ {
							p := image.Pt(x, run.Rect.Min.Y)
							if _, twice := pixels[p]; twice && value <= 100 {
								t.Fatalf("pixel painted twice: %v", p)
							}
							pixels[p] = run
						}
					}
					for y := range 4 {
						rb, g := [4]uint8{131, 255, 197, 131}[y], [4]uint8{129, 255, 194, 129}[y]
						row := color.RGBA{A: 255}
						switch {
						case kind == terrain.ManaBar:
							row.B = rb
						case value < 25:
							row.R = rb
						case value < 50:
							row.R, row.G = rb, g
						default:
							row.G = g
						}
						for x := 4; x < width-4; x++ {
							got, painted := pixels[bar.Min.Add(image.Pt(x, y))]
							if x-4 < fill {
								if !painted || got.Color != row || got.HalfAdd != faded {
									t.Errorf("width=%d kind=%d value=%d faded=%v row=%d x=%d got=%+v painted=%v want=%v", width, kind, value, faded, y, x, got, painted, row)
								}
							} else if faded {
								if painted {
									t.Errorf("faded remainder painted at %d,%d", x, y)
								}
							} else if !painted || got.Color != grey[y] || got.HalfAdd {
								t.Errorf("selected remainder at %d,%d=%+v want %v", x, y, got, grey[y])
							}
						}
						if value > 100 {
							p := bar.Min.Add(image.Pt(4+fill-1, y))
							if got := pixels[p]; got.Color != row || got.HalfAdd != faded {
								t.Errorf("overfull endpoint %v=%+v want %v", p, got, row)
							}
						}
					}
				}
			}
		}
	}
	prefix := []terrain.StatusBarRun{{Rect: image.Rect(0, 0, 1, 1), Color: color.RGBA{A: 255}}}
	for _, kind := range []terrain.StatusBarKind{-1, 2} {
		if got := terrain.AppendStatusBarRuns(prefix, kind, image.Rect(0, 0, 32, 4), 100, 100, false); len(got) != 1 {
			t.Errorf("unknown kind appended %d runs", len(got)-1)
		}
	}
	if got := terrain.AppendStatusBarRuns(prefix, terrain.HealthBar, image.Rect(0, 0, 7, 4), 1, 100, false); len(got) != 1 {
		t.Errorf("narrow bar appended %d runs", len(got)-1)
	}
}

func TestStatusBarBlendMatchesPackedHalfWords(t *testing.T) {
	for _, source := range []color.RGBA{{128, 0, 0, 255}, {255, 0, 0, 255}, {192, 0, 0, 255}, {128, 128, 0, 255}, {255, 255, 0, 255}, {192, 192, 0, 255}, {0, 128, 0, 255}, {0, 255, 0, 255}, {0, 192, 0, 255}, {0, 0, 128, 255}, {0, 0, 255, 255}, {0, 0, 192, 255}} {
		for word := 0; word < 65536; word++ {
			r, g, b := word/2048, word/32%64, word%32
			under := color.RGBA{uint8(r * 255 / 31), uint8(g * 255 / 63), uint8(b * 255 / 31), 255}
			want := color.RGBA{uint8((r/2 + int(source.R)/8/2) * 255 / 31), uint8((g/2 + int(source.G)/4/2) * 255 / 63), uint8((b/2 + int(source.B)/8/2) * 255 / 31), 255}
			if got := terrain.StatusBarBlend(source, under); got != want {
				t.Fatalf("source=%v word=%04x got=%v want=%v", source, word, got, want)
			}
		}
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
