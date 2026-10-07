package data

import (
	"fmt"
	"strings"
)

// Shield and Armor are a piece's resolved value, past ResolveWeapon's own
// three fields — Name, Row and Code — and an armour's own Slot. Both
// classes carry the defence and absorption authored in runtime columns 9 and
// 10, scaled through the same shape/material ladders and rounding rules. AN
// ARMOUR CARRIES A DEFENCE AND AN ABSORPTION: the two columns its own row
// states beyond Slot, each scaled by its shape and its material at that
// attribute's own factor- ladder slot — the same shape ResolveWeapon's own
// Defence already takes, one column pair and one ladder slot pair further
// along. A COMMON TYPE FOR ALL THREE PIECES WAS REJECTED (0128 plan D-6,
// unrevisited here): it would have to carry the union of what a weapon, a
// shield and an armour state, and at least one of its three arms would sit
// empty on every value this package ever builds — the one thing all three
// DO share, the code, is already ItemCode, a type of its own.

// shieldItemClass is field B of every item code ResolveShield composes. A
// SHIELD'S SLOT IS NOT READ FROM A COLUMN: unlike an armour, whose
// destination is its own row's Slot cell, a shield's is fixed by its class
// alone — the Shields collection carries no such column to read even if
// this file wanted to.
const shieldItemClass = 2

// Shield is one Shields row a name resolved to.
type Shield struct {
	// Name is the literal this shield was resolved from — kept, as
	// Weapon.Name is, so a refusal and a report can say which one.
	Name string

	// Row is the Shields collection's own index for this piece — the same
	// findByName result Code's field D already carries, read a second time
	// rather than searched for again (mirrors Weapon.Row).
	Row int32

	// Code is the item code this shield's own name composes: field A the
	// material index, field B shieldItemClass, field C the shape index and
	// field D this Row — the same four-field shape ResolveWeapon composes,
	// with the class fixed rather than read off a column.
	Code ItemCode

	// Defence and Absorption are the shield's authored armour block. The
	// original fills these from the same runtime columns and factor ladders as
	// an armour: defence rounds after adding one half, absorption truncates.
	Defence    int32
	Absorption int32

	// Weight is the row's own weight column scaled by shape and material —
	// itemWeight's one arithmetic, the same one an armour's and a weapon's
	// weight takes. It is what a carried unit of this piece adds to a
	// container's sum and what a worn one adds to its wearer's own weight.
	Weight int32
}

type Armor struct {
	// Name is the literal this armour was resolved from. EMPTY WHEN THE PIECE
	// WAS ANSWERED FROM A BARE CODE INSTEAD (ArmorFromCode below): a code
	// carries no literal to attach, and inventing one — recomposing shape,
	// material and row name and handing the join back to the name resolver,
	// WeaponFromCode's own trick — is exactly what ArmorFromCode's own doc
	// rules out for an armour. Past Name, a piece answered by ArmorFromCode
	// reads identically to the same piece answered by ResolveArmor, field for
	// field (0136 AC-3).
	Name string

	// Row is the Armors collection's own index for this piece.
	Row int32

	// Code is the item code this armour's own name composes: field A the
	// material index, field C the shape index and field D this Row, exactly
	// as Shield's own Code is built — field B DIFFERS, and is documented on
	// Slot below, because an armour's slot is not a class constant the way
	// a shield's is.
	Code ItemCode

	Slot int32

	// Defence and Absorption are the piece's own worth: the row's param 9 and
	// param 10, each scaled by the PRODUCT of the piece's shape and material
	// factor at that attribute's own factor-ladder slot — 6 for defence
	// (scaleDefenceSlot, shared with a Weapon's own Defence), 7 for absorption
	// (scaleAbsorptionSlot, an armour's alone).
	//
	// THE TWO DO NOT ROUND ALIKE: Defence adds one half before truncating
	// toward zero, Absorption does not — see fillArmor's own doc for the
	// exact arithmetic and the revert check T1 is written to fail. EACH IS
	// STORED SIXTEEN BITS WIDE, AN UNSIGNED CONVERSION AND NOT A REFUSAL: a
	// negative product would wrap rather than clamp, though no row the shipped
	// tables carry, on either lawful root, ever produces one.
	Defence    int32
	Absorption int32

	// Weight is the row's own weight column scaled by shape and material, read
	// by fillArmor beside Defence and Absorption and therefore identical
	// whether the piece was answered from a name or from a bare code.
	Weight int32
}

