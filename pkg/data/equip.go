package data

import "strings"

// EquipSlots is how many equipment slots a character carries: twelve, the
// original's own width.
const EquipSlots = 12

// Equipment is a character's twelve equipment slots, addressed by the
// original's OWN NUMBERING — 1 to 12 — rather than by a Go index. A
// thirteenth slot and a slot numbered 0 are both absent, and every method
// below refuses them instead of clamping.
//
// AN OCCUPIED SLOT CARRIES THE ITEM'S WHOLE CODE, NOT A BARE ROW: the
// sixteen-bit data.ItemCode the item that fills the slot either resolved to
// or arrived carrying, unchanged — a code ARRIVES AS ONE VALUE, it is not
// assembled here from a row and three other guesses. There is one accessor
// pair for it, Code and SetCode, and no second spelling beside them: plan
// D-3 rejected keeping both a code and a row accessor because two spellings
// of one fact go stale the moment a caller sets only one.
//
// ZERO IS EMPTY, AND IT IS THE ORIGINAL'S OWN ENCODING RATHER THAN A GO
// CONVENIENCE. The sender writes a code whose field D is 0 into an
// unoccupied slot instead of omitting it, and no real item resolves to row
// 0 — row 0 of the shipped collection carries no name. A reader who does not
// know this reads the zero as Go's zero value standing in for "nothing was
// set"; it is instead a fact about the wire format this type reproduces —
// and the type's own zero value still carries it, because ItemCode(0) has
// every field at 0, D included.
//
// NOTHING HERE ENFORCES THAT ONLY SLOT 1 IS FILLED. The width is the
// original's and worth having, because a slot NUMBER is the only thing that
// makes "the first slot" mean anything; what the other eleven carry is not
// decoded for a character this tree builds. A rule confining them to empty
// would only have to be undone by whichever story fills them next, so the
// eleven empty slots are a disclosure and not a restriction.
//
// A SLOT'S CODE IS A LOADER VALUE: it reaches no simulation type, no byte
// form and no digest, the same as the Weapon.Row and Weapon.Code it is built
// from.
type Equipment struct {
	code [EquipSlots]ItemCode
}

// validEquipSlot reports that n is one of the original's twelve slot
// numbers, 1 to 12. It is the one test Code, SetCode and Occupied all share,
// so the three cannot come to disagree about which n is a slot at all.
func validEquipSlot(n int) bool { return n >= 1 && n <= EquipSlots }

// EquipSlotFor answers ITEM CODE c's OWN equipment slot — a field of the
// code itself, with no Equipment behind it — and whether c names one at
// all (AC-1). It reads field B, and field B alone: B IS BOTH THE SLOT AND
// THE CLASS AT ONCE (spec Terms "Item code"), the same fact itemcode.go's
// own field doc already states — a reader who has only half of that fact
// in mind would otherwise read this function as answering a category, not a
// slot.
//
// n IS c.B() ITSELF, UNCLAMPED: the function writes no per-class arm and no
// constant naming any one particular slot, because every slot number that
// reaches a caller is a number an item's own code already carried. It reuses
// validEquipSlot — the one bound Code, SetCode and Occupied already share
// — rather than re-spelling it, so ItemClassCarried (14) and a B of 0 fall
// out of that single comparison with no arm of their own: 14 is past
// EquipSlots and 0 is below it. The two named non-slots are therefore a
// CONSEQUENCE of one test, not two special cases written beside it.
func EquipSlotFor(c ItemCode) (int, bool) {
	n := c.B()
	return n, validEquipSlot(n)
}

// Code is the item code equipment slot n carries, and whether n named a slot
// at all. n outside 1..12 answers (0, false) rather than a clamp — the
// same refusal SetCode and Occupied give it.
func (e Equipment) Code(n int) (ItemCode, bool) {
	if !validEquipSlot(n) {
		return 0, false
	}
	return e.code[n-1], true
}

// SetCode writes code into equipment slot n, and reports that n named a slot
// to write it into. n outside 1..12 leaves e untouched and answers false —
// the same refusal Code and Occupied give it.
func (e *Equipment) SetCode(n int, code ItemCode) bool {
	if !validEquipSlot(n) {
		return false
	}
	e.code[n-1] = code
	return true
}

func (e Equipment) Occupied(n int) (occupied, ok bool) {
	code, ok := e.Code(n)
	if !ok {
		return false, false
	}
	return code.D() != 0, true
}

// HeroBody is defined in appearance.go and reused here unchanged (SC-4): the
// body list below is a list of the same names that law already classifies,
// and this file adds no second type for one name.

