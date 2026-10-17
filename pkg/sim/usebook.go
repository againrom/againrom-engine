package sim

// BookReader is the part of an actor or party member that reading a one-spell
// book touches: whether it may read, and its spellbook. A World actor and a
// town party member each project into it, so both routes share one admission
// and one transaction.
type BookReader struct {
	Alive bool
	Mage  bool
	Book  Spellbook
	Known uint32
}

// EntityBookReader projects a World actor.
func EntityBookReader(e Entity) BookReader {
	return BookReader{Alive: e.Alive(), Mage: isMage(e), Book: e.Book, Known: e.KnownSpells}
}

func (r BookReader) entity() Entity {
	var e Entity
	e.Book, e.KnownSpells = r.Book, r.Known
	return e
}

// Admits reports whether the reader may use up a book of spell: alive, a mage
// and not yet knowing the spell.
func (r BookReader) Admits(spell uint16) bool {
	return r.Alive && r.Mage && !knowsSpell(r.entity(), uint32(spell))
}

// ReadBookSpell is the one transaction of reading a book: admit, learn the
// spell into r, then run consume with the learned view. A refusal or a failed
// consume leaves r unchanged and reports false.
func ReadBookSpell(r *BookReader, spell uint16, table []SpellRule, consume func(learned BookReader) bool) bool {
	if !r.Admits(spell) {
		return false
	}
	learned := *r
	e := learned.entity()
	LearnBookSpell(&e, spell, table)
	learned.Book, learned.Known = e.Book, e.KnownSpells
	if !consume(learned) {
		return false
	}
	*r = learned
	return true
}

// readBook applies one deterministic use of a carried one-spell book. The
// container index is resolved at Step time, like equip and drop. Every refusal
// is before mutation: the reader must be alive and a mage who does not know
// the spell, the element must exist, and its complete instance must be a
// readable BookSpell.
func (w *World) readBook(i, cindex int) {
	if !w.sourceMutationReady(i) {
		return
	}
	if cindex < 0 || cindex >= len(w.carried[i]) {
		return
	}
	item := w.carried[i][cindex].Instance()
	spell, ok := item.BookSpell()
	if !ok {
		return
	}
	reader := EntityBookReader(w.entities[i])
	if !reader.Admits(spell) {
		return
	}
	live := w
	if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	read := ReadBookSpell(&reader, spell, w.spells, func(learned BookReader) bool {
		oldBook, oldKnown := w.entities[i].Book, w.entities[i].KnownSpells
		w.entities[i].Book, w.entities[i].KnownSpells = learned.Book, learned.Known
		w.refreshSavedBookRoots(i)
		if w.consumeCarriedUnit(i, cindex) {
			return true
		}
		w.entities[i].Book, w.entities[i].KnownSpells = oldBook, oldKnown
		return false
	})
	if read && w != live {
		*live = *w
	}
}
