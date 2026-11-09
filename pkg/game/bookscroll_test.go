package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// bookScrollTable is every spell row the second game's book names, so the
// book is the fixed 24-cell catalog.
func bookScrollTable() []sim.SpellRule {
	var table []sim.SpellRule
	for id := uint16(1); id <= 29; id++ {
		table = append(table, sim.SpellRule{ID: id, TargetsUnit: true})
	}
	return table
}

func bookScrollItem(spell uint16) sim.ItemInstance {
	return sim.ItemInstance{Code: scrollUseOrdinary, Kind: 4, Price: 100, Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(spell) | 50<<16}}}
}

func bookScrollBook(t *testing.T, selected []sim.Entity, scrolls ...sim.BookScroll) []ui.SpellEntry {
	t.Helper()
	book, fixed := selectedSpellbook(sim.Rules{}, selected, bookScrollTable(), nil, ui.Words{}, nil, data.SpellPresentationOf(rules.SpellGameSecond).BookRows, scrolls...)
	if !fixed || len(book) != 24 {
		t.Fatalf("book fixed %v cells %d", fixed, len(book))
	}
	return book
}

// In the second game a carried Heal scroll shows at Heal's cell, the first
// row's fifth (R2-SESSION-229, R2-ENGINE-766), and the entry carries what the
// item cast needs; every other cell stays as it was.
func TestAFighterBookListsACarriedScrollAtItsSpellsCell(t *testing.T) {
	mw := scrollUseMission(t, true, bookScrollItem(24))
	fighter := releaseEntity(t, mw, 7)
	scrolls := mw.bookScrolls([]sim.Entity{fighter})
	if len(scrolls) != 1 || scrolls[0].Spell != 24 || scrolls[0].Index != 0 {
		t.Fatalf("book scrolls %+v", scrolls)
	}
	book := bookScrollBook(t, []sim.Entity{fighter}, scrolls...)
	bare := bookScrollBook(t, []sim.Entity{fighter})
	if e := book[4]; e.ID != 24 || !e.Unavailable || e.ItemCast == nil || e.ItemCast.Index != 0 || e.ItemCast.Key != itemCastKey(bookScrollItem(24)) {
		t.Fatalf("cell 4: %+v item cast %+v", book[4], book[4].ItemCast)
	}
	book[4].ItemCast = nil
	if !reflect.DeepEqual(book, bare) {
		t.Fatal("a cell other than the scroll's changed")
	}
}

// Two scrolls of one spell list the first; a second spell gets its own cell.
func TestABookScrollListsTheFirstOfEachSpell(t *testing.T) {
	second := bookScrollItem(24)
	second.Price = 101
	mw := scrollUseMission(t, true, bookScrollItem(24), bookScrollItem(1), second)
	fighter := releaseEntity(t, mw, 7)
	got := mw.bookScrolls([]sim.Entity{fighter})
	if len(got) != 2 || got[0].Spell != 24 || got[0].Index != 0 || got[0].Item.Price != 100 || got[1].Spell != 1 {
		t.Fatalf("book scrolls %+v", got)
	}
}

// The first game's book lists no scroll: AI-SPELLPOP-287 fills availability
// from known spells alone.
func TestAFirstGameBookListsNoScroll(t *testing.T) {
	mw := scrollUseMission(t, false, bookScrollItem(1))
	fighter := releaseEntity(t, mw, 7)
	if got := mw.bookScrolls([]sim.Entity{fighter}); got != nil {
		t.Fatalf("first-game book scrolls %+v", got)
	}
	for _, e := range bookScrollBook(t, []sim.Entity{fighter}) {
		if e.ItemCast != nil || !e.Unavailable {
			t.Fatalf("first-game cell %+v", e)
		}
	}
}

// A mage with no scroll has the book of today. A known spell keeps its own
// cell with a scroll of it in the pack; a scroll of a spell he does not know
// fills that cell.
func TestABookScrollKeepsAMagesKnownCell(t *testing.T) {
	mage := sim.Entity{ID: 7, Owner: sim.SelfSlot, KnownSpells: 1 << 24}
	bare := bookScrollBook(t, []sim.Entity{mage})
	for i, e := range bare {
		if e.ItemCast != nil || e.Unavailable != (e.ID != 24) {
			t.Fatalf("mage cell %d: %+v", i, e)
		}
	}
	scrolls := []sim.BookScroll{{Spell: 24, Index: 0, Item: bookScrollItem(24)}, {Spell: 1, Index: 1, Item: bookScrollItem(1)}}
	book := bookScrollBook(t, []sim.Entity{mage}, scrolls...)
	if book[4].ItemCast != nil || book[4].Unavailable {
		t.Fatalf("the known spell's cell changed: %+v", book[4])
	}
	if book[0].ItemCast == nil || book[0].ItemCast.Index != 1 || !book[0].Unavailable {
		t.Fatalf("the unknown spell's cell: %+v", book[0])
	}
	book[0].ItemCast = nil
	if !reflect.DeepEqual(book, bare) {
		t.Fatal("a mage's other cells changed")
	}
}

// A group lists no scroll, and neither does a unit the player does not own.
func TestABookScrollNeedsOneOwnedUnit(t *testing.T) {
	mw := scrollUseMission(t, true, bookScrollItem(1))
	fighter := releaseEntity(t, mw, 7)
	enemy := releaseEntity(t, mw, 8)
	if got := mw.bookScrolls([]sim.Entity{fighter, enemy}); got != nil {
		t.Fatalf("a group listed %+v", got)
	}
	if got := mw.bookScrolls([]sim.Entity{enemy}); got != nil {
		t.Fatalf("an enemy listed %+v", got)
	}
}

// The book's identity is the pack's: the entry's cell and key pass the release
// sink, the fighter pays no mana, and the scroll is consumed by its own cast.
func TestABookScrollCastSpendsThePackScroll(t *testing.T) {
	table := scrollUseTable()
	item := mapload.ItemInstanceFromCode(scrollUseOrdinary, table)
	mw := scrollUseMission(t, true, item)
	if !mw.view.SelectUnit(7) {
		t.Fatal("the fighter cannot be selected")
	}
	mw.pushSpellbook()
	fighter := releaseEntity(t, mw, 7)
	book, _ := selectedSpellbook(mw.world.Rules(), []sim.Entity{fighter}, mw.world.Spells(), nil, ui.Words{}, nil, nil, mw.bookScrolls([]sim.Entity{fighter})...)
	if len(book) != 1 || book[0].ItemCast == nil {
		t.Fatalf("book %+v", book)
	}
	mw.useScroll(7, book[0].ItemCast.Index, book[0].ItemCast.Key, 8, 0, 0, false)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScroll {
		t.Fatalf("the release queued %+v", mw.pending)
	}
	for n := 0; n < 120; n++ {
		mw.tick()
		if items, _ := mw.world.CarriedItems(7); len(items) == 0 && len(mw.world.ScrollCasts()) == 0 {
			break
		}
	}
	items, _ := mw.world.CarriedItems(7)
	f, enemy := releaseEntity(t, mw, 7), releaseEntity(t, mw, 8)
	if len(items) != 0 || enemy.HP >= 100 || f.Mana != 0 {
		t.Fatalf("pack %+v enemy health %d fighter mana %d", items, enemy.HP, f.Mana)
	}
}
