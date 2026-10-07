package mapload_test

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The extent every case below is stated over, and its interior. Twenty-four on
// both axes is the smallest square with an interior at all — the ring is eight
// cells deep from each side — and its interior is the 8x8 block at columns and
// rows 8..15, which is 64 cells: room for the 56-cell walk of AC-2 and for every
// other case one cell at a time.
const (
	passW, passH = 24, 24
	passCells    = passW * passH // 576
	passInterior = 64
	passRing     = passCells - passInterior // 512
)

// The words every case is written over, each with the arithmetic that puts
// it in its arm.
const (
	// openWord: i 0x041 = 65, g 1, b 0, s 1. Outside the water range, bit 13
	// clear, not group 7 — no arm reaches it.
	openWord uint16 = 0x0041
	// impassableWord: openWord with bit 13 set. Arm (a), and only arm (a).
	impassableWord uint16 = 0x2041
	// waterWord: i 0x208 = 520, inside [512, 768). Arm (b). Its group is 8, so
	// the mountain arm has nothing to say about it either way.
	waterWord uint16 = 0x0208
	// mountainWord: i 0x1d1 = 465, g 7, b 1, s 1, blend level 5. Arm (c).
	mountainWord uint16 = 0x01D1
)

// passBlend is the blend level of every (blend column, sub-cell) pair of strip
// group 7, TRANSCRIBED BY HAND from the contract's own table.
//
// It is written out here rather than read from the derivation for the reason
// every hand-written expectation in this tree is: read off the production table
// this walk would agree with whatever that table said, and one wrong number in
// fifty-six moves routes on real maps with nothing to point at.
var passBlend = [4][14]uint8{
	{2, 3, 2, 4, 3, 4, 2, 2, 2, 2, 4, 4, 4, 4},
	{3, 5, 3, 3, 1, 3, 2, 4, 2, 2, 4, 2, 4, 4},
	{2, 3, 2, 4, 3, 4, 2, 4, 2, 2, 4, 2, 4, 4},
	{5, 5, 5, 5, 5, 5, 2, 2, 2, 2, 4, 4, 4, 4},
}

// groupWord is the tile word naming strip group g at blend column b and sub-cell
// s, assembled from the field widths rather than written out as a hex literal
// per case: fifty-six literals would be fifty-six chances to typo the fixture
// itself, and what this file's expectations are stated over is the level table,
// not the packing.
func groupWord(g, b, s int) uint16 { return uint16(g<<6 | b<<4 | s) }

func passIsBorder(x, y, w, h int) bool {
	return x < 8 || y < 8 || x >= w-8 || y >= h-8
}

// passLay is where case k sits: across the 8x8 interior of a 24x24 map, row by
// row, so every case stands well clear of the ring and no two share a cell.
func passLay(k int) (x, y int) { return 8 + k%8, 8 + k/8 }

// passMap is a w*h map with full zero planes, the fixture everything below
// starts from. It carries the extent and the two planes the derivation reads and
// nothing else.
func passMap(w, h int) *alm.Map {
	return &alm.Map{Width: w, Height: h, Tiles: make([]uint16, w*h), Overlay: make([]uint8, w*h)}
}

// passWant is the expected plane: 0x03 on every border cell, whatever a
// named override says on any other, and 0x00 elsewhere.
func passWant(w, h int, over map[[2]int]byte) func(x, y int) byte {
	return func(x, y int) byte {
		if passIsBorder(x, y, w, h) {
			return 0x03
		}
		if b, ok := over[[2]int{x, y}]; ok {
			return b
		}
		return 0x00
	}
}

// passCheck compares a derived plane against that expectation cell by cell.
func passCheck(t *testing.T, label string, got []byte, w, h int, want func(x, y int) byte) {
	t.Helper()
	if len(got) != w*h {
		t.Fatalf("%s: the plane is %d byte(s), want %d for a %dx%d extent", label, len(got), w*h, w, h)
	}
	bad := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g, k := got[y*w+x], want(x, y)
			if g == k {
				continue
			}
			bad++
			if bad <= 8 {
				t.Errorf("%s: cell (%d,%d) derived as %#02x, want %#02x", label, x, y, g, k)
			}
		}
	}
	if bad > 8 {
		t.Errorf("%s: %d cells differ in all", label, bad)
	}
}

