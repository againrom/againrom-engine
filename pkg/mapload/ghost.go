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

// ghostTemplate is the first exact-name `Ghost` Units row resolved through
// the creature arm, at no difficulty (MAGIC-249, MAGIC-250), or the zero
// template, which raises nothing, when the row is absent or does not resolve.
func ghostTemplate(t *Table) sim.GhostTemplate {
	if t == nil {
		return sim.GhostTemplate{}
	}
	i := findUnitByName(t.units(), ghostRowName)
	if i == data.NotFound {
		return sim.GhostTemplate{}
	}
	def, worn, err := unitRowDefinition(t, i, DifficultyNormal)
	if err != nil {
		return sim.GhostTemplate{}
	}
	return sim.GhostTemplate(unitRowBlock(def.TypeID, i, def, worn, t).actorDefinition(t))
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
