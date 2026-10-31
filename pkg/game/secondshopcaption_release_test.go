package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// shelfStockKind is what a shelf's items are.
type shelfStockKind int

const (
	holdsNothing shelfStockKind = iota
	holdsArmour
	holdsWeapons
	holdsEnchanted
	holdsMagicAndBooks
	holdsArmourAndWeapons
	holdsMagic
	holdsBooks
	holdsMixed
)

func (h shelfStockKind) String() string {
	return [...]string{"nothing", "armour and shields", "weapons", "enchanted equipment", "magic items and books",
		"armour with some weapons", "magic items", "books", "mixed"}[h]
}

func classifyShelf(items []secondshop.Item) shelfStockKind {
	if len(items) == 0 {
		return holdsNothing
	}
	var armour, weapon, magic, book, plain, spelled int
	for _, it := range items {
		switch it.Table {
		case secondshop.Armor, secondshop.Shield:
			armour++
		case secondshop.Weapon:
			weapon++
		case secondshop.Magic:
			magic++
		case secondshop.Book:
			book++
		}
		if it.Table == secondshop.Armor || it.Table == secondshop.Shield || it.Table == secondshop.Weapon {
			if len(it.Effects) > 0 {
				spelled++
			} else {
				plain++
			}
		}
	}
	switch {
	case magic > 0 && book > 0 && armour+weapon == 0:
		return holdsMagicAndBooks
	case magic > 0 && book+armour+weapon == 0:
		return holdsMagic
	case book > 0 && magic+armour+weapon == 0:
		return holdsBooks
	case armour+weapon > 0 && magic+book == 0 && plain == 0 && spelled > 0:
		return holdsEnchanted
	case armour > 0 && weapon == 0 && magic+book == 0:
		return holdsArmour
	case weapon > 0 && armour == 0 && magic+book == 0:
		return holdsWeapons
	case armour > 0 && weapon > 0 && magic+book == 0:
		return holdsArmourAndWeapons
	}
	return holdsMixed
}

// captionSlot is the main text line that names each kind of stock.
var captionSlot = map[shelfStockKind]int{
	holdsArmour: 62, holdsWeapons: 63, holdsEnchanted: 64, holdsMagic: 64, holdsMagicAndBooks: 65, holdsBooks: 65,
	holdsArmourAndWeapons: 62,
}

// secondShelfCaptionWant is each town's shelves: their stock and caption line.
var secondShelfCaptionWant = map[int][4]struct {
	holds shelfStockKind
	slot  int
}{
	1: {{holdsArmour, 62}, {holdsWeapons, 63}, {holdsEnchanted, 64}, {holdsMagicAndBooks, 65}},
	2: {{holdsArmour, 62}, {holdsEnchanted, 64}, {holdsMagicAndBooks, 65}, {holdsWeapons, 63}},
	3: {{holdsArmourAndWeapons, 62}, {holdsEnchanted, 64}, {holdsMagicAndBooks, 65}, {holdsEnchanted, 64}},
}

// TestReleaseSecondShopShelfCaptions hovers every shown shelf of the three
// second-game shops and requires the caption that names what the shelf holds.
func TestReleaseSecondShopShelfCaptions(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("town %d", id), func(t *testing.T) {
			secondGameRoot(t)
			var f *FrontEnd
			var app *ui.App
			var s *townScreen
			if id == 1 {
				f, app, s, _ = secondShopTownOne(t)
			} else {
				f, app, s, _ = secondShopLaterTown(t, id)
			}
			c := f.Town.second
			if id == 3 {
				c.bank[770] = 1
				c.talkTo(innOffer{kind: 0, npc: 675, topic: 78})
				c.shop.filled = 0
			}
			secondShopEnter(t, f, app, s)
			captions := map[string]int{}
			for k := 0; k < 4; k++ {
				want := secondShelfCaptionWant[id][k]
				secondShopActivate(t, f, app, fmt.Sprintf("SHELF %d", k+1))
				x, y, err := app.HeadlessShopPoint("shelf_pick", k)
				if err != nil {
					t.Fatalf("shelf %d is not shown: %v", k, err)
				}
				if err := app.HeadlessPointer("hover", x, y); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 80; i++ {
					if err := app.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
				}
				var items []secondshop.Item
				for _, it := range s.deal().Shelf(ShopShelf(k)) {
					items = append(items, secondItem(it))
				}
				holds := classifyShelf(items)
				hover := strings.Fields(routeText(strings.Join(app.HeadlessTooltipLines(), " ")))
				text := strings.Fields(routeText(f.Words.Hover[want.slot]))
				t.Logf("town %d shelf %d holds %v (%d items), hovers %q, line %d is %q", id, k, holds, len(items), strings.Join(hover, " "), want.slot, strings.Join(text, " "))
				if holds != want.holds {
					t.Errorf("town %d shelf %d holds %v, want %v", id, k, holds, want.holds)
				}
				if got := strings.Join(hover, " "); got != strings.Join(text, " ") {
					t.Errorf("town %d shelf %d hovers %q, want line %d %q", id, k, got, want.slot, strings.Join(text, " "))
				}
				if named, ok := captionSlot[holds]; ok && named != want.slot {
					t.Errorf("town %d shelf %d: stock %v is named by line %d, the table wants %d", id, k, holds, named, want.slot)
				}
				captions[strings.Join(hover, " ")]++
			}
			// Loss control: one caption repeated fails.
			if len(captions) < 3 {
				t.Errorf("town %d shows %d distinct captions: %v", id, len(captions), captions)
			}
		})
	}
}
