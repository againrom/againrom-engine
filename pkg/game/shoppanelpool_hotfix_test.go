package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestShopCardPoolMaximumFollowsTheDerivedMaximum is the hotfix's own witness.
// partyPanelSubject filled BOTH halves of the health and mana pairs from the
// spawn pool. A generated member spawns at his maximum, so the two reads were
// indistinguishable; a source-restored character carries his own current
// health, and for him the card stated the current value as the maximum and
// then never moved when an equipped Body item raised that maximum.
//
// The fixture's stored pair is 40/90, so the FIRST assertion alone fails on the
// pre-fix expression, and the pair moves again after the equip.
func TestShopCardPoolMaximumFollowsTheDerivedMaximum(t *testing.T) {
	f, screen, item := sourceCityEquipment(t)
	item.Effects = []sim.ItemEffect{{Kind: 2, Operand: 10}}
	member := screen.shopPartyMember(screen.shopMemberIndex())
	member.Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(item, 2)}
	member.Carry.ItemInstances = []sim.ItemInstance{item, item}
	cityShopGraph(t, f)

	stored := func() (health, healthMax int) {
		t.Helper()
		s := screen.shopPartyMember(screen.shopMemberIndex()).Carry.LiveLoad.Inventory.Source
		return int(int16(s.Stats[8])), int(int16(s.Stats[9]))
	}

	before := screen.townCharacterView().Subject
	health, healthMax := stored()
	if health == healthMax {
		t.Fatalf("the fixture stores %d/%d and cannot witness the pair", health, healthMax)
	}
	if before.HP != health || before.MaxHP != healthMax {
		t.Fatalf("the card states %d/%d, want the stored %d/%d", before.HP, before.MaxHP, health, healthMax)
	}

	if action := screen.shopEquipFromPack(1); action.Msg != "worn" {
		t.Fatalf("shopEquipFromPack = %+v, want \"worn\"", action)
	}

	after := screen.townCharacterView().Subject
	if after.Char.Body <= before.Char.Body {
		t.Fatalf("Body went %d to %d wearing a Body item", before.Char.Body, after.Char.Body)
	}
	health, healthMax = stored()
	if after.HP != health || after.MaxHP != healthMax {
		t.Fatalf("the card states %d/%d after the equip, want the stored %d/%d", after.HP, after.MaxHP, health, healthMax)
	}
	if after.MaxHP == before.MaxHP {
		t.Fatalf("the maximum stayed %d while Body moved %d to %d", after.MaxHP, before.Char.Body, after.Char.Body)
	}
}
