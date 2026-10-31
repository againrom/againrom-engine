package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondBookUnit is the bag's book of spell as a mission holds it.
func secondBookUnit(t *testing.T, spell int) sim.ItemInstance {
	t.Helper()
	u, ok := secondshop.Tables{}.Instance(secondshop.Item{Table: secondshop.Book, Spell: spell, Price: 1000})
	if !ok {
		t.Fatalf("no bag unit for the book of spell %d", spell)
	}
	return secondshop.MissionUnit(u, secondMissionItem(nil))
}

// secondBookMission is a second-game mission whose one member, a mage or a
// fighter with a book in his spellbook, carries items.
func secondBookMission(t *testing.T, mage bool, items ...sim.ItemInstance) *mapWorld {
	t.Helper()
	e := sim.Entity{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, Book: sim.Spellbook{State: sim.BookPresent}}
	if mage {
		e.Mana, e.MaxMana = 20, 20
	}
	script, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	spells := []sim.SpellRule{{ID: 5, ManaCost: 7, School: 1, MaxRange: 9}, {ID: 26, ManaCost: 30, School: 5, MaxRange: 12}}
	w, err := sim.NewStockedSpelledWorld(1051, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{e}, script, sim.Relations{}, nil, []sim.Stock{{ID: 7, ItemInstances: items}}, spells)
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	if !w.SecondGame() {
		t.Fatal("not a second-game World")
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Hero: eqHero(), Mage: mage}},
		Start: mapload.Start{IDs: []sim.EntityID{7}}}
	return openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
}

// In a second-game mission a mage uses a book from his pack, as in the first
// game: the next tick he knows the spell, in its book slot, and the book is
// used up. SAVE's World form and its LOAD keep the spell and the hash.
func TestSecondGameMageLearnsABookInAMission(t *testing.T) {
	mw := secondBookMission(t, true, secondBookUnit(t, 5))
	mw.enqueueEquip(0)
	if want := sim.ReadBook(7, sim.ItemSlot(0)); len(mw.pending) != 1 || mw.pending[0] != want {
		t.Fatalf("pending = %+v, want [%+v]", mw.pending, want)
	}
	mw.tick()
	e := releaseEntity(t, mw, 7)
	if e.KnownSpells != 1<<5 || e.Book.Slots[4] != (sim.BookSpell{Range: 9, ManaCost: 7}) {
		t.Fatalf("known %#x, slot 5 %+v; want spell 5 learned into slot 5", e.KnownSpells, e.Book.Slots[4])
	}
	if items, _ := mw.world.CarriedItems(7); len(items) != 0 {
		t.Fatalf("carried %+v, want the book used up", items)
	}
	raw, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != mw.world.Hash() || !back.SecondGame() {
		t.Fatalf("LOAD hash %#x, SAVE hash %#x, second game %v", back.Hash(), mw.world.Hash(), back.SecondGame())
	}
	for _, got := range back.Entities() {
		if got.ID == 7 && (got.KnownSpells != e.KnownSpells || got.Book != e.Book) {
			t.Fatalf("LOAD known %#x book %+v, want %#x %+v", got.KnownSpells, got.Book, e.KnownSpells, e.Book)
		}
	}
}

// The refusals are the first game's: a fighter, a known spell; and a spell
// the book has no slot for (DIV-4369). Each keeps the book and the reader.
func TestSecondGameMissionBookRefusals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mage  bool
		items []sim.ItemInstance
		known uint32
	}{
		{"fighter", false, []sim.ItemInstance{secondBookUnit(t, 5)}, 0},
		{"spell 29", true, []sim.ItemInstance{secondBookUnit(t, 29)}, 0},
		{"known", true, []sim.ItemInstance{secondBookUnit(t, 26), secondBookUnit(t, 26)}, 1 << 26},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw := secondBookMission(t, tc.mage, tc.items...)
			if tc.known != 0 {
				mw.enqueueEquip(0)
				mw.tick()
			}
			before, _ := mw.world.CarriedItems(7)
			reader := releaseEntity(t, mw, 7)
			if reader.KnownSpells != tc.known {
				t.Fatalf("setup: known %#x, want %#x", reader.KnownSpells, tc.known)
			}
			mw.enqueueEquip(0)
			if len(mw.pending) != 0 {
				t.Fatalf("pending = %+v, want none", mw.pending)
			}
			sim.Step(mw.world, []sim.Command{sim.ReadBook(7, sim.ItemSlot(0))})
			after, _ := mw.world.CarriedItems(7)
			got := releaseEntity(t, mw, 7)
			if !reflect.DeepEqual(after, before) || got.KnownSpells != reader.KnownSpells || got.Book != reader.Book {
				t.Fatalf("a refused read changed carried %+v -> %+v, reader %+v -> %+v", before, after, reader, got)
			}
		})
	}
}

// In a second-game town the shown mage reads a bag book dropped on his doll,
// as in the first game's shop: he knows the spell and the book is used up.
// A known spell and a spell with no slot are refused and change nothing.
func TestSecondGameMageLearnsABookInTheShop(t *testing.T) {
	bagBook := func(spell int) sim.ItemInstance {
		u, _ := secondshop.Tables{}.Instance(secondshop.Item{Table: secondshop.Book, Spell: spell, Price: 1000})
		return u
	}
	doll := ui.ShopControl{Kind: ui.ShopControlDoll}
	s, c := secondDollShop(t, bagBook(29), bagBook(5), bagBook(5))
	s.sess.Carried[0].Mage = true
	before := secondDollNow(s, c)

	s.ShopDrag(secondDollPack(s, 0), doll)
	if got := secondDollNow(s, c); !reflect.DeepEqual(got, before) {
		t.Fatalf("the book of spell 29 was read: known %#x, bag %+v", got.party[0].KnownSpells, got.bag)
	}
	s.ShopDrag(secondDollPack(s, 1), doll)
	after := secondDollNow(s, c)
	if after.party[0].KnownSpells != 1<<5 || len(after.bag) != 2 || after.bag[0].Code != before.bag[0].Code || after.bag[1].Code != before.bag[2].Code {
		t.Fatalf("known %#x, bag %+v; want spell 5 and one book of it used up", after.party[0].KnownSpells, after.bag)
	}
	if after.gold != before.gold || !reflect.DeepEqual(after.table, before.table) || !reflect.DeepEqual(after.shelves, before.shelves) {
		t.Fatal("reading a bag book moved gold, the table or the stock")
	}
	s.ShopDrag(secondDollPack(s, 1), doll)
	if got := secondDollNow(s, c); !reflect.DeepEqual(got, after) {
		t.Fatalf("the book of a known spell was read: bag %+v", got.bag)
	}
	if err := captureSecondCampaign(c).validate(); err != nil {
		t.Fatalf("SAVE refuses the bag after the read: %v", err)
	}
}
