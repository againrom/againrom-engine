package sim

import (
	"reflect"
	"testing"
)

func TestCurrentBlockAbsenceLeavesTerrainAndOtherMotionState(t *testing.T) {
	w := sourceBindingWorld1111(t)
	w.savedMotion = &savedActorMotionState{Blocks: []SavedActorBlock{{Cell: 0x0101, Dyn: 67, Static: 3}, {Cell: 0x0202, Dyn: 64}},
		Cells: []SavedActorCell{{Cell: 0x0101, Payload: [52]byte{3, 7}}}}
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x0101, Residue0: 13}})
	before, policy := w.Hash(), w.CurrentPolicy()
	for _, rows := range [][]uint16{{0x0101, 0x0101}, {0x0303}} {
		if err := w.RestoreCurrentCarrierAbsence(nil, nil, nil, rows); err == nil || w.Hash() != before {
			t.Fatal("invalid block removal changed state", rows, err)
		}
	}
	cells, records := append([]SavedActorCell(nil), w.savedMotion.Cells...), w.SavedCellRecords()
	if err := w.RestoreCurrentCarrierAbsence(nil, nil, nil, []uint16{0x0101}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.savedMotion.Blocks, []SavedActorBlock{{Cell: 0x0202, Dyn: 64}}) || !reflect.DeepEqual(w.savedMotion.Cells, cells) || !reflect.DeepEqual(w.SavedCellRecords(), records) || !reflect.DeepEqual(w.CurrentPolicy(), policy) {
		t.Fatal("block presence altered terrain or another cell owner")
	}
}