// TestEachArmBlocksItsOwnCellAndNothingElse is AC-1 (SC-1): six interior cells,
// one per given, every byte compared whole and every unnamed cell required to be
// what the border rule alone makes it.
//
// The sixth cell is "nothing at all" read the strict way: BOTH PLANES STOP
// BEFORE IT, so the map carries no tile word and no overlay byte for that cell
// at all. Written as a zero word in a full plane the case would only repeat the
// first.
func TestEachArmBlocksItsOwnCellAndNothingElse(t *testing.T) {
	const row = 8
	cases := []struct {
		x       int
		word    uint16
		overlay uint8
		want    byte
		what    string
	}{
		{8, openWord, 0, 0x00, "a plain land word"},
		{9, impassableWord, 0, 0x01, "bit 13"},
		{10, waterWord, 0, 0x01, "a word inside [512,768)"},
		{11, mountainWord, 0, 0x01, "group 7 at b=1, s=1"},
		{12, openWord, 0x5a, 0x01, "a nonzero overlay"},
		{13, 0, 0, 0x00, "nothing at all"},
	}

	// The planes end exactly at the sixth cell's index.
	end := row*passW + cases[len(cases)-1].x
	m := &alm.Map{Width: passW, Height: passH,
		Tiles: make([]uint16, end), Overlay: make([]uint8, end)}
	over := map[[2]int]byte{}
	for _, c := range cases[:len(cases)-1] {
		m.Tiles[row*passW+c.x] = c.word
		m.Overlay[row*passW+c.x] = c.overlay
		over[[2]int{c.x, row}] = c.want
	}
	over[[2]int{cases[len(cases)-1].x, row}] = cases[len(cases)-1].want

	got := mapload.Passability(m)
	passCheck(t, "the six arms", got, passW, passH, passWant(passW, passH, over))

	// ...and named one at a time, so a failure says which given moved.
	for _, c := range cases {
		if g := got[row*passW+c.x]; g != c.want {
			t.Errorf("%s: cell (%d,%d) derived as %#02x, want %#02x", c.what, c.x, row, g, c.want)
		}
	}
}

// TestAnyNonzeroSceneryCodeBlocksAndZeroDecidesNothing is AC-5 (SC-1): four
// overlay codes on cells whose word blocks nothing, and code 0 on a cell whose
// word blocks — each its own case.
//
// The sixth cell is the control the fifth needs: code 0 on a word that blocks
// nothing must stay open, or "the fifth is exactly what its word made it" would
// hold just as well of an arm that blocked on every code including zero.
func TestAnyNonzeroSceneryCodeBlocksAndZeroDecidesNothing(t *testing.T) {
	const row = 9
	cases := []struct {
		x       int
		word    uint16
		overlay uint8
		want    byte
	}{
		{8, openWord, 1, 0x01},
		{9, openWord, 82, 0x01},
		{10, openWord, 246, 0x01},
		{11, openWord, 250, 0x01},
		{12, impassableWord, 0, 0x01},
		{13, openWord, 0, 0x00},
	}

	m := passMap(passW, passH)
	over := map[[2]int]byte{}
	for _, c := range cases {
		m.Tiles[row*passW+c.x] = c.word
		m.Overlay[row*passW+c.x] = c.overlay
		over[[2]int{c.x, row}] = c.want
	}

	got := mapload.Passability(m)
	passCheck(t, "the scenery codes", got, passW, passH, passWant(passW, passH, over))
	for _, c := range cases {
		if g := got[row*passW+c.x]; g != c.want {
			t.Errorf("overlay code %d over word %#04x derived as %#02x, want %#02x",
				c.overlay, c.word, g, c.want)
		}
	}
}

