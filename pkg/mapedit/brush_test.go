package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/mapedit"
)

// testClass is an independent reading of the movement classifier
// (TERR-PASS-050): water is the raw range 0x200..0x2ff of the masked index,
// otherwise the strip group's pair and the five-level blend select the class.
func testClass(w uint16) uint8 {
	i := int(w & 0x3ff)
	if i&0x300 == 0x200 {
		return mapedit.ClassWater
	}
	pairs := map[int][2]uint8{
		0: {mapedit.ClassGrass, mapedit.ClassLand}, 1: {mapedit.ClassCracked, mapedit.ClassLand},
		2: {mapedit.ClassSand, mapedit.ClassLand}, 3: {mapedit.ClassSavanna, mapedit.ClassLand},
		4: {mapedit.ClassStones, mapedit.ClassLand}, 5: {mapedit.ClassCracked, mapedit.ClassStones},
		6: {mapedit.ClassFlowers, mapedit.ClassSavanna}, 7: {mapedit.ClassMountain, mapedit.ClassStones},
		12: {mapedit.ClassRoad, mapedit.ClassLand},
	}
	levels := [4][14]uint8{
		{2, 3, 2, 4, 3, 4, 2, 2, 2, 2, 4, 4, 4, 4},
		{3, 5, 3, 3, 1, 3, 2, 4, 2, 2, 4, 2, 4, 4},
		{2, 3, 2, 4, 3, 4, 2, 4, 2, 2, 4, 2, 4, 4},
		{5, 5, 5, 5, 5, 5, 2, 2, 2, 2, 4, 4, 4, 4},
	}
	p, ok := pairs[i>>6&15]
	if !ok || i&15 >= 14 {
		return 0xff
	}
	if levels[i>>4&3][i&15] <= 2 {
		return p[1]
	}
	return p[0]
}

const brushSide = 24

// brushMap is a brushSide x brushSide map holding only the three required
// records, every tile the given word and every height the given byte.
func brushMap(tile uint16, height byte) []byte {
	n := brushSide * brushSide
	meta := make([]byte, metaSize)
	putU32(meta, metaW, brushSide)
	putU32(meta, metaH, brushSide)
	putU32(meta, metaBitmask, 0x7f)
	tiles := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		putU16(tiles, 2*i, tile)
	}
	alts := bytes.Repeat([]byte{height}, n)
	out := make([]byte, fileHeaderSize)
	putU32(out, fhMagic, fileMagic)
	putU32(out, fhHdrLen, fileHeaderSize)
	putU32(out, fhDataSize, uint32(4*n+72))
	putU32(out, fhRecordCount, 3)
	putU32(out, fhFormatVersion, formatVersion990)
	for tid, p := range [][]byte{meta, tiles, alts} {
		hdr := make([]byte, recordHeaderSize)
		putU32(hdr, recTag, recordTag)
		putU32(hdr, recHdrLen, recordHdrLen)
		putU32(hdr, recPayloadSize, uint32(len(p)))
		putU32(hdr, recTypeID, uint32(tid))
		putU32(hdr, recPerMapConst, fixPerMapConst)
		out = append(append(out, hdr...), p...)
	}
	return out
}

// planes reads the tile and height planes and the mask word out of a map image
// by the container layout.
func planes(t *testing.T, data []byte) (tiles []uint16, alts []byte, mask uint32) {
	t.Helper()
	fs := walkFrame(t, data)
	tf, af, mf := fs.byType(t, 1), fs.byType(t, 2), fs.byType(t, 0)
	for i := 0; i < brushSide*brushSide; i++ {
		tiles = append(tiles, u16At(data, tf.payloadOff+2*i))
	}
	return tiles, data[af.payloadOff : af.payloadOff+brushSide*brushSide], u32At(data, mf.payloadOff+metaBitmask)
}

// Derive keeps the class of the cell it words, whatever surrounds it.
func TestDeriveKeepsTheCellsClass(t *testing.T) {
	for c := uint8(1); c <= mapedit.ClassRoad; c++ {
		for x := uint8(1); x <= mapedit.ClassRoad; x++ {
			for mask := 0; mask < 256; mask++ {
				var around [8]uint8
				for d := range around {
					around[d] = c
					if mask>>d&1 == 1 {
						around[d] = x
					}
				}
				w := mapedit.Derive(c, around, mask, int(x))
				if got := testClass(w); got != c {
					t.Fatalf("class %d with %d at %08b: word %#x classifies as %d", c, x, mask, w, got)
				}
			}
		}
	}
}

