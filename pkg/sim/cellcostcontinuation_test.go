package sim

import "testing"

func TestCurrentOmittedPlaneRowsValidateBeforeMutation(t *testing.T) {
	w, p := cellPlanesFixture(t, false)
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	zero, cost, dynamic := uint8(0), uint8(17), uint8(16)
	valid := CurrentCellPlaneResidue{Cell: 0x0806, Static: &zero, Dynamic: &dynamic}
	before := w.Hash()
	for _, bad := range [][]CurrentCellPlaneResidue{
		{valid, valid},
		{valid, {Cell: 0x0909}},
		{valid, {Cell: 0x100f, Cost: &cost}},
		{valid, {Cell: 0x0909, Dynamic: &dynamic}},
		{valid, {Cell: 0x1010, Static: &zero}},
	} {
		if err := w.RestoreCurrentCellCosts(nil, bad...); err == nil || w.Hash() != before {
			t.Fatal("invalid omitted-plane binding partially changed the World", err)
		}
	}
	if err := w.RestoreCurrentCellCosts(nil, valid, CurrentCellPlaneResidue{Cell: 0xffff, Cost: &cost}); err != nil {
		t.Fatal(err)
	}
	if w.savedCellPlanes.Dynamic[0x0806] != 16 || w.savedCellPlanes.Cost[0xffff] != 17 {
		t.Fatal("raw u16 plane keyspace was clamped to map bounds")
	}
	var fresh World
	if err := fresh.UnmarshalBinary(mustMarshal(t, w)); err != nil || fresh.Hash() != w.Hash() {
		t.Fatal("valid sparse rows broke native state", err)
	}
}