// armorSlotColumn is the runtime column of an Armors row this file reads —
// param 4, the row's own Slot cell (spec Terms "Slot column").
const armorSlotColumn = 4

// armorDefenceColumn and armorAbsorptionColumn are the two further columns
// fillArmor reads, since 0136 (spec Terms "Defence column / absorption
// column"): param 9 and param 10. scaleAbsorptionSlot is the factor-ladder
// slot absorption is scaled at — an armour's own, no weapon column reads
// it. scaleDefenceSlot, defence's own ladder slot, is already declared in
// weapon.go for a WEAPON's Defence and is reused rather than re-spelled
// here: the ladder is the identical nine-double record either way, and a
// second constant naming the same index would be a second name for one fact.
const (
	armorDefenceColumn    = 9
	armorAbsorptionColumn = 10
	scaleAbsorptionSlot   = 7
)

// ResolveShield turns a shield's NAME into the row it names.
//
// The residue must then name a Shields row EXACTLY. A residue naming no row
// is refused, with an error in ResolveWeapon's own house style — a shield
// whose Slot column would be anything other than shieldItemClass does not
// exist, so the refusal here has one arm and not three.
func ResolveShield(name string, shapes, materials ScaleTable, shields Collection) (Shield, error) {
	rest := stripSuffix(name)

	shape, _, rest := takePrefix(rest, shapes)
	material, materialName, rest := materialPrefix(rest, materials)
	rest = impliedShapePrefix(materialName, rest)

	if i := strings.Index(rest, " Shield"); i > 0 {
		rest = rest[:i]
	}

	row := findByName(shields, rest)
	if row == NotFound {
		return Shield{}, fmt.Errorf("data: shield %q: no Shields entry named %q", name, rest)
	}

	return fillShield(name, shape, material, row, shapes, materials, shields), nil
}

// fillShield is the one arithmetic a shield is worth. Given the three indices
// its code carries, it composes the code and reads defence, absorption and
// weight through the same scaling helpers an armour uses. ResolveShield and
// ShieldFromCode both go through it, so the name and code paths cannot drift.
//
// IT CANNOT FAIL. Unlike an armour's fill there is no Slot cell to read and
// no column whose absence changes what the piece IS, so a row too short for
// the weight column weighs its scaled zero (itemWeight's own convention) and
// the caller's only refusal remains the name or the code that named no row.
func fillShield(name string, shape, material, row int, shapes, materials ScaleTable, shields Collection) Shield {
	code := ComposeItemCode(material, shieldItemClass, shape, row)
	p := shieldRowParams(shields, row)
	defence, absorption := wearProtection(p, shape, material, shapes, materials)
	weight := itemWeight(p, shape, material, shapes, materials)
	return Shield{Name: name, Row: int32(row), Code: code,
		Defence: defence, Absorption: absorption, Weight: weight}
}

// shieldRowParams is row's parameter cells, or nil for a row outside the
// collection or one nothing was written into. It is shieldFromCode's bound
// check and fillShield's own, in one place.
func shieldRowParams(shields Collection, row int) []int32 {
	if row < 0 || row >= shields.Len() {
		return nil
	}
	return shields.EntryParams(row)
}

// ShieldFromCode turns an item code BACK into the shield it names — the
// bare-code counterpart to ResolveShield, and the shield arm of
// ItemCodeWeight's own dispatch.
//
// IT READS THE THREE INDICES STRAIGHT OFF THE CODE, ArmorFromCode's own
// treatment and not WeaponFromCode's: a shield's name parse has the same
// two re-attachment steps an armour's has — impliedShapePrefix and the
// trailing ` Shield` cut — so recomposing a name and handing it back to
// ResolveShield would apply both a second time.
//
// FIELD B IS THE CLASS GATE, checked before the collection opens: a B other
// than shieldItemClass names a weapon, an armour or no equipment at all, and
// is refused with the zero value. Field D must name a WRITTEN row; the
// collection's reserved zeroth entry and any index past its end are refused
// on the same terms, because a shield answered from an unwritten row would
// carry a scaled zero that reads exactly like a real weightless shield.
//
// The result carries no Name, for ArmorFromCode's own reason. Past Name it
// reads identically to what ResolveShield answers for the name the code was
// composed from.
func ShieldFromCode(c ItemCode, shapes, materials ScaleTable, shields Collection) (Shield, error) {
	if b := c.B(); b != shieldItemClass {
		return Shield{}, fmt.Errorf("data: item code 0x%04x: field B is %d, not a shield's class %d",
			uint16(c), b, shieldItemClass)
	}
	row := c.D()
	if len(shieldRowParams(shields, row)) == 0 {
		return Shield{}, fmt.Errorf("data: item code 0x%04x: row %d names no written Shields entry",
			uint16(c), row)
	}
	return fillShield("", c.C(), c.A(), row, shapes, materials, shields), nil
}