func TestDeriveShapes(t *testing.T) {
	g, l := mapedit.ClassGrass, mapedit.ClassLand
	around := func(c uint8, at ...int) (a [8]uint8) {
		for i := range a {
			a[i] = c
		}
		for _, d := range at {
			a[d] = l
		}
		return a
	}
	// Directions in reading order: NW N NE W E SW S SE.
	const nw, n, ne, w, e, sw, s, se = 0, 1, 2, 3, 4, 5, 6, 7
	shape := func(c uint8, a [8]uint8) (mapedit.Shape, uint16) {
		word := mapedit.Derive(c, a, 3, 5)
		sh, group, ok := mapedit.WordShape(word)
		if !ok {
			t.Fatalf("Derive gave %#x, which has no shape", word)
		}
		return sh, group
	}
	for _, tc := range []struct {
		name string
		got  mapedit.Shape
		want mapedit.Shape
	}{
		{"uniform", must(shape(g, around(g)))(), mapedit.Shape{Primary: true, Kind: mapedit.ShapePure}},
		{"Land at N side", must(shape(g, around(g, nw, n, ne)))(), mapedit.Shape{Primary: true, Kind: mapedit.ShapeEdge, Dir: 0}},
		{"Land at E side", must(shape(g, around(g, ne, e, se)))(), mapedit.Shape{Primary: true, Kind: mapedit.ShapeEdge, Dir: 1}},
		{"Land at W side", must(shape(g, around(g, nw, w, sw)))(), mapedit.Shape{Primary: true, Kind: mapedit.ShapeEdge, Dir: 3}},
		{"Land cell, Grass at W SW S", must(shape(l, with(around(l), g, w, sw, s)))(), mapedit.Shape{Kind: mapedit.ShapeTip, Dir: 2}},
		{"Land cell, no Grass", must(shape(l, around(l)))(), mapedit.Shape{Kind: mapedit.ShapePure}},
		{"Land at S plus corners", must(shape(g, around(g, sw, s, se)))(), mapedit.Shape{Primary: true, Kind: mapedit.ShapeEdge, Dir: 2}},
		{"Land cell, Grass at E S SE", must(shape(l, with(around(l), g, e, s, se)))(), mapedit.Shape{Kind: mapedit.ShapeTip, Dir: 3}},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: shape %+v, want %+v", tc.name, tc.got, tc.want)
		}
	}
	// A water cell shows an outer corner where land lies at two adjacent
	// cardinals; a land cell has no such word and takes the inner corner.
	wt := mapedit.ClassWater
	if sh, _ := shape(wt, around(wt, se)); sh != (mapedit.Shape{Primary: true, Kind: mapedit.ShapeInner, Dir: 3}) {
		t.Errorf("water with land at SE only: %+v", sh)
	}
	if sh, _ := shape(wt, around(wt, n, w)); sh != (mapedit.Shape{Primary: true, Kind: mapedit.ShapeOuter, Dir: 0}) {
		t.Errorf("water with land at N and W: %+v", sh)
	}
}

func with(a [8]uint8, c uint8, at ...int) [8]uint8 {
	for _, d := range at {
		a[d] = c
	}
	return a
}

