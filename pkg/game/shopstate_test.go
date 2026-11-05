package game

import (
	"reflect"
	"testing"
)

func TestShopStateCatalogRecordsAndEmptyUpdates(t *testing.T) {
	s := NewShop(777)
	s.deriveRecords([]int{1, 17})
	if len(s.records) != 2 || s.records[17][ShelfArmour].Max != 777 || s.records[17][ShelfWeapons].Draws != 100 {
		t.Fatal("catalog population was limited by the four native records", s.records)
	}
	before := s.records.clone()
	s.depart(50)
	s.stage(90)
	s.addClass4()
	if !reflect.DeepEqual(s.records, before) {
		t.Fatal("chapter profile applied Scenario writes")
	}
	s.filled = 17
	s.patch([]shopRecordPatch{{record: 17, category: 0, field: shopRecordMax, value: 888}}, 0)
	if s.filled != 0 || s.records[17][0].Max != 888 {
		t.Fatal("a catalog record beyond the SAV mapping did not invalidate stock")
	}
	clone := cloneShopMutation(s)
	clone.records[17][0].Max++
	if s.records[17][0].Max != 888 {
		t.Fatal("a detached transaction changed the original records")
	}
	s.deriveRecords([]int{1 << 30})
	if len(s.records) != 3 || s.records[1<<30][0].Max != 777 {
		t.Fatal("a sparse catalog ID allocated a numeric range")
	}
}
