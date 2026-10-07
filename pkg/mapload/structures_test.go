package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// The four positions the contract reads, spelt here from the contract and NOT
// imported from the implementation: a test that took the production constants
// would agree with whatever they became.
const (
	slotSizeX    = 0
	slotSizeY    = 1
	slotBlocking = 4
	slotAttach   = 5
	bldRowWidth  = 6
)

// bldRow is one Buildings row: the four positions the contract reads, and no
// more. bldRowOf is the same with an explicit width, for the short-row case.
func bldRow(w, h int, blocking, attach uint32) []int32 {
	return bldRowOf(bldRowWidth, w, h, blocking, attach)
}

func bldRowOf(width, w, h int, blocking, attach uint32) []int32 {
	p := make([]int32, width)
	for i := range p {
		p[i] = -1
	}
	set := func(slot int, v int32) {
		if slot < width {
			p[slot] = v
		}
	}
	set(slotSizeX, int32(w))
	set(slotSizeY, int32(h))
	set(slotBlocking, int32(blocking))
	set(slotAttach, int32(attach))
	return p
}

// bldTable builds a Buildings collection whose entry 0 is the reserved unwritten
// one and whose other entries sit at the indices given — every gap between them
// filled with an unwritten row, so a key that names a gap is still in range.
func bldTable(rows map[int][]int32) *mapload.Table {
	high := 0
	for i := range rows {
		if i > high {
			high = i
		}
	}
	c := make(defCollection, high+1)
	for i, p := range rows {
		c[i] = defEntry{name: "e", params: p}
	}
	return &mapload.Table{Buildings: c}
}

// structMap is a map carrying the placements given and nothing else. The extent
// is generous enough that a footprint anchored inside it stays inside.
func structMap(objs ...alm.Object) *alm.Map {
	return &alm.Map{Width: 24, Height: 24, Tiles: make([]uint16, 24*24),
		Overlay: make([]uint8, 24*24), Objects: objs}
}

// place is one placement at cell (col,row): the fixed-point anchor is the cell
// shifted up by 8 plus the half-cell every shipped record carries, so the
// fraction the derivation drops is exercised rather than left at zero.
func place(kind uint32, col, row int, ext []byte) alm.Object {
	return alm.Object{X: uint32(col)<<8 | 0x80, Y: uint32(row)<<8 | 0x80, Kind: kind, Field0C: 1, Ext: ext}
}

// ext builds an eight-byte extension carrying w and h at the two bytes that
// decide, and a distinctive value in every byte that must not.
func ext(w, h byte) []byte {
	return []byte{w, 0x5a, 0x5a, 0x5a, h, 0xa5, 0xa5, 0xa5}
}

