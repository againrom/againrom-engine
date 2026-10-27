package sim

import (
	"fmt"
	"slices"
)

// OriginalActorSpellbook changes only the persisted book and membership.
// The importer resolves identity; it must not refresh source parameters.
type OriginalActorSpellbook struct {
	ID          EntityID
	KnownSpells uint32
	Book        Spellbook
	// CreatureSpells replaces the actor's slots when HasCreatureSpells. An
	// all-zero source over an actor constructed with class slots keeps the
	// class value: no original record of such a class holds a zero block.
	CreatureSpells    [CreatureSpellSlots]CreatureSpell
	HasCreatureSpells bool
}

// spellsPastBook reports that known names a spell past the book's slots and
// that each such spell is a row of the world's spell table. That is the one
// native membership an original book holds: the second game's spell 29
// (DIV-4174).
func (w *World) spellsPastBook(known uint32) bool {
	if known&1 != 0 {
		return false
	}
	found := false
	for id := len(Spellbook{}.Slots) + 1; id < SpellIDLimit; id++ {
		if known&(1<<id) == 0 {
			continue
		}
		if !w.admitsSpellID(id) {
			return false
		}
		found = true
	}
	return found
}

// admitsSpellID reports whether id can stand as the spell of an actor's book,
// its admitted book selection or an effect it carries: 0 (none), a slot of the
// book, or a spell past the slots that is a row of the world's spell table
// (DIV-4174).
func (w *World) admitsSpellID(id int) bool {
	if id >= 0 && id <= len(Spellbook{}.Slots) {
		return true
	}
	return id > 0 && id < SpellIDLimit && slices.ContainsFunc(w.spells, func(r SpellRule) bool { return int(r.ID) == id })
}

// SpellIDLimit bounds every spell id: an actor's known spells are a 32-bit
// mask. A reader of a saved id checks only this bound and leaves the World to
// decide a spell past the book's slots by its table.
const SpellIDLimit = 32

func (w *World) ImportOriginalActorSpellbooks(books []OriginalActorSpellbook) error {
	if w == nil {
		return fmt.Errorf("original actor spellbooks: no world")
	}
	indices := make([]int, len(books))
	seen := make(map[EntityID]bool, len(books))
	for i, book := range books {
		if seen[book.ID] {
			return fmt.Errorf("original actor spellbooks: repeated entity %d", book.ID)
		}
		seen[book.ID] = true
		index := indexOfEntity(w.entities, book.ID)
		if index < 0 {
			return fmt.Errorf("original actor spellbooks: missing entity %d", book.ID)
		}
		e := w.entities[index]
		if !e.Alive() && !originalDyingEntity(e) && !w.originalCorpseEntity(book.ID) {
			return fmt.Errorf("original actor spellbooks: entity %d is not living, dying or a loaded corpse", book.ID)
		}
		switch book.Book.State {
		case BookAbsent, BookPresent:
		case BookNativeAbsent, BookNativePresent:
			if !w.spellsPastBook(book.KnownSpells) {
				return fmt.Errorf("original actor spellbooks: entity %d has a native book mode without a table spell past the book", book.ID)
			}
		default:
			return fmt.Errorf("original actor spellbooks: entity %d has no source presence", book.ID)
		}
		if err := book.Book.Validate(book.KnownSpells); err != nil {
			return fmt.Errorf("original actor spellbooks: entity %d: %w", book.ID, err)
		}
		indices[i] = index
	}
	for i, book := range books {
		e := &w.entities[indices[i]]
		e.KnownSpells, e.Book = book.KnownSpells, book.Book
		if book.HasCreatureSpells && (book.CreatureSpells != [CreatureSpellSlots]CreatureSpell{} || !e.hasCreatureSpells()) {
			e.CreatureSpells = book.CreatureSpells
		}
	}
	return nil
}
