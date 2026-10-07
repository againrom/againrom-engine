package sav

import "testing"

// TestCellsDecodesTheCompleteRecord locks the field layout SAV-CELLREC-032/
// SAV-CELLLOAD-109/110/111 name against the standard fixture, whose only
// nonzero payload byte (rec[7] = 0x99, "undecoded, and must survive a round
// trip") lands inside Ground's second byte.
func TestCellsDecodesTheCompleteRecord(t *testing.T) {
	f := open(t, standard())
	cells, present, err := f.Cells()
	if err != nil || !present {
		t.Fatalf("Cells: present=%v err=%v", present, err)
	}
	if len(cells) != 2 {
		t.Fatalf("want 2 records, got %d", len(cells))
	}
	for i, want := range []uint16{0x0810, 0x1234} {
		c := cells[i]
		if c.Key != want {
			t.Fatalf("record %d key = %#x, want %#x", i, c.Key, want)
		}
		if c.Ground != 0x9900 {
			t.Fatalf("record %d Ground = %#x, want 0x9900", i, c.Ground)
		}
		if c.BaselineCost != 0 || c.BaselineStatic != 0 || c.LayerCount != 0 || c.Residue0 != 0 ||
			c.Air != 0 || c.Building != 0 || c.Sack != 0 || c.SpellEffects != [6]uint32{} ||
			c.Trigger != [6]byte{} || c.Residue1 != [2]byte{} {
			t.Fatalf("record %d has an unexpected nonzero field: %+v", i, c)
		}
	}
}

// TestSetCellRoundTripsEveryField writes a fully populated record over one
// with a different one and confirms every field, including both residue
// spans, reads back exactly what was written and leaves the sibling record
// and the key untouched.
func TestSetCellRoundTripsEveryField(t *testing.T) {
	f := open(t, standard())
	want := Cell{
		Key:            0x1234, // SetCell writes the key too; use the record's own.
		BaselineCost:   11,
		BaselineStatic: 22,
		LayerCount:     3,
		Residue0:       4,
		Ground:         0x11223344,
		Air:            0x22334455,
		Building:       0x33445566,
		Sack:           0x44556677,
		SpellEffects:   [6]uint32{1, 2, 3, 4, 5, 6},
		Trigger:        [6]byte{26, 7, 8, 9, 10, 11},
		Residue1:       [2]byte{0xaa, 0xbb},
	}
	if err := f.SetCell(1, want); err != nil {
		t.Fatalf("SetCell: %v", err)
	}
	cells, present, err := f.Cells()
	if err != nil || !present {
		t.Fatalf("Cells: present=%v err=%v", present, err)
	}
	if cells[1] != want {
		t.Fatalf("record 1 = %+v, want %+v", cells[1], want)
	}
	if cells[0].Key != 0x0810 || cells[0].Ground != 0x9900 {
		t.Fatalf("record 0 was disturbed: %+v", cells[0])
	}
	// A round trip through Marshal/Open must reproduce the written bytes:
	// SetCell's own record-index addressing must agree with the decoder's.
	written, err := Open(f.Marshal())
	if err != nil {
		t.Fatalf("re-open written file: %v", err)
	}
	again, present, err := written.Cells()
	if err != nil || !present {
		t.Fatalf("Cells (re-decoded): present=%v err=%v", present, err)
	}
	if again[1] != want {
		t.Fatalf("re-decoded record 1 = %+v, want %+v", again[1], want)
	}
}

func TestSetCellRefusesAnOutOfRangeIndex(t *testing.T) {
	f := open(t, standard())
	if err := f.SetCell(-1, Cell{}); err == nil {
		t.Fatal("want an error for a negative index, got none")
	}
	if err := f.SetCell(2, Cell{}); err == nil {
		t.Fatal("want an error for an index past the count, got none")
	}
	s := standard()
	s.noWorld = true
	noWorld := open(t, s)
	if err := noWorld.SetCell(0, Cell{}); err == nil {
		t.Fatal("want an error for a save with no world half, got none")
	}
}
