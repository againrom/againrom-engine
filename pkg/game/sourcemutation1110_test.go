package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSourceMutation1110CityRefusalsKeepItemsPurseAndSource(t *testing.T) {
	for _, action := range []string{"buy", "individual return", "pack potion", "shelf potion", "table potion"} {
		t.Run(action, func(t *testing.T) {
			f := city1102Front(t, false)
			screen := f.TownScreen().(*townScreen)
			item := sim.ItemInstance{Code: 0xe07, Kind: 3, Price: 10, WeightPresent: true, Weight: 7, Effects: []sim.ItemEffect{{Kind: 2, Operand: 1}}}
			if !screen.setShopPackItemInstances([]sim.ItemInstance{item}) {
				t.Fatal("fixture pack")
			}
			f.Shop.table = []ShopPlace{{ShopItem: shopItemFromInstance(item, 1), Mine: action == "individual return"}}
			f.Shop.shelves[ShelfMagic] = []ShopItem{shopItemFromInstance(item, 1)}
			screen.shopChosen = 2
			member := screen.shopPartyMember(screen.shopMemberIndex())
			if member.Carry.LiveLoad.Inventory.Source.Class != 2 {
				t.Fatal("not a retained Human")
			}
			member.Carry.LiveLoad.Capacity = 0
			member.Carry.LiveLoad.Movement.Capacity = 0
			member.Carry.LiveLoad.Inventory.Source.Stats[7] = 0
			party, table, shelf, gold := mapload.CloneParty(f.Carried), f.Shop.Table(), f.Shop.Shelf(ShelfMagic), f.Town.Gold()
			switch action {
			case "buy":
				screen.shopBuy()
			case "individual return":
				screen.shopOffTable(0, false)
			case "pack potion":
				screen.shopEquipFromPack(0)
			case "shelf potion":
				screen.shopEquipFromShelf(0)
			case "table potion":
				screen.shopEquipFromTable(0)
			}
			if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(table, f.Shop.Table()) || !reflect.DeepEqual(shelf, f.Shop.Shelf(ShelfMagic)) || gold != f.Town.Gold() {
				t.Fatal("source refusal committed item/purse/state prefix")
			}
		})
	}
}

func TestSourceMutation1110CityCommitFailureNeverUsesNativeFallback(t *testing.T) {
	f := city1102Front(t, false)
	screen := f.TownScreen().(*townScreen)
	member := screen.shopPartyMember(screen.shopMemberIndex())
	result := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	result.Carry.LiveLoad.Inventory.Source.Stats[7] = 0
	result.Hero.Body = 100
	before := mapload.CloneParty([]mapload.PartyMember{*member})[0]
	if commitTownPotion(member, result) || !reflect.DeepEqual(*member, before) {
		t.Fatal("source commit failure took native fallback")
	}
}
