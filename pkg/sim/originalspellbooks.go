package sim

import "fmt"

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
		if !e.Alive() && !originalDyingEntity(e) {
			return fmt.Errorf("original actor spellbooks: entity %d is not living or dying", book.ID)
		}
		if book.Book.State != BookAbsent && book.Book.State != BookPresent {
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