// AC-1 — resolution against a table with a reserved entry 0, a full entry, a
// short entry and one whose width exceeds a byte.
func TestFootprintResolution(t *testing.T) {
	t.Parallel()

	const wideX = 0x103 // low byte 3
	tbl := bldTable(map[int][]int32{
		1:  bldRow(3, 2, 0b010, 0b111),
		2:  bldRowOf(5, 4, 4, 0, 0), // five parameters: too short to read position 5
		3:  bldRowOf(bldRowWidth, wideX, 2, 0, 0b1111),
		33: bldRow(1, 1, 1, 1),
	})

	m := structMap(
		place(0, 10, 10, nil),
		place(1, 10, 11, nil),
		place(2, 10, 12, nil),
		place(3, 10, 13, nil),
		place(34, 10, 14, nil),
		place(0x121, 10, 15, ext(9, 9)), // low byte 0x21, but not the extension kind
	)

	// The table's own rows, copied before the call: nothing here may write
	// through the collection it was handed.
	before := make([][]int32, 0, 4)
	for _, i := range []int{1, 2, 3, 33} {
		row := tbl.Buildings.EntryParams(i)
		before = append(before, append([]int32(nil), row...))
	}

	got, c := mapload.Footprints(m, tbl)

	if len(got) != 3 {
		t.Fatalf("resolved %d footprints, want 3: %+v", len(got), got)
	}
	if got[0].Width != 3 || got[0].Height != 2 || got[0].Blocking != 0b010 || got[0].Attach != 0b111 {
		t.Errorf("key 1 resolved to %+v", got[0])
	}
	if got[0].Col != 10 || got[0].Row != 11 {
		t.Errorf("key 1 anchored at (%d,%d), want (10,11)", got[0].Col, got[0].Row)
	}
	if got[1].Width != 3 {
		t.Errorf("key 3 width %d, want the low byte of %#x", got[1].Width, wideX)
	}
	// 0x121 names entry 33 by its low byte and carries NO extension, so its
	// extents are that entry's 1x1 and not the record's 9x9.
	if got[2].Width != 1 || got[2].Height != 1 {
		t.Errorf("key 0x121 resolved to %dx%d, want the 1x1 entry 33 names", got[2].Width, got[2].Height)
	}

	want := mapload.StructureCounts{Resolved: 3, Unresolved: 2, Short: 1}
	if c != want {
		t.Errorf("counts %+v, want %+v", c, want)
	}
	if n := c.Resolved + c.Unresolved + c.Short + c.ZeroExtent; n != len(m.Objects) {
		t.Errorf("placement counters sum to %d over %d placements", n, len(m.Objects))
	}

	for n, i := range []int{1, 2, 3, 33} {
		after := tbl.Buildings.EntryParams(i)
		if len(after) != len(before[n]) {
			t.Fatalf("entry %d changed width", i)
		}
		for j := range after {
			if after[j] != before[n][j] {
				t.Errorf("entry %d parameter %d moved %d -> %d", i, j, before[n][j], after[j])
			}
		}
	}
}

func TestStructureBuilderKeepsUnequalFootprintMasks(t *testing.T) {
	tbl := bldTable(map[int][]int32{
		1:    bldRow(3, 2, 0b010, 0b111),
		0x21: bldRow(1, 1, 1, 1),
	})
	m := structMap(place(1, 10, 11, nil), place(0x21, 20, 21, ext(3, 4)))
	got := mapload.Structures(m, tbl)
	if len(got) != 2 || got[0].Width != 3 || got[0].Height != 2 || got[0].Blocking != 0b010 || got[0].Attach != 0b111 {
		t.Fatalf("table footprint: %+v", got)
	}
	if got[1].Width != 3 || got[1].Height != 4 || got[1].Blocking != 0 || got[1].Attach != ^uint32(0) {
		t.Fatalf("extension footprint: %+v", got[1])
	}
}

