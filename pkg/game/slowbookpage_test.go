package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/rules"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestSecondGameBookPageListsNoSlow(t *testing.T) {
	var table []sim.SpellRule
	var known uint32
	e := sim.Entity{Book: sim.Spellbook{State: sim.BookPresent}}
	for id := uint16(1); id <= 29; id++ {
		table = append(table, sim.SpellRule{ID: id, ManaCost: 5, MaxRange: 4})
		known |= 1 << id
		*e.Book.Slot(id) = sim.BookSpell{Range: 4, ManaCost: 5}
	}
	e.KnownSpells = known
	rows := rules.SpellBookRows(rules.SpellGameSecond)
	book, fixed := selectedSpellbook(sim.Rules{}, []sim.Entity{e}, table, nil, ui.Words{}, nil, rows)
	if !fixed || len(book) != len(rows) {
		t.Fatalf("fixed %v, %d cells; want %d", fixed, len(book), len(rows))
	}
	for _, c := range book {
		if c.ID == 29 {
			t.Fatalf("the page lists Slow: %+v", c)
		}
	}
	if e.KnownSpells&(1<<29) == 0 || e.Book.Slot(29).ManaCost != 5 {
		t.Fatal("the hero lost Slow from his book state")
	}
	if rules.GameProfile(rules.SpellGameSecond).ShopBook(29) {
		t.Fatal("the second game's shop book list holds Slow")
	}
}

func TestReleaseSecondOriginalBookPageListsNoSlow(t *testing.T) {
	if _, err := os.Stat(slowBookCorpus); err != nil {
		t.Skipf("no save corpus: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(slowBookCorpus, "game0005.sav"))
	if err != nil {
		t.Skipf("no corpus file: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0005.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondGameFront(t)
	app := f.App("second-mission")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	groundAppLoad(t, app, list, "game0005.sav")
	if f.live == nil {
		t.Fatal("the LOAD opened no mission")
	}
	mw := f.live
	for _, e := range mw.world.Entities() {
		if e.Owner != 1 || e.KnownSpells&(1<<29) == 0 || !e.Book.HasInstances() {
			continue
		}
		book, _ := selectedSpellbook(mw.world.Rules(), []sim.Entity{e}, mw.world.Spells(), mw.spellNames, mw.view.Words(), mw.spellIcon, mw.spells().BookRows)
		for _, c := range book {
			if c.ID == 29 {
				t.Fatalf("the page lists Slow: %+v", c)
			}
		}
		return
	}
	t.Fatal("no Player-1 actor holds Slow in a book")
}
