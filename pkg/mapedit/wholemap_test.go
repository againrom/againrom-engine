package mapedit_test

import (
	"bytes"
	"math/rand"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// terrainSide is the side of the whole-map test map.
const terrainSide = 48

// terrainMap is a 48x48 Grass map at height 64 with a lake, a sand field, a
// mountain and a stony patch painted by the brush, a blocked cell, two units,
// and objects in a row of cells: transitions of several strip groups, and
// records and planes a whole-map command must leave as they are.
func terrainMap(t *testing.T) *mapedit.Editor {
	t.Helper()
	m, err := mapedit.NewMap(alm.NewMapSpec{Width: terrainSide, Height: terrainSide, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		class uint8
		at    mapedit.Cell
		r     int
	}{{mapedit.ClassWater, mapedit.Cell{X: 20, Y: 22}, 4}, {mapedit.ClassSand, mapedit.Cell{X: 34, Y: 14}, 3},
		{mapedit.ClassMountain, mapedit.Cell{X: 12, Y: 36}, 2}, {mapedit.ClassStones, mapedit.Cell{X: 36, Y: 36}, 3}} {
		b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(s.class), Radius: s.r}
		if _, err := m.Paint(b, []mapedit.Cell{s.at}, testClass); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := m.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 1}, []mapedit.Cell{{X: 30, Y: 30}}, nil); err != nil {
		t.Fatal(err)
	}
	for _, x := range []uint32{10, 26} {
		rec, err := m.NewUnitRecord(x<<8|128, 12<<8|128, 27, 3, 2)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.PlaceUnit(rec); err != nil {
			t.Fatal(err)
		}
	}
	for x := 8; x < 40; x += 3 {
		if err := m.SetCell(mapedit.GridRaw, mapedit.GridOverlay, x, 28, uint16(uint8(x))); err != nil {
			t.Fatal(err)
		}
	}
	again, err := mapedit.New(m.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return again
}

// source is the random source a whole-map command takes for seed.
func source(seed int64) func(int) int { return rand.New(rand.NewSource(seed)).Intn }

// outsidePayload is data without the payload of the record of typeId tid.
func outsidePayload(t *testing.T, data []byte, tid uint32) []byte {
	t.Helper()
	f := walkFrame(t, data).byType(t, tid)
	out, err := remainder(data, []region{{off: f.payloadOff, n: f.payloadSize}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Random terrain: one seed gives one map twice, another seed another; only
// the tile plane changes, every changed cell keeps its class, its shape and
// its strip group and so every corner, and bit 13; one Undo gives back the
// bytes.
func TestRandomTerrainFollowsItsSeedAndKeepsEveryCorner(t *testing.T) {
	run := func(seed int64) (*mapedit.Editor, []byte) {
		m := terrainMap(t)
		before := m.Bytes()
		changed, err := m.RandomizeTerrain(source(seed))
		if err != nil || !changed {
			t.Fatalf("seed %d: changed %v, %v", seed, changed, err)
		}
		return m, before
	}
	a, before := run(1)
	b, _ := run(1)
	c, _ := run(2)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("one seed gave two maps")
	}
	if bytes.Equal(a.Bytes(), c.Bytes()) {
		t.Fatal("two seeds gave one map")
	}
	after := a.Bytes()
	if !bytes.Equal(outsidePayload(t, before, 1), outsidePayload(t, after, 1)) {
		t.Fatal("random terrain wrote outside the tile plane")
	}
	old, err := alm.Open(before)
	if err != nil {
		t.Fatal(err)
	}
	now, err := alm.Open(after)
	if err != nil {
		t.Fatal(err)
	}
	changed, mixed := 0, 0
	for i, w := range old.Tiles {
		v := now.Tiles[i]
		if v == w {
			continue
		}
		changed++
		ws, wg, wok := mapedit.WordShape(w)
		vs, vg, vok := mapedit.WordShape(v)
		if !wok || !vok || ws != vs || wg != vg || testClass(w) != testClass(v) || w&0x2000 != v&0x2000 {
			t.Fatalf("cell %d: %#x became %#x", i, w, v)
		}
		if ws.Kind != mapedit.ShapePure {
			mixed++
		}
	}
	if changed == 0 || mixed == 0 {
		t.Fatalf("%d cells changed, %d of them transitions", changed, mixed)
	}
	if now.Tiles[30*terrainSide+30]&0x2000 == 0 {
		t.Fatal("the blocked cell lost its flag")
	}
	if !a.Undo() || !bytes.Equal(a.Bytes(), before) {
		t.Fatal("one Undo did not give back the map")
	}
}

// Random heights: one seed gives one map twice, another seed another; only
// the height plane changes, within 0..127; one Undo gives back the bytes.
func TestRandomHeightsFollowsItsSeedAndWritesOnlyHeights(t *testing.T) {
	run := func(seed int64) (*mapedit.Editor, []byte) {
		m := terrainMap(t)
		before := m.Bytes()
		changed, err := m.RandomizeHeights(source(seed))
		if err != nil || !changed {
			t.Fatalf("seed %d: changed %v, %v", seed, changed, err)
		}
		return m, before
	}
	a, before := run(5)
	b, _ := run(5)
	c, _ := run(6)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("one seed gave two maps")
	}
	if bytes.Equal(a.Bytes(), c.Bytes()) {
		t.Fatal("two seeds gave one map")
	}
	if !bytes.Equal(outsidePayload(t, before, 2), outsidePayload(t, a.Bytes(), 2)) {
		t.Fatal("random heights wrote outside the height plane")
	}
	now, err := alm.Open(a.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	lo, hi := 255, 0
	for _, h := range now.Altitudes {
		lo, hi = min(lo, int(h)), max(hi, int(h))
	}
	if hi > alm.MaxHeight || hi-lo < 16 {
		t.Fatalf("heights %d..%d", lo, hi)
	}
	if !a.Undo() || !bytes.Equal(a.Bytes(), before) {
		t.Fatal("one Undo did not give back the map")
	}
}

// Neither command runs without a source, and a refusal writes nothing.
func TestWholeMapCommandsRefuseNoSource(t *testing.T) {
	m := terrainMap(t)
	before := m.Bytes()
	if _, err := m.RandomizeTerrain(nil); err == nil {
		t.Fatal("random terrain ran without a source")
	}
	if _, err := m.RandomizeHeights(nil); err == nil {
		t.Fatal("random heights ran without a source")
	}
	if !bytes.Equal(m.Bytes(), before) || m.CanUndo() {
		t.Fatal("a refusal wrote or recorded")
	}
}
