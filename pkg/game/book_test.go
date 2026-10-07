package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func gameSpellBook(spell uint32) sim.ItemInstance {
	return sim.ItemInstance{
		Code:    0x0e17,
		Kind:    5,
		Effects: []sim.ItemEffect{{Kind: 42, Operand: spell}},
	}
}

func bookMission(t *testing.T, mage bool, items ...sim.ItemInstance) *mapWorld {
	t.Helper()
	e := sim.Entity{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10}
	if mage {
		e.Mana, e.MaxMana = 20, 20
	}
	w, err := sim.NewStockedWorld(1051,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{e}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, ItemInstances: items}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{
		Number: 1,
		Map:    m,
		World:  w,
		Party:  []mapload.PartyMember{{Hero: eqHero(), Mage: mage}},
		Start:  mapload.Start{IDs: []sim.EntityID{7}},
	}
	return openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
}

func TestInventoryUseOfAReadableBookQueuesReadAndLearnsOnTheNextTick(t *testing.T) {
	book := gameSpellBook(26)
	mw := bookMission(t, true, book)

	mw.enqueueEquip(0)

	want := sim.Command{Kind: sim.KindReadBook, Entity: 7, X: 0}
	if len(mw.pending) != 1 || mw.pending[0] != want {
		t.Fatalf("pending = %+v, want [%+v]", mw.pending, want)
	}
	if got := releaseEntity(t, mw, 7).KnownSpells; got != 0 {
		t.Fatalf("KnownSpells changed before tick to %#x", got)
	}
	mw.tick()
	if got := releaseEntity(t, mw, 7).KnownSpells; got != uint32(1)<<26 {
		t.Fatalf("KnownSpells after tick = %#x, want Teleport", got)
	}
	if items, _ := mw.world.CarriedItems(7); len(items) != 0 {
		t.Fatalf("carried after tick = %#v, want the book consumed", items)
	}
}

func TestInventoryBookUseRefusesAFighterAndMalformedBook(t *testing.T) {
	valid := gameSpellBook(26)
	bad := gameSpellBook(32)
	for _, tc := range []struct {
		name string
		mage bool
		item sim.ItemInstance
	}{
		{name: "fighter", item: valid},
		{name: "malformed", mage: true, item: bad},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := bookMission(t, tc.mage, tc.item)
			before, _ := mw.world.CarriedItems(7)
			mw.enqueueEquip(0)
			if len(mw.pending) != 0 {
				t.Fatalf("pending = %+v, want none", mw.pending)
			}
			after, _ := mw.world.CarriedItems(7)
			if !reflect.DeepEqual(after, before) || releaseEntity(t, mw, 7).KnownSpells != 0 {
				t.Fatalf("refusal mutated item/spells: before %#v after %#v entity %+v",
					before, after, releaseEntity(t, mw, 7))
			}
		})
	}
}