// TestGroupSevenBlocksExactlyAtBlendLevelThreeOrMore is AC-2 (SC-2): all 56
// group-7 cells, one per (blend column, sub-cell) pair, each expected from the
// hand table above; exactly 35 block; and the straddling pair is named.
func TestGroupSevenBlocksExactlyAtBlendLevelThreeOrMore(t *testing.T) {
	m := passMap(passW, passH)
	over := map[[2]int]byte{}
	type place struct{ x, y, b, s int }
	var placed []place

	k := 0
	for b := 0; b < 4; b++ {
		for s := 0; s < 14; s++ {
			x, y := passLay(k)
			m.Tiles[y*passW+x] = groupWord(7, b, s)
			want := byte(0x00)
			if passBlend[b][s] >= 3 {
				want = 0x01
			}
			over[[2]int{x, y}] = want
			placed = append(placed, place{x, y, b, s})
			k++
		}
	}
	if k != 56 {
		t.Fatalf("the walk placed %d cells, want all 56 group-7 pairs", k)
	}

	got := mapload.Passability(m)
	passCheck(t, "the group-7 walk", got, passW, passH, passWant(passW, passH, over))

	blocked := 0
	for _, p := range placed {
		g := got[p.y*passW+p.x]
		if g == 0x01 {
			blocked++
		}
		if want := over[[2]int{p.x, p.y}]; g != want {
			t.Errorf("group 7, b=%d, s=%d (level %d) derived as %#02x, want %#02x",
				p.b, p.s, passBlend[p.b][p.s], g, want)
		}
	}
	if blocked != 35 {
		t.Errorf("%d of the 56 group-7 cells block ground, want 35 — the ones at level 3 or more", blocked)
	}

	// The pair either side of the compare, named: b=1 s=0 is level 3 and blocks,
	// b=1 s=6 is level 2 and does not. A threshold moved by one in either
	// direction moves exactly one of these two.
	for _, c := range []struct {
		s     int
		level uint8
		want  byte
	}{{0, 3, 0x01}, {6, 2, 0x00}} {
		if passBlend[1][c.s] != c.level {
			t.Fatalf("the hand table has b=1, s=%d at level %d, not the %d this pair is stated over",
				c.s, passBlend[1][c.s], c.level)
		}
		var x, y int
		for _, p := range placed {
			if p.b == 1 && p.s == c.s {
				x, y = p.x, p.y
			}
		}
		if g := got[y*passW+x]; g != c.want {
			t.Errorf("group 7, b=1, s=%d is level %d and derived as %#02x, want %#02x — the compare is "+
				"at 3 or more", c.s, c.level, g, c.want)
		}
	}
}

// TestTheRejectedSubCellsTheUnwrittenGroupsAndTheIgnoredBits is AC-3 (SC-3):
// group 7 at sub-cells 14 and 15, strip groups 13, 14 and 15, and bits 10, 11,
// 12, 14 and 15 added to a group-7 BLOCKING word and to an open one.
//
// The blocking word must be group 7 and nothing else. The sim-side split and the
// render tier's — (w & 0x1fff) >> 6 — agree on every word that reaches neither
// arm and on every word inside the water range, so an ignored-bit case written
// over any other word would leave a split swapped for the render tier's alive.
// Over a group-7 word the render split reads a different group and the cell
// opens, which is what makes these ten cells discriminate at all.
//
// None of the five bits occurs on a shipped cell, so the corpus witnesses none
// of this and a synthetic word is the only instrument there is.
func TestTheRejectedSubCellsTheUnwrittenGroupsAndTheIgnoredBits(t *testing.T) {
	m := passMap(passW, passH)
	over := map[[2]int]byte{}
	type probe struct {
		x, y int
		word uint16
		want byte
		what string
	}
	var probes []probe

	add := func(word uint16, want byte, what string) {
		x, y := passLay(len(probes))
		m.Tiles[y*passW+x] = word
		over[[2]int{x, y}] = want
		probes = append(probes, probe{x, y, word, want, what})
	}

	// The rejected sub-cells: group 7 reaches the table only at 13 or below.
	add(groupWord(7, 0, 14), 0x00, "group 7 at sub-cell 14")
	add(groupWord(7, 0, 15), 0x00, "group 7 at sub-cell 15")
	// The groups the original never wrote a table for.
	add(groupWord(13, 0, 0), 0x00, "strip group 13")
	add(groupWord(14, 0, 0), 0x00, "strip group 14")
	add(groupWord(15, 0, 0), 0x00, "strip group 15")

	const allFive = uint16(1<<10 | 1<<11 | 1<<12 | 1<<14 | 1<<15)
	for _, base := range []struct {
		word uint16
		want byte
		what string
	}{{mountainWord, 0x01, "a group-7 blocking word"}, {openWord, 0x00, "an open word"}} {
		for _, bit := range []uint16{1 << 10, 1 << 11, 1 << 12, 1 << 14, 1 << 15, allFive} {
			add(base.word|bit, base.want, fmt.Sprintf("%s plus bits %#04x", base.what, bit))
		}
	}

	got := mapload.Passability(m)
	passCheck(t, "the rejected words and the ignored bits", got, passW, passH, passWant(passW, passH, over))
	for _, p := range probes {
		if g := got[p.y*passW+p.x]; g != p.want {
			t.Errorf("%s (word %#04x) derived as %#02x, want %#02x", p.what, p.word, g, p.want)
		}
	}
	if len(probes) != 17 {
		t.Errorf("the file placed %d probes, want 17 — five rejected words and twelve ignored-bit cases", len(probes))
	}
}

