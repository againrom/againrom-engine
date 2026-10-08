package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// ghostRowName is the definition-table `Units` entry a Control Spirit cast
// raises.
//
// It is a NAME and not a subscript, on the claim's own terms: the row's index
// is a property of the shipped table and moves with it, while the name is what
// the original's code carries.
const ghostRowName = "Ghost"

// ghostTemplate is the sim-side GhostTemplate resolved off t's Units
// collection, or the zero value when the table ships no such row.
//
// It is sited here because pkg/sim may not import pkg/data (spellRules, spell.go).
//
// Difficulty adjusts only health, to-hit and defence, and the raise takes all
// three from the corpse (MAGIC-250), so the adjusted values are persisted but
// unread.
//
// A row NewUnitDef cannot parse gives the zero template, which raises nothing,
// rather than failing a mission load over a spell.
func ghostTemplate(t *Table, diff Difficulty) sim.GhostTemplate {
	if t == nil {
		return sim.GhostTemplate{}
	}
	units := t.units()
	i, d, ok := ghostRow(units)
	if !ok {
		return sim.GhostTemplate{}
	}
	if adj, err := Adjust(d, diff); err == nil {
		d = adj
	}
	c := d.Combat()
	// A raised Ghost uses the same equipped-weapon active slot as an ordinary
	// placement of its Units row. Only the slot is taken here: this template's
	// pre-existing choice of row combat numbers is unchanged by the resistance
	// story.
	if w := unitWeapon(units.EntryStrings(i), t); w != nil {
		c.SkillSlot = data.FoldWeapon(c, w, d.ToHit).SkillSlot
	}
	return sim.GhostTemplate{
		NativeBasis: nativeInitialBase(nil).WithModifier([64]byte{}).WithBody(uint16(d.Body)),
		// THE CLASS KEY IS THE ROW'S TypeID COLUMN. It is the same number a
		// placement carries in ClassID and that Resolve keys FindUnit on, and
		// it is what pkg/game indexes units.reg with — so the row and the
		// artwork are named by one value rather than by two that could drift.
		Class:         d.TypeID,
		TypeID:        d.TypeID,
		Domain:        domainFor(d),
		Speed:         d.Speed,
		RotationSpeed: d.RotationSpeed,
		ScanRange:     sightOf(d.ScanRange),
		Reach:         reachOf(c.Reach),
		TokenSize:     uint8(d.TokenSize),
		DyingTime:     d.DyingTime,
		XPValue:       d.XPValue,
		Withdraw:      d.Withdraw,
		Wimpy:         d.Wimpy,
		Protection:    d.Protection,
		Resistance:    data.DamageKindResistance(d.Resistance),
		XPSlot:        uint8(c.SkillSlot),
		ToHit:         c.ToHit,
		Defence:       c.Defence,
		Absorption:    c.Absorption,
		DamageBase:    c.DamageBase,
		DamageSpread:  c.DamageSpread,
		AttackCharge:  c.AttackChargeTime,
		AttackRelax:   c.AttackRelaxTime,
		AlwaysHits:    c.AlwaysHits,
	}
}

// ghostRow is the Units row named Ghost, the one a Control Spirit cast
// constructs its actor from (MAGIC-SING-019), and its definition. ok is false
// when the table ships no such row or the row does not parse.
func ghostRow(units data.Collection) (int, data.UnitDef, bool) {
	i := findUnitByName(units, ghostRowName)
	if i == data.NotFound {
		return i, data.UnitDef{}, false
	}
	d, err := data.NewUnitDef(units.EntryName(i), units.EntryParams(i))
	return i, d, err == nil
}

// findUnitByName walks c ascending from index 1 and returns the first entry
// whose own name equals key exactly, or data.NotFound.
//
// It matches data.FindHumanByName: index 0 is reserved, a nil collection has no
// entries, and an empty name matches nothing.
func findUnitByName(c data.Collection, key string) int {
	if c == nil || key == "" {
		return data.NotFound
	}
	for i := 1; i < c.Len(); i++ {
		if n := c.EntryName(i); n != "" && n == key {
			return i
		}
	}
	return data.NotFound
}
