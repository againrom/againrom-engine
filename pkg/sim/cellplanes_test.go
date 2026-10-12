package sim

import "testing"

// The builder folds the native fields over the installed base at mission start
// and overlays only Block records for a loaded document.
func TestCellPlanesFoldAndDocument(t *testing.T) {
	b := Bounds{Width: 16, Height: 16}
	n := int(b.Width * b.Height)
	grid, cost, height, overlay := make([]byte, n), make([]byte, n), make([]byte, n), make([]byte, n)
	for i := range n {
		cost[i], height[i] = byte(i%7+1), byte(i%11)
	}
	grid[1*16+9] = 1 // ground blocked
	grid[2*16+9] = 2 // raw domain 2: bit 2 too
	overlay[4*16+9] = 1
	w, err := NewTerrainWorld(1, b, ModeCanonical, Terrain{Block: grid, Cost: cost, Height: height}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := &SavedCellPlanes{Costs: [11]byte{255, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}
	base.Static[0x0000], base.Dynamic[0x0000] = 0x1f, 0x1f
	base.Static[0x0409], base.Dynamic[0x0409] = 5, 5
	base.Cost[0x2020], base.CostKnown[0x2020] = 9, 1

	p := w.CellPlanes(CellPlaneStart{Base: base, Overlay: overlay})
	for key, want := range map[uint16]byte{0x0000: 0x18, 0x0109: 1, 0x0209: 6, 0x0409: 4, 0x0509: 0} {
		if p.Static[key] != want || p.Dynamic[key] != want {
			t.Errorf("cell %04x: Static %#x Dynamic %#x, want %#x", key, p.Static[key], p.Dynamic[key], want)
		}
	}
	if p.Cost[0x0105] != cost[1*16+5] || p.Height[0x0105] != height[1*16+5] || p.CostKnown[0x0105] != 1 {
		t.Errorf("cell 0105 does not carry the native cost and height")
	}
	if p.Cost[0x2020] != 9 || p.CostKnown[0x2020] != 1 || p.Costs != base.Costs {
		t.Errorf("out-of-bounds base bytes or the cost table moved")
	}
	if p.CostKnown[0x0010] != 0 {
		t.Errorf("a cell past the bounds became known")
	}

	d := w.CellPlanes(CellPlaneStart{Base: base, Document: true, Blocks: []SavedActorBlock{{Cell: 0x0105, Dyn: 0x61, Static: 0x21}}})
	if d.Static[0x0105] != 0x21 || d.Dynamic[0x0105] != 0x61 || d.Static[0x0109] != 0 || d.Cost[0x0105] != 0 || d.Static[0x0000] != 0x1f {
		t.Errorf("a document start folded native fields or lost its Block record")
	}
	if z := w.CellPlanes(CellPlaneStart{}); z.Costs != ([11]byte{}) || z.Static[0x0109] != 1 {
		t.Errorf("a start without an install is not the zero base with the native fold")
	}
}
