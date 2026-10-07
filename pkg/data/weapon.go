package data

import (
	"fmt"
	"math"
	"strings"
)

// ScaleTable is a collection whose entries carry the nine doubles — the shape
// table and the material table, which between them supply every factor a
// weapon's row is scaled by.
//
// It is declared HERE and not imported from the parser, like Collection beside
// it, so this tier reads a table without depending on who walked the file.
type ScaleTable interface {
	Len() int
	EntryName(i int) string
	EntryDoubles(i int) []float64
}

// The slots of a Weapons row this contract reads. They are RUNTIME column
// numbers, which are also the parameter array's own indices — the file's title
// array carries one entry more than the row has cells, so column i is title
// i+1, and a consumer indexing by title is off by one on everything.
//
// Four of these are anchored at instruction level by different uses on
// different fields, which is what makes the map a reading rather than a guess.
const (
	weaponAttackTypeSlot  = 5
	weaponPhysicalMinSlot = 6
	weaponPhysicalMaxSlot = 7
	weaponToHitSlot       = 8
	weaponDefenceSlot     = 9
	weaponRangeSlot       = 0xb
	weaponChargeSlot      = 0xc
	weaponRelaxSlot       = 0xd
)

// The slots of a shape's or material's nine-double record this contract reads.
// The ladder is fixed by the record's own size — a tenth double would end past
// it — and not by a fit, which is what the ladder it replaces was.
const (
	scaleDamageSlot  = 4
	scaleToHitSlot   = 5
	scaleDefenceSlot = 6
)

// meleeAttackTypes is the exclusive upper bound of the attack types that take
// FoldWeapon's melee arm. Ten and above takes the ranged arm instead
// (Weapon.Ranged). That arm takes accuracy from General, routes exact types
// 11/12 into the existing third damage component, and takes none of the four
// additions the melee arm takes. This threshold therefore decides which of
// the two complete equip arms a row takes; every row on either side resolves.
const meleeAttackTypes = 0xa

// emptyCell is the value a Weapons row stores for a column with nothing in it.
const emptyCell int32 = -1

// defaultRange is what an empty range cell means. The fill writes the column
// verbatim or this, and nothing else reads the cell.
const defaultRange int32 = 1

// weaponItemClass is field B of every item code ResolveWeapon composes: the
// equipment slot a weapon occupies and also its class (docs/0110-inventory
// spec Terms "Item code"). Only a weapon can occupy a slot in this build
// (spec Scope, "armours, shields and magic items"), so this is the only
// class value this file ever writes.
const weaponItemClass = 1

// Weapon is one Weapons row scaled by its shape and its material: what an item
// of that name CARRIES, as opposed to what its row ships.
//
// The two scales are the whole difference and they are large. A row ships
// 23..40 and the item built from it carries 5 and a spread of 3 — so a row's
// own columns are NOT commensurable with anything a hero holds, and the only
// numbers that may meet a stat are the ones in this struct.
//
// DamageBase and DamageSpread are a BASE AND A SPREAD, not a minimum and a
// maximum: the fill's second byte subtracts the first. For one equipped weapon
// the two encodings carry the same information; they diverge the moment a
// second contributor exists, because a fold sums bases and spreads separately
// and two maxima never add.
type Weapon struct {
	// Name is the literal this weapon was resolved from, kept so a refusal and
	// a report can say which one.
	Name string

	// Row is the definition row this weapon resolved to inside the Weapons
	// collection — findByName's own index, carried onto the struct rather than
	// searched for a second time. The collection is one-based and index
	// 0 is its reserved, never-written entry, so a real weapon's Row is never
	// 0.
	//
	// IT REACHES NO ENTITY, NO BYTE FORM AND NO DIGEST: nothing in this tree
	// adds it to a simulation type, encodes it or hashes it. It is a LOADER
	// value, read because the resolution that already ran has it, for whoever
	// assembles an equipment slot from a party member's starting weapon.
	Row int32

	Code ItemCode

	// DamageBase and DamageSpread are added to the wielder's own pair. Both are
	// byte-wide in the original and are modelled as such.
	DamageBase   int32
	DamageSpread int32

	// ToHit and Defence are added to the wielder's to-hit and defence. Defence
	// is the column the shipped titles spell `#.deIrnce`; it is nonzero on four
	// of the twenty-seven shipped weapon rows.
	ToHit   int32
	Defence int32

	// AttackType is the row's own kind. It selects the equip arm — and, on the
	// melee arm, it IS the skill slot the wielder's derive reads, which is why
	// a blade weapon trains and is trained by the blade skill.
	AttackType int32

	// ChargeTime and RelaxTime are ASSIGNED over the wielder's cadence rather
	// than added to it, so an armed unit's cadence is its weapon's alone.
	ChargeTime int32
	RelaxTime  int32

	// Range is the row's reach column, one when the cell is empty. Hero.Derive
	// reads it now: a generated character's reach is his weapon's Range,
	// exactly as this field states it.
	Range int32

	SpellName  string
	SpellPower int32

	// Weight is the row's own weight column scaled by shape and material —
	// itemWeight's one arithmetic, shared with an armour and a shield.
	// HERO-EQUIP-017 names this as the weapon fill's `w+0x4a` from column 3,
	// and it is the value every equip path hands to the wielder's own weight
	// accumulator.
	Weight int32
}