// AC-4, resolution half — the extension's two extents, its dead bytes, its sum
// gate and the two shapes that yield nothing.
func TestExtensionResolution(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{
		0x21: bldRow(1, 1, 0b1, 0b1),
	})

	t.Run("the record's extents win and both sets are replaced", func(t *testing.T) {
		m := structMap(place(0x21, 2, 3, ext(3, 4)))
		got, c := mapload.Footprints(m, tbl)
		if len(got) != 1 {
			t.Fatalf("resolved %d, want 1", len(got))
		}
		f := got[0]
		if f.Width != 3 || f.Height != 4 {
			t.Errorf("extents %dx%d, want 3x4 from the record", f.Width, f.Height)
		}
		if f.Attach != ^uint32(0) {
			t.Errorf("attach set %#x, want every bit", f.Attach)
		}
		if f.Blocking != 0 {
			t.Errorf("blocking set %#x, want none", f.Blocking)
		}
		if c.Resolved != 1 {
			t.Errorf("counts %+v", c)
		}
	})

	t.Run("the six other bytes decide nothing", func(t *testing.T) {
		quiet := []byte{3, 0, 0, 0, 4, 0, 0, 0}
		a, _ := mapload.Footprints(structMap(place(0x21, 2, 3, ext(3, 4))), tbl)
		b, _ := mapload.Footprints(structMap(place(0x21, 2, 3, quiet)), tbl)
		if a[0] != b[0] {
			t.Errorf("the noise bytes moved the footprint: %+v vs %+v", a[0], b[0])
		}
	})

	t.Run("both extent bytes zero takes the entry's own row", func(t *testing.T) {
		m := structMap(place(0x21, 2, 3, ext(0, 0)))
		got, c := mapload.Footprints(m, tbl)
		if len(got) != 1 {
			t.Fatalf("resolved %d, want 1 — a zero sum must fall through to the entry", len(got))
		}
		f := got[0]
		if f.Width != 1 || f.Height != 1 || f.Blocking != 0b1 || f.Attach != 0b1 {
			t.Errorf("got %+v, want entry 0x21's own 1x1 row and both its sets", f)
		}
		if c.ZeroExtent != 0 {
			t.Errorf("counts %+v, want no zero-extent skip", c)
		}
	})

	t.Run("one zero extent byte yields nothing", func(t *testing.T) {
		_, c := mapload.Footprints(structMap(place(0x21, 2, 3, ext(0, 4))), tbl)
		if (c != mapload.StructureCounts{ZeroExtent: 1}) {
			t.Errorf("counts %+v, want one zero-extent skip", c)
		}
	})

	t.Run("an extension resolving no entry yields nothing", func(t *testing.T) {
		bare := bldTable(map[int][]int32{1: bldRow(1, 1, 0, 1)})
		_, c := mapload.Footprints(structMap(place(0x21, 2, 3, ext(3, 4))), bare)
		if (c != mapload.StructureCounts{Unresolved: 1}) {
			t.Errorf("counts %+v, want one unresolved", c)
		}
	})

	t.Run("a zero extent in the entry itself yields nothing", func(t *testing.T) {
		zero := bldTable(map[int][]int32{1: bldRow(0, 4, 0, 1)})
		_, c := mapload.Footprints(structMap(place(1, 2, 3, nil)), zero)
		if (c != mapload.StructureCounts{ZeroExtent: 1}) {
			t.Errorf("counts %+v, want one zero-extent skip", c)
		}
	})
}

func TestFootprintResolutionTotal(t *testing.T) {
	t.Parallel()

	tbl := bldTable(map[int][]int32{1: bldRow(2, 2, 0b11, 0b1111)})
	tables := map[string]*mapload.Table{
		"nil table":     nil,
		"empty table":   {},
		"no buildings":  {Units: defCollection{}},
		"one entry":     tbl,
		"zero-length":   {Buildings: defCollection{}},
		"reserved only": {Buildings: defCollection{{}}},
	}

	var objs []alm.Object
	for k := 0; k < 300; k++ {
		objs = append(objs, place(uint32(k), k%20, k%20, nil))
	}
	objs = append(objs, place(0x21, 1, 1, nil), place(0x21, 1, 1, []byte{1, 2}))
	m := structMap(objs...)

	for name, tb := range tables {
		first, c := mapload.Footprints(m, tb)
		second, c2 := mapload.Footprints(m, tb)
		if len(first) != len(second) || c != c2 {
			t.Errorf("%s: two calls disagreed", name)
		}
		for i := range first {
			if first[i] != second[i] {
				t.Errorf("%s: footprint %d differs between calls", name, i)
			}
		}
		if n := c.Resolved + c.Unresolved + c.Short + c.ZeroExtent; n != len(m.Objects) {
			t.Errorf("%s: placement counters sum to %d over %d placements", name, n, len(m.Objects))
		}
		if c.Resolved != len(first) {
			t.Errorf("%s: %d resolved but %d footprints", name, c.Resolved, len(first))
		}
	}

	if got, c := mapload.Footprints(nil, tbl); got != nil || (c != mapload.StructureCounts{}) {
		t.Errorf("a nil map resolved %v / %+v, want nothing", got, c)
	}
}

// planeOf derives the plane for one map against one table, and returns it beside
// the arms-only plane the same map yields — so every assertion below is about
// what the STRUCTURE moved and not about the terrain under it.
func planeOf(t *testing.T, m *alm.Map, tbl *mapload.Table) (got, arms []byte) {
	t.Helper()
	return mapload.PassabilityWith(m, tbl), mapload.Passability(m)
}

