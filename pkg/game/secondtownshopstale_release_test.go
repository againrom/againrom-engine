package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestReleaseSecondTownDruidStaleStockSave writes a SAV between the druid inn
// conversation and the next entry to the shop: the stock drawn before the
// conversation is stale in the SAV, and a separate process that LOADs it and
// enters the shop draws a fourth category with goods (DIV-3045).
func TestReleaseSecondTownDruidStaleStockSave(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_STALE_INPUT"); path != "" {
		f, app := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
		c := f.Town.second
		if c.shop.filled == 3 || len(c.shop.stock[3]) != 0 || c.bank[770] != 2 {
			t.Fatalf("cold LOAD: filled %d, fourth category %d entries, slot 770 %d", c.shop.filled, len(c.shop.stock[3]), c.bank[770])
		}
		secondShopEnter(t, f, app, f.TownScreen().(*secondCampaignScreen))
		secondShopClassFour(t, c)
		t.Logf("fresh-process LOAD: fourth category %d entries", len(c.shop.stock[3]))
		return
	}
	f, app, s, _ := secondShopLaterTown(t, 3)
	c := f.Town.second
	c.bank[770] = 1
	secondShopEnter(t, f, app, s)
	if len(c.shop.stock[3]) != 0 || c.shop.filled != 3 {
		t.Fatalf("before the conversation: fourth category %d entries, filled %d", len(c.shop.stock[3]), c.shop.filled)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
	c.talkTo(secondInnOption{kind: 0, npc: 675, topic: 78})
	dir := secondMissionSaveDirectory(t)
	secondTownNamedSave(t, f, app, dir, "Stale")
	runSpellWitnessChild(t, filepath.Join(dir, secondSaveFile(t, dir, "Stale")), "AGAINROM_SECOND_STALE_INPUT")
}

// TestReleaseSecondTownKaargDepartureStale: a departure that changes the record of
// the location the stock was drawn for makes the next entry draw again, and
// one that leaves the record equal keeps the stock.
func TestReleaseSecondTownKaargDepartureStale(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopLaterTown(t, 2)
	c := f.Town.second
	secondShopEnter(t, f, app, s)
	if c.shop.filled != 2 || len(c.shop.stock[0]) == 0 {
		t.Fatalf("first visit: filled %d", c.shop.filled)
	}
	first := c.shop.stock
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
	// Loss control: a departure that writes nothing to record 2 keeps the stock.
	c.shop.depart(99)
	if c.shop.filled != 2 {
		t.Fatal("a departure that changes no record marked the stock stale")
	}
	c.current = secondLocation{1, 30}
	c.completeBank(c.bank)
	if c.shop.filled == 2 || c.bank[768] != 40 {
		t.Fatalf("mission 30 departure: filled %d, stage %d", c.shop.filled, c.bank[768])
	}
	c.current = secondLocation{2, 2}
	secondShopEnter(t, f, app, s)
	if c.shop.filled != 2 || len(c.shop.stock[0]) == 0 || reflect.DeepEqual(first, c.shop.stock) {
		t.Fatalf("next entry did not draw again: filled %d", c.shop.filled)
	}
}