// Melee reports whether this weapon takes the equip arm that feeds the first
// damage pair — every row below meleeAttackTypes.
func (w Weapon) Melee() bool { return w.AttackType < meleeAttackTypes }

// Ranged reports whether this weapon's row takes the ranged equip arm —
// meleeAttackTypes and above. It is Melee's own complement, kept as its own
// predicate because FoldWeapon reads the ranged sense directly rather than
// negating Melee at its one call site.
func (w Weapon) Ranged() bool { return w.AttackType >= meleeAttackTypes }

// takeSuffix splits name at its first `{` into the two halves stripSuffix
// used to keep only one of: rest is exactly what stripSuffix has always
// answered, and suffix is everything from that `{` to the end of the string
// — the half a `{…}` name used to have thrown away outright.
//
// No `{` in name answers suffix == "", which takeCastSpell also refuses
// (FR-1a): an absent attachment and a malformed one carry the weapon the
// same way, through the same zero pair.
func takeSuffix(name string) (rest, suffix string) {
	if i := strings.IndexByte(name, '{'); i >= 0 {
		return strings.TrimSpace(name[:i]), name[i:]
	}
	return strings.TrimSpace(name), ""
}

// stripSuffix removes a trailing `{…}` from name — everything from its
// first `{` to the end of the string — and trims what is left.
func stripSuffix(name string) string {
	rest, _ := takeSuffix(name)
	return rest
}

// ResolveWeapon turns a weapon's NAME into the numbers an item of that name
// carries, against the three collections the original's own resolver reads.
//
// The name is a leading shape word, then a leading material word, then the
// rest as the Weapons row — each of the first two optional, both taken by
// itemparse.go's takePrefix, the ONE walk this package now states and
// ResolveShield and ResolveArmor share unchanged. THE SPLIT RULE IS OURS:
// research states the shape of the parse and not how the search matches, and
// a shipped literal proves a material may be more than one word (`Uncommon
// Magic Wood Short Bow`, whose table holds both `Wood` and `Magic Wood`).
// SINCE 0128 THE MATCH ITSELF IS THE ORIGINAL'S OWN DESCENDING SUBSTRING
// FIND (FR-2a), not the longest-word-boundary-prefix match this file carried
// before: a table is searched from its LAST entry down to its first, and the
// first entry whose name occurs ANYWHERE in what is left wins — the
// table's own order breaks a tie by construction, since the walk stops at
// the first hit.
//
// A NAME WITH NO SHAPE WORD RESOLVES THROUGH INDEX 0, which is what the item
// constructor's own zeroed byte does — not through a table entry this tree went
// looking for by name. Nothing here believes anything about what the shipped
// tables are called.
//
// The scale is the PRODUCT of the two records' factors, and it is applied
// per attribute out of a different slot of the same nine-double ladder.
//
// It fails, with the zero value, on a remainder that names no row and on a
// row too short for the slots this contract reads. A ROW'S ATTACK TYPE NO
// LONGER REFUSES IT: a ranged row is scaled exactly as a melee row is, and
// it is Weapon.Ranged() — read by the equip fold, not by this function —
// that tells the two kinds apart. The refusal that remains is still by name:
// a weapon resolved in part would give its wielder a silent zero, which is
// the defect this whole story exists to remove.
//
// SINCE 0139 THE `{…}` SUFFIX IS READ, NOT ONLY STRIPPED: takeSuffix keeps
// the half stripSuffix used to throw away, and takeCastSpell (itemparse.go)
// turns it into SpellName and SpellPower on the result. The attachment
// governs only those two fields — it plays no part in which Weapons row
// rest resolves against, so a malformed one yields a weapon carrying no
// spell rather than a refusal (FR-1a).
func ResolveWeapon(name string, shapes, materials ScaleTable, weapons Collection) (Weapon, error) {
	rest, suffix := takeSuffix(name)
	spellName, spellPower, _ := takeCastSpell(suffix)

	shape, _, rest := takePrefix(rest, shapes)
	material, _, rest := materialPrefix(rest, materials)

	row := findByName(weapons, rest)
	if row == NotFound {
		return Weapon{}, fmt.Errorf("data: weapon %q: no Weapons entry named %q", name, rest)
	}
	p := weapons.EntryParams(row)
	if len(p) <= weaponRelaxSlot {
		return Weapon{}, fmt.Errorf("data: weapon %q: row %q carries %d cell(s); slots 0 to %d are read",
			name, rest, len(p), weaponRelaxSlot)
	}

	kind := p[weaponAttackTypeSlot]

	// One factor per attribute, each the PRODUCT of the two records' double at
	// that attribute's own slot of the shared ladder.
	factor := func(slot int) float64 {
		return scale(shapes, shape, slot) * scale(materials, material, slot)
	}

	// The base is rounded and stored FIRST, and the spread then subtracts THAT
	// byte rather than the unrounded product. The two differ, and the shipped
	// sheet agrees with this one.
	dmg := factor(scaleDamageSlot)
	base := damageByte(float64(p[weaponPhysicalMinSlot]) * dmg)

	// Code composes the three indices this resolution already has — material,
	// shape and row — into the one word an item of this name carries as its
	// appearance. Field C, the shape index, is authored; A and D are the same
	// collection indices already found above.
	code := ComposeItemCode(material, weaponItemClass, shape, row)

	return Weapon{
		Name:         name,
		Row:          int32(row),
		Code:         code,
		DamageBase:   base,
		DamageSpread: damageByte(float64(p[weaponPhysicalMaxSlot])*dmg - float64(base)),
		ToHit:        ftol(float64(p[weaponToHitSlot])*factor(scaleToHitSlot) + 0.5),
		Defence:      ftol(float64(p[weaponDefenceSlot])*factor(scaleDefenceSlot) + 0.5),
		AttackType:   kind,
		ChargeTime:   p[weaponChargeSlot],
		RelaxTime:    p[weaponRelaxSlot],
		Range:        cellOr(p[weaponRangeSlot], defaultRange),
		SpellName:    spellName,
		SpellPower:   spellPower,
		Weight:       itemWeight(p, shape, material, shapes, materials),
	}, nil
}

