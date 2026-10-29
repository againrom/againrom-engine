package town

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const innCatalogHead = `{"catalog": "fixture", "towns": [
  {"id": 1, "description": "one", "hooks": "square-hooks", "offers": "bank", "map-point": {"record": true}},
  {"id": 2, "description": "two", "hooks": "square-hooks", "offers": "bank", "map-point": {"record": true}}]`

// Inn rows decode with every key, and each broken row is refused with its
// key named, through the catalog's own decode.
func TestInnRowsDecodeAndRefuse(t *testing.T) {
	c, err := DecodeCatalog([]byte(innCatalogHead+`, "offer-table": "inn", "offers": [
  {"kind": 3, "topic": 7, "npc": 40, "town": 2, "stages": [5, 6], "after": 4,
   "zero": [9], "nonzero": [10], "one": [11], "any": [12, 13]},
  {"topic": 0, "npc": 41}]}`), catalogVocab)
	if err != nil {
		t.Fatal(err)
	}
	want := []OfferRow{
		{Kind: OfferMission, Topic: 7, NPC: 40, Town: 2, Stages: []int{5, 6}, After: 4,
			Zero: []int{9}, Nonzero: []int{10}, One: []int{11}, Any: []int{12, 13}},
		{Kind: OfferTalk, NPC: 41},
	}
	if !reflect.DeepEqual(c.Offers, want) {
		t.Fatalf("decoded %+v, want %+v", c.Offers, want)
	}
	if !c.Offers[0].Equal(want[0]) || c.Offers[0].Equal(want[1]) || InnTable.Key(c.Offers[0]) != (OfferKey{Table: InnTable, Topic: 7, NPC: 40}) {
		t.Fatal("Equal or Key does not follow the row")
	}
	inn := func(topic, npc int) OfferKey { return OfferKey{Table: InnTable, Topic: topic, NPC: npc} }
	for _, tc := range []struct {
		rows, want string
		key        OfferKey
	}{
		{`{"kind": 1, "topic": 1, "npc": 1}`, "kind 1", inn(1, 1)},
		{`{"topic": -1, "npc": 1}`, "topic -1", inn(-1, 1)},
		{`{"topic": 65536, "npc": 1}`, "topic 65536", inn(65536, 1)},
		{`{"topic": 1, "npc": 0}`, "npc 0", inn(1, 0)},
		{`{"topic": 1, "npc": 1, "town": 3}`, "town 3", inn(1, 1)},
		{`{"topic": 1, "npc": 1, "after": -1}`, "after -1", inn(1, 1)},
		{`{"topic": 1, "npc": 1}, {"topic": 1, "npc": 1, "kind": 3}`, "named twice", inn(1, 1)},
		{`{"topic": 1, "npc": 1, "stages": [0]}`, "stage 0", inn(1, 1)},
		{`{"topic": 1, "npc": 1, "any": [-2]}`, "slot -2", inn(1, 1)},
	} {
		_, err := DecodeCatalog([]byte(innCatalogHead+`, "offer-table": "inn", "offers": [`+tc.rows+`]}`), catalogVocab)
		var oe *OfferError
		if err == nil || !strings.Contains(err.Error(), "inn row") || !strings.Contains(err.Error(), tc.want) || !errors.As(err, &oe) || oe.Row != tc.key {
			t.Errorf("%s: %v", tc.rows, err)
		}
	}
	if _, err := DecodeCatalog([]byte(innCatalogHead+`, "offers": [{"topic": 1, "npc": 1}]}`), catalogVocab); err == nil || !strings.Contains(err.Error(), "offer table") {
		t.Errorf("rows without a table: %v", err)
	}
}

