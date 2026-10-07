package data

// The wear rule (FR-2a): whether a character of a given class may use a
// given item.
//
// IT IS ONE PREDICATE OVER THREE BITS and nothing else — two bits of the item
// and one bit of the character. No slot takes part, no level, no statistic and
// no material: a mage is refused a breastplate for exactly the same reason he
// is refused a dagger, and a fighter is refused a robe for the mirror of it.
//
// THE ENFORCEMENT IS THE CALLER'S, AND IT IS DELIBERATELY NOT IN pkg/sim.
// The original applies this rule in its window and in no other place: its
// equip command, its equip routines and every callee they reach read neither
// the character's class bit nor the item's column, so a simulation that
// applied the rule would be stricter than the game it reconstructs. This
// file therefore answers a question; it moves nothing.

// SutableForColumn is the item's own Data.bin parameter index the two bits are
// read from — a RUNTIME COLUMN NUMBER, i.e. the parameter array's own index,
// so it is the column the file's title array names one to its right. The
// shipped title there is `sutableFor`, measured identical on both preserved
// roots.
//
// pkg/mapload/spawn.go SPELLS THE SAME NUMBER SEPARATELY, as
// weaponCarrySlot, and the two are not shared. That constant reads this
// column for a different question — whether a dead body leaves its weapon
// behind — and the two readings were established by different evidence.
// Joining them behind one exported name would make either one impossible to
// narrow without moving the other.
const SutableForColumn = 0xf

// Suitability is one item's two sutableFor bits, read apart.
//
// BOTH CLEAR IS A REAL STATE and not an absence: an item saying neither is an
// item nobody may use, which is what the three shipped emplaced weapons say.
// Both set is the other real state, and it is what a ring, an amulet, bare
// hands and the plasma sword say.
type Suitability struct {
	// Fighter is bit 0 of the column: usable by a character whose mage flag
	// is clear.
	Fighter bool

	// Mage is bit 1 of the column: usable by a character whose mage flag is
	// set.
	Mage bool
}

// Allows reports whether a character may use this item, mage being that
// character's own class flag.
//
// This is the original's own 13-instruction predicate written out: the item's
// fighter bit against a character who is not a mage, or the item's mage bit
// against one who is.
func (s Suitability) Allows(mage bool) bool {
	if mage {
		return s.Mage
	}
	return s.Fighter
}

// SuitabilityFromParams reads the two bits off one collection row's parameter
// array. ok is false only for a nil array, which is a row that was never read;
// see SuitabilityFromCode for what a caller does with that.
//
// A ROW SHORTER THAN THE COLUMN ANSWERS THE ZERO Suitability, USABLE BY
// NEITHER, and answers it with ok true. The row has been read and its
// sixteenth cell is not there to say anything, which in the original leaves
// the item's descriptor bits at the zero its producer just cleared them to.
// A short row is a statement; a missing row is not.
//
// THE VALUE IS BIT TESTED AS IT STANDS, NEGATIVES INCLUDED. The original's
// three producers clear the descriptor byte and then copy bit 0 and bit 1 of
// the cell independently, with no ordering between them and no range check,
// so a cell of -1 sets both bits and reads as usable by both. One shipped
// row holds -1.
func SuitabilityFromParams(p []int32) (Suitability, bool) {
	if p == nil {
		return Suitability{}, false
	}
	if len(p) <= SutableForColumn {
		return Suitability{}, true
	}
	v := p[SutableForColumn]
	return Suitability{Fighter: v&1 != 0, Mage: v&2 != 0}, true
}

// SuitabilityFromCode reads one item code's two bits, resolving the code to a
// row of whichever of the three equipment collections its class names: field B
// is weaponItemClass for a weapon, shieldItemClass for a shield and one of the
// twelve equipment slots for an armour, and field D is the row index in every
// one of the three.
//
// ok IS FALSE FOR A CODE THIS TREE CANNOT REACH A ROW FOR, and a caller must
// read that as "no answer" rather than as a refusal. Three cases reach it: a
// field B naming none of the three collections — item class 14, the
// carried class, among them — a nil collection, and a row index outside
// the collection. Refusing on any of those would turn a gap in this build's
// item resolution into a silent equip refusal, which is a rule the original
// does not have.
//
// IT DOES NOT GO THROUGH WeaponFromCode OR ArmorFromCode. Those two resolve a
// whole item — scaling its damage by shape and material, refusing a row too
// short to carry the columns they read — and this function needs one cell of one
// row. Reaching for them would make the answer depend on two scale tables that
// have nothing to do with who may wear the thing, and would refuse a shield
// outright, since no resolver in this tree answers one from a bare code.
func SuitabilityFromCode(c ItemCode, weapons, shields, armors Collection) (Suitability, bool) {
	var coll Collection
	switch b := c.B(); {
	case b == weaponItemClass:
		coll = weapons
	case b == shieldItemClass:
		coll = shields
	case validEquipSlot(b):
		coll = armors
	default:
		return Suitability{}, false
	}
	if coll == nil {
		return Suitability{}, false
	}
	row := c.D()
	if row < 0 || row >= coll.Len() {
		return Suitability{}, false
	}
	return SuitabilityFromParams(coll.EntryParams(row))
}

// AllowsItem is the whole rule at one call: whether a character with the given
// mage flag may use the item c names. unknown reports that the item's row could
// not be read, and a caller that enforces the rule must not refuse on it.
func AllowsItem(c ItemCode, mage bool, weapons, shields, armors Collection) (allowed, known bool) {
	s, ok := SuitabilityFromCode(c, weapons, shields, armors)
	if !ok {
		return true, false
	}
	return s.Allows(mage), true
}
