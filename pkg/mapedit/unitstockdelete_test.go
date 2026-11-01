package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// stockMaps returns a fresh map for each loot head width the format has.
func stockMaps(t *testing.T) map[string]*mapedit.Editor {
	t.Helper()
	out := map[string]*mapedit.Editor{}
	for name, version := range map[string]uint32{"short head": alm.LootWideHeadVersion - 1, "wide head": alm.LootWideHeadVersion} {
		raw := freshRecords(t).Bytes()
		binary.LittleEndian.PutUint32(raw[alm.FileFormatVersion:], version)
		e, err := mapedit.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = e
	}
	return out
}

// stockedUnits places n units, gives each a stock holding item 0x0101+i and
// returns their indices.
func stockedUnits(t *testing.T, e *mapedit.Editor, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		u, err := e.NewUnitRecord(uint32(10+i)*256+128, 11*256+128, 40, 2, 2)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.PlaceUnit(u); err != nil {
			t.Fatal(err)
		}
		if _, err := e.AddUnitStock(i); err != nil {
			t.Fatal(err)
		}
		if err := e.SetField(mapedit.KindLoot, i, "additem", 0x0101+int64(i)); err != nil {
			t.Fatal(err)
		}
	}
}

// stockItems maps each unit's loot link to the first item of the stock it
// names, 0 for no link.
func stockItems(t *testing.T, e *mapedit.Editor) []int {
	t.Helper()
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Loot()
	if err != nil {
		t.Fatal(err)
	}
	out := make([]int, len(m.Units))
	for i := range m.Units {
		rec, _ := e.UnitRecord(i)
		link := binary.LittleEndian.Uint32(rec[alm.UnitLoot:])
		if link == 0 {
			continue
		}
		if int(link) > len(l.Records) {
			t.Fatalf("unit %d links stock %d of %d", i, link, len(l.Records))
		}
		r := l.Records[link-1]
		if r.Owner != uint32(m.Units[i].UnitID) {
			t.Fatalf("unit %d links a stock of owner %d", i, r.Owner)
		}
		out[i] = int(r.Elements[0].ItemCode())
	}
	return out
}

func TestDeleteUnitRemovesItsStockInOneUndoStep(t *testing.T) {
	for name, e := range stockMaps(t) {
		t.Run(name, func(t *testing.T) {
			stockedUnits(t, e, 3)
			before := e.Bytes()
			if err := e.DeleteUnit(1); err != nil {
				t.Fatal(err)
			}
			after := e.Bytes()
			if e.Count(mapedit.KindUnit) != 2 || e.Count(mapedit.KindLoot) != 2 {
				t.Fatal("unit and stock not both removed")
			}
			if got := stockItems(t, e); got[0] != 0x0101 || got[1] != 0x0103 {
				t.Fatalf("surviving links name %x", got)
			}
			if !e.Undo() || !bytes.Equal(e.Bytes(), before) {
				t.Fatal("one undo did not restore the bytes")
			}
			if got := stockItems(t, e); got[0] != 0x0101 || got[1] != 0x0102 || got[2] != 0x0103 {
				t.Fatalf("restored links name %x", got)
			}
			if !e.Redo() || !bytes.Equal(e.Bytes(), after) {
				t.Fatal("redo did not remove both again")
			}
		})
	}
}

func TestDeleteUnitRemovesOnlyOneHistoryEntry(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 1)
	if err := e.DeleteUnit(0); err != nil {
		t.Fatal(err)
	}
	if !e.Undo() {
		t.Fatal("no undo")
	}
	// The next undo reverts the last authoring step, not half of the delete.
	if e.Count(mapedit.KindUnit) != 1 || e.Count(mapedit.KindLoot) != 1 {
		t.Fatal("one undo left half of the delete")
	}
}

