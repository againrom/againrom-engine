package sim

import (
	"encoding/json"
	"fmt"

	"againrom/pkg/rules"
)

// BookState distinguishes old table-backed native actors from a source book.
// No producer infers a first book from mana or from learned membership.
type BookState uint8

const (
	BookLegacy BookState = iota
	BookAbsent
	BookPresent
	// These native modes keep membership that has no ordinary Spell slot.
	// Their low slots retain ordinary present/absent semantics independently.
	BookNativePresent
	BookNativeAbsent
)

// BookSpell contains exactly the persistent instance parameters. Damage and
// duration are scratch derived at application, not serialized Spell fields.
type BookSpell struct {
	Range, Defensive uint8
	ManaCost         uint16
}

// BookIDs is the highest spell id a book holds. Ids 1..len(Slots) have a slot
// in the entity record; the ids past them live in Beyond and ride in their own
// section of the World form, so a World without one keeps its bytes.
const BookIDs = BookWireSlots + 1

// BookWireSlots is the number of book slots the first game's SAV format and
// the entity record hold; every reader of that format bounds by it.
const BookWireSlots = 28

// bookIDMask is the known-spell bits of the ids a book holds.
const bookIDMask = uint32(1)<<(BookIDs+1) - 2

type Spellbook struct {
	State  BookState
	Slots  [BookWireSlots]BookSpell // index is spell ID minus one; KnownSpells owns membership
	Beyond [BookIDs - BookWireSlots]BookSpell
}

// Slot is the instance of spell id in the book, nil for an id no book holds.
// It is the one place that turns an id into book storage.
func (b *Spellbook) Slot(id uint16) *BookSpell {
	switch {
	case id >= 1 && int(id) <= len(b.Slots):
		return &b.Slots[id-1]
	case int(id) > len(b.Slots) && int(id) <= BookIDs:
		return &b.Beyond[int(id)-len(b.Slots)-1]
	}
	return nil
}

// BookSpellCount is the highest spell number a book of game g holds: the
// game's spell count, at most BookIDs.
func BookSpellCount(g rules.SpellGame) int {
	return min(len(rules.SpellProfile(g)), BookIDs)
}

func (b Spellbook) HasInstances() bool {
	return b.State == BookPresent || b.State == BookNativePresent
}

func (b Spellbook) WirePresent(known uint32) bool {
	return b.HasInstances() || b.State == BookLegacy && known != 0
}

func (b Spellbook) hasNativeSpells() bool {
	return b.State == BookNativePresent || b.State == BookNativeAbsent
}

func (b Spellbook) Validate(known uint32) error {
	if b.State > BookNativeAbsent {
		return fmt.Errorf("invalid spellbook state %d", b.State)
	}
	if b.State == BookLegacy {
		if b.Slots != ([BookWireSlots]BookSpell{}) || b.Beyond != ([BookIDs - BookWireSlots]BookSpell{}) {
			return fmt.Errorf("legacy spellbook has instance parameters")
		}
		return nil
	}
	if b.hasNativeSpells() != (known & ^bookIDMask != 0) || !b.HasInstances() && known&bookIDMask != 0 {
		return fmt.Errorf("invalid source spellbook membership %#x", known)
	}
	for id := uint16(1); id <= BookIDs; id++ {
		if known&(uint32(1)<<id) == 0 && *b.Slot(id) != (BookSpell{}) {
			return fmt.Errorf("unlearned spell %d has parameters", id)
		}
	}
	return nil
}

// BookRuleFor resolves only a book source. The table remains immutable and
// item/weapon/script callers continue to use it directly.
func BookRuleFor(e Entity, rule SpellRule) (SpellRule, bool) {
	if !knowsSpell(e, uint32(rule.ID)) {
		return SpellRule{}, false
	}
	slot := e.Book.Slot(rule.ID)
	if e.Book.State == BookLegacy || e.Book.hasNativeSpells() && slot == nil {
		return rule, true
	}
	if !e.Book.HasInstances() || slot == nil {
		return SpellRule{}, false
	}
	// A spell past the entity record's slots that holds no parameters is a
	// book that predates them: it keeps the table row's values.
	if int(rule.ID) > len(e.Book.Slots) && *slot == (BookSpell{}) {
		return rule, true
	}
	s := *slot
	rule.MaxRange, rule.ManaCost = s.Range, int32(int16(s.ManaCost))
	rule.Defensive = s.Defensive != 0
	rule.bookInstance, rule.bookDefensive = true, s.Defensive
	return rule, true
}

func (w *World) bookSpell(e Entity, id uint32) (SpellRule, bool) {
	rule, ok := w.findSpell(id)
	if !ok {
		return SpellRule{}, false
	}
	return BookRuleFor(e, rule)
}

