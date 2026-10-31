package sim

import (
	"testing"
)

// The two games as a World names them; a sim test imports only the standard
// library.
var (
	bookFirstGame  = (&World{}).SpellGame()
	bookSecondGame = (&World{rom2: &rom2ScriptState{}}).SpellGame()
)

// Each game's book form reads its own items: the first game's a kind-5 item
// with one teach effect, the second game's a book ID. Neither reads the
// other's book.
func TestBookFormsReadEachGamesBooks(t *testing.T) {
	const firstGame, secondGame, unknownGame = 0, 1, 2
	if bookFirstGame == bookSecondGame {
		t.Fatalf("a second-game World names game %d, as the first", bookSecondGame)
	}
	first := ItemInstance{Code: 0x0e17, Kind: 5, Effects: []ItemEffect{{Kind: 42, Operand: 26}}}
	for _, tc := range []struct {
		name  string
		game  int
		item  ItemInstance
		book  bool
		spell uint16
		ok    bool
	}{
		{"first book", firstGame, first, true, 26, true},
		{"first book, no effect", firstGame, ItemInstance{Code: 0x0e17, Kind: 5}, true, 0, false},
		{"first book, spell 32", firstGame, ItemInstance{Kind: 5, Effects: []ItemEffect{{Kind: 42, Operand: 32}}}, true, 0, false},
		{"first book, two effects", firstGame, ItemInstance{Kind: 5, Effects: []ItemEffect{{Kind: 42, Operand: 2}, {Kind: 42, Operand: 3}}}, true, 0, false},
		{"first reads no book ID", firstGame, ItemInstance{Code: 0x0e85, Effects: []ItemEffect{{Kind: 42, Operand: 5}}}, false, 0, false},
		{"second book 1", secondGame, ItemInstance{Code: 0x0e81}, true, 1, true},
		{"second book 26", secondGame, ItemInstance{Code: 0x0e9a, Effects: []ItemEffect{{Kind: 42, Operand: 26}}}, true, 26, true},
		{"second book 29", secondGame, ItemInstance{Code: 0x0e9d}, true, 29, true},
		{"second book, spell 0", secondGame, ItemInstance{Code: 0x0e80}, true, 0, false},
		{"second book, spell 127", secondGame, ItemInstance{Code: 0x0eff}, true, 0, false},
		{"second scroll row", secondGame, ItemInstance{Code: 0x0e03, Kind: 4}, false, 0, false},
		{"second ID with material", secondGame, ItemInstance{Code: 0x1e81}, false, 0, false},
		{"second weapon", secondGame, ItemInstance{Code: 0x2101, Kind: 2}, false, 0, false},
		{"second part 13", secondGame, ItemInstance{Code: 0x0d81}, false, 0, false},
		{"second reads no first book", secondGame, first, false, 0, false},
		{"unknown game", unknownGame, first, false, 0, false},
	} {
		game := bookFirstGame
		switch tc.game {
		case secondGame:
			game = bookSecondGame
		case unknownGame:
			game = bookSecondGame + 8
		}
		spell, ok := tc.item.BookSpell(game)
		if book := tc.item.IsBook(game); book != tc.book || ok != tc.ok || ok && spell != tc.spell {
			t.Errorf("%s: book %v, spell %d %v; want book %v, spell %d %v", tc.name, book, spell, ok, tc.book, tc.spell, tc.ok)
		}
	}
}

// The second game's book teaches only a spell with a book slot (DIV-4369);
// the first game's admission is as before for every spell its reader names.
func TestBookAdmissionRefusesASecondGameSpellWithNoSlot(t *testing.T) {
	r := BookReader{Alive: true, Mage: true, Book: Spellbook{State: BookPresent}}
	for spell := uint16(1); spell < 32; spell++ {
		r.Game = bookFirstGame
		if !r.Admits(spell) {
			t.Errorf("first game refuses spell %d", spell)
		}
		r.Game = bookSecondGame
		if got, want := r.Admits(spell), spell <= 28; got != want {
			t.Errorf("second game admits spell %d: %v, want %v", spell, got, want)
		}
	}
	for _, game := range [...]BookReader{{Game: bookFirstGame}, {Game: bookSecondGame}} {
		game := game.Game
		dead := BookReader{Mage: true, Game: game}
		fighter := BookReader{Alive: true, Game: game}
		knows := BookReader{Alive: true, Mage: true, Book: Spellbook{State: BookPresent}, Known: 1 << 5, Game: game}
		if dead.Admits(5) || fighter.Admits(5) || knows.Admits(5) {
			t.Errorf("game %d admits a dead reader %v, a fighter %v or a known spell %v", game, dead.Admits(5), fighter.Admits(5), knows.Admits(5))
		}
	}
}

// A second-game mage reads a book from his pack: the spell goes into its
// book slot, the book is used up, and the World's byte form keeps both.
func TestSecondGameWorldReadsABookIntoItsSlot(t *testing.T) {
	spells := []SpellRule{{ID: 5, ManaCost: 7, School: 1, MaxRange: 9, Defensive: true}}
	mage := Entity{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, Mana: 20, MaxMana: 20, Book: Spellbook{State: BookPresent}}
	book := ItemInstance{Code: 0x0e85, Price: 1000, Effects: []ItemEffect{{Kind: 42, Operand: 5}}}
	slow := ItemInstance{Code: 0x0e9d, Price: 1000, Effects: []ItemEffect{{Kind: 42, Operand: 29}}}
	s, err := NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{mage}, s,
		Relations{}, nil, []Stock{{ID: 7, ItemInstances: []ItemInstance{slow, book}}}, spells)
	if err != nil {
		t.Fatal(err)
	}
	if !w.SecondGame() {
		t.Fatal("not a second-game World")
	}
	Step(w, []Command{ReadBook(7, ItemSlot(0))})
	if items, _ := w.CarriedItems(7); len(items) != 2 || w.entities[0].KnownSpells != 0 {
		t.Fatalf("the book of spell 29 was read: carried %+v, known %#x", items, w.entities[0].KnownSpells)
	}
	Step(w, []Command{ReadBook(7, ItemSlot(1))})
	e := w.entities[0]
	if e.KnownSpells != 1<<5 || e.Book.Slots[4] != (BookSpell{Range: 9, Defensive: 1, ManaCost: 7}) {
		t.Fatalf("known %#x, slot 5 %+v; want spell 5 in slot 5 with its rule's range, side and cost", e.KnownSpells, e.Book.Slots[4])
	}
	if items, _ := w.CarriedItems(7); len(items) != 1 || items[0].Code != slow.Code {
		t.Fatalf("carried %+v, want the book used up and the book of spell 29 kept", items)
	}
	Step(w, []Command{ReadBook(7, ItemSlot(0))})
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() || back.entities[0].KnownSpells != e.KnownSpells || back.entities[0].Book != e.Book {
		t.Fatalf("after the round trip hash %#x (was %#x), known %#x, book %+v", back.Hash(), w.Hash(), back.entities[0].KnownSpells, back.entities[0].Book)
	}
}