// AC-2 — five of six cells attach, row-major from the anchor running right and
// down; two of the five close and three open; the sixth is untouched.
func TestFootprintWalkRowMajor(t *testing.T) {
	t.Parallel()

	// 3x2. Attach names every cell but the last (bit 5); blocking names bits 1
	// and 3 — the second cell of the top row and the first of the bottom.
	tbl := bldTable(map[int][]int32{1: bldRow(3, 2, 0b001010, 0b011111)})
	const ax, ay = 10, 10
	m := structMap(place(1, ax, ay, nil))

	got, arms := planeOf(t, m, tbl)
	c := mapload.StructureCensus(m, tbl)

	// The interior of this fixture is open before the pass, so a closed byte
	// below is the structure's doing and nothing else's.
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 3; dx++ {
			if arms[(ay+dy)*24+ax+dx] != 0 {
				t.Fatalf("the fixture's cell (%d,%d) was not open before the pass", ax+dx, ay+dy)
			}
		}
	}

	want := map[[2]int]byte{
		{0, 0}: 0, {1, 0}: 1, {2, 0}: 0,
		{0, 1}: 1, {1, 1}: 0,
	}
	for off, w := range want {
		if b := got[(ay+off[1])*24+ax+off[0]]; b != w {
			t.Errorf("offset %v came out %#02x, want %#02x", off, b, w)
		}
	}
	if b := got[(ay+1)*24+ax+2]; b != arms[(ay+1)*24+ax+2] {
		t.Errorf("the unnamed sixth cell moved to %#02x", b)
	}

	wantC := mapload.StructureCounts{Resolved: 1, Attached: 5, Closed: 2, Opened: 3}
	if c != wantC {
		t.Errorf("counts %+v, want %+v", c, wantC)
	}
}

// AC-3 — an 11x4 footprint: cell 32 shares cell 0's bit, in both sets.
func TestFootprintAliasesPast32Cells(t *testing.T) {
	t.Parallel()

	// Bit 0 alone in both sets. Cell 32 is (dx=10, dy=2): 2*11+10 == 32, and
	// 32 mod 32 == 0, so it is the ONLY other cell either set names.
	tbl := bldTable(map[int][]int32{1: bldRow(11, 4, 1, 1)})
	const ax, ay = 6, 6
	m := structMap(place(1, ax, ay, nil))

	got, _ := planeOf(t, m, tbl)
	c := mapload.StructureCensus(m, tbl)

	if b := got[(ay+2)*24+ax+10]; b&1 == 0 {
		t.Errorf("cell 32 came out %#02x — an index taken without the modulus leaves it untouched", b)
	}
	if b := got[ay*24+ax]; b&1 == 0 {
		t.Errorf("cell 0 came out %#02x", b)
	}
	wantC := mapload.StructureCounts{Resolved: 1, Attached: 2, Closed: 2}
	if c != wantC {
		t.Errorf("counts %+v, want %+v — exactly cells 0 and 32", c, wantC)
	}
}

// AC-4, walk half — an extension placement occupies its whole rectangle and
// opens every cell of it, subtracting whatever the arms had put there.
func TestExtensionOpensItsWholeRectangle(t *testing.T) {
	t.Parallel()

	// The entry closes its single cell; the record's own 3x4 must override both
	// the extents and both sets. The rectangle is laid ON THE BORDER RING, which
	// the arms close to a ground and an air mover, so the subtraction is visible
	// and the air bit's immunity is testable in the same fixture.
	tbl := bldTable(map[int][]int32{0x21: bldRow(1, 1, 0b1, 0b1)})
	const ax, ay = 2, 2
	m := structMap(place(0x21, ax, ay, ext(3, 4)))

	got, arms := planeOf(t, m, tbl)
	c := mapload.StructureCensus(m, tbl)

	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 3; dx++ {
			i := (ay+dy)*24 + ax + dx
			if arms[i]&1 == 0 {
				t.Fatalf("the fixture's cell (%d,%d) was already open", ax+dx, ay+dy)
			}
			if got[i]&1 != 0 {
				t.Errorf("cell (%d,%d) came out %#02x, want bit 0 subtracted", ax+dx, ay+dy, got[i])
			}
			if got[i]&2 != arms[i]&2 {
				t.Errorf("cell (%d,%d) moved its air bit", ax+dx, ay+dy)
			}
		}
	}
	wantC := mapload.StructureCounts{Resolved: 1, Attached: 12, Opened: 12}
	if c != wantC {
		t.Errorf("counts %+v, want %+v", c, wantC)
	}
}

