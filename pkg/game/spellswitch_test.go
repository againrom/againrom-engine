package game

import (
	"bytes"
	"math/rand"
	"os"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// spellSwitchData is the spell-switch fixture mod's spells file: Fire Ball
// off, Wall of Fire at position 1, Fire Arrow at 3, 23 positions.
func spellSwitchData(t *testing.T) mod.SpellData {
	t.Helper()
	src, err := os.ReadFile("../modrt/testdata/mods/spell-switch/data/spells.toml")
	if err != nil {
		t.Fatal(err)
	}
	d, err := mod.ParseSpells("spell-switch", mod.SpellsFile, src, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func bookIDs(book []ui.SpellEntry) []uint32 {
	out := make([]uint32, len(book))
	for i, e := range book {
		out[i] = e.ID
	}
	return out
}

func TestSpellSwitchBookRowsOnBothGames(t *testing.T) {
	table := bookScrollTable()
	e := sim.Entity{Mind: 30, Book: sim.Spellbook{State: sim.BookPresent}}
	for id := uint16(1); id <= 28; id++ {
		sim.LearnBookSpell(&e, id, table)
	}
	for _, spells := range []string{"rom1", "rom2"} {
		tab := &mapload.Table{Edition: &base.Edition{Spells: spells}}
		ground := spellBookRows(tab)
		if rows := spellBookOrder(tab, ground); !slices.Equal(rows, ground) {
			t.Fatalf("%s: without a mod the book leaves the game's own rows", spells)
		}
		plain, fixed := selectedSpellbook(sim.Rules{}, []sim.Entity{e}, table, nil, ui.Words{}, nil, ground)
		markMovedSpells(plain, ground, ground)
		if !fixed || len(plain) != 24 || slices.ContainsFunc(plain, func(s ui.SpellEntry) bool { return s.Moved }) {
			t.Fatalf("%s: plain book %v", spells, bookIDs(plain))
		}
		tab.Mods.Spells = spellSwitchData(t)
		rows := spellBookOrder(tab, ground)
		book, fixed := selectedSpellbook(sim.Rules{}, []sim.Entity{e}, table, nil, ui.Words{}, nil, rows)
		markMovedSpells(book, rows, ground)
		want := []uint32{3, 0, 1}
		for _, id := range ground[3:23] {
			want = append(want, uint32(id))
		}
		if !fixed || !slices.Equal(bookIDs(book), want) {
			t.Errorf("%s: book %v, want %v", spells, bookIDs(book), want)
		}
		for i, s := range book {
			if s.Moved != (i == 0 || i == 2) {
				t.Errorf("%s: cell %d spell %d moved %v", spells, i, s.ID, s.Moved)
			}
		}
	}
}

// scrollSwitchTable holds a priced Fire scroll row for every first-game spell.
func scrollSwitchTable() *mapload.Table {
	spells := make(dbCollection, 29)
	for i := 1; i < len(spells); i++ {
		p := make([]int32, 22)
		p[2], p[20] = 1, 10
		spells[i] = dbEntry{name: rules.SpellProfile(rules.SpellGameFirst)[i-1].Name, params: p}
	}
	items := make(dbCollection, 2)
	items[1].name = "Scroll_Fire"
	return &mapload.Table{Spells: spells, MagicItems: items}
}

func TestSpellSwitchShopStocksNoBookOrScrollOfAnOffSpell(t *testing.T) {
	bookSpells := func(tab *mapload.Table) []uint16 {
		var out []uint16
		for _, it := range shopBookPool(tab, 1000) {
			s, _ := it.Instance().BookSpell(rules.SpellGameFirst)
			out = append(out, s)
		}
		return out
	}
	scrollSpells := func(tab *mapload.Table) []uint16 {
		var out []uint16
		for _, it := range shopScrollPool(tab, 1000, rand.New(rand.NewSource(1))) {
			out = append(out, uint16(it.Instance().Effects[0].Operand))
		}
		return out
	}
	books := shopBookFixtureTable()
	plainBooks := bookSpells(books)
	books.Mods.Spells = spellSwitchData(t)
	if got, want := bookSpells(books), slices.DeleteFunc(slices.Clone(plainBooks), func(s uint16) bool { return s == 2 }); !slices.Contains(plainBooks, 2) || !slices.Equal(got, want) {
		t.Errorf("books %v under the mod, %v without", got, plainBooks)
	}
	scrolls := scrollSwitchTable()
	plainScrolls := scrollSpells(scrolls)
	scrolls.Mods.Spells = spellSwitchData(t)
	// Fire Ball is off; Shield (18) held position 24, past the 23 positions.
	want := slices.DeleteFunc(slices.Clone(plainScrolls), func(s uint16) bool { return s == 2 || s == 18 })
	if got := scrollSpells(scrolls); len(plainScrolls) != 27 || !slices.Equal(got, want) {
		t.Errorf("scrolls %v under the mod, %v without", got, plainScrolls)
	}
}

func TestSpellSwitchRefusalsNameTheRowAndTheLimit(t *testing.T) {
	f := modSpellFront()
	d := spellEdits(mod.SpellRow{Target: "Fire Arrow", BookPosition: mod.SpellInt{Set: true, Val: 24, Line: 9}})
	d.Global.BookSize, d.Global.BookSizeMod, d.Global.BookSizeFile = mod.SpellInt{Set: true, Val: 23, Line: 2}, "m", mod.SpellsFile
	if err := f.SetModSpells(d); err == nil || err.Error() != `mod "m": data/spells.toml:9: book_position of Fire Arrow is 24, outside 1..23` {
		t.Errorf("position: %v", err)
	}
	f = modSpellFront()
	d = mod.SpellData{Global: mod.SpellGlobal{BookSize: mod.SpellInt{Set: true, Val: 30, Line: 2}, BookSizeMod: "big", BookSizeFile: mod.SpellsFile}}
	if err := f.SetModSpells(d); err == nil || err.Error() != `mod "big": data/spells.toml:2: book_size is 30, outside 1..24` {
		t.Errorf("size: %v", err)
	}
	if !f.Table.Mods.Spells.Empty() {
		t.Error("a refused mod reached the table")
	}
}

// A spell the mage knew before the mod is hidden, not erased: the save
// written under the mod is the save written without it, and loaded without
// the mod its book shows the spell again. The fixture mage knows Fire Arrow;
// its table is filled to the first game's 28 rows so the book is the drawn one.
func TestSpellSwitchHidesAKnownSpellAndTheSaveKeepsIt(t *testing.T) {
	f, load := cityBookFixture(t, 99, false, false, false)
	rows := f.Table.Spells.(dbCollection)
	for id := len(rows); id <= 28; id++ {
		rows = append(rows, dbEntry{name: rules.SpellProfile(rules.SpellGameFirst)[id-1].Name, params: rows[1].params})
	}
	f.Table.Spells = rows
	known := func(f *FrontEnd) []uint32 {
		s := f.TownScreen().(*townScreen)
		s.room, s.shopBook = roomShop, true
		book := s.ShopScreen().Spells
		if len(book) != 24 {
			t.Fatalf("the town book has %d cells", len(book))
		}
		var out []uint32
		for _, e := range book {
			if e.ID != 0 && !e.Unavailable {
				out = append(out, e.ID)
			}
		}
		return out
	}
	control := cityProjectionSave(t, f)
	plain := known(f)
	if !slices.Contains(plain, 1) {
		t.Fatalf("the fixture mage does not show Fire Arrow: %v", plain)
	}
	f.Table.Mods.Spells = mod.SpellData{Rows: []mod.SpellRow{{Mod: "m", File: mod.SpellsFile, Line: 1, Target: "Fire_Arrow", Off: mod.SpellBool{Set: true, Val: true, Line: 2}}}}
	if hidden := known(f); slices.Contains(hidden, 1) {
		t.Fatalf("the book under the mod shows Fire Arrow: %v", hidden)
	}
	saved := cityProjectionSave(t, f)
	if !bytes.Equal(saved, control) {
		t.Fatal("the save written under the mod differs from the save written without it")
	}
	f.Table.Mods.Spells = mod.SpellData{}
	if got := known(load(saved)); !slices.Equal(got, plain) {
		t.Fatalf("loaded without the mod the book shows %v, want %v", got, plain)
	}
}

// An autocast set before the mod turned its spell off stops, and the unit
// keeps the spell; with the spell on, or with no mod, the setting stands.
func TestSpellSwitchStopsAnAutocastOfAnOffSpell(t *testing.T) {
	for name, rows := range map[string][]uint16{"no mod": nil, "on": {3, 1, 2}, "off": {3, 2}} {
		mw := scrollUseMission(t, true)
		mw.pending = append(mw.pending, sim.Autocast(7, 1))
		mw.tick()
		if got := releaseEntity(t, mw, 7).AutoSpell; got != 1 {
			t.Fatalf("%s: setup: autocast is %d", name, got)
		}
		mw.bookRows = rows
		mw.pushSpellbook()
		mw.tick()
		got := releaseEntity(t, mw, 7).AutoSpell
		if want := uint16(1); name == "off" && got != 0 || name != "off" && got != want {
			t.Errorf("%s: autocast is %d after the mod", name, got)
		}
	}
}