// TestTheWaterRangeIsReadWhole is AC-4 (SC-3): the range is a test on the index
// and on nothing else, so no sub-case escapes it.
//
// All three words carry strip group 8, which the mountain arm never blocks, so
// what blocks these cells is the range itself. The corpus carries none of them:
// a low nibble of 8 or more inside the range occurs on 0 shipped cells, and the
// blend column of the third is the case a classifier read would have decided
// differently.
func TestTheWaterRangeIsReadWhole(t *testing.T) {
	cases := []struct {
		word uint16
		what string
	}{
		{0x0208, "sub-cell 8 inside the range"},
		{0x0209, "sub-cell 9 inside the range"},
		{0x0214, "sub-cell 4 with (i & 0x30) == 0x10"},
	}

	m := passMap(passW, passH)
	over := map[[2]int]byte{}
	for k, c := range cases {
		x, y := passLay(k)
		if i := int(c.word) & 0x3ff; i < 512 || i >= 768 {
			t.Fatalf("%s: word %#04x has index %d, which is outside [512,768) — the fixture is wrong",
				c.what, c.word, i)
		}
		m.Tiles[y*passW+x] = c.word
		over[[2]int{x, y}] = 0x01
	}

	got := mapload.Passability(m)
	passCheck(t, "the water range", got, passW, passH, passWant(passW, passH, over))
	for k, c := range cases {
		x, y := passLay(k)
		if g := got[y*passW+x]; g != 0x01 {
			t.Errorf("%s (word %#04x) derived as %#02x, want 0x01", c.what, c.word, g)
		}
	}
}

// TestTheBorderCoversTheOuterEightRingsAndNothingElse is AC-6 (SC-4): over a
// 24x24 map the border cells are exactly the cells within 8 of an edge, each
// carrying ground and air, and no interior cell carries air at all; over a 16x24
// and a 24x16 every cell is border.
//
// The counts are hand-computed and stated: 24x24 is 576 cells of which the 8x8
// interior is 64, so 512 carry the air bit. A ring one cell shallower would
// leave 100 interior cells and 476 on the ring, and a ring one deeper 36 and
// 540 — so this count alone separates a depth of 7, 8 and 9.
func TestTheBorderCoversTheOuterEightRingsAndNothingElse(t *testing.T) {
	t.Run("24x24", func(t *testing.T) {
		got := mapload.Passability(passMap(passW, passH))
		passCheck(t, "the border", got, passW, passH, passWant(passW, passH, nil))

		air, ground, interiorAir := 0, 0, 0
		for y := 0; y < passH; y++ {
			for x := 0; x < passW; x++ {
				b := got[y*passW+x]
				if b&0x02 != 0 {
					air++
					if !passIsBorder(x, y, passW, passH) {
						interiorAir++
					}
				}
				if b&0x01 != 0 {
					ground++
				}
			}
		}
		if air != passRing {
			t.Errorf("%d cells carry the air bit, want %d — the 8-deep ring of a %d-cell map",
				air, passRing, passCells)
		}
		if ground != passRing {
			t.Errorf("%d cells block ground, want %d — on a map whose planes are empty the ring is the "+
				"only arm there is", ground, passRing)
		}
		if interiorAir != 0 {
			t.Errorf("%d interior cells carry the air bit, want 0 — the border is its only writer", interiorAir)
		}
	})

	// A map at 16 or below on either axis is border throughout: the two sides of
	// that axis meet.
	for _, c := range []struct{ w, h int }{{16, 24}, {24, 16}} {
		t.Run(fmt.Sprintf("%dx%d", c.w, c.h), func(t *testing.T) {
			got := mapload.Passability(passMap(c.w, c.h))
			if len(got) != c.w*c.h {
				t.Fatalf("the plane is %d byte(s), want %d", len(got), c.w*c.h)
			}
			for i, b := range got {
				if b != 0x03 {
					t.Fatalf("cell %d of a %dx%d map derived as %#02x, want 0x03 — every cell of it is "+
						"border", i, c.w, c.h, b)
				}
			}
		})
	}
}

