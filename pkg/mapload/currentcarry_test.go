package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestMaterializePartyCarryKeepsSpawnAndCurrentHoldings(t *testing.T) {
	hero := data.NewHero(data.Spread{Body: 25, Reaction: 27, Mind: 31, Spirit: 23}, 2)
	member := mapload.PartyMember{ID: "hero", Hero: hero, CarriedItems: []sim.ItemInstance{{Code: 0x111, Kind: 2, Price: 73}},
		WornItems: [sim.EquipSlots]sim.ItemInstance{{Code: 0x112, Kind: 2, Price: 42}}}
	member.Hero.Skill[4] = 12
	before, hp, mana := mapload.PartySpawnWithTable(member, nil)
	next := mapload.MaterializePartyCarry(member, nil)
	after, gotHP, gotMana := mapload.PartySpawnWithTable(next, nil)
	if !reflect.DeepEqual(before, after) || hp != gotHP || mana != gotMana || next.Carry == nil || next.Carry.LiveLoad != nil || next.Carry.SkillXP != member.Hero.Reward().SkillXP {
		t.Fatal("current Carry changed spawn or invented actor load")
	}
	if !reflect.DeepEqual(mapload.MemberCarriedItems(member, nil), mapload.MemberCarriedItems(next, nil)) || !reflect.DeepEqual(mapload.MemberItemEquipment(member, nil), mapload.MemberItemEquipment(next, nil)) {
		t.Fatal("current Carry changed holdings")
	}
	next.Carry.ItemInstances[0].Price++
	if member.Carry != nil || member.CarriedItems[0].Price != 73 {
		t.Fatal("current Carry aliases input")
	}
}

// A flat pack becomes one cell per equal item; another code and an enchanted
// bow keep their own cells (ITEM-STACK-003, ITEM-MERGE-129).
func TestMaterializePartyCarryFoldsEqualUnits(t *testing.T) {
	bow := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 267}
	cheap := sim.ItemInstance{Code: 0x8114, Kind: 2, Price: 133}
	magic := sim.ItemInstance{Code: 0x8134, Kind: 2, Price: 4276, Effects: []sim.ItemEffect{{Kind: 12, Operand: 5}}}
	member := mapload.PartyMember{ID: "archer", Hero: data.NewHero(data.Spread{Body: 25, Reaction: 27, Mind: 31, Spirit: 23}, 2),
		CarriedItems: []sim.ItemInstance{bow, cheap, magic, bow}}
	next := mapload.MaterializePartyCarry(member, nil)
	want := []sim.ItemStack{sim.StackItem(bow, 2), sim.StackItem(cheap, 1), sim.StackItem(magic, 1)}
	if !reflect.DeepEqual(next.Carry.OrderedStacks, want) {
		t.Fatalf("pack cells %+v, want %+v", next.Carry.OrderedStacks, want)
	}
	if err := mapload.ValidatePartyLoad(next); err != nil {
		t.Fatal(err)
	}
	units := []sim.ItemInstance{bow, bow, cheap, magic}
	if got := mapload.MemberCarriedItems(next, nil); !reflect.DeepEqual(got, units) || !reflect.DeepEqual(next.Carry.Items, []uint16{0x8134, 0x8134, 0x8114, 0x8134}) {
		t.Fatalf("pack units %+v codes %x, want the cells in order", got, next.Carry.Items)
	}
	if len(member.CarriedItems) != 4 || !reflect.DeepEqual(member.CarriedItems[1], cheap) {
		t.Fatal("folding changed the input member")
	}
}