// A neighbour whose class shares no strip is read as the cell's hub, and a
// water neighbour is ignored by a land cell.
func TestDeriveHubAndWater(t *testing.T) {
	var uniform [8]uint8
	for i := range uniform {
		uniform[i] = mapedit.ClassGrass
	}
	toSand, toLand, toWater := uniform, uniform, uniform
	toSand[1], toLand[1], toWater[1] = mapedit.ClassSand, mapedit.ClassLand, mapedit.ClassWater
	if a, b := mapedit.Derive(mapedit.ClassGrass, toSand, 1, 1), mapedit.Derive(mapedit.ClassGrass, toLand, 1, 1); a != b {
		t.Errorf("Grass beside Sand %#x, beside Land %#x: the same hub reading is expected", a, b)
	}
	if got, want := mapedit.Derive(mapedit.ClassGrass, toWater, 1, 1), mapedit.Derive(mapedit.ClassGrass, uniform, 1, 1); got != want {
		t.Errorf("Grass beside Water %#x, uniform %#x", got, want)
	}
	// Flowers and Mountain have no Land strip; their hubs are Savanna and Stones.
	flowersLand := [8]uint8{mapedit.ClassFlowers, mapedit.ClassLand, mapedit.ClassFlowers, mapedit.ClassFlowers, mapedit.ClassFlowers, mapedit.ClassFlowers, mapedit.ClassFlowers, mapedit.ClassFlowers}
	flowersSav := flowersLand
	flowersSav[1] = mapedit.ClassSavanna
	if a, b := mapedit.Derive(mapedit.ClassFlowers, flowersLand, 1, 1), mapedit.Derive(mapedit.ClassFlowers, flowersSav, 1, 1); a != b {
		t.Errorf("Flowers beside Land %#x, beside Savanna %#x", a, b)
	}
}

func must[A, B any](a A, b B) func() A { return func() A { return a } }

// A stroke is one history step that writes the tile plane, and the type-0 mask
// word when the planes' groups change, and nothing else.
func TestTileStrokeWritesOnlyTheTilePlane(t *testing.T) {
	in := brushMap(0x14, 40)
	ed, err := mapedit.New(in)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassMountain), Radius: 2},
		[]mapedit.Cell{{X: 12, Y: 12}}, testClass)
	if err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	out := ed.Bytes()
	fs := walkFrame(t, in)
	tf := fs.byType(t, 1)
	mf := fs.byType(t, 0)
	for i := range in {
		if in[i] == out[i] {
			continue
		}
		inTiles := i >= tf.payloadOff && i < tf.payloadOff+tf.payloadSize
		inMask := i >= mf.payloadOff+metaBitmask && i < mf.payloadOff+metaBitmask+4
		if !inTiles && !inMask {
			t.Fatalf("byte %#x changed outside the tile plane and the mask word", i)
		}
	}
	tiles, _, mask := planes(t, out)
	if mask != 1|1<<7 {
		t.Errorf("mask = %#x, want groups 0 and 7", mask)
	}
	mountain := 0
	for y := 0; y < brushSide; y++ {
		for x := 0; x < brushSide; x++ {
			c := testClass(tiles[y*brushSide+x])
			inDisc := (x-12)*(x-12)+(y-12)*(y-12) <= 4
			if inDisc != (c == mapedit.ClassMountain) {
				t.Fatalf("cell (%d,%d): class %d, in disc %v", x, y, c, inDisc)
			}
			if c == mapedit.ClassMountain {
				mountain++
			}
			if tiles[y*brushSide+x]&^0x23ff != 0 {
				t.Fatalf("cell (%d,%d) word %#x carries bits 10..12 or 14..15", x, y, tiles[y*brushSide+x])
			}
		}
	}
	if mountain != 13 {
		t.Errorf("%d mountain cells, want the 13 of a radius-2 disc", mountain)
	}
	// Cells beyond the disc and its ring are untouched.
	for i := 0; i < brushSide*brushSide; i++ {
		x, y := i%brushSide, i/brushSide
		if (x-12)*(x-12)+(y-12)*(y-12) > 2*2+2*2+2 && tiles[i] != 0x14 {
			t.Fatalf("cell (%d,%d) outside the stroke's reach changed to %#x", x, y, tiles[i])
		}
	}
	if !ed.Undo() || !bytes.Equal(ed.Bytes(), in) || ed.CanUndo() {
		t.Fatal("one Undo does not restore the exact bytes as the only step")
	}
	if !ed.Redo() || !bytes.Equal(ed.Bytes(), out) {
		t.Fatal("Redo does not restore the stroke's bytes")
	}
}