// WeaponFromCode turns an item code BACK into the weapon it names — the
// exact reverse of the composition Code documents on Weapon and
// ResolveWeapon performs at the end of its own resolution (AC-2).
//
// A weapon's code carries the three collection indices its own composition
// wrote into it, nothing this function has to recompute: field A the
// material, field C the shape, field D the row. Each is turned back into
// that collection's own entry NAME — shapes.EntryName(c.C()),
// materials.EntryName(c.A()), weapons.EntryName(c.D()) — and the NON-EMPTY
// names are joined with a single space IN THAT ORDER: shape, then material,
// then the row's own name, because that is the order ResolveWeapon's own
// prefix walk consumes them in (stripSuffix, then a shape word, then a
// material word, then the remainder). The joined string is handed to
// ResolveWeapon ITSELF rather than scaled here a second time: THIS FUNCTION
// HOLDS NO ARITHMETIC OF ITS OWN for a scale, a damage byte or a cadence —
// there is one weapon resolver in this tree, and re-entering it is the whole
// of the pay-off, because a name the resolver accepts and the code it was
// composed from now agree BY CONSTRUCTION rather than by two calculations
// kept in step by hand. The result therefore reads identically to what
// ResolveWeapon answers for the name the code was built from, field for
// field, that name apart — a code's own entry index and a table's index-0
// default resolve through the same factor either way, so a name the join
// recomposes need not be the literal one ResolveWeapon first saw.
//
// A code whose field B is not weaponItemClass names no weapon at all — some
// other slot, or the class carried and never worn — and is refused before
// either table is read. Past that gate this function invents no error of its
// own: a row index of 0, an entry with no name, or a joined string the
// resolver's own search does not match all come back as RESOLVEWEAPON'S OWN
// error, because there is nothing this function knows about the code that
// ResolveWeapon does not already know about the name it composes.
func WeaponFromCode(c ItemCode, shapes, materials ScaleTable, weapons Collection) (Weapon, error) {
	if c.B() != weaponItemClass {
		return Weapon{}, fmt.Errorf("data: item code 0x%04x: field B is %d, not the weapon class %d",
			uint16(c), c.B(), weaponItemClass)
	}

	shapeName := ""
	if shapes != nil && c.C() < shapes.Len() {
		shapeName = shapes.EntryName(c.C())
	}
	materialName := ""
	if materials != nil && c.A() < materials.Len() {
		materialName = materials.EntryName(c.A())
	}
	weaponName := ""
	if weapons != nil && c.D() < weapons.Len() {
		weaponName = weapons.EntryName(c.D())
	}
	var parts []string
	for _, name := range []string{shapeName, materialName, weaponName} {
		if name != "" {
			parts = append(parts, name)
		}
	}

	return ResolveWeapon(strings.Join(parts, " "), shapes, materials, weapons)
}