// AC-5 — a clash refuses one cell, abandons the remainder of that footprint, and
// leaves what it had already attached attached. Reversing the record order
// reverses which placement is which.
func TestOverlapRefusesAndAbandons(t *testing.T) {
	t.Parallel()

	// 4x1, every cell named by both sets. The later placement starts three
	// columns along, so its FIRST cell is free and its SECOND clashes with the
	// earlier one's last.
	tbl := bldTable(map[int][]int32{1: bldRow(4, 1, 0b1111, 0b1111)})
	const ax, ay = 10, 10

	m := structMap(place(1, ax, ay, nil), place(1, ax+4, ay, nil))
	got, _ := planeOf(t, m, tbl)
	c := mapload.StructureCensus(m, tbl)
	if (c != mapload.StructureCounts{Resolved: 2, Attached: 8, Closed: 8}) {
		t.Fatalf("the disjoint control clashed: %+v", c)
	}
	_ = got

	// Now overlap by one: the later footprint's second cell is the earlier's
	// fourth.
	over := structMap(place(1, ax, ay, nil), place(1, ax+3, ay, nil))
	ogot, arms := planeOf(t, over, tbl)
	oc := mapload.StructureCensus(over, tbl)

	wantC := mapload.StructureCounts{Resolved: 2, Attached: 4, Closed: 4, Refused: 1, Abandoned: 3}
	if oc != wantC {
		t.Errorf("counts %+v, want %+v", oc, wantC)
	}
	if n := oc.Attached + oc.Dropped + oc.Refused + oc.Abandoned; n != 8 {
		t.Errorf("cell counters sum to %d over 8 named cells", n)
	}
	for dx := 0; dx < 4; dx++ {
		if ogot[ay*24+ax+dx]&1 == 0 {
			t.Errorf("the earlier footprint lost cell %d", dx)
		}
	}
	// The later footprint attached nothing: its first cell IS the earlier one's
	// fourth, so the clash is at dx 0 and every cell behind it is abandoned.
	for dx := 4; dx < 7; dx++ {
		i := ay*24 + ax + dx
		if ogot[i] != arms[i] {
			t.Errorf("cell %d past the clash moved anyway", dx)
		}
	}

	// The walk's ORDER, which a single-row fixture cannot see: a 3x2 footprint
	// whose clash sits at the last cell of its TOP row. Row by row the clash is
	// the third cell walked, so two attach and the three behind it are
	// abandoned; column by column it would be the fifth, and four would attach.
	rows := bldTable(map[int][]int32{
		1: bldRow(1, 1, 0b1, 0b1),
		2: bldRow(3, 2, 0b111111, 0b111111),
	})
	const bx, by = 14, 14
	order := structMap(place(1, bx+2, by, nil), place(2, bx, by, nil))
	oc2 := mapload.StructureCensus(order, rows)
	if (oc2 != mapload.StructureCounts{Resolved: 2, Attached: 3, Closed: 3, Refused: 1, Abandoned: 3}) {
		t.Errorf("counts %+v — the rectangle is not walked row by row", oc2)
	}

	// Reversed: the placement that was later goes first, keeps all four, and the
	// other one now clashes on its own last cell — three attached, one refused.
	rev := structMap(place(1, ax+3, ay, nil), place(1, ax, ay, nil))
	rgot, _ := planeOf(t, rev, tbl)
	rc := mapload.StructureCensus(rev, tbl)
	if (rc != mapload.StructureCounts{Resolved: 2, Attached: 7, Closed: 7, Refused: 1}) {
		t.Errorf("reversed counts %+v", rc)
	}
	for dx := 0; dx < 7; dx++ {
		if rgot[ay*24+ax+dx]&1 == 0 {
			t.Errorf("reversed: cell %d did not attach", dx)
		}
	}
}