// TestTheDerivationIsTotalOverEveryMapShape is AC-10's derivation half: no map
// shape makes it raise, panic or answer a length other than the extent's.
//
// The long-plane case is written so that a derivation reading past the extent
// would be caught: every word past cell 576 is an impassable one, so an interior
// cell that picked one up would block, and every one before it is open.
func TestTheDerivationIsTotalOverEveryMapShape(t *testing.T) {
	// A map with no tile plane and no overlay plane at all.
	bare := &alm.Map{Width: passW, Height: passH}

	// A tile plane one cell short of the first interior cell: (8,8) carries a
	// water word, and everything from (9,8) on is past the plane's end.
	short := &alm.Map{Width: passW, Height: passH, Tiles: make([]uint16, 8*passW+9)}
	short.Tiles[8*passW+8] = waterWord

	// A tile plane fifty words longer than the extent, every extra word blocking.
	long := &alm.Map{Width: passW, Height: passH, Tiles: make([]uint16, passCells+50)}
	for i := passCells; i < len(long.Tiles); i++ {
		long.Tiles[i] = impassableWord
	}

	// An overlay plane and no tile plane: the arms are independent.
	overlayOnly := &alm.Map{Width: passW, Height: passH, Overlay: make([]uint8, passCells)}
	overlayOnly.Overlay[9*passW+9] = 200

	cases := []struct {
		what  string
		m     *alm.Map
		cells int
		over  map[[2]int]byte
	}{
		{"no map at all", nil, 0, nil},
		{"no tile plane and no overlay plane", bare, passCells, nil},
		{"a tile plane shorter than the extent", short, passCells, map[[2]int]byte{{8, 8}: 0x01}},
		{"a tile plane longer than the extent", long, passCells, nil},
		{"an overlay plane and no tile plane", overlayOnly, passCells, map[[2]int]byte{{9, 9}: 0x01}},
		{"a width of 0", &alm.Map{Width: 0, Height: passH}, 0, nil},
		{"a height of 0", &alm.Map{Width: passW, Height: 0}, 0, nil},
		{"a negative height", &alm.Map{Width: passW, Height: -1}, 0, nil},
		{"a negative width", &alm.Map{Width: -3, Height: passH}, 0, nil},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			got := mapload.Passability(c.m)
			if len(got) != c.cells {
				t.Fatalf("the plane is %d byte(s), want %d", len(got), c.cells)
			}
			if c.cells == 0 {
				return
			}
			passCheck(t, c.what, got, passW, passH, passWant(passW, passH, c.over))
		})
	}
}

func TestTheDerivationReadsTheExtentAndTheTwoPlanesAlone(t *testing.T) {
	plain := passMap(passW, passH)
	plain.Tiles[10*passW+10] = mountainWord
	plain.Overlay[11*passW+11] = 4

	loud := passMap(passW, passH)
	copy(loud.Tiles, plain.Tiles)
	copy(loud.Overlay, plain.Overlay)
	loud.Name = "loud"
	loud.Description = "everything else filled in"
	loud.Angle, loud.MetaRecordWord = 1.5, 0x40200000
	loud.Altitudes = make([]uint8, passCells)
	for i := range loud.Altitudes {
		loud.Altitudes[i] = uint8(i)
	}
	loud.Objects = []alm.Object{{X: 0x0900, Y: 0x0900, Kind: 0x21}}
	loud.Units = []alm.Unit{{X: 0x0A00, Y: 0x0A00, ClassID: 7}}
	loud.Groups = []alm.Group{{Scalar: 5000, Name: "g"}}
	loud.Triggers = alm.Triggers{EntryCount: 3}
	loud.Meta.Count6 = 99

	a, b := mapload.Passability(plain), mapload.Passability(loud)
	if len(a) != len(b) {
		t.Fatalf("the two planes are %d and %d bytes", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("the two planes differ at cell %d: %#02x against %#02x — something outside the extent "+
				"and the two planes reached a grid byte", i, a[i], b[i])
		}
	}
	// ...and the fixture is not vacuously equal: both carry the two blocked
	// interior cells the plain map was given.
	if a[10*passW+10] != 0x01 || a[11*passW+11] != 0x01 {
		t.Fatalf("the shared plane derives (10,10) as %#02x and (11,11) as %#02x, want both 0x01 — with "+
			"neither blocked the comparison above is between two empty planes",
			a[10*passW+10], a[11*passW+11])
	}
}