// Painting a cell over a region already showing a class in the same shapes
// writes no byte and records no step.
func TestStrokeThatChangesNothingRecordsNothing(t *testing.T) {
	in := brushMap(0x14, 40)
	ed, _ := mapedit.New(in)
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassLand), Radius: 1},
		[]mapedit.Cell{{X: 10, Y: 10}}, testClass)
	if err != nil || changed || ed.CanUndo() || !bytes.Equal(ed.Bytes(), in) {
		t.Fatalf("Land over Land: changed=%v err=%v undo=%v", changed, err, ed.CanUndo())
	}
	changed, err = ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 0, Radius: 1}, []mapedit.Cell{{X: 10, Y: 10}}, nil)
	if err != nil || changed {
		t.Fatalf("a zero height change: changed=%v err=%v", changed, err)
	}
}

// A rejected stroke has written and recorded nothing.
func TestRejectedStrokeRecordsNothing(t *testing.T) {
	in := brushMap(0x14, 40)
	ed, _ := mapedit.New(in)
	bad := []struct {
		name string
		b    mapedit.Brush
		at   []mapedit.Cell
	}{
		{"outside", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 2}, []mapedit.Cell{{X: brushSide, Y: 0}}},
		{"negative", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 2}, []mapedit.Cell{{X: -1, Y: 0}}},
		{"no cells", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 2}, nil},
		{"class 0", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 0}, []mapedit.Cell{{X: 1, Y: 1}}},
		{"class 11", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 11}, []mapedit.Cell{{X: 1, Y: 1}}},
		{"class 258", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 258}, []mapedit.Cell{{X: 1, Y: 1}}},
		{"radius", mapedit.Brush{Plane: mapedit.PlaneTile, Value: 2, Radius: 99}, []mapedit.Cell{{X: 1, Y: 1}}},
		{"plane", mapedit.Brush{Plane: 9, Value: 2}, []mapedit.Cell{{X: 1, Y: 1}}},
	}
	for _, tc := range bad {
		if _, err := ed.Paint(tc.b, tc.at, testClass); err == nil {
			t.Errorf("%s: no error", tc.name)
		}
	}
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: 2}, []mapedit.Cell{{X: 1, Y: 1}}, nil); err == nil {
		t.Error("a tile stroke without a classifier was accepted")
	}
	if ed.CanUndo() || !bytes.Equal(ed.Bytes(), in) {
		t.Fatal("a rejected stroke changed the model")
	}
}

// A stroke on the last cell and the top rows follows the save normalisation
// (ALM-SAVE-238): the top eight rows take the last cell's word.
func TestTileStrokeNormalisesTheTopRows(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 40))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassGrass)},
		[]mapedit.Cell{{X: brushSide - 1, Y: brushSide - 1}}, testClass); err != nil {
		t.Fatal(err)
	}
	tiles, _, _ := planes(t, ed.Bytes())
	last := tiles[len(tiles)-1]
	for i := 0; i < 8*brushSide; i++ {
		if tiles[i] != last {
			t.Fatalf("top-row cell %d is %#x, the last cell is %#x", i, tiles[i], last)
		}
	}
	// A stroke in the middle leaves the top rows alone.
	ed, _ = mapedit.New(brushMap(0x14, 40))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassGrass)},
		[]mapedit.Cell{{X: 12, Y: 12}}, testClass); err != nil {
		t.Fatal(err)
	}
	tiles, _, _ = planes(t, ed.Bytes())
	if tiles[0] != 0x14 {
		t.Fatalf("a stroke in the middle changed the top row: %#x", tiles[0])
	}
}

func TestHeightStrokeWritesOnlyTheHeightPlane(t *testing.T) {
	in := brushMap(0x14, 40)
	ed, _ := mapedit.New(in)
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 10, Radius: 1}, []mapedit.Cell{{X: 12, Y: 12}}, nil)
	if err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	out := ed.Bytes()
	af := walkFrame(t, in).byType(t, 2)
	for i := range in {
		if in[i] != out[i] && (i < af.payloadOff || i >= af.payloadOff+af.payloadSize) {
			t.Fatalf("byte %#x changed outside the height plane", i)
		}
	}
	_, alts, mask := planes(t, out)
	if mask != 0x7f {
		t.Errorf("a height stroke moved the mask to %#x", mask)
	}
	for i, h := range alts {
		x, y := i%brushSide, i/brushSide
		want := byte(40)
		if (x-12)*(x-12)+(y-12)*(y-12) <= 1 {
			want = 50
		}
		if h != want {
			t.Fatalf("height (%d,%d) = %d, want %d", x, y, h, want)
		}
	}
	if !ed.Undo() || !bytes.Equal(ed.Bytes(), in) || ed.CanUndo() {
		t.Fatal("one Undo does not restore the heights")
	}
}

