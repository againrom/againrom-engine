package sim

import "fmt"

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

type Spellbook struct {
	State BookState
	Slots [28]BookSpell // index is spell ID minus one; KnownSpells owns membership
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
		if b.Slots != ([28]BookSpell{}) {
			return fmt.Errorf("legacy spellbook has instance parameters")
		}
		return nil
	}
	if b.hasNativeSpells() != (known & ^uint32(0x1ffffffe) != 0) || !b.HasInstances() && known&0x1ffffffe != 0 {
		return fmt.Errorf("invalid source spellbook membership %#x", known)
	}
	for i, slot := range b.Slots {
		if known&(uint32(1)<<uint(i+1)) == 0 && slot != (BookSpell{}) {
			return fmt.Errorf("unlearned spell %d has parameters", i+1)
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
	if e.Book.State == BookLegacy || e.Book.hasNativeSpells() && (rule.ID < 1 || rule.ID > 28) {
		return rule, true
	}
	if !e.Book.HasInstances() || rule.ID < 1 || rule.ID > 28 {
		return SpellRule{}, false
	}
	s := e.Book.Slots[rule.ID-1]
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
	if e.Book.State == BookLegacy || e.Book.hasNativeSpells() && (id < 1 || id > 28) {
		if id < 32 {
			e.KnownSpells |= uint32(1) << id
		}
		return
	}
	if !e.Book.HasInstances() || id < 1 || id > 28 || knowsSpell(*e, uint32(id)) {
		return
	}
	for _, rule := range table {
		if rule.ID == id {
			defensive := uint8(0)
			if rule.Defensive {
				defensive = 1
			}
			e.Book.Slots[id-1] = BookSpell{rule.MaxRange, defensive, uint16(rule.ManaCost)}
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
		if rule.ID < 1 || rule.ID > 28 || !knowsSpell(*e, uint32(rule.ID)) {
			continue
		}
		var level int32
		if rule.School < skillSlots {
			level = e.Skill[rule.School]
		}
		e.Book.Slots[rule.ID-1].Range = uint8(spellRangeUnder(r, rule, spellPowerUnder(r, rule, level, e.Mind)))
	}
}

// RefreshOriginalBook projects the signed original range producer from ordinary
// level and Mind words. It keeps mana and Defensive instance operands.
func RefreshOriginalBook(e *Entity, table []SpellRule) {
	if !e.Book.HasInstances() {
		return
	}
	for _, rule := range table {
		if rule.ID < 1 || rule.ID > 28 || !knowsSpell(*e, uint32(rule.ID)) {
			continue
		}
		var level int32
		if rule.School < skillSlots {
			level = e.Skill[rule.School]
		}
		e.Book.Slots[rule.ID-1].Range = originalSpellRange(rule, level, e.Mind)
	}
}

func originalSpellRange(rule SpellRule, level, mind int32) uint8 {
	level, mind = int32(int16(level)), int32(int16(mind))
	if rule.School > 0 && rule.School < skillSlots {
		level = min(100, level)
	}
	power := min(int32(100), max(int32(0), level+mind-30))
	return uint8(spellRange(rule, power))
}

func (w *World) refreshAppliedBook(ci int, id uint32) {
	if !w.entities[ci].Book.HasInstances() {
		return
	}
	if rule, ok := w.findSpell(id); ok && id >= 1 && id <= 28 {
		RefreshBook(w.rules, &w.entities[ci], []SpellRule{rule})
		w.refreshSavedBookRoots(ci)
	}
}