// AC-6 — the pass overrides whichever arm closed a cell, never touches an air
// bit, and never sets a bit above bit 1.
func TestPassOverridesEveryArm(t *testing.T) {
	t.Parallel()

	// A 4x1 footprint laid across four cells the arms close for four different
	// reasons: water, mountain, scenery and the border ring. Its blocking set
	// names the first two and not the last two.
	tbl := bldTable(map[int][]int32{1: bldRow(4, 1, 0b0011, 0b1111)})

	const row = 12
	m := passMap(24, 24)
	// water: the tile index inside [512,768). mountain: strip group 7 at a
	// blend level at or above 3. scenery: a nonzero overlay byte. The fourth
	// cell is the ring itself, at column 23.
	m.Tiles[row*24+20] = 600
	m.Tiles[row*24+21] = 7<<6 | 1<<4 | 1 // group 7, blend column 1, sub-cell 1
	m.Overlay[row*24+22] = 3
	m.Objects = []alm.Object{place(1, 20, row, nil)}

	arms := mapload.Passability(m)
	for dx := 0; dx < 4; dx++ {
		if arms[row*24+20+dx]&1 == 0 {
			t.Fatalf("the fixture's cell at column %d was not closed by an arm", 20+dx)
		}
	}
	if arms[row*24+23]&2 == 0 {
		t.Fatalf("the fixture's fourth cell is not on the ring")
	}

	got := mapload.PassabilityWith(m, tbl)
	for dx := 0; dx < 4; dx++ {
		i := row*24 + 20 + dx
		wantClosed := dx < 2
		if closed := got[i]&1 != 0; closed != wantClosed {
			t.Errorf("column %d came out %#02x, want bit 0 %v", 20+dx, got[i], wantClosed)
		}
		if got[i]&2 != arms[i]&2 {
			t.Errorf("column %d moved its air bit: %#02x from %#02x", 20+dx, got[i], arms[i])
		}
		if got[i]&^0x03 != 0 {
			t.Errorf("column %d set a reserved bit: %#02x", 20+dx, got[i])
		}
	}
}

// AC-7 — a footprint overhanging the last column and row, and one anchored
// wholly outside.
func TestFootprintOutsideTheExtent(t *testing.T) {
	t.Parallel()

	// 3x2, every cell attached. The blocking set names both cells of the first
	// two columns and NEITHER cell of the third — which is the column that
	// overhangs, so a walk that wrapped instead of dropping would OPEN a ring
	// cell of the next row and be visible against it.
	tbl := bldTable(map[int][]int32{1: bldRow(3, 2, 0b011011, 0b111111)})

	// Anchored so that one column hangs off the far edge: the 3x2 at (22,22) of
	// a 24x24 map keeps its first two columns on both rows and drops its third.
	over := structMap(place(1, 22, 22, nil))
	got, arms := planeOf(t, over, tbl)
	c := mapload.StructureCensus(over, tbl)

	wantC := mapload.StructureCounts{Resolved: 1, Attached: 4, Closed: 4, Dropped: 2}
	if c != wantC {
		t.Errorf("counts %+v, want %+v", c, wantC)
	}
	// No cell of the next row is touched: the two cells a wrapping walk would
	// reach are (0,23) and (1,23), both on the ring and both closed to a ground
	// and an air mover — an opened one would read 0x02.
	for _, i := range []int{23 * 24, 23*24 + 1} {
		if arms[i] != 0x03 {
			t.Fatalf("the fixture's cell %d is not a ring cell", i)
		}
		if got[i] != arms[i] {
			t.Errorf("cell %d came out %#02x — the walk wrapped into the next row", i, got[i])
		}
	}

	// Wholly outside: no byte moves at all.
	outside := structMap(place(1, 400, 400, nil))
	ogot, oarms := planeOf(t, outside, tbl)
	for i := range oarms {
		if ogot[i] != oarms[i] {
			t.Fatalf("an off-map placement moved cell %d", i)
		}
	}
	oc := mapload.StructureCensus(outside, tbl)
	if (oc != mapload.StructureCounts{Resolved: 1, Dropped: 6}) {
		t.Errorf("counts %+v, want six dropped cells and nothing else", oc)
	}
}

