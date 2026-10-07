package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func sourceCityEquipment(t *testing.T) (*FrontEnd, *townScreen, sim.ItemInstance) {
	t.Helper()
	f := city1102Front(t, false)
	screen := f.TownScreen().(*townScreen)
	m := screen.shopPartyMember(screen.shopMemberIndex())
	m.OriginalHuman = nil // fixture is a retained live basis, not original-city export provenance
	f.originalCity = nil
	s := sim.SourceActor{Class: 2, Fighter: true, TypeID: 33, EquipmentRuntimePresent: true, Reach: 1, AttackCharge: 8, AttackRelax: 4,
		Stats:  [14]uint16{10, 15, 10, 10, 37, 0, 500, 201, 40, 90, 100, 0, 0, 50},
		Attack: [24]byte{99, 0, 7, 0, 44, 0, 14: 8, 15: 9, 16: 1}, Base: [24]byte{0, 0, 0xbc, 1, 9, 0}}
	s.Modifier[4], s.Modifier[6], s.Modifier[8], s.Modifier[18], s.Modifier[22], s.Modifier[42] = 2, 200, 4, 5, 2, 3
	m.Carry.LiveLoad = &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, ContainerPresent: true, Accumulator: 406, Source: s},
		Load: 500, Capacity: 201, Speed: 37, Movement: sim.HumanMovement{Present: true, RawSpeed: 37, NativeSpeed: 37, Load: 500, Capacity: 201}}
	item := sim.ItemInstance{Code: 0x0701, Kind: 1, WeightPresent: true, Weight: 2, Price: 10,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7, Defence: [22]byte{7}}}
	m.Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(item, 2)}
	m.Carry.Items = []uint16{item.Code, item.Code}
	m.Carry.ItemInstances = []sim.ItemInstance{item, item}
	m.Carry.EquippedItems = [sim.EquipSlots]sim.ItemInstance{}
	m.Carry.Equipped = [sim.EquipSlots]uint16{}
	m.Hero = mapload.SourceHumanState(s, 406).Hero()
	f.Shop.table = nil
	f.Shop.shelves[ShelfArmour] = []ShopItem{shopItemFromInstance(item, 2)}
	screen.shopChosen = 0
	return f, screen, item
}

func TestSourceEquipment1110CityAllOriginsNativeNextRemoval(t *testing.T) {
	for _, origin := range []string{"pack", "shelf", "mine table", "merchant table"} {
		t.Run(origin, func(t *testing.T) {
			f, screen, item := sourceCityEquipment(t)
			gold := f.Town.Gold()
			switch origin {
			case "pack":
				if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
					t.Fatal(a)
				}
			case "shelf":
				if a := screen.shopEquipFromShelf(0); a.Msg == "" || len(f.Shop.Shelf(ShelfArmour)) != 1 || f.Shop.Shelf(ShelfArmour)[0].Count != 1 {
					t.Fatal(a)
				}
			default:
				f.Shop.table = []ShopPlace{{ShopItem: shopItemFromInstance(item, 2), Mine: origin == "mine table"}}
				want := "worn"
				if origin == "merchant table" {
					want = fmt.Sprintf("bought and worn for 10 gold; you have %d left", gold-10)
				}
				if a := screen.shopEquipFromTable(0); a.Msg != want || f.Shop.Table()[0].Count != 1 {
					t.Fatal(a)
				}
			}
			m := screen.shopPartyMember(screen.shopMemberIndex())
			wantLoad, wantCount, wantGold := int32(205), 2, gold
			if origin == "pack" {
				wantLoad, wantCount = 204, 1
			}
			if origin == "shelf" || origin == "merchant table" {
				wantGold -= 10
			}
			check := func(m mapload.PartyMember) {
				t.Helper()
				state := m.Carry.LiveLoad
				if state.Load != wantLoad || state.Capacity != 301 || state.Speed != 15 ||
					binary.LittleEndian.Uint16(state.Inventory.Source.Defence[:]) != 15 || state.Inventory.Source.Stats[8] != 24 ||
					m.Carry.Equipped[6] != item.Code || len(m.Carry.ItemInstances) != wantCount || f.Town.Gold() != wantGold {
					t.Fatal("literal city result", state, m.Carry, f.Town.Gold())
				}
			}
			check(*m)
			snapshot, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := EncodeSave(snapshot, "source equipment city")
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, err := DecodeSave(raw)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded.Party, snapshot.Party) {
				t.Fatal("city native drops retained equipment or source actor")
			}
			// A fresh town instance gets only decoded state, no producer closure.
			f.Carried = mapload.CloneParty(decoded.Party)
			screen = f.TownScreen().(*townScreen)
			if a := screen.shopUnequipToTable(7); a.Msg != "on the table" {
				t.Fatal(a)
			}
			m = screen.shopPartyMember(screen.shopMemberIndex())
			if m.Carry.Equipped[6] != 0 || m.Carry.LiveLoad.Load != wantLoad-2 || len(f.Shop.Table()) == 0 {
				t.Fatal("fresh native next removal")
			}
		})
	}
}

func TestSourceEquipment1110CityRefusalsAreAtomic(t *testing.T) {
	for _, action := range []string{"pack", "shelf", "table", "doll", "doll table"} {
		for _, fault := range []string{"capacity", "unsupported effect", "late effect"} {
			t.Run(action+"/"+fault, func(t *testing.T) {
				f, screen, item := sourceCityEquipment(t)
				m := screen.shopPartyMember(screen.shopMemberIndex())
				switch fault {
				case "capacity":
					m.Carry.LiveLoad.Capacity = 0
					m.Carry.LiveLoad.Movement.Capacity = 0
					m.Carry.LiveLoad.Inventory.Source.Stats[7] = 0
				case "unsupported effect":
					item.SourceEquipment.EffectsUnsupported = true
				case "late effect":
					item.Effects = []sim.ItemEffect{{Kind: 2, Operand: 1}, {Kind: 1, Operand: 2}}
				}
				m.Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(item, 1)}
				m.Carry.ItemInstances = []sim.ItemInstance{item}
				m.Carry.Items = []uint16{item.Code}
				if action == "doll" || action == "doll table" {
					m.Carry.EquippedItems[6] = item
					m.Carry.Equipped[6] = item.Code
				}
				f.Shop.shelves[ShelfArmour] = []ShopItem{shopItemFromInstance(item, 1)}
				f.Shop.table = []ShopPlace{{ShopItem: shopItemFromInstance(item, 1)}}
				party, trade, gold := mapload.CloneParty(f.Carried), cloneShopMutation(f.Shop), f.Town.Gold()
				switch action {
				case "pack":
					screen.shopEquipFromPack(1)
				case "shelf":
					screen.shopEquipFromShelf(0)
				case "table":
					screen.shopEquipFromTable(0)
				case "doll":
					screen.shopUnequipDoll(7)
				case "doll table":
					screen.shopUnequipToTable(7)
				}
				if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(trade, cloneShopMutation(f.Shop)) || gold != f.Town.Gold() {
					t.Fatal("refusal committed prefix")
				}
			})
		}
	}
}
