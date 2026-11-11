package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// A worn item lifted off the shop's doll leaves the picture the moment it is
// lifted, returns with a cancelled lift, and stays off once dropped in the
// pack. The shown figure is asked through ShopScreen, the one view both games'
// shop rooms draw the doll from.
func checkLiftedDoll(t *testing.T, s *townScreen, slot int, drop func()) {
	t.Helper()
	ordinary := s.ShopScreen()
	if ordinary.Figure == nil {
		t.Fatal("the doll has no figure to lift an item off")
	}

	s.ShopSuppressDoll(slot)
	lifted := s.ShopScreen()
	if s.shopSuppressFigure == nil {
		t.Fatal("lifting a worn item composed no figure without it")
	}
	if lifted.Figure != s.shopSuppressFigure || lifted.Figure == ordinary.Figure {
		t.Error("the doll still shows the lifted item")
	}
	if lifted.SlotMask != s.shopSuppressMask {
		t.Error("the doll's slot map still names the lifted item")
	}
	if lifted.Character.Figure != lifted.Figure {
		t.Error("the character pane still shows the lifted item")
	}

	s.ShopSuppressDoll(0)
	if back := s.ShopScreen(); back.Figure != ordinary.Figure || back.Character.Figure != ordinary.Figure {
		t.Error("a cancelled lift did not bring the item back onto the doll")
	}

	s.ShopSuppressDoll(slot)
	drop()
	s.ShopSuppressDoll(0)
	if item, ok := s.shopEquippedItem(0, slot); ok && !item.Empty() {
		t.Fatal("the dropped item is still worn")
	}
	after := s.ShopScreen()
	if after.Figure == lifted.Figure || after.Figure == nil {
		t.Error("the doll after the drop is not the unit's own picture")
	}
	if after.Figure != after.Character.Figure {
		t.Error("the character pane differs from the doll after the drop")
	}
}

func TestShopDollRedrawsWhenAWornItemIsLiftedFirstGame(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Archives = shopFigureArchive(t)
	click(s, ui.ShopControlShelfPick, roomArmour)
	idx := shopWearableShelfIndex(t, f, ShelfArmour)
	code := f.Shop.Shelf(ShelfArmour)[idx].Code
	slot, _ := EquipTarget(code, f.Table)
	s.shelfBase = idx
	s.shopEquipFromShelf(0)
	checkLiftedDoll(t, s, slot, func() {
		s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1},
			ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(0)})
	})
}

func TestShopDollRedrawsWhenAWornItemIsLiftedSecondGame(t *testing.T) {
	f, _, screen := secondCampaignFixtureFiles(t, false)
	f.Table = secondDollTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hero.Worn[1] = secondDollTarge
	hero.WornItems[1] = mapload.ItemInstanceFromCode(secondDollTarge, f.Table)
	f.Carried = []mapload.PartyMember{hero}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}}, secondTownShop
	*c.heroBag() = []sim.ItemInstance{secondDollAxe}
	f.Town.second, f.Town.gold = c, 500
	f.Archives.Containers = shopFigureArchive(t).Containers
	screen.bindShop(screen.bankShopRow())
	screen.shop.shown = nil
	screen.composed = newTownSceneState(nil)
	d, ok := screen.composedTown()
	if !ok {
		t.Fatal("no composed town")
	}
	screen.composed.art[d.Town+"/shop"] = &townRoomArt{scene: &town.Art{}}
	checkLiftedDoll(t, screen, 2, func() {
		screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: 1},
			ui.ShopControl{Kind: ui.ShopControlPackCell, Index: screen.packCell(0)})
	})
}
