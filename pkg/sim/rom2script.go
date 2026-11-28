package sim

import (
	"slices"
	"sort"
)

type ScriptDialect uint8

const (
	ScriptROM1 ScriptDialect = iota
	ScriptROM2
)

const (
	ScriptCheckPlayerStructures     int32 = 20
	ScriptCheckCellObject           int32 = 22
	ScriptCheckScenario             int32 = 23
	ScriptCheckObjective            int32 = 24
	ScriptCheckCellEffect           int32 = 25
	ScriptCheckUnitEffect           int32 = 26
	ScriptCheckCentered             int32 = 27
	ScriptInstantCounter            int32 = 1
	ScriptInstantSetScenario        int32 = 35
	ScriptInstantObjective          int32 = 36
	ScriptInstantDiagnostic         int32 = 37
	ScriptInstantTakeItemFromMap    int32 = 38
	ScriptInstantClearGroupActivity int32 = 39
	ScriptInstantMoveItem           int32 = 11
	ScriptInstantRelocateUnit       int32 = 27
	ScriptInstantOutpostCountdown   int32 = 31
)

type rom2ScriptState struct {
	Scenario ScenarioBank
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

// scriptOpRows are the op bodies the second game adds to the shared ones. A
// check row returns the register value and whether its unit exists; an instant
// row applies the op. An op without a row runs the shared body.
type scriptOpRows struct {
	check   map[int32]func(*World, ScriptCheck) (int32, bool)
	instant map[int32]func(*World, ScriptInstant, *castObs)
}

// secondGameRows is the one table of the second game's op bodies. Every
// profile holds these same maps, so profiles compare equal by content.
var secondGameRows = secondGameScriptRows()

func secondGameScriptRows() scriptOpRows {
	return scriptOpRows{
		check: map[int32]func(*World, ScriptCheck) (int32, bool){
			// R2-ENGINE-1452: count the player's structures with positive signed +0x42.
			ScriptCheckPlayerStructures: func(w *World, c ScriptCheck) (int32, bool) {
				if !c.HasPlayer {
					return 0, false
				}
				var n int32
				for _, st := range w.structures {
					if st.Owner == c.Player && int16(st.Field42) > 0 {
						n++
					}
				}
				return n, true
			},
			// R2-ENGINE-1450: cell p1*256+p0; if occupied relocate to (low8 p2, low8 p3) and return 1.
			ScriptCheckCellObject: func(w *World, c ScriptCheck) (int32, bool) {
				key := cellKey(c.Args[0], c.Args[1])
				x, y := int32(key&0xff), int32(key>>8)
				ground, _ := w.cellLayerOccupants(x, y)
				if len(ground) == 0 {
					return 0, true
				}
				w.relocateActor(ground[0], c.Args[2]&0xff, c.Args[3]&0xff, nil)
				return 1, true
			},
			ScriptCheckScenario: func(w *World, c ScriptCheck) (int32, bool) {
				v, _ := w.ROM2ScenarioValue(c.Args[0])
				return v, true
			},
			ScriptCheckObjective: func(w *World, c ScriptCheck) (int32, bool) {
				index := int64(752) + int64(c.Args[0]) // R2-ENGINE-045, DIV-2354
				if index < 0 || index >= 1024 {
					return 0, true
				}
				v, _ := w.ROM2ScenarioValue(int32(index))
				return v, true
			},
			ScriptCheckCellEffect: func(w *World, c ScriptCheck) (int32, bool) {
				if w.rom2CellEffect(cellKey(c.Args[0], c.Args[1]), c.Args[2]) {
					return 1, true
				}
				return 0, true
			},
			ScriptCheckUnitEffect: func(w *World, c ScriptCheck) (int32, bool) {
				e, ok := w.scriptEntity(c.Unit, c.HasUnit)
				if !ok {
					return 0, false
				}
				for _, effect := range w.attached {
					if effect.Target == e.ID && uint32(effect.Spell)&31 == uint32(c.Args[0])&31 {
						return 1, true
					}
				}
				return 0, true
			},
			ScriptCheckCentered: func(w *World, c ScriptCheck) (int32, bool) {
				e, ok := w.scriptEntity(c.Unit, c.HasUnit)
				if !ok {
					return 0, false
				}
				if e.X == c.Args[1] && e.Y == c.Args[2] && w.rom2Centered(e) {
					return 1, true
				}
				return 0, true
			},
		},
		instant: map[int32]func(*World, ScriptInstant, *castObs){
			// R2-ENGINE-1450: add 1 to register 902.
			ScriptInstantCounter: func(w *World, _ ScriptInstant, _ *castObs) {
				w.setRegisterAt(902, w.registerAt(902)+1)
			},
			ScriptInstantMessage: func(_ *World, in ScriptInstant, obs *castObs) {
				obs.recordScriptMessage(in.Args[0])
			},
			ScriptInstantSetScenario: func(w *World, in ScriptInstant, _ *castObs) {
				w.setROM2Scenario(in.Args[0], in.Args[1])
			},
			ScriptInstantObjective: func(w *World, in ScriptInstant, obs *castObs) {
				if event := WriteObjective(&w.rom2.Scenario, int64(in.Args[0]), in.Args[1]); event != objectiveNoEvent {
					obs.recordScriptMessage(event)
				}
			},
			ScriptInstantDiagnostic: func(*World, ScriptInstant, *castObs) {},
			ScriptInstantTakeItemFromMap: func(w *World, in ScriptInstant, _ *castObs) {
				if !in.HasItem {
					return
				}
				for i := range w.entities {
					if w.entities[i].OffMap {
						continue
					}
					w.runInstant(ScriptInstant{Op: ScriptInstantTakeItem, Unit: w.entities[i].ID, HasUnit: true, Item: in.Item, HasItem: true})
				}
			},
			// R2-ENGINE-1701: p0 the group number, from a group slot when
			// the node names one, p1 the countdown.
			ScriptInstantOutpostCountdown: func(w *World, in ScriptInstant, _ *castObs) {
				group, at := uint32(in.Args[0]), 1
				if in.HasGroup || in.LateGroup {
					group, at = in.Group, 0
				}
				w.setOutpostCountdown(group, in.Args[at])
			},
			ScriptInstantClearGroupActivity: func(w *World, in ScriptInstant, _ *castObs) {
				if in.HasGroup {
					if owner, ok := w.scriptGroupOwner(in.Group); ok {
						w.setROM2GroupForced(owner, in.Group, false)
					}
				}
			},
			ScriptInstantLose: func(w *World, in ScriptInstant, _ *castObs) {
				w.lost = uint32(in.Args[0])
			},
			// R2-ENGINE-1699: one unit of the item leaves U's bag (the whole
			// element at count 1, else one split off) and joins V's bag.
			ScriptInstantMoveItem: func(w *World, in ScriptInstant, _ *castObs) {
				w.moveScriptItem(in)
			},
			// R2-ENGINE-1700: the cell move to (low8 p0, low8 p1), then a
			// computer unit's Guard order, whether or not the cell took it.
			ScriptInstantRelocateUnit: func(w *World, in ScriptInstant, obs *castObs) {
				if !in.HasUnit {
					return
				}
				i := indexOfEntity(w.entities, in.Unit)
				if i < 0 || !w.entities[i].Alive() || w.entities[i].OffMap {
					return
				}
				w.relocateActor(i, in.Args[0]&0xff, in.Args[1]&0xff, obs)
				if w.computerControlledUnit(w.entities[i]) {
					w.standScriptedUnit(i, actorStateGuard, keepOrderDestination)
				}
			},
		},
	}
}

func (w *World) ROM2ScenarioValue(index int32) (int32, bool) {
	if w.rom2 == nil || index < 0 || index >= int32(len(w.rom2.Scenario)) {
		return 0, false
	}
	return w.rom2.Scenario[index], true
}

func (w *World) setROM2Scenario(index int32, value int32) {
	if _, ok := w.ROM2ScenarioValue(index); ok {
		SetScenario(&w.rom2.Scenario, int64(index), value)
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
	w.rom2, w.game = nil, newGameProfile(w.script.Dialect())
	// A World with no program has no game yet; the loader rebuilds it.
	if w.script != nil {
		w.dropUnkeptStructureOwners(w.structures)
	}
	w.settleBuildingCasters()
	if !w.game.holdsScriptState {
		return
	}
	w.rom2 = &rom2ScriptState{Scenario: NewGameScenarioBank()} // R2-SESSION-023
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

func (w *World) rom2Centered(e Entity) bool {
	if w.savedMotion != nil {
		for _, m := range w.savedMotion.Motions {
			if m.Entity == e.ID && m.Current {
				return m.Position.centred()
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
		l := int32(areaRow(SecondGameSpellArm(e.Spell)).layer) // R2-ENGINE-053, DIV-2354
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

// moveScriptItem is action 11: the first element of U's bag holding the
// node's item gives up one unit, the whole element at count 1, and that unit
// joins V's bag, merged as a pickup merges (R2-ENGINE-1699). It uses the
// give-all body's load mutation, so a refusal leaves both bags as they were.
func (w *World) moveScriptItem(in ScriptInstant) {
	gi, ok := w.scriptItemHolder(in)
	if !ok || !in.HasUnit2 {
		return
	}
	ri := indexOfEntity(w.entities, in.Unit2)
	k := slices.IndexFunc(w.carried[gi], func(st ItemStack) bool { return st.Code == in.Item })
	if ri < 0 || ri == gi || k < 0 || !w.hasActorContainer(ri) {
		return
	}
	giverLoad, receiverLoad := w.beginLoadMutation(gi), w.beginLoadMutation(ri)
	if !w.sourceMutationReady(gi) || !w.sourceMutationReady(ri) {
		return
	}
	live := w
	if w.savedObjects != nil || w.entities[gi].ActorLoad.Source.Class != 0 || w.entities[ri].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(gi)
		n.carried[ri] = cloneStacks(w.carried[ri])
		w = &n
	}
	item, ok := w.takeCarriedObject(gi, k, w.carried[gi][k].Count <= 1, ri)
	if !ok || !w.addCarried(ri, item, gi) {
		return
	}
	if !w.finishLoadMutation(ri, receiverLoad) || !w.finishLoadMutation(gi, giverLoad) || !w.savedMutationValid() {
		return
	}
	if w != live {
		*live = *w
	}
}
