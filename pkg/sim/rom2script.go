package sim

import "sort"

type ScriptDialect uint8

const (
	ScriptROM1 ScriptDialect = iota
	ScriptROM2
)

const (
	ScriptCheckScenario             int32 = 23
	ScriptCheckObjective            int32 = 24
	ScriptCheckCellEffect           int32 = 25
	ScriptCheckUnitEffect           int32 = 26
	ScriptCheckCentered             int32 = 27
	ScriptInstantSetScenario        int32 = 35
	ScriptInstantObjective          int32 = 36
	ScriptInstantDiagnostic         int32 = 37
	ScriptInstantTakeItemFromMap    int32 = 38
	ScriptInstantClearGroupActivity int32 = 39
)

type rom2ScriptState struct {
	Scenario [1024]int32
	Groups   []rom2GroupActivity
}

type rom2GroupActivity struct {
	Owner  uint32
	Group  uint32
	Forced bool
}

func NewROM2Script(checks []ScriptCheck, instants []ScriptInstant, triggers []ScriptTrigger) (*Script, error) {
	return newScript(ScriptROM2, checks, instants, triggers)
}

func (s *Script) Dialect() ScriptDialect {
	if s == nil {
		return ScriptROM1
	}
	return s.dialect
}

func rom2CheckSupported(op int32) bool { return op >= ScriptCheckScenario && op <= ScriptCheckCentered }
func rom2InstantSupported(op int32) bool {
	return op >= ScriptInstantSetScenario && op <= ScriptInstantClearGroupActivity
}

func (w *World) ROM2ScenarioValue(index int32) (int32, bool) {
	if w.rom2 == nil || index < 0 || index >= int32(len(w.rom2.Scenario)) {
		return 0, false
	}
	return w.rom2.Scenario[index], true
}

func (w *World) setROM2Scenario(index int32, value int32) {
	if _, ok := w.ROM2ScenarioValue(index); ok {
		w.rom2.Scenario[index] = value
	}
}

func (w *World) ROM2ScenarioState() ([1024]int32, bool) {
	if w.rom2 == nil {
		return [1024]int32{}, false
	}
	return w.rom2.Scenario, true
}

func (w *World) SetROM2ScenarioState(bank [1024]int32) bool {
	if w.rom2 == nil {
		return false
	}
	w.rom2.Scenario = bank
	return true
}

func (w *World) initializeROM2Script() {
	if w.script.Dialect() != ScriptROM2 {
		return
	}
	w.rom2 = &rom2ScriptState{}
	w.rom2.Scenario[768] = 10
	for _, g := range w.groups {
		w.rom2.Groups = append(w.rom2.Groups, rom2GroupActivity{Owner: g.owner, Group: g.group})
	}
	for _, c := range w.script.checks {
		if c.Op == ScriptCheckGroupCount {
			continue
		}
		w.forceROM2References(c.Group, c.HasGroup, c.Unit, c.HasUnit, c.Unit2, c.HasUnit2)
	}
	for _, in := range w.script.instants {
		w.forceROM2References(in.Group, in.HasGroup, in.Unit, in.HasUnit, in.Unit2, in.HasUnit2)
	}
}

func (w *World) forceROM2References(group uint32, hasGroup bool, unit EntityID, hasUnit bool, unit2 EntityID, hasUnit2 bool) {
	if hasGroup {
		if owner, ok := w.scriptGroupOwner(group); ok {
			w.setROM2GroupForced(owner, group, true)
		}
	}
	for _, ref := range []struct {
		id  EntityID
		has bool
	}{{unit, hasUnit}, {unit2, hasUnit2}} {
		if !ref.has {
			continue
		}
		if i := indexOfEntity(w.entities, ref.id); i >= 0 {
			w.setROM2GroupForced(w.entities[i].Owner, w.entities[i].Group, true)
		}
	}
}

