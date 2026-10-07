package data

// ItemNames is the shipped item-name table (research ITEM-DISPNAME-036,
// ITEM-NAMEKEY-037, ITEM-NAMEPOP-038): a code's raw sixteen bits are the
// lookup key, not an index a bare-code recovery like WeaponFromCode's or
// ArmorFromCode's has to reassemble a name from. THE NAME IS STORED, NOT
// COMPOSED — research establishes this positively: the key for shape
// Elven, material Adamantium, Armors row Amulet reads "Adamantium Amulet"
// on the EN root, the tier word dropped rather than joined on, and the
// same fields at a different material read "Beard" or "Magic Beard". This
// type therefore adds no arithmetic of its own: a lookup either finds the
// line the install shipped or it does not.
//
// MOST CODES THE ENCODING CAN EXPRESS CARRY NO LINE (ITEM-NAMEPOP-038: 416
// of 6 064 reachable keys). A miss is not a defect in this type; it is the
// shipped table's own shape, and a caller falls back to something else —
// pkg/game's itemName does.
type ItemNames map[ItemCode]string

// NameFor is the stored name n holds for code c, and whether it holds one.
// A nil n answers ("", false) for every code, the ordinary Go read of a nil
// map: no caller needs a guard of its own before calling this method.
func (n ItemNames) NameFor(c ItemCode) (string, bool) {
	s, ok := n[c]
	return s, ok
}
