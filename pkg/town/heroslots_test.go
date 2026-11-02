package town

import (
	"strings"
	"testing"
)

// An inn row names its starting heroes by class and sex, read in the
// catalog's hero slots; a tavern row with a hero, a row with a hero but no
// slots and broken slots are refused.
func TestInnRowsNameTheirStartingHeroes(t *testing.T) {
	slots := `, "hero-slots": {"mage": 5, "female": 6}`
	c, err := DecodeCatalog([]byte(innCatalogHead+`, "offer-table": "inn"`+slots+`, "offers": [
  {"kind": 3, "topic": 7, "npc": 40, "class": "mage", "sex": "female"},
  {"kind": 3, "topic": 8, "npc": 40, "class": "fighter"},
  {"kind": 3, "topic": 9, "npc": 40}]}`), catalogVocab)
	if err != nil {
		t.Fatal(err)
	}
	if *c.HeroSlots != (HeroSlots{Mage: 5, Female: 6}) {
		t.Fatalf("slots %+v", c.HeroSlots)
	}
	if r := c.Offers; r[0].Class != "mage" || r[0].Sex != "female" || r[1].Class != "fighter" || r[1].Sex != "" || r[2].Class != "" {
		t.Fatalf("rows %+v", r)
	}
	if c.Offers[0].Equal(OfferRow{Kind: 3, Topic: 7, NPC: 40, Class: "mage"}) {
		t.Fatal("Equal ignores the sex")
	}
	for _, tc := range []struct{ head, rows, want string }{
		{`, "offer-table": "inn"`, `{"topic": 1, "npc": 1, "class": "mage"}`, "the catalog names no hero-slots"},
		{`, "offer-table": "tavern"` + slots, `{"npc": 1, "town": 1, "stages": [20], "sex": "male"}`, "a tavern row is offered to every starting hero"},
		{`, "offer-table": "inn", "hero-slots": {"mage": -1, "female": 6}`, `{"topic": 1, "npc": 1}`, "hero-slots"},
		{`, "offer-table": "inn", "hero-slots": {"mage": 6, "female": 6}`, `{"topic": 1, "npc": 1}`, "hero-slots"},
	} {
		if _, err := DecodeCatalog([]byte(innCatalogHead+tc.head+`, "offers": [`+tc.rows+`]}`), catalogVocab); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s %s: %v, want %q", tc.head, tc.rows, err, tc.want)
		}
	}
}