func TestHeightStrokeFalloffClampAndStep(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 120))
	// Clamp at 127.
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 60, Radius: 0}, []mapedit.Cell{{X: 5, Y: 5}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	if alts[5*brushSide+5] != 127 {
		t.Fatalf("clamped height = %d, want 127", alts[5*brushSide+5])
	}
	// Cosine fall-off: the centre rises most, the rim least, never below zero change.
	ed, _ = mapedit.New(brushMap(0x14, 20))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 30, Radius: 4, Falloff: mapedit.FalloffCosine}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ = planes(t, ed.Bytes())
	centre, near, rim := int(alts[12*brushSide+12]), int(alts[12*brushSide+14]), int(alts[12*brushSide+16])
	if !(centre > near && near > rim && rim >= 20) {
		t.Fatalf("fall-off: centre %d, two cells out %d, rim %d", centre, near, rim)
	}
	// The step to the row below is bounded at 31 inside the disc.
	ed, _ = mapedit.New(brushMap(0x14, 10))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 100, Radius: 3}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ = planes(t, ed.Bytes())
	for i := 0; i+brushSide < len(alts); i++ {
		if d := int(alts[i+brushSide]) - int(alts[i]); d > 31 {
			t.Fatalf("cell %d: the step to the row below is %d", i, d)
		}
	}
}

// The terrain stroke is deterministic: the same stroke on the same bytes gives
// the same bytes, and painting a band then repainting its class is stable.
func TestTileStrokeIsDeterministic(t *testing.T) {
	run := func() []byte {
		ed, _ := mapedit.New(brushMap(0x14, 40))
		var path []mapedit.Cell
		for x := 4; x < 20; x++ {
			path = append(path, mapedit.Cell{X: x, Y: 12})
		}
		if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassWater), Radius: 1}, path, testClass); err != nil {
			t.Fatal(err)
		}
		return ed.Bytes()
	}
	a, b := run(), run()
	if !bytes.Equal(a, b) {
		t.Fatal("two identical strokes gave different bytes")
	}
	tiles, _, _ := planes(t, a)
	for x := 4; x < 20; x++ {
		if c := testClass(tiles[12*brushSide+x]); c != mapedit.ClassWater {
			t.Fatalf("band cell %d is class %d", x, c)
		}
	}
}

// A stroke keeps bit 13 on unpainted rewritten cells, including the top rows.
func TestTileStrokeKeepsTheImpassableFlagOfUnpaintedCells(t *testing.T) {
	const flag = 0x2000
	in := brushMap(0x11, 40)
	tf := walkFrame(t, in).byType(t, 1)
	flagged := []int{12*brushSide + 12, 12*brushSide + 11, 3*brushSide + 5, brushSide*brushSide - 2}
	for _, i := range flagged {
		putU16(in, tf.payloadOff+2*i, 0x11|flag)
	}
	ed, err := mapedit.New(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassMountain)},
		[]mapedit.Cell{{X: 13, Y: 12}, {X: brushSide - 1, Y: brushSide - 1}}, testClass); err != nil {
		t.Fatal(err)
	}
	tiles, _, _ := planes(t, ed.Bytes())
	rewritten := 0
	for _, i := range []int{flagged[0], flagged[1], flagged[3]} {
		if tiles[i]&^flag != 0x11 {
			rewritten++
		}
		if tiles[i]&flag == 0 {
			t.Errorf("cell %d lost its impassable flag: %#x", i, tiles[i])
		}
	}
	if rewritten == 0 {
		t.Error("the stroke rewrote no flagged neighbour, so the test measures nothing")
	}
	if tiles[flagged[2]]&flag == 0 || tiles[flagged[2]]&^flag != tiles[0] {
		t.Errorf("top-row flagged cell %#x, top row cell 0 %#x", tiles[flagged[2]], tiles[0])
	}
	if tiles[0]&flag != 0 {
		t.Errorf("an unflagged top-row cell took the flag: %#x", tiles[0])
	}
	if p := tiles[12*brushSide+13]; p&flag != 0 {
		t.Errorf("the painted cell carries the flag: %#x", p)
	}
	if last := tiles[len(tiles)-1]; last&flag != 0 {
		t.Errorf("the painted last cell carries the flag: %#x", last)
	}
}