// ResolveArmor turns an armour's NAME into the row it names, the slot its
// own row states, and — since 0136 — the defence and absorption its own
// row states too.
//
// A residue naming no row is refused, the same as ResolveShield's own
// refusal. Past that, fillArmor's OWN refusal runs — see its doc for the
// one arithmetic ResolveArmor and ArmorFromCode now share and for why that
// refusal's own threshold is the row's Slot cell and NOT the wider
// absorption column ArmorFromCode itself additionally demands.
func ResolveArmor(name string, shapes, materials ScaleTable, armors Collection) (Armor, error) {
	rest := stripSuffix(name)

	shape, _, rest := takePrefix(rest, shapes)
	material, materialName, rest := materialPrefix(rest, materials)
	rest = impliedShapePrefix(materialName, rest)

	row := findByName(armors, rest)
	if row == NotFound {
		return Armor{}, fmt.Errorf("data: armour %q: no Armors entry named %q", name, rest)
	}

	piece, err := fillArmor(shape, material, row, shapes, materials, armors)
	if err != nil {
		return Armor{}, fmt.Errorf("data: armour %q: %w", name, err)
	}
	piece.Name = name
	return piece, nil
}

// armorRowLen is Armors row row's own written width, or -1 when armors is
// nil, row is outside armors' own bounds, or row names an entry no fixture
// or shipped file ever writes to (EntryName empty — the reserved index 0 of
// a one-based collection, or a hole a corrupt D field could name past it).
//
// IT IS THE ONE BOUNDS-CHECKED READ fillArmor AND ArmorFromCode BOTH SHARE,
// so neither ever indexes a Collection the other has already learned cannot
// answer. A row's own true width, once known non-negative, is what each
// caller then measures against ITS OWN threshold — fillArmor's is
// armorSlotColumn, unchanged since before this story; ArmorFromCode's own is
// wider, and is that function's own refusal to make, not this one's.
func armorRowLen(armors Collection, row int) int {
	if armors == nil || row < 0 || row >= armors.Len() || armors.EntryName(row) == "" {
		return -1
	}
	return len(armors.EntryParams(row))
}

// armorColumn is p[i], or 0 for a row too short to carry it. It is the
// leniency fillArmor extends to the defence and absorption columns alone —
// NEVER to armorSlotColumn, which fillArmor's own length refusal below
// guarantees is present before this is ever called — so a row ResolveArmor
// has always resolved, carrying Slot and nothing past it, keeps resolving
// exactly as before 0136, at a defence and an absorption of zero rather
// than a refusal this task would otherwise be widening for every caller at
// once.
func armorColumn(p []int32, i int) int32 {
	if i < len(p) {
		return p[i]
	}
	return 0
}