// AC-10 — the polarity, on its own terms and referring to no requirement: two
// neighbouring cells the arms leave OPEN, both attach-named by one footprint,
// its blocking set naming the first and not the second. The named cell must come
// out closed and the unnamed one open. A build with the two arms exchanged
// answers the reverse on exactly this pair, and on nothing else here.
func TestBlockingSetPolarity(t *testing.T) {
	t.Parallel()

	// 2x1: bit 0 named by the blocking set, bit 1 not; both cells attached.
	tbl := bldTable(map[int][]int32{1: bldRow(2, 1, 0b01, 0b11)})
	const ax, ay = 12, 12
	m := structMap(place(1, ax, ay, nil))

	arms := mapload.Passability(m)
	named, unnamed := ay*24+ax, ay*24+ax+1
	if arms[named] != 0 || arms[unnamed] != 0 {
		t.Fatalf("the fixture's pair was not open before the pass: %#02x %#02x",
			arms[named], arms[unnamed])
	}

	got := mapload.PassabilityWith(m, tbl)
	if got[named]&1 == 0 {
		t.Errorf("the cell the blocking set NAMES came out %#02x — it must be closed to a ground mover",
			got[named])
	}
	if got[unnamed]&1 != 0 {
		t.Errorf("the cell the blocking set does NOT name came out %#02x — it must be open",
			got[unnamed])
	}

	c := mapload.StructureCensus(m, tbl)
	if c.Closed != 1 || c.Opened != 1 {
		t.Errorf("counts %+v, want one closed and one opened", c)
	}
}

func TestStructurePassProperties(t *testing.T) {
	t.Parallel()

	const w, h = 24, 24
	sets := []uint32{0, 1, 0xffffffff, 0x0f0f0f0f, 0x80000001}
	extents := [][2]int{{1, 1}, {3, 2}, {11, 4}, {255, 1}, {1, 255}, {8, 8}}
	anchors := [][2]int{{0, 0}, {12, 12}, {23, 23}, {22, 0}, {0, 22}, {500, 500}, {23, 0}}

	for si, blocking := range sets {
		for ai, attach := range sets {
			for ei, e := range extents {
				tbl := bldTable(map[int][]int32{1: bldRow(e[0], e[1], blocking, attach)})
				named := 0
				for dy := 0; dy < e[1]; dy++ {
					for dx := 0; dx < e[0]; dx++ {
						if attach>>((dy*e[0]+dx)%32)&1 == 1 {
							named++
						}
					}
				}
				for _, a := range anchors {
					m := passMap(w, h)
					m.Objects = []alm.Object{place(1, a[0], a[1], nil)}

					arms := mapload.Passability(m)
					got := mapload.PassabilityWith(m, tbl)
					c := mapload.StructureCensus(m, tbl)

					if len(got) != w*h {
						t.Fatalf("plane is %d bytes, want %d", len(got), w*h)
					}
					if n := c.Attached + c.Dropped + c.Refused + c.Abandoned; n != named {
						t.Fatalf("sets %d/%d extent %d anchor %v: counters sum to %d over %d named cells",
							si, ai, ei, a, n, named)
					}
					if c.Closed+c.Opened != c.Attached {
						t.Fatalf("closed+opened %d != attached %d", c.Closed+c.Opened, c.Attached)
					}
					for i := range got {
						if got[i]&^0x03 != 0 {
							t.Fatalf("byte %d is %#02x — a bit above bit 1", i, got[i])
						}
						if got[i]&0x02 != arms[i]&0x02 {
							t.Fatalf("byte %d moved its air bit", i)
						}
					}
					again := mapload.PassabilityWith(m, tbl)
					for i := range got {
						if got[i] != again[i] {
							t.Fatalf("two derivations disagreed at byte %d", i)
						}
					}
				}
			}
		}
	}
}