// BodyList is the shipped body-name payload, held verbatim and in order: one
// HeroBody per line of the install's own text. Entry i is line i,
// zero-based; the original addresses it with a definition row LESS ONE,
// which is HeroBodyFor's whole job below.
//
// NO COPY OF A SHIPPED LIST IS HELD IN THIS REPOSITORY. Every BodyList a
// test builds is a fixture of a handful of synthetic entries; the real
// payload is read from an install at run time by pkg/game, one tier up.
type BodyList []HeroBody

// ParseBodyList splits data into the ordered list the shipped payload holds,
// over CRLF or bare LF (the parse half — the read from an install is
// pkg/game's, one tier up).
//
// EVERY LINE IS AN ENTRY, INCLUDING AN EMPTY ONE. The shipped payload
// carries one, and dropping it would shift every entry after it one row out
// of step with the original's own indexing — which is exactly what would
// make an occupied slot's derived name disagree with the install it came
// from. A TRAILING NEWLINE ADDS NO FINAL ENTRY: the newline that ends the
// last real line is a terminator, not a separator that introduces an empty
// line after it. The two rules together are also why an EMPTY data answers
// an EMPTY list rather than a list of one empty entry — there is no
// terminator and nothing before it either.
//
// The original's loader does the same (ANIM-108): the shipped payload has 26
// entries, entry 22 blank, and a definition row of 27 or more indexes past the
// end, which HeroBodyFor answers as no name.
func ParseBodyList(data []byte) BodyList {
	lines := strings.Split(string(data), "\n")
	// The trailing element of a "\n"-split is empty exactly when data ends in
	// a newline (the ordinary case) or data is empty (the degenerate one);
	// either way it is not a line and must not become an entry.
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	list := make(BodyList, len(lines))
	for i, l := range lines {
		list[i] = HeroBody(strings.TrimSuffix(l, "\r"))
	}
	return list
}

func HeroBodyFor(l BodyList, e Equipment) (HeroBody, bool) {
	code, _ := e.Code(1) // slot 1 always exists; the discarded bool is a fact
	// about the slot number 1, not about this derivation.
	row := code.D()
	if row == 0 {
		row = 1 // the empty-slot arm and the row-1 arm are the same lookup.
	}
	i := row - 1
	if i < 0 || i >= len(l) || l[i] == "" {
		return "", false
	}
	return l[i], true
}

// figureHeldLastNames is the six body names HERO-FIGURE-060's own predicate
// compares slot 1's resolved body against — every two-handed and ranged body
// in heroBodyClass's own chain except mage, which the original's own
// compositor never reaches with this predicate.
var figureHeldLastNames = map[HeroBody]bool{
	BodyBowman: true, BodyArcher: true, BodyCrossbowman: true,
	BodyAxeman2H: true, BodySwordsman2H: true, BodyMageStaff: true,
}

// FigureHeldLast is which of the two held equipment slots — 1, a weapon or
// staff, or 2, a shield — the doll paints LAST, on top of the other and of
// every other occupied slot (0151-layers-and-names T7).
//
// DECODED (HERO-FIGURE-060): the compositor resolves slot 1's own body name
// — the SAME derivation HeroBodyFor already performs, off the SAME shipped
// list, `main/text/heropicture.txt` (HERO-APPEAR-052; ReadBodyList,
// pkg/game/bodylist.go) — and paints drawable index 0 last when that name
// is one of figureHeldLastNames, index 1 last otherwise. HERO-FIGURE-058
// gives drawable index = equipment slot − 1, so index 0 is slot 1 and
// index 1 is slot 2. This function relies on the High half only: which index
// paints last. It reads the two slots as 1 and 2 because that is
// EquipSlotFor's and wear.go's own established naming for them, not because
// this function re-proves the Medium reading.
//
// AN EMPTY SLOT 1 NEVER MATCHES: HeroBodyFor's own empty-slot arm answers
// l[0], "unarmed" on the shipped list (HERO-APPEAR-052), which is not one of
// the six names, and an l HeroBodyFor cannot resolve at all (nil, or a
// row past its end) answers ("", false) the same way — both fall through to
// the shield-last arm, exactly as a character holding no weapon at all
// should.
//
// AUTHORED FIRST, REPLACED ON THE OWNER'S RULING (0151-layers-and-names T7):
// T1 painted slot 1 last unconditionally, disclosed against this exact
// decoded swap because the owner's own defect report stated no shield
// exception. Shown the swap, the owner ruled to implement it as decoded: a
// shield ends up drawn over the sword either way, because a shield in this
// game reaches full character height, so the swap's shield-last arm does not
// reintroduce the "boots cover the sword" defect T1 fixed — the shield
// covering part of the weapon is correct, not the same bug.
func FigureHeldLast(l BodyList, e Equipment) int {
	if name, ok := HeroBodyFor(l, e); ok && figureHeldLastNames[name] {
		return 1
	}
	return 2
}
