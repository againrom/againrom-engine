package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The gold line is install string 88, the gain and install string 89, each
// after one space; a word the install does not state falls back to the
// authored one (ITEM-PICKTEXT-145).
func TestPickupGoldLineStatesTheGainBetweenTheInstallWords(t *testing.T) {
	words := ui.AuthoredWords()
	if got, want := pickupGoldLine(&words, 500), "Picked up 500 gold"; got != want {
		t.Fatalf("pickupGoldLine(500) = %q, want %q", got, want)
	}
	words.PickedUpGold, words.PickedUpGoldUnit = "Took", "coins"
	if got, want := pickupGoldLine(&words, 1), "Took 1 coins"; got != want {
		t.Fatalf("pickupGoldLine(1) with installed words = %q, want %q", got, want)
	}
	words.PickedUpGold = ""
	if got, want := pickupGoldLine(&words, 2147483647), "Picked up 2147483647 coins"; got != want {
		t.Fatalf("pickupGoldLine with string 88 missing = %q, want %q", got, want)
	}
}

// The gold line states the signed rise of the purse and nothing else: a purse
// that did not rise, or whose new total reads negative, posts no gold line
// (ITEM-PICKTEXT-145).
func TestPurseGainIsTheSignedRiseAndNothingElse(t *testing.T) {
	for _, c := range []struct {
		name          string
		before, after uint32
		want          int32
	}{
		{"rise", 100, 600, 500},
		{"from empty", 0, 1, 1},
		{"unchanged", 600, 600, 0},
		{"fall", 600, 100, 0},
		{"new total reads negative", 0x7fffff00, 0x80000100, 0},
		{"both totals read negative", 0x80000000, 0x80000064, 100},
	} {
		if got := purseGain(c.before, c.after); got != c.want {
			t.Errorf("%s: purseGain(%#x, %#x) = %d, want %d", c.name, c.before, c.after, got, c.want)
		}
	}
}

// One take states the gold line and then its carried Items in the given order;
// a take that gained no gold states none.
func TestGoldPickupKeepsItemTextAndOrder(t *testing.T) {
	table := eqDefsTable(t)
	table.Names = data.ItemNames{data.ItemCode(eqSwordCode): "Gold", data.ItemCode(eqNoSlotCode): "Arrow"}
	reached := []sim.ItemStack{sim.PlainStack(eqSwordCode, 2), sim.PlainStack(eqNoSlotCode, 1)}
	words := ui.AuthoredWords()
	items := []string{"Picked up Gold (now 2 pieces)", "Picked up Arrow"}
	if got := pickupLinesForTake(reached, 0, table, &words); !slices.Equal(got, items) {
		t.Fatalf("a take with no gold stated %q, want %q", got, items)
	}
	if got, want := pickupLinesForTake(reached, 500, table, &words), append([]string{"Picked up 500 gold"}, items...); !slices.Equal(got, want) {
		t.Fatalf("a take with gold stated %q, want %q", got, want)
	}
	if got := pickupLinesForTake(nil, 500, nil, &words); !slices.Equal(got, []string{"Picked up 500 gold"}) {
		t.Fatalf("a gold-only take stated %q, want the gold line alone", got)
	}
}

// The message line changes only after the transfer succeeded: a take posts the
// gold line and the item line once, and an empty ground under the hero posts
// nothing more and moves no gold.
func TestGoldPickupPostsOnlyAfterSuccessfulTransfer(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 20, MaxHP: 20}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 500, Items: []uint16{eqSwordCode, eqSwordCode}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	mw.invParty.table = eqDefsTable(t)
	before := w.Purse(sim.SelfSlot)
	mw.takeSackFor(7)
	rows := mw.view.MessageLines()
	if w.Purse(sim.SelfSlot) != before+500 || len(w.Sacks()) != 0 || len(rows) != 2 {
		t.Fatal("pickup did not transfer and post once", w.Purse(sim.SelfSlot), rows)
	}
	if rows[0].Text != "Picked up 500 gold" || rows[1].Text != "Picked up Sword (now 2 pieces)" {
		t.Fatalf("the take posted %+v, want the gold line then the item line", rows)
	}
	mw.takeSackFor(7)
	if len(mw.view.MessageLines()) != 2 || w.Purse(sim.SelfSlot) != before+500 {
		t.Fatal("empty ground repeated the gold line or the transfer")
	}
}