// The same validator refuses each broken tavern row by its tavern key.
func TestTavernRowsAreTheReducedShape(t *testing.T) {
	if got := ChapterOffer(1, 20, 5, 0); !got.Equal(OfferRow{NPC: 5, Town: 1, Stages: []int{20}}) || got.Chapter() != 20 {
		t.Errorf("talk %+v", got)
	}
	if got := ChapterOffer(1, 20, 5, 21); !got.Equal(OfferRow{Kind: OfferMission, Topic: 21, NPC: 5, Town: 1, Stages: []int{20}}) {
		t.Errorf("mission %+v", got)
	}
	head := innCatalogHead + `, "offer-table": "tavern", "offers": [`
	c, err := DecodeCatalog([]byte(head+`{"kind": 3, "topic": 21, "npc": 0, "town": 2, "stages": [20]}]}`), catalogVocab)
	if err != nil || len(c.Offers) != 1 || TavernTable.Key(c.Offers[0]).String() != "town 2 chapter 20 npc 0" {
		t.Fatalf("a tavern row: %v", err)
	}
	tavern := func(id, chapter, npc int) OfferKey {
		return OfferKey{Table: TavernTable, Town: id, Chapter: chapter, NPC: npc}
	}
	for _, tc := range []struct {
		rows, want string
		key        OfferKey
	}{
		{`{"npc": 1, "stages": [20]}`, "town 0 is not a town of the catalog", tavern(0, 20, 1)},
		{`{"npc": 1, "town": 1, "stages": [20, 30]}`, "2 chapters are stated", tavern(1, 0, 1)},
		{`{"npc": 1, "town": 1, "stages": [-1]}`, "chapter -1 is not positive", tavern(1, -1, 1)},
		{`{"npc": -1, "town": 1, "stages": [20]}`, "npc -1 is outside the range 0 to", tavern(1, 20, -1)},
		{`{"kind": 3, "topic": 65536, "npc": 1, "town": 1, "stages": [20]}`, "mission 65536 is outside", tavern(1, 20, 1)},
		{`{"topic": 21, "npc": 1, "town": 1, "stages": [20]}`, "kind 0 does not follow mission 21", tavern(1, 20, 1)},
		{`{"npc": 1, "town": 1, "stages": [20], "zero": [3]}`, "no stage floor or bank gate", tavern(1, 20, 1)},
		{`{"npc": 1, "town": 1, "stages": [20], "after": 3}`, "no stage floor or bank gate", tavern(1, 20, 1)},
		{`{"npc": 1, "town": 1, "stages": [20]}, {"npc": 1, "town": 2, "stages": [20]}`, "npc 1 already speaks in chapter 20", tavern(2, 20, 1)},
	} {
		_, err := DecodeCatalog([]byte(head+tc.rows+`]}`), catalogVocab)
		var oe *OfferError
		if err == nil || !strings.Contains(err.Error(), "tavern row "+tc.key.String()) || !strings.Contains(err.Error(), tc.want) || !errors.As(err, &oe) || oe.Row != tc.key {
			t.Errorf("%s: %v", tc.rows, err)
		}
	}
	if _, err := DecodeCatalog([]byte(innCatalogHead+`, "offer-table": "shop"}`), catalogVocab); err == nil || !strings.Contains(err.Error(), "neither") {
		t.Errorf("an unknown table: %v", err)
	}
}

// The mercenary rule answers its price ladder and its levels from data: a
// changed step, start, factor or level moves exactly the answers it owns.
func TestMercenaryRuleIsData(t *testing.T) {
	r := &MercenaryRule{StockTypes: 2, StockIDPhase: 3, PriceFrom: 5, PriceStep: 5, PriceLadder: []int{1, 2, 3},
		Levels: []MercenaryLevel{{From: 10, Level: 2}, {From: 20, Level: 5}}}
	if err := r.validate("fixture"); err != nil {
		t.Fatal(err)
	}
	for m, want := range map[int]int{-6: 0, 4: 0, 5: 1, 9: 1, 10: 2, 19: 3, 20: 0, 64: 0} {
		if got := r.UnitPrice(m); got != want {
			t.Errorf("UnitPrice(%d) = %d, want %d", m, got, want)
		}
	}
	for m, want := range map[int]int{-1: 1, 9: 1, 10: 2, 19: 2, 20: 5, 64: 5} {
		if got := r.Level(m); got != want {
			t.Errorf("Level(%d) = %d, want %d", m, got, want)
		}
	}
	moved := *r
	moved.PriceLadder = []int{1, 8, 3}
	if moved.UnitPrice(10) != 8 || moved.UnitPrice(5) != 1 {
		t.Error("a changed factor did not move its own price alone")
	}
	var none *MercenaryRule
	if none.UnitPrice(10) != 0 || none.Level(64) != 1 || none.validate("fixture") != nil {
		t.Error("a nil rule answers a price or a level")
	}
	for _, bad := range []MercenaryRule{
		{StockTypes: 0, PriceStep: 1},
		{StockTypes: 1, StockIDPhase: -1, PriceStep: 1},
		{StockTypes: 1, PriceStep: 0},
		{StockTypes: 1, PriceStep: 1, PriceLadder: []int{-1}},
		{StockTypes: 1, PriceStep: 1, Levels: []MercenaryLevel{{From: 2, Level: 2}, {From: 2, Level: 3}}},
		{StockTypes: 1, PriceStep: 1, Levels: []MercenaryLevel{{From: 2, Level: 0}}},
	} {
		if err := bad.validate("fixture"); err == nil || !strings.Contains(err.Error(), "mercenaries") {
			t.Errorf("%+v: %v", bad, err)
		}
	}
	c, err := DecodeCatalog([]byte(innCatalogHead+`, "mercenaries": {"stock-types": 2, "price-step": 0}}`), catalogVocab)
	if err == nil || c != nil {
		t.Error("a catalog with a broken mercenary rule decoded")
	}
}