// fillArmor is the ONE ARITHMETIC an armour piece is worth: given the three
// indices a piece's code already carries, or a name already resolved to, it
// reads the row's Slot, defence and absorption columns and scales the latter
// two by the shape's and the material's own factor at their own
// factor-ladder slot — exactly the shape ResolveWeapon's own Defence and
// ToHit already take, one package tier down.
//
// BOTH ResolveArmor AND ArmorFromCode CALL THIS AND NOTHING ELSE COMPUTES A
// PIECE'S VALUE: ResolveArmor has already turned a name into (shape,
// material, row) by the time it calls this; ArmorFromCode (below) reads the
// same three off a code's own fields C, A, D. Neither repeats a scale, a
// rounding rule or a column index of its own past this point.
//
// THE REFUSAL HERE IS armorSlotColumn'S OWN, UNCHANGED SINCE BEFORE 0136: a
// row too short to carry Slot refuses, exactly as ResolveArmor always
// refused it.
//
// DEFENCE AND ABSORPTION DO NOT ROUND ALIKE: defence is the scaled product
// PLUS ONE HALF, truncated toward zero — ResolveWeapon's own rounding,
// ftol(x+0.5) — while absorption is the scaled product truncated ALONE,
// with no half added. A row whose two columns hold the same value and whose
// two ladder slots hold the same factor therefore need not answer the same
// two numbers; deleting the "+ 0.5" below is this task's own named revert
// check.
//
// EACH IS STORED SIXTEEN BITS WIDE, AS AN UNSIGNED CONVERSION AND NOT A
// REFUSAL: int32(uint16(ftol(...))) keeps the low sixteen bits of the
// truncated product, wrapping a negative one rather than clamping it —
// damageByte's own precedent, one width up. Measured over every row × shape
// × material combination the shipped tables allow, identical on both lawful
// roots: defence spans [0.1395, 101.9059], absorption spans [0.0000,
// 4.7159], and no row carries a negative column at all — so the wrap is
// reachable by no shipped combination, which is what makes this modelling
// rather than clamping.
func fillArmor(shape, material, row int, shapes, materials ScaleTable, armors Collection) (Armor, error) {
	n := armorRowLen(armors, row)
	if n < 0 {
		return Armor{}, fmt.Errorf("armour: row %d names no written Armors entry", row)
	}
	if n <= armorSlotColumn {
		return Armor{}, fmt.Errorf("armour: row %d carries %d cell(s); slot %d is read",
			row, n, armorSlotColumn)
	}
	p := armors.EntryParams(row)
	slot := p[armorSlotColumn]

	// Field B tracks Slot only inside the twelve real slots (validEquipSlot,
	// equip.go's own bound — reused rather than re-spelled, so this file and
	// EquipSlotFor can never come to disagree about which n is a slot at
	// all). Outside it, field B is forced to 0 rather than masked: see Slot's
	// doc comment above for why a mask alone is not enough.
	var class int
	if validEquipSlot(int(slot)) {
		class = int(slot)
	}
	code := ComposeItemCode(material, class, shape, row)

	defence, absorption := wearProtection(p, shape, material, shapes, materials)

	weight := itemWeight(p, shape, material, shapes, materials)

	return Armor{Row: int32(row), Code: code, Slot: slot,
		Defence: defence, Absorption: absorption, Weight: weight}, nil
}

// wearProtection is the shared armour-block fill. Shield and armour use the
// same two runtime columns, ladder slots, widths and deliberately different
// rounding rules (ITEM-ARMFILL-032).
func wearProtection(p []int32, shape, material int, shapes, materials ScaleTable) (defence, absorption int32) {
	defFactor := scale(shapes, shape, scaleDefenceSlot) * scale(materials, material, scaleDefenceSlot)
	absFactor := scale(shapes, shape, scaleAbsorptionSlot) * scale(materials, material, scaleAbsorptionSlot)
	defence = int32(uint16(ftol(float64(armorColumn(p, armorDefenceColumn))*defFactor + 0.5)))
	absorption = int32(uint16(ftol(float64(armorColumn(p, armorAbsorptionColumn)) * absFactor)))
	return defence, absorption
}

func ArmorFromCode(c ItemCode, shapes, materials ScaleTable, armors Collection) (Armor, error) {
	b := c.B()
	if b == weaponItemClass || b == shieldItemClass || !validEquipSlot(b) {
		return Armor{}, fmt.Errorf("data: item code 0x%04x: field B is %d, not an armour's equipment slot",
			uint16(c), b)
	}

	row := c.D()
	if n := armorRowLen(armors, row); n >= 0 && n <= armorAbsorptionColumn {
		return Armor{}, fmt.Errorf("data: item code 0x%04x: row %d carries %d cell(s); absorption %d is read",
			uint16(c), row, n, armorAbsorptionColumn)
	}

	piece, err := fillArmor(c.C(), c.A(), row, shapes, materials, armors)
	if err != nil {
		return Armor{}, fmt.Errorf("data: item code 0x%04x: %w", uint16(c), err)
	}
	if !validEquipSlot(int(piece.Slot)) {
		return Armor{}, fmt.Errorf("data: item code 0x%04x: row %d states Slot %d, not a wearable place",
			uint16(c), row, piece.Slot)
	}
	return piece, nil
}