func (w *World) setROM2GroupForced(owner, group uint32, forced bool) {
	if w.rom2 == nil {
		return
	}
	i := sort.Search(len(w.rom2.Groups), func(i int) bool {
		g := w.rom2.Groups[i]
		return g.Owner > owner || g.Owner == owner && g.Group >= group
	})
	if i < len(w.rom2.Groups) && w.rom2.Groups[i].Owner == owner && w.rom2.Groups[i].Group == group {
		w.rom2.Groups[i].Forced = forced
	}
}

func (w *World) rom2Check(c ScriptCheck) (int32, bool) {
	switch c.Op {
	case ScriptCheckScenario:
		v, _ := w.ROM2ScenarioValue(c.Args[0])
		return v, true
	case ScriptCheckObjective:
		index := int64(752) + int64(c.Args[0])
		if index < 0 || index >= 1024 {
			return 0, true
		}
		v, _ := w.ROM2ScenarioValue(int32(index))
		return v, true
	case ScriptCheckCellEffect:
		if w.rom2CellEffect(cellKey(c.Args[0], c.Args[1]), c.Args[2]) {
			return 1, true
		}
		return 0, true
	case ScriptCheckUnitEffect, ScriptCheckCentered:
		e, ok := w.scriptEntity(c.Unit, c.HasUnit)
		if !ok {
			return 0, false
		}
		if c.Op == ScriptCheckUnitEffect {
			for _, effect := range w.attached {
				if effect.Target == e.ID && uint32(effect.Spell)&31 == uint32(c.Args[0])&31 {
					return 1, true
				}
			}
			return 0, true
		}
		if e.X == c.Args[1] && e.Y == c.Args[2] && w.rom2Centered(e) {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (w *World) rom2Centered(e Entity) bool {
	if w.savedMotion != nil {
		for _, m := range w.savedMotion.Motions {
			if m.Entity == e.ID && m.Current {
				return m.Position.FineX == 128 && m.Position.FineY == 128
			}
		}
	}
	return e.Transit == 0
}

func (w *World) rom2CellEffect(key uint16, layer int32) bool {
	if layer < 0 || layer >= 6 {
		return false
	}
	for _, e := range w.effects {
		l := int32(-1)
		switch e.Spell {
		case 3:
			l = 0
		case 6:
			l = 2
		case 17:
			l = 3
		case 15:
			l = 4
		case 14:
			l = 5
		}
		if l != layer {
			continue
		}
		for _, painted := range e.Cells {
			if painted == key {
				return true
			}
		}
	}
	return false
}

func (w *World) rom2Instant(in ScriptInstant, obs *castObs) bool {
	switch in.Op {
	case ScriptInstantSetScenario:
		w.setROM2Scenario(in.Args[0], in.Args[1])
	case ScriptInstantObjective:
		index, mode := int64(752)+int64(in.Args[0]), in.Args[1]
		if index < 0 || index >= 1024 {
			return true
		}
		old := w.rom2.Scenario[index]
		switch mode {
		case 1:
			if old == 0 {
				w.rom2.Scenario[index] = 1
			}
		case 2:
			if old != 4 {
				if old&2 == 0 {
					obs.recordScriptMessage(253)
				}
				w.rom2.Scenario[index] = 3
			}
		case 4:
			obs.recordScriptMessage(254)
			w.rom2.Scenario[index] = 5
		default:
			w.rom2.Scenario[index] = mode
		}
	case ScriptInstantDiagnostic:
	case ScriptInstantTakeItemFromMap:
		if !in.HasItem {
			return true
		}
		for i := range w.entities {
			if w.entities[i].OffMap {
				continue
			}
			w.runInstant(ScriptInstant{Op: ScriptInstantTakeItem, Unit: w.entities[i].ID, HasUnit: true, Item: in.Item, HasItem: true})
		}
	case ScriptInstantClearGroupActivity:
		if in.HasGroup {
			if owner, ok := w.scriptGroupOwner(in.Group); ok {
				w.setROM2GroupForced(owner, in.Group, false)
			}
		}
	case ScriptInstantLose:
		w.lost = uint32(in.Args[0])
	default:
		return false
	}
	return true
}
