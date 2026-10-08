package game

import (
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCityShopSoldStaffKeepsCurrentSourceWithoutOwnedSpell(t *testing.T) {
	for _, form := range []string{"source_to_plain", "source_to_source", "plain_to_plain"} {
		t.Run(form, func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			f.Carried[0].ID = "hero"
			params := chargenWeaponParams(3)
			params[2], params[3], params[6], params[7] = 733, 5, 0, 0
			params[11], params[12], params[13], params[14], params[15] = 5, 8, 4, 2, 2
			f.Table.Weapons = dbCollection{{}, {name: "Staff", params: params}}
			materials := fixtureScale{names: make([]string, 9), doubles: make([][]float64, 9)}
			materials.names[8] = "Wood"
			for i := range materials.doubles {
				materials.doubles[i] = []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
			}
			f.Table.Materials = materials
			plain := sim.ItemInstance{Code: 0x8101, Kind: 2, Price: 733, Effects: []sim.ItemEffect{{Kind: 41, Operand: 65537}}}
			if code := data.ItemCode(plain.Code); code.B() != shopWeaponClass || code.A() != 8 || code.D() != 1 {
				t.Fatal("staff code has invalid class, material or definition row", code)
			}
			built := mapload.SourceConstructedItem(plain.Clone(), f.Table)
			if !built.WeightPresent || built.Weight != 5 || built.Kind != 2 || built.SourceEquipment.Class != sim.SourceWeapon || built.SourceEquipment.DefinitionRow != 1 || !built.SourceEquipment.Definition.Present || built.SourceEquipment.Definition.Suitable != 2 || built.SourceEquipment.Spell.Present {
				t.Fatal("staff constructor did not provide the current absent-Spell control", built)
			}
			incoming, target := built.Clone(), plain.Clone()
			if form == "source_to_source" {
				target = built.Clone()
			} else if form == "plain_to_plain" {
				incoming = plain.Clone()
			}
			f.Shop = NewShop(1000)
			f.Shop.tbl = f.Table
			f.Shop.shelves[ShelfMagic] = []ShopItem{shopItemFromInstance(target, 1)}
			cityShopSetTestPack(t, f, 0, sim.StackItem(incoming, 1))
			cityShopGraph(t, f)
			if action := screen.shopFromPack(0, true); strings.Contains(action.Msg, "cannot") {
				t.Fatal("current staff staging failed", action)
			}
			if places := f.Shop.Table(); len(places) != 1 || !staffSaleValueEqual(places[0].Instance(), incoming) || places[0].SourceEquipment.Spell.Present {
				t.Fatal("staging changed the current staff value or invented an owned Spell", places)
			}
			gold := f.Town.Gold()
			if action := screen.shopSell(); strings.Contains(action.Msg, "cannot") || f.Town.Gold()-gold != 367 || len(f.Shop.Table()) != 0 {
				t.Fatal("current staff sale did not commit its exact payout", action, f.Town.Gold()-gold)
			}
			wantLots, wantTarget, wantIncoming := 1, int32(2), int32(2)
			if form == "source_to_plain" {
				wantLots, wantTarget, wantIncoming = 2, 1, 1
			}
			var targetUnits, incomingUnits int32
			stock := f.Shop.Shelf(ShelfMagic)
			for _, lot := range stock {
				if staffSaleValueEqual(lot.Instance(), target) {
					targetUnits += lot.Count
				}
				if staffSaleValueEqual(lot.Instance(), incoming) {
					incomingUnits += lot.Count
				}
				if lot.SourceEquipment.Spell.Present {
					t.Fatal("sale invented an owned Spell on the shelf", lot)
				}
			}
			if len(stock) != wantLots || targetUnits != wantTarget || incomingUnits != wantIncoming || len(f.Shop.Shelf(ShelfWeapons)) != 0 {
				t.Fatal("sale replaced current source operands with generated stock", form, stock, targetUnits, incomingUnits)
			}
		})
	}
}
