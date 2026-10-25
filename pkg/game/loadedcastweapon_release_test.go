package game

import (
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestReleaseLoadedCastWeaponUnequipsToTable: table take-off, no node edge.
func TestReleaseLoadedCastWeaponUnequipsToTable(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Hero", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.gold = 1000000
	f.Shop = NewShop(3000)
	f.Shop.Generate(f.Table, 1)
	screen := f.TownScreen().(*townScreen)
	screen.room, screen.shopMember = roomShop, 0
	bought := false
	for k, r := range screen.rack() {
		if !r.stocked || bought {
			continue
		}
		for idx, it := range f.Shop.shelves[r.shelf] {
			in := it.Instance()
			if _, _, ok := in.CastSpell(); !ok || data.ItemCode(in.Code).B() != 1 {
				continue
			}
			screen.shopChosen, screen.shelfBase = k, 0
			if a := screen.shopEquipFromShelf(idx); a.Msg != "" && !strings.Contains(a.Msg, "cannot") {
				bought = true
				break
			}
		}
	}
	if !bought {
		t.Skip("no fighter castSpell weapon in the generated stock")
	}
	raw, stage, err := writerCensusSave(f, false, t.TempDir())
	if err != nil {
		t.Fatal(stage, err)
	}
	h := releaseFront(t)
	if _, town, err := h.RestoreOriginal(raw); err != nil || !town {
		t.Fatal(town, err)
	}
	root, err := cityMutationParty(h.Town.cityObjects, h.Carried[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	worn := mapload.MemberItemEquipment(h.Carried[0], h.Table)[0]
	if _, _, cast := worn.CastSpell(); !cast || !worn.SourceEquipment.Spell.Present {
		t.Fatalf("control: loaded weapon %#x class %d spell %v", worn.Code, worn.SourceEquipment.Class, worn.SourceEquipment.Spell.Present)
	}
	row, err := cityMutationSource(h.Town.cityObjects, root.Worn[0], cityItemLocation{})
	if err != nil || h.Town.cityObjects.Items[row].Spell == 0 {
		t.Fatalf("control: the worn node holds no Spell edge (%v)", err)
	}
	hs := h.TownScreen().(*townScreen)
	hs.room, hs.shopMember = roomShop, 0
	if a := hs.shopUnequipToTable(1); !strings.Contains(a.Msg, "on the table") {
		t.Fatalf("unequip to the table: %q", a.Msg)
	}
	if len(h.Shop.table) != 1 {
		t.Fatalf("table holds %d items", len(h.Shop.table))
	}
	placed := h.Shop.table[0].ShopItem
	row, err = cityMutationSource(h.Town.cityObjects, placed.CityID, cityItemLocation{})
	if err != nil || h.Town.cityObjects.Items[row].Spell != 0 {
		t.Fatalf("table node %d keeps its Spell edge (%v)", placed.CityID, err)
	}
}
