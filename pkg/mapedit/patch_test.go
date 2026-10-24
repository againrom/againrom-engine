package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// reliefMap is terrainMap with heights that differ cell by cell, so a pasted
// height is told apart from the one it covers.
func reliefMap(t *testing.T) *mapedit.Editor {
	t.Helper()
	m := terrainMap(t)
	if _, err := m.RandomizeHeights(source(3)); err != nil {
		t.Fatal(err)
	}
	again, err := mapedit.New(m.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	return again
}

// A patch over the lake's edge and the row of objects, pasted elsewhere,
// puts its tiles, heights and objects there cell for cell and writes nothing
// outside the three planes; one Undo gives back the bytes. Pasted at the
// map's edge it is cut there.
func TestPastePatchReproducesTerrainHeightsAndObjects(t *testing.T) {
	m := reliefMap(t)
	src, err := alm.Open(m.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	const x0, y0, w, h = 14, 24, 10, 6
	p, err := m.CopyPatch(x0, y0, w, h)
	if err != nil {
		t.Fatal(err)
	}
	trees := 0
	for _, b := range p.Objects {
		if b != 0 {
			trees++
		}
	}
	if trees == 0 {
		t.Fatal("the patch holds no objects")
	}
	for _, at := range []mapedit.Cell{{X: 30, Y: 10}, {X: terrainSide - 4, Y: terrainSide - 3}} {
		before := m.Bytes()
		changed, err := m.PastePatch(p, at)
		if err != nil || !changed {
			t.Fatalf("paste at %v: changed %v, %v", at, changed, err)
		}
		now, err := alm.Open(m.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		old, _ := alm.Open(before)
		for y := range terrainSide {
			for x := range terrainSide {
				i := y*terrainSide + x
				px, py := x-at.X, y-at.Y
				if px >= 0 && py >= 0 && px < w && py < h {
					s := (y0+py)*terrainSide + x0 + px
					if now.Tiles[i] != src.Tiles[s] || now.Altitudes[i] != src.Altitudes[s] || now.Overlay[i] != src.Overlay[s] {
						t.Fatalf("paste at %v: cell (%d,%d) is not the patch's", at, x, y)
					}
					continue
				}
				if y >= 8 && now.Tiles[i] != old.Tiles[i] || now.Altitudes[i] != old.Altitudes[i] || now.Overlay[i] != old.Overlay[i] {
					t.Fatalf("paste at %v: cell (%d,%d) outside the patch changed", at, x, y)
				}
			}
		}
		rest := func(data []byte) []byte {
			fs := walkFrame(t, data)
			var rs []region
			for _, tid := range []uint32{1, 2, 3} {
				f := fs.byType(t, tid)
				rs = append(rs, region{off: f.payloadOff, n: f.payloadSize})
			}
			out, err := remainder(data, rs)
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		if !bytes.Equal(rest(before), rest(m.Bytes())) {
			t.Fatalf("paste at %v wrote outside the three planes", at)
		}
		if !m.Undo() || !bytes.Equal(m.Bytes(), before) {
			t.Fatalf("paste at %v: one Undo did not give back the map", at)
		}
	}
}

// A paste over the top eight rows leaves them the last cell's word, the save
// normalisation every tile edit takes.
func TestPastePatchKeepsTheSaveNormalisation(t *testing.T) {
	m := reliefMap(t)
	p, err := m.CopyPatch(16, 20, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.PastePatch(p, mapedit.Cell{X: 4, Y: 2}); err != nil {
		t.Fatal(err)
	}
	now, err := alm.Open(m.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	last := now.Tiles[len(now.Tiles)-1]
	for i := range 8 * terrainSide {
		if now.Tiles[i]&^0x2000 != last&^0x2000 {
			t.Fatalf("top-row cell %d holds %#x, not the last cell's %#x", i, now.Tiles[i], last)
		}
	}
}

// A patch off the grid, a patch whose planes do not match its size, and a
// patch with objects onto a map with no type-3 record are refused and write
// nothing; the same patch without objects pastes there.
func TestPatchRefusals(t *testing.T) {
	m := reliefMap(t)
	if _, err := m.CopyPatch(44, 0, 8, 2); err == nil {
		t.Fatal("a patch off the grid was copied")
	}
	p, err := m.CopyPatch(26, 26, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	bad := p
	bad.Heights = bad.Heights[1:]
	before := m.Bytes()
	if _, err := m.PastePatch(bad, mapedit.Cell{X: 2, Y: 10}); err == nil || !bytes.Equal(m.Bytes(), before) {
		t.Fatalf("a short patch: %v", err)
	}
	bare, err := mapedit.New(brushMap(0x14, 40))
	if err != nil {
		t.Fatal(err)
	}
	plain := bare.Bytes()
	if _, err := bare.PastePatch(p, mapedit.Cell{X: 2, Y: 10}); err == nil || !bytes.Equal(bare.Bytes(), plain) {
		t.Fatalf("objects onto a map without a type-3 record: %v", err)
	}
	p.Objects = make([]byte, len(p.Objects))
	if changed, err := bare.PastePatch(p, mapedit.Cell{X: 2, Y: 10}); err != nil || !changed {
		t.Fatalf("an empty object layer onto a map without one: %v %v", changed, err)
	}
}
