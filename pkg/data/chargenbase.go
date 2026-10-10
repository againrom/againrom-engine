package data

// ChargenBaseNames is the four shipped Humans rows a generated character may
// start from, in ARCHETYPE order — fighter male, fighter female, mage
// male, mage female: the order archetypeSlot indexes into directly, and the
// order ChargenBase falls back through, name by name, when the archetype's
// own slot does not resolve.
//
// It is a FUNCTION and not a package variable, for ChargenSpread's own
// reason (chargen.go): a slice returned off a variable is aliased and
// mutable from anywhere, and "these four names in this order" would then be
// true by convention rather than by construction. SESS-HERO-014; the
// fallback is DIV-2799.
func ChargenBaseNames() []string {
	return []string{"PC_Danath", "PC_Naira", "PC_Fergard", "PC_Reniesta"}
}

// archetypeSlot is the one expression that turns an archetype choice into an
// index into ChargenBaseNames: the four published names are already in
// archetype order, so the slot is 2*class + female, and nothing decomposes a
// shipped column to recover a choice the player already made. It exists so
// that expression is written once rather than at every call site that needs
// it.
func archetypeSlot(class, female bool) int {
	slot := 0
	if class {
		slot = 2
	}
	if female {
		slot++
	}
	return slot
}

// ChargenBase is the base row a generated character with archetype (class,
// female) starts from:
//
// THE COLLECTION INDEX COMES BACK TOO, beside the parsed row (0134 D-13):
// both arms report the index they actually landed on, the archetype's own
// slot and the published-order fallback alike, because a caller reading a
// generated character's starting CLOTHES needs the row's own trailing
// strings — c.EntryStrings(i) — and those live on the entry the search
// resolved, not on anything NewHumanDef parsed off it. A second search by
// the archetype's own NAME would not do: after this function's own
// fallback, the name that resolved and the archetype that was asked for can
// be two different rows, and re-searching by name would silently read the
// wrong one's equipment. i is NotFound (0) exactly when the bool is false,
// and is meaningless — never read — in that arm.
//
// NOTHING HERE WALKS A TYPE ID.
func ChargenBase(c Collection, class, female bool) (HumanDef, int, bool) {
	names := ChargenBaseNames()
	if d, i, ok := resolveHumanByName(c, names[archetypeSlot(class, female)]); ok {
		return d, i, true
	}
	for _, name := range names {
		if d, i, ok := resolveHumanByName(c, name); ok {
			return d, i, true
		}
	}
	return HumanDef{}, NotFound, false
}

// resolveHumanByName folds FindHumanByName and NewHumanDef into the one
// question ChargenBase asks twice — at the archetype's own slot and, on
// fallback, at each of the four names in turn: does this name resolve to a
// row this tree can parse? A name not present in c, or a row too short for
// NewHumanDef to accept, answers false rather than an error or a panic
// (R-3). ITS SECOND RETURN IS THE INDEX THE NAME RESOLVED AT (0134 D-13),
// NotFound in either failing arm, so ChargenBase's own two call sites can
// report exactly the entry they parsed rather than search for it again.
func resolveHumanByName(c Collection, name string) (HumanDef, int, bool) {
	i := FindHumanByName(c, name)
	if i == NotFound {
		return HumanDef{}, NotFound, false
	}
	d, err := NewHumanDef(name, c.EntryParams(i))
	if err != nil {
		return HumanDef{}, NotFound, false
	}
	return d, i, true
}

// Profile is the ONE expression that turns a Humans row into a data.Profile:
// the two column flags off the row's own health and mana maxima, and the
// flag the pool graph multiplies by — Profile.Fighter — set for a row
// carrying NO mana column.
//
// THE CONDITION AND ITS DIRECTION ARE DECODED, not authored — the reading
// below is no longer weighed against a live competing one. `HERO-HP-072`
// (High for the condition and the direction; Medium for the naming) locates
// the runtime class bit the health arm's own doubling gates on: the arm
// doubles when that bit is CLEAR, and on the placement path the bit is set
// exactly when the actor's mana column is positive. Fighter is this tree's
// name for "the bit is clear", so d.ManaMax == 0 below is that condition
// stated over the row this actor is built from.
//
// WHICH ARCHETYPE THE BIT NAMES STAYS UNDECIDED — `HERO-HP-072`'s Medium
// half — and nothing here depends on it: Fighter is the graph's own class
// flag, not a resolved "swordsman" or "mage". The shipped numbers in this
// file's own header — PC_Danath and PC_Naira carrying no mana column,
// PC_Fergard and PC_Reniesta carrying 70 each — corroborate the CONDITION
// (a row with no mana column is the one this tree hands the doubled
// health), not a naming this story does not make.
//
// INVERTING THIS IS ONE EXPRESSION: swap d.ManaMax == 0 below for
// d.ManaMax != 0 and nothing else in this file, in recompute.go or in a
// caller changes shape. It would now also be a claim to argue against, not
// merely a value to flip.
func (d HumanDef) Profile() Profile {
	return Profile{
		Fighter:      d.ManaMax == 0,
		HealthColumn: d.HealthMax != 0,
		ManaColumn:   d.ManaMax != 0,
		Rider:        RiderTypeID(d.TypeID),
	}
}