// censusMap is the map the census is stated over: every arm placed on a cell
// whose count is known by hand, INCLUDING two arms inside the ring, so the five
// per-arm counts plainly do not sum to either blocked count.
//
//	interior (8,8)  bit 13                  -> impassable
//	interior (9,8)  a water word            -> water
//	interior (10,8) group 7, b=1, s=1       -> mountain
//	interior (11,8) overlay 7               -> scenery
//	interior (12,8) bit 13 AND overlay 9    -> impassable and scenery, one cell
//	border   (0,0)  a water word            -> water, on a cell already 0x03
//	border   (1,0)  overlay 3               -> scenery, likewise
func censusMap() *alm.Map {
	m := passMap(passW, passH)
	m.Tiles[8*passW+8] = impassableWord
	m.Tiles[8*passW+9] = waterWord
	m.Tiles[8*passW+10] = mountainWord
	m.Overlay[8*passW+11] = 7
	m.Tiles[8*passW+12] = impassableWord
	m.Overlay[8*passW+12] = 9
	m.Tiles[0] = waterWord
	m.Overlay[1] = 3
	return m
}

func TestTheCensusReportsEveryNumberFR8Asks(t *testing.T) {
	c := mapload.Census(censusMap())

	// 512 ring cells at 0x03, five blocked interior cells at 0x01, the other 59
	// interior cells open, and nothing at 0x02 — the border is the air bit's only
	// writer and it sets the ground bit with it.
	want := mapload.Counts{
		Value:      [4]int{59, 5, 0, 512},
		Ground:     517,
		Air:        512,
		Impassable: 2,
		Water:      2,
		Mountain:   1,
		Scenery:    3,
		Border:     512,
	}
	if c != want {
		t.Errorf("the census is\n %+v\nwant\n %+v", c, want)
	}

	total := c.Value[0] + c.Value[1] + c.Value[2] + c.Value[3]
	if total != passCells {
		t.Errorf("the per-value counts sum to %d, want one entry per cell (%d)", total, passCells)
	}
	if arms := c.Impassable + c.Water + c.Mountain + c.Scenery + c.Border; arms == c.Ground {
		t.Errorf("the five arm counts sum to %d, exactly the %d cells that block ground — on this fixture "+
			"they overlap and must not, or the counts are not being taken with the others absent",
			arms, c.Ground)
	}
}

func TestTheCensusAndThePlaneComeFromOneClassifier(t *testing.T) {
	group7 := passMap(passW, passH)
	for k := 0; k < 56; k++ {
		x, y := passLay(k)
		group7.Tiles[y*passW+x] = groupWord(7, k/14, k%14)
	}

	for _, c := range []struct {
		what string
		m    *alm.Map
	}{
		{"the census fixture", censusMap()},
		{"an empty 24x24", passMap(passW, passH)},
		{"the group-7 walk", group7},
		{"a 16x24, border throughout", passMap(16, 24)},
		{"no map at all", nil},
		{"a width of 0", &alm.Map{Width: 0, Height: passH}},
	} {
		t.Run(c.what, func(t *testing.T) {
			plane := mapload.Passability(c.m)
			got := mapload.Census(c.m)

			var want mapload.Counts
			for _, b := range plane {
				want.Value[b]++
				if b&0x01 != 0 {
					want.Ground++
				}
				if b&0x02 != 0 {
					want.Air++
				}
			}
			if got.Value != want.Value || got.Ground != want.Ground || got.Air != want.Air {
				t.Errorf("the census reports values %v, ground %d, air %d; the plane holds values %v, "+
					"ground %d, air %d", got.Value, got.Ground, got.Air, want.Value, want.Ground, want.Air)
			}
		})
	}
}