func bookAffords(e Entity, rule SpellRule) bool {
	if !isMage(e) && bookCaster(e) {
		return true
	}
	mana := e.Mana
	if rule.bookInstance {
		mana = int32(int16(mana))
	}
	return int64(rule.ManaCost) <= int64(mana)
}

func debitBook(e *Entity, rule SpellRule) {
	if !isMage(*e) && bookCaster(*e) {
		return
	}
	if rule.bookInstance {
		e.setCurrentMana(int32(int16(uint16(e.Mana) - uint16(rule.ManaCost))))
	} else {
		e.setCurrentMana(e.Mana - rule.ManaCost)
	}
}

// LearnBookSpell is the no-book / already-known gate of teachSpell. A new
// Spell receives constructor defaults, not another actor's cached parameters.
func LearnBookSpell(e *Entity, id uint16, table []SpellRule) {
	slot := e.Book.Slot(id)
	if e.Book.State == BookLegacy || e.Book.hasNativeSpells() && slot == nil {
		if id < 32 {
			e.KnownSpells |= uint32(1) << id
		}
		return
	}
	if !e.Book.HasInstances() || slot == nil || knowsSpell(*e, uint32(id)) {
		return
	}
	for _, rule := range table {
		if rule.ID == id {
			defensive := uint8(0)
			if rule.Defensive {
				defensive = 1
			}
			*slot = BookSpell{rule.MaxRange, defensive, uint16(rule.ManaCost)}
			e.KnownSpells |= uint32(1) << id
			return
		}
	}
}

// RefreshBook is the derive's existing-book walk. Only cached range is a
// persistent output: mana/Defensive retain their saved instance values.
func RefreshBook(r Rules, e *Entity, table []SpellRule) {
	if !e.Book.HasInstances() {
		return
	}
	for _, rule := range table {
		slot := e.Book.Slot(rule.ID)
		if slot == nil || !knowsSpell(*e, uint32(rule.ID)) {
			continue
		}
		var level int32
		if rule.School < skillSlots {
			level = e.Skill[rule.School]
		}
		slot.Range = uint8(spellRangeOf(r, rule, spellPowerUnder(r, rule, level, e.Mind), rangeCast))
	}
}

// RefreshOriginalBook projects the signed original range producer from ordinary
// level and Mind words. It keeps mana and Defensive instance operands.
func RefreshOriginalBook(e *Entity, table []SpellRule) {
	if !e.Book.HasInstances() {
		return
	}
	for _, rule := range table {
		slot := e.Book.Slot(rule.ID)
		if slot == nil || !knowsSpell(*e, uint32(rule.ID)) {
			continue
		}
		var level int32
		if rule.School < skillSlots {
			level = e.Skill[rule.School]
		}
		slot.Range = originalSpellRange(rule, level, e.Mind)
	}
}

func originalSpellRange(rule SpellRule, level, mind int32) uint8 {
	level, mind = int32(int16(level)), int32(int16(mind))
	if rule.School > 0 && rule.School < skillSlots {
		level = min(100, level)
	}
	power := min(int32(100), max(int32(0), level+mind-30))
	return uint8(spellRangeOf(Rules{}, rule, power, rangeCast))
}

// RefreshPlayerBooks runs the book walk on every actor the player owns, as the
// resume after LOAD does before the first cast (MAGIC-420): a slot keeps the
// range its SAV stored only until then. It reports how many books it walked.
func (w *World) RefreshPlayerBooks(player uint32) int {
	walked := 0
	for i := range w.entities {
		if w.entities[i].Owner != player || !w.entities[i].Book.HasInstances() {
			continue
		}
		RefreshBook(w.rules, &w.entities[i], w.spells)
		w.refreshSavedBookRoots(i)
		walked++
	}
	return walked
}

func (w *World) refreshAppliedBook(ci int, id uint32) {
	if !w.entities[ci].Book.HasInstances() {
		return
	}
	if rule, ok := w.findSpell(id); ok && id >= 1 && int(id) <= BookSpellCount(w.SpellGame()) {
		RefreshBook(w.rules, &w.entities[ci], []SpellRule{rule})
		w.refreshSavedBookRoots(ci)
	}
}

// MarshalJSON writes the beyond-slot spells only when one is held, so a book
// of the first 28 spells keeps the form its records have always had.
func (b Spellbook) MarshalJSON() ([]byte, error) {
	out := struct {
		State  BookState
		Slots  [BookWireSlots]BookSpell
		Beyond *[BookIDs - BookWireSlots]BookSpell `json:",omitempty"`
	}{State: b.State, Slots: b.Slots}
	if b.Beyond != ([BookIDs - BookWireSlots]BookSpell{}) {
		out.Beyond = &b.Beyond
	}
	return json.Marshal(out)
}
