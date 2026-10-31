package sim

import "againrom/pkg/rules"

// bookForm is how one game's item names the spell its book teaches. A form
// with an effect kind is an item of kind whose one effect of that kind names
// the spell. A form with a part is an item ID of that part with the row bit
// set and no material bits, the spell in the row's other bits. slotted
// teaches only a spell the reader's book has a slot for.
type bookForm struct {
	kind, effect uint8
	part, row    uint16
	slotted      bool
}

// The first game's book is the decoded Book object kind 5 with one teach
// effect 42. The second game's is the bag's book ID, part 14 with row bit 7
// (DIV-3043's codes); it is read by the first game's rule on owner direction
// and teaches only spells 1..28 (DIV-4369).
var bookForms = [...]bookForm{
	rules.SpellGameFirst:  {kind: 5, effect: 42},
	rules.SpellGameSecond: {part: 14, row: 0x80, slotted: true},
}

func bookFormOf(game rules.SpellGame) (bookForm, bool) {
	if int(game) >= len(bookForms) {
		return bookForm{}, false
	}
	return bookForms[game], true
}

// IsBook reports whether the item is a book of game's form, readable or not.
func (i ItemInstance) IsBook(game rules.SpellGame) bool {
	f, ok := bookFormOf(game)
	if !ok {
		return false
	}
	if f.effect != 0 {
		return i.Kind == f.kind
	}
	return i.Code>>8&0xf == f.part && i.Code&f.row != 0 && i.Code>>12 == 0
}

// BookSpell returns the one spell a readable book of game's form teaches. A
// book naming no spell, more than one, or one outside the known-spell mask is
// not a readable one-spell book.
func (i ItemInstance) BookSpell(game rules.SpellGame) (uint16, bool) {
	if !i.IsBook(game) {
		return 0, false
	}
	f, _ := bookFormOf(game)
	if f.effect == 0 {
		spell := i.Code & (f.row - 1)
		return spell, spell != 0 && spell < 32
	}
	var spell uint16
	found := false
	for _, e := range i.Effects {
		if e.Kind != f.effect {
			continue
		}
		if found || e.Operand == 0 || e.Operand >= 32 {
			return 0, false
		}
		spell, found = uint16(e.Operand), true
	}
	return spell, found
}

// bookTeaches reports whether a book of game's form may teach spell.
func bookTeaches(game rules.SpellGame, spell uint16) bool {
	f, ok := bookFormOf(game)
	return ok && (!f.slotted || spell >= 1 && int(spell) <= len(Spellbook{}.Slots))
}