func TestDeleteStockedUnitsInEitherOrderKeepsEveryLink(t *testing.T) {
	for _, order := range [][2]int{{0, 0}, {1, 0}, {2, 1}, {2, 0}, {0, 1}, {1, 1}} {
		for name, e := range stockMaps(t) {
			stockedUnits(t, e, 3)
			base := e.Bytes()
			survivor := []int{0x0101, 0x0102, 0x0103}
			for _, i := range order {
				if err := e.DeleteUnit(i); err != nil {
					t.Fatalf("%s %v: %v", name, order, err)
				}
				survivor = append(survivor[:i], survivor[i+1:]...)
				got := stockItems(t, e)
				for k := range survivor {
					if got[k] != survivor[k] {
						t.Fatalf("%s %v: links %x, want %x", name, order, got, survivor)
					}
				}
			}
			if e.Count(mapedit.KindUnit) != 1 || e.Count(mapedit.KindLoot) != 1 {
				t.Fatalf("%s %v: records left", name, order)
			}
			if !e.Undo() || !e.Undo() || !bytes.Equal(e.Bytes(), base) {
				t.Fatalf("%s %v: two undos did not return the bytes", name, order)
			}
		}
	}
}

func TestDeleteLaterStockOwnerFirstThenEarlier(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 3)
	if err := e.DeleteUnit(2); err != nil {
		t.Fatal(err)
	}
	if got := stockItems(t, e); got[0] != 0x0101 || got[1] != 0x0102 {
		t.Fatalf("links %x", got)
	}
	if err := e.DeleteUnit(0); err != nil {
		t.Fatal(err)
	}
	if got := stockItems(t, e); len(got) != 1 || got[0] != 0x0102 {
		t.Fatalf("links %x", got)
	}
}

func TestDeleteEarlierStockOwnerFirstThenLater(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 3)
	if err := e.DeleteUnit(0); err != nil {
		t.Fatal(err)
	}
	if got := stockItems(t, e); got[0] != 0x0102 || got[1] != 0x0103 {
		t.Fatalf("links %x", got)
	}
	if err := e.DeleteUnit(1); err != nil {
		t.Fatal(err)
	}
	if got := stockItems(t, e); len(got) != 1 || got[0] != 0x0102 {
		t.Fatalf("links %x", got)
	}
}

func TestDeleteUnitKeepsStockOfASharedIdentity(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 1)
	twin, _ := e.UnitRecord(0)
	twin = append([]byte(nil), twin...)
	binary.LittleEndian.PutUint32(twin[alm.UnitLoot:], 0)
	binary.LittleEndian.PutUint32(twin[alm.UnitX:], 12*256+128)
	if _, err := e.PlaceUnit(twin); err != nil {
		t.Fatal(err)
	}
	if err := e.DeleteUnit(1); err != nil {
		t.Fatal(err)
	}
	if e.Count(mapedit.KindLoot) != 1 {
		t.Fatal("a stock the other unit still owns was removed")
	}
	if got := stockItems(t, e); got[0] != 0x0101 {
		t.Fatalf("links %x", got)
	}
	if err := e.DeleteUnit(0); err != nil {
		t.Fatal(err)
	}
	if e.Count(mapedit.KindLoot) != 0 {
		t.Fatal("the last owner left its stock behind")
	}
}

func TestDeleteUnitWithoutStockOrWithIDZeroChangesOnlyTheUnit(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 1)
	u, _ := e.NewUnitRecord(20*256+128, 20*256+128, 40, 2, 2)
	if _, err := e.PlaceUnit(u); err != nil {
		t.Fatal(err)
	}
	z, _ := e.NewUnitRecord(21*256+128, 20*256+128, 40, 2, 2)
	binary.LittleEndian.PutUint16(z[alm.UnitID:], 0)
	if _, err := e.PlaceUnit(z); err != nil {
		t.Fatal(err)
	}
	// A stock owned by identity 0 would be a ground sack; it stays.
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(128, 128, 0x0101)); err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{2, 1} {
		raw := e.Bytes()
		if err := e.DeleteUnit(i); err != nil {
			t.Fatal(err)
		}
		if e.Count(mapedit.KindLoot) != 2 || e.Count(mapedit.KindUnit) != i {
			t.Fatalf("delete of unit %d touched stock", i)
		}
		e.Undo()
		if !bytes.Equal(raw, e.Bytes()) {
			t.Fatal("undo differs")
		}
		if err := e.DeleteUnit(i); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteSackStillRenumbersUnitLinks(t *testing.T) {
	e := freshRecords(t)
	stockedUnits(t, e, 2)
	if err := e.Delete(mapedit.KindLoot, 0); err != nil {
		t.Fatal(err)
	}
	got := stockItems(t, e)
	if got[0] != 0 || got[1] != 0x0102 {
		t.Fatalf("links %x", got)
	}
}
