package sim

import "testing"

func TestCurrentStructureCellAbsenceIsAtomicAndOrdinaryAnchored(t *testing.T) {
	w := savedGroupWorld(t)
	if err := w.ImportOriginalStructures(nil, nil, []SavedStructureCell{{Cell: 0x0101, BaselineCost: 7, BaselineStatic: 3}, {Cell: 0x0102, BaselineCost: 9, BaselineStatic: 2}}, w.grid); err != nil {
		t.Fatal(err)
	}
	first := CurrentAbsentStructureCell{Cell: 0x0101, Cost: 7, Static: 3}
	for _, tail := range []CurrentAbsentStructureCell{first, {Cell: 0xffff}} {
		hash := w.Hash()
		if err := w.RestoreAbsentStructureCells([]CurrentAbsentStructureCell{first, tail}); err == nil || w.Hash() != hash {
			t.Fatal("invalid late absence changed earlier cells", err)
		}
	}
	if err := w.RestoreAbsentStructureCells([]CurrentAbsentStructureCell{first, {Cell: 0x0102, Cost: 8, Static: 2}}); err != nil {
		t.Fatal(err)
	}
	_, cells, present := w.SavedStructures()
	if !present || len(cells) != 1 || cells[0].Cell != 0x0102 || cells[0].BaselineCost != 9 {
		t.Fatal("absence erased an edited ordinary cell or its carrier", cells, present)
	}
}