// objectMap is brushMap with a type-3 record of the given code in every cell.
func objectMap(tile uint16, height, code byte) []byte {
	in := brushMap(tile, height)
	n := brushSide * brushSide
	rec := make([]byte, recordHeaderSize)
	putU32(rec, recTag, recordTag)
	putU32(rec, recHdrLen, recordHdrLen)
	putU32(rec, recPayloadSize, uint32(n))
	putU32(rec, recTypeID, 3)
	putU32(rec, recPerMapConst, fixPerMapConst)
	in = append(in, rec...)
	in = append(in, bytes.Repeat([]byte{code}, n)...)
	putU32(in, fhRecordCount, 4)
	putU32(in, fhDataSize, uint32(5*n+72))
	return in
}

func objectCells(t *testing.T, data []byte) []byte {
	t.Helper()
	f := walkFrame(t, data).byType(t, 3)
	return data[f.payloadOff : f.payloadOff+brushSide*brushSide]
}

func TestObjectStrokeWritesTheClassCodeOverTheDiscAndNothingElse(t *testing.T) {
	in := objectMap(0x14, 40, 0)
	ed, err := mapedit.New(in)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneObject, Value: 7, Radius: 2}, []mapedit.Cell{{X: 12, Y: 12}}, nil)
	if err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	out := ed.Bytes()
	of := walkFrame(t, in).byType(t, 3)
	for i := range in {
		if in[i] != out[i] && (i < of.payloadOff || i >= of.payloadOff+of.payloadSize) {
			t.Fatalf("byte %#x changed outside the type-3 grid", i)
		}
	}
	for i, c := range objectCells(t, out) {
		x, y := i%brushSide, i/brushSide
		want := byte(0)
		if (x-12)*(x-12)+(y-12)*(y-12) <= 4 {
			want = 7
		}
		if c != want {
			t.Fatalf("object (%d,%d) = %d, want %d", x, y, c, want)
		}
	}
	if !ed.Undo() || !bytes.Equal(ed.Bytes(), in) || ed.CanUndo() {
		t.Fatal("one Undo does not restore the object bytes")
	}
}

func TestObjectClearRowWritesTheEmptyValue(t *testing.T) {
	in := objectMap(0x14, 40, 9)
	ed, _ := mapedit.New(in)
	if changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneObject, Value: 0, Radius: 1}, []mapedit.Cell{{X: 5, Y: 5}}, nil); err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	cells := objectCells(t, ed.Bytes())
	for i, c := range cells {
		x, y := i%brushSide, i/brushSide
		want := byte(9)
		if (x-5)*(x-5)+(y-5)*(y-5) <= 1 {
			want = 0
		}
		if c != want {
			t.Fatalf("object (%d,%d) = %d, want %d", x, y, c, want)
		}
	}
}

func TestObjectStrokeThatChangesNothingRecordsNothing(t *testing.T) {
	in := objectMap(0x14, 40, 9)
	ed, _ := mapedit.New(in)
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneObject, Value: 9, Radius: 2}, []mapedit.Cell{{X: 8, Y: 8}}, nil)
	if err != nil || changed || ed.CanUndo() || !bytes.Equal(ed.Bytes(), in) {
		t.Fatalf("same code: changed=%v err=%v undo=%v", changed, err, ed.CanUndo())
	}
}

func TestObjectStrokeIsRejectedWithoutATypeThreeRecord(t *testing.T) {
	in := brushMap(0x14, 40)
	ed, _ := mapedit.New(in)
	for _, v := range []int{5, 256, -1} {
		if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneObject, Value: v}, []mapedit.Cell{{X: 1, Y: 1}}, nil); err == nil {
			t.Errorf("code %d: no error", v)
		}
	}
	if ed.CanUndo() || !bytes.Equal(ed.Bytes(), in) {
		t.Fatal("a rejected stroke changed the model")
	}
}