// WeaponRange resolves name to its weapon's RANGE ALONE: for ANY attack
// type, computing no damage, no to-hit and no cadence, and refusing no row
// on the grounds of its attack type. The second result reports whether the
// name resolved; a name that does not answers (0, false), never a partial
// number.
//
// It takes ResolveWeapon's own first step and its own prefix walk — strip,
// shape, material, findByName — with a different tail and a different length
// guard: `weaponRangeSlot`, not `weaponRelaxSlot`, because a row that carries
// a range and no cadence is a row this function can still answer.
//
// ResolveWeapon now admits every attack type too. This remains a separate
// entry point because a range-only caller may accept a short row that carries
// the range cell but not the later cadence cells the complete resolver needs;
// it also avoids scaling and constructing fields that caller will discard.
func WeaponRange(name string, shapes, materials ScaleTable, weapons Collection) (int32, bool) {
	rest := stripSuffix(name)

	_, _, rest = takePrefix(rest, shapes)
	_, _, rest = takePrefix(rest, materials)

	row := findByName(weapons, rest)
	if row == NotFound {
		return 0, false
	}
	p := weapons.EntryParams(row)
	if len(p) <= weaponRangeSlot {
		return 0, false
	}
	return cellOr(p[weaponRangeSlot], defaultRange), true
}

// scale is table t's entry i's double at slot.
//
// An ABSENT table, or one with no entry there, contributes 1 — the identity, so
// a caller holding half a table gets the other half's scaling rather than zero.
// That is a property of a missing table and never of a missing NAME: index 0 of
// a present table is a real entry and its own factor is what an unnamed shape
// resolves through.
func scale(t ScaleTable, i, slot int) float64 {
	if t == nil || i < 0 || i >= t.Len() {
		return 1
	}
	d := t.EntryDoubles(i)
	if slot >= len(d) {
		return 1
	}
	return d[slot]
}

// findByName returns the index of the first entry of c named exactly n, or
// NotFound. It walks ascending and skips the unwritten entries a collection's
// count allocates, which is the shape every search in this package holds to.
func findByName(c Collection, n string) int {
	if c == nil || n == "" {
		return NotFound
	}
	for i := 0; i < c.Len(); i++ {
		if name := c.EntryName(i); name != "" && name == n {
			return i
		}
	}
	return NotFound
}

// damageByte rounds half up, truncates, and takes the low byte — the width of
// the store, on the weapon's fill and on the wielder's derive alike. Modelling
// the width costs one conversion; refusing the overflow instead would invent a
// rule the original does not have. No shipped row and no legal stat reaches it.
func damageByte(f float64) int32 { return int32(uint8(ftol(f + 0.5))) }

// cellOr is v, or alt when the cell is the format's empty one.
func cellOr(v, alt int32) int32 {
	if v == emptyCell {
		return alt
	}
	return v
}

// ftol truncates toward zero, which is what the image's own conversion does:
// it sets the rounding control to truncate before every store. Rounding to
// nearest anywhere in this chain moves a band edge by a whole point.
func ftol(f float64) int32 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return int32(math.Trunc(f))
}

// AttackTypeFromCode is one weapon code's own AttackType cell, read off its
// definition row and nothing else.
//
// IT DOES NOT REACH FOR THE SCALE TABLES, for SuitabilityFromCode's reason: a
// weapon's kind is a cell of its row, neither scale touches it, and a caller
// that needs only the kind should not be refused because a shape index is out
// of range. WeaponFromCode is what a caller wanting the SCALED weapon uses;
// this answers the one column, and it is the same column that function reads.
//
// ok is false for a code whose field B is not the weapon class, for a nil or
// too-short collection, for a row index outside it, and for a row too short to
// carry the column. A caller must read that as "no answer" rather than as a
// kind of its own.
func AttackTypeFromCode(c ItemCode, weapons Collection) (int32, bool) {
	if c.B() != weaponItemClass || weapons == nil {
		return 0, false
	}
	row := c.D()
	if row < 0 || row >= weapons.Len() {
		return 0, false
	}
	p := weapons.EntryParams(row)
	if len(p) <= weaponAttackTypeSlot {
		return 0, false
	}
	return p[weaponAttackTypeSlot], true
}
