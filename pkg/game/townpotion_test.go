package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var townPotionShapes = []struct {
	name   string
	effect sim.ItemEffect
}{
	{"Body", sim.ItemEffect{Kind: 2, Operand: 3}},
	{"Reaction", sim.ItemEffect{Kind: 4, Operand: 3}},
	{"Mind", sim.ItemEffect{Kind: 3, Operand: 3}},
	{"Spirit", sim.ItemEffect{Kind: 5, Operand: 3}},
	{"Healing", sim.ItemEffect{Kind: 6, Operand: 50}},
	{"Mana", sim.ItemEffect{Kind: 9, Operand: 50}},
	{"Health Regeneration", sim.ItemEffect{Kind: 8, Mode: 1, Operand: 100 | 960<<16}},
	{"Mana Regeneration", sim.ItemEffect{Kind: 11, Mode: 1, Operand: 100 | 960<<16}},
	{"Absorption", sim.ItemEffect{Kind: 16, Mode: 1, Operand: 50 | 480<<16}},
}

// townPotionMember is a wounded mage, as an original SAV can hold.
func townPotionMember(f *FrontEnd) {
	m := &f.Carried[0]
	m.ID, m.Name, m.StartingHero, m.PlayerCharacter = "hero", "Hero", true, true
	m.Mage, m.Profile = true, data.Profile{HealthColumn: true, ManaColumn: true}
	m.Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	m.Saved = &mapload.Saved{HP: 17, MaxHP: 257, Mana: 19, MaxMana: 113}
}

func townPotionUnchanged(t *testing.T, before, after mapload.PartyMember, table *mapload.Table) {
	t.Helper()
	if !reflect.DeepEqual(before.Hero, after.Hero) || !reflect.DeepEqual(before.Saved, after.Saved) || after.PotionEffect != nil ||
		before.KnownSpells != after.KnownSpells || !reflect.DeepEqual(before.Profile, after.Profile) {
		t.Fatalf("town potion changed the member:\nbefore %+v %+v\nafter  %+v %+v effect %+v", before.Hero, before.Saved, after.Hero, after.Saved, after.PotionEffect)
	}
	var bh, ah *mapload.NativeCarryHistory
	if before.Carry != nil {
		bh = before.Carry.NativeHistory
	}
	if after.Carry != nil {
		ah = after.Carry.NativeHistory
	}
	if !reflect.DeepEqual(bh, ah) {
		t.Fatalf("town potion changed the native history: %+v -> %+v", bh, ah)
	}
	bd, bhp, bmp := mapload.PartyDisplayWithTable(before, table)
	ad, ahp, amp := mapload.PartyDisplayWithTable(after, table)
	if bd != ad || bhp != ahp || bmp != amp {
		t.Fatalf("town potion changed the sheet: %+v %d %d -> %+v %d %d", bd, bhp, bmp, ad, ahp, amp)
	}
}

func TestTownPotionIsUsedUpAndChangesNothing(t *testing.T) {
	for _, shape := range townPotionShapes {
		for _, topology := range []bool{false, true} {
			for _, route := range []string{"pack", "table", "shelf"} {
				name := shape.name + "/" + route + map[bool]string{false: "/values", true: "/city objects"}[topology]
				t.Run(name, func(t *testing.T) {
					f, s := shopRoom(t, nil)
					f.Town.open = true
					townPotionMember(f)
					potion := sim.ItemInstance{Code: 0xe08, Kind: 3, Price: 10, Effects: []sim.ItemEffect{shape.effect}}
					if route == "pack" {
						cityShopSetTestPack(t, f, 0, sim.StackItem(potion, 1))
					} else {
						cityShopSetTestPack(t, f, 0)
					}
					if topology {
						cityShopGraph(t, f)
					}
					s.shopChosen = 2
					f.Shop.shelves[ShelfMagic] = []ShopItem{shopItemFromInstance(potion, 1)}
					f.Shop.table = nil
					if route == "table" {
						f.Shop.table = []ShopPlace{{ShopItem: shopItemFromInstance(potion, 1), Mine: true}}
					}
					before := mapload.CloneParty(f.Carried)[0]
					gold := f.Town.Gold()
					var action ui.TownAction
					switch route {
					case "pack":
						action = s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlDoll})
					case "table":
						action = s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll})
					case "shelf":
						action = s.shopEquipFromShelf(0)
					}
					paid := 0
					if route == "shelf" {
						paid = int(potion.Price)
					}
					if action.Msg == "" || len(s.shopPackStacks()) != 0 || len(f.Shop.Table()) != 0 || f.Town.Gold() != gold-paid ||
						route == "shelf" && len(f.Shop.Shelf(ShelfMagic)) != 0 {
						t.Fatalf("potion not used up: %+v pack %v table %v gold %d->%d", action, s.shopPackStacks(), f.Shop.Table(), gold, f.Town.Gold())
					}
					townPotionUnchanged(t, before, f.Carried[0], f.Table)
					if !topology || route != "pack" {
						return
					}
					fresh := cityShopColdLoad(t, f)
					townPotionUnchanged(t, before, fresh.Carried[0], fresh.Table)
					m := &f.Carried[0]
					w, st := townPotionMissionStart(t, fresh.Carried[:1])
					control, cst := townPotionMissionStart(t, []mapload.PartyMember{*m})
					if len(w.ActiveEffects()) != 0 {
						t.Fatalf("next mission carries a town potion: %+v", w.ActiveEffects())
					}
					got, _ := w.Entity(st.IDs[0])
					want, _ := control.Entity(cst.IDs[0])
					if got.HealthRegeneration != want.HealthRegeneration || got.ManaRegeneration != want.ManaRegeneration || got.Absorption != want.Absorption {
						t.Fatalf("next mission actor differs from the unchanged member: %+v want %+v", got, want)
					}
				})
			}
		}
	}
}

func townPotionMissionStart(t *testing.T, party []mapload.PartyMember) (*sim.World, mapload.Start) {
	t.Helper()
	w, st, err := mapload.StartMission(townPotionMap(), nil, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	return w, st
}

func townPotionMap() *alm.Map {
	return &alm.Map{Width: 32, Height: 32, Tiles: make([]uint16, 1024), Overlay: make([]uint8, 1024)}
}
