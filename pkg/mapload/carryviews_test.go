package mapload

import (
	"testing"

	"againrom/pkg/sim"
)

func TestCarryViewsFollowCurrentOwnershipAcrossBoundary(t *testing.T) {
	item := sim.PlainItem(0xe01)
	item.ObjectID = 91
	weapon := sim.PlainItem(0x123)
	weapon.ObjectID = 93
	var stocks []sim.Stock
	for _, id := range []sim.EntityID{7, 9} {
		stock := sim.Stock{ID: id, ItemInstances: []sim.ItemInstance{item}}
		stock.EquippedItems[0] = weapon
		stocks = append(stocks, stock)
	}
	w, err := sim.NewStockedWorld(7, sim.Bounds{Width: 5, Height: 5}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, HP: 40, MaxHP: 40, X: 1, Y: 1}, {ID: 9, Owner: sim.SelfSlot, TypeID: sim.HumanTypeID, HP: 40, MaxHP: 40, X: 3, Y: 3}},
		nil, sim.Relations{}, nil, stocks)
	if err != nil {
		t.Fatal(err)
	}
	party := []PartyMember{{ID: "entry", Worn: [sim.EquipSlots]uint16{0x456}, Carried: []uint16{0xe02}}}
	roster := map[sim.EntityID]PartyMember{9: {ID: "joiner", Worn: [sim.EquipSlots]uint16{0x456}, Carried: []uint16{0xe02}}}
	within, ids := CarryRosterIDs(party, w, []sim.EntityID{7}, roster)
	if len(within) != 2 || len(ids) != 2 || ids[0] != 7 || ids[1] != 9 {
		t.Fatal("carry membership or order changed", ids)
	}
	for _, p := range within {
		if p.Worn[0] != weapon.Code || p.WornItems[0].ObjectID != 93 || len(p.Carried) != 1 || p.Carried[0] != item.Code || p.CarriedItems[0].ObjectID != 91 {
			t.Fatal("entry or joiner retained obsolete holdings", p)
		}
	}
	within[0].Carried[0]++
	within[0].CarriedItems[0].Code++
	if within[0].Carry.Items[0] != item.Code || within[0].Carry.ItemInstances[0].Code != item.Code || party[0].Carried[0] != 0xe02 {
		t.Fatal("compatibility view aliases current ownership or the entry template")
	}
	for _, p := range CarryRoster(party, w, []sim.EntityID{7}, roster) {
		if p.CarriedItems[0].ObjectID != 0 || p.WornItems[0].ObjectID != 0 || p.Carry.ItemInstances[0].ObjectID != 0 || p.Carry.EquippedItems[0].ObjectID != 0 {
			t.Fatal("a local object handle escaped into the next mission")
		}
	}
	items, _ := w.CarriedItems(7)
	if items[0].ObjectID != 91 || items[0].Code != item.Code {
		t.Fatal("carry mutated the running World's ownership")
	}
}
