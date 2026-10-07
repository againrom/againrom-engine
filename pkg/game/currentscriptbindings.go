package game

import (
	"againrom/pkg/base"
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Party roles arrive after ordinary actor admission and identity relocation.
// Use their exact current IDs to complete the authored role references; a
// reference already resolved by ordinary LOAD keeps its own endpoint.
func restoreCurrentScriptBindings(ms *Mission, table *mapload.Table) error {
	if ms.Map == nil || ms.World.Script() == nil || len(ms.Party) == 0 {
		return nil
	}
	if len(ms.Party) != len(ms.Start.IDs) {
		return fmt.Errorf("current script party has incomplete entity bindings")
	}
	refs := campaignScriptPartyRefs(ms.Map, table, ms.Party, func(i int) sim.EntityID { return ms.Start.IDs[i] })
	compile := mapload.CompileScript
	if table != nil && table.Game == base.GameROM2 {
		compile = mapload.CompileROM2Script
	}
	roles, _, err := compile(ms.Map, refs)
	if err != nil {
		return err
	}
	if err := restoreCurrentScriptRoles(ms.World, roles); err != nil {
		return err
	}
	return restoreMission40CompanionObjective(ms)
}

func restoreMission40CompanionObjective(ms *Mission) error {
	if ms == nil || ms.Number != 40 || ms.World == nil || ms.World.Script() == nil || ms.World.Script().Dialect() == sim.ScriptROM2 {
		return nil
	}
	for i, member := range ms.Party {
		if member.CompanionNPC != 22 && member.ID != "npc:22" {
			continue
		}
		if i >= len(ms.Start.IDs) {
			return fmt.Errorf("mission 40 companion lacks a runtime identity")
		}
		unit := ms.Start.IDs[i]
		current := ms.World.Script()
		for _, check := range current.Checks() {
			if check.Op != sim.ScriptCheckVIP {
				continue
			}
			if check.HasUnit && check.Unit == unit {
				return nil
			}
			return fmt.Errorf("mission 40 companion objective names a different unit")
		}
		next, err := current.WithVIP(unit)
		if err != nil {
			return err
		}
		return ms.World.RestoreScriptProgram(next)
	}
	return nil
}

func restoreCurrentScriptRoles(w *sim.World, roles *sim.Script) error {
	current := w.Script()
	if current == nil || roles == nil || current.Dialect() != roles.Dialect() || !slices.Equal(current.Triggers(), roles.Triggers()) {
		return nil
	}
	checks, instants := current.Checks(), current.Instants()
	roleChecks, roleInstants := roles.Checks(), roles.Instants()
	if len(checks) != len(roleChecks) || len(instants) != len(roleInstants) {
		return nil
	}
	changed := false
	fill := func(id *sim.EntityID, present *bool, role sim.EntityID, hasRole bool) {
		if !*present && hasRole {
			*id, *present, changed = role, true, true
		}
	}
	for i, role := range roleChecks {
		c := &checks[i]
		shape := role
		shape.Unit, shape.HasUnit, shape.Unit2, shape.HasUnit2 = c.Unit, c.HasUnit, c.Unit2, c.HasUnit2
		shape.Structure, shape.HasStructure = c.Structure, c.HasStructure
		if shape != *c {
			return nil
		}
		fill(&c.Unit, &c.HasUnit, role.Unit, role.HasUnit)
		fill(&c.Unit2, &c.HasUnit2, role.Unit2, role.HasUnit2)
	}
	for i, role := range roleInstants {
		in := &instants[i]
		shape := role
		shape.Unit, shape.HasUnit, shape.Unit2, shape.HasUnit2 = in.Unit, in.HasUnit, in.Unit2, in.HasUnit2
		shape.Structure, shape.HasStructure = in.Structure, in.HasStructure
		if shape != *in {
			return nil
		}
		fill(&in.Unit, &in.HasUnit, role.Unit, role.HasUnit)
		fill(&in.Unit2, &in.HasUnit2, role.Unit2, role.HasUnit2)
	}
	if !changed {
		return nil
	}
	compile := sim.NewScript
	if current.Dialect() == sim.ScriptROM2 {
		compile = sim.NewROM2Script
	}
	next, err := compile(checks, instants, current.Triggers())
	if err != nil {
		return err
	}
	if !w.RestoreScriptBindings(current, next) {
		return fmt.Errorf("current script role completion changed non-binding state")
	}
	return nil
}
