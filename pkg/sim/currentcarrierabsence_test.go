package sim

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestCurrentCarrierAbsenceIsIndependentAndAtomic(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}})
	if err != nil {
		t.Fatal(err)
	}
	tails := []CellTail{{X: 1, Y: 1}, {X: 2, Y: 2, Bytes: [6]byte{25, 31, 2, 3, 4, 5}}}
	records := []SavedCellRecord{{Cell: 0x0101, Residue0: 19}, {Cell: 0x0202, Residue1: [2]byte{7, 11}}}
	diaries := []SavedDiary{{Owner: SavedDiaryOwner{Player: true}, Length: 3}, {Owner: SavedDiaryOwner{Actor: 1}, Length: 7}}
	if err := w.ImportOriginalCellTails(tails); err != nil {
		t.Fatal(err)
	}
	w.SetSavedCellRecords(records)
	w.SetSavedDiaries(diaries)
	before := w.Hash()
	for _, fault := range []struct {
		tails, records []uint16
		diaries        []SavedDiaryOwner
	}{
		{tails: []uint16{0x0101, 0x0101}},
		{tails: []uint16{0x0101}, records: []uint16{0x0303}},
		{records: []uint16{0x0202, 0x0202}},
		{tails: []uint16{0x0101}, records: []uint16{0x0202}, diaries: []SavedDiaryOwner{{Actor: 1}, {Actor: 1}}},
		{diaries: []SavedDiaryOwner{{Player: true, Actor: 1}}},
	} {
		if err := w.RestoreCurrentCarrierAbsence(fault.tails, fault.records, fault.diaries); err == nil || w.Hash() != before {
			t.Fatal("invalid absent carrier changed World", fault, err)
		}
	}
	if err := w.RestoreCurrentCarrierAbsence([]uint16{0x0101}, []uint16{0x0202}, []SavedDiaryOwner{{Actor: 1}}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.CellTails(), tails[1:]) || !reflect.DeepEqual(w.SavedCellRecords(), records[:1]) || !reflect.DeepEqual(w.SavedDiaries(), diaries[:1]) {
		t.Fatal("absence removed a different cell projection or Diary owner")
	}
}

func TestCurrentSackKeyAbsenceDoesNotChangeOtherCellValues(t *testing.T) {
	for _, carriers := range []uint8{CurrentSackKeyRecord, CurrentSackKeyMotion, CurrentSackKeyRecord | CurrentSackKeyMotion} {
		w := sourceBindingWorld1111(t)
		w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x0101, Sack: 91, Residue0: 7, Ground: SavedCellActorSlot{Key: 41}}, {Cell: 0x0202, Sack: 92, Residue1: [2]byte{3, 5}}})
		w.savedMotion = &savedActorMotionState{Cells: []SavedActorCell{{Cell: 0x0101}, {Cell: 0x0202}}}
		binary.LittleEndian.PutUint32(w.savedMotion.Cells[0].Payload[16:], 91)
		binary.LittleEndian.PutUint32(w.savedMotion.Cells[1].Payload[16:], 92)
		w.savedMotion.Cells[0].Payload[3], w.savedMotion.Cells[1].Payload[32] = 17, 29
		before := w.Hash()
		for _, rows := range [][]CurrentSackKeyAbsence{{{0x0101, 92, carriers}}, {{0x0303, 91, carriers}}, {{0x0101, 91, carriers}, {0x0101, 91, carriers}}, {{0x0101, 91, 0}}, {{0x0101, 91, 4}}} {
			if err := w.RestoreCurrentSackKeyAbsence(rows); err == nil || w.Hash() != before {
				t.Fatal("invalid cell Sack absence changed state", rows, err)
			}
		}
		want := w.SavedCellRecords()
		wantMotion := cloneActorMotions(w.savedMotion)
		if carriers&CurrentSackKeyRecord != 0 {
			want[0].Sack = 0
		}
		if carriers&CurrentSackKeyMotion != 0 {
			binary.LittleEndian.PutUint32(wantMotion.Cells[0].Payload[16:], 0)
		}
		if err := w.RestoreCurrentSackKeyAbsence([]CurrentSackKeyAbsence{{0x0101, 91, carriers}}); err != nil || !reflect.DeepEqual(w.SavedCellRecords(), want) || !reflect.DeepEqual(w.savedMotion, wantMotion) {
			t.Fatal("Sack absence changed another field", err)
		}
	}
}

func TestCurrentMotionCellAbsenceIsAtomicAndKeepsOtherCarriers(t *testing.T) {
	w := sourceBindingWorld1111(t)
	w.savedMotion = &savedActorMotionState{Cells: []SavedActorCell{{Cell: 0x0101}, {Cell: 0x0202}}, Blocks: []SavedActorBlock{{Cell: 0x0101, Dyn: 64}}}
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x0101, Sack: 91}})
	w.savedMotion.Cells[0].Payload[3] = 71
	old := cloneActorMotions(w.savedMotion)
	before := w.Hash()
	for _, rows := range [][][]uint16{{nil, {0x0303}}, {nil, {0x0101, 0x0101}}, {{0x0101}, {0x0101}, nil}} {
		if err := w.RestoreCurrentCarrierAbsence(nil, nil, nil, rows...); err == nil || w.Hash() != before {
			t.Fatal("malformed motion absence changed current World", rows, err)
		}
	}
	if err := w.RestoreCurrentCarrierAbsence(nil, nil, nil, nil, []uint16{0x0101}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.savedMotion.Cells, old.Cells[1:]) || !reflect.DeepEqual(w.savedMotion.Blocks, old.Blocks) || w.SavedCellRecords()[0].Sack != 91 || old.Cells[0].Payload[3] != 71 {
		t.Fatal("motion absence changed another carrier or aliased its input")
	}
}

func TestCurrentPhysicalCrossingUpdatesOnlyPresentActorResidue(t *testing.T) {
	for _, present := range []bool{false, true} {
		w, m, cells, blocks := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
		importMotion1115(t, w, m, cells, blocks)
		if present {
			w.SetSavedCellRecords([]SavedCellRecord{{Cell: cells[0].Cell, Ground: SavedCellActorSlot{Key: 0x1004, Entity: 7, Bound: true}, Air: SavedCellActorSlot{Key: 44}, Residue0: 9}, {Cell: cells[1].Cell, Air: SavedCellActorSlot{Key: 45}, Residue1: [2]byte{3, 7}}})
		}
		Step(w, nil)
		if w.entities[0].X != 16 {
			t.Fatal("fixture did not physically cross")
		}
		got := w.SavedCellRecords()
		if !present {
			if len(got) != 0 {
				t.Fatal("crossing invented residue carrier")
			}
			continue
		}
		if got[0].Ground != (SavedCellActorSlot{}) || got[1].Ground != (SavedCellActorSlot{Key: 0x1004, Entity: 7, Bound: true}) || got[0].Air.Key != 44 || got[1].Air.Key != 45 || got[0].Residue0 != 9 || got[1].Residue1 != ([2]byte{3, 7}) {
			t.Fatal("physical crossing changed wrong residue fields", got)
		}
	}
}
