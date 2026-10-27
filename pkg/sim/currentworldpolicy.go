package sim

import (
	"fmt"
	"slices"
)

type CurrentGroupPolicy struct {
	Owner, Selector        uint32
	Base, Order            uint8
	CommandedX, CommandedY int32
	RoamCounter            uint8
}

type CurrentTerrainCell struct {
	Cell         uint16
	Cost, Height *uint8 `json:",omitempty"`
}

type CurrentSpellPolicy struct {
	ID                    uint16
	Delivery, EffectSpeed int32
	EffectDuration        *uint16 `json:",omitempty"`
}

// CurrentWorldPolicy distinguishes execution policies which the original file
// grammar cannot express. Its planes and counters are current values, never an
// ALM reset or a second encoded World. Actor/structure identities stay in their
// normal object bindings.
type CurrentWorldPolicy struct {
	TickHigh                                                                   uint32
	ClockKnown                                                                 bool
	EntityIDFloor                                                              *uint64 `json:",omitempty"`
	Mode                                                                       Mode
	Ghost                                                                      *GhostTemplate       `json:",omitempty"`
	ItemWeights                                                                *[]ItemWeight        `json:",omitempty"`
	Terrain                                                                    Terrain              `json:"-"`
	TerrainCells                                                               []CurrentTerrainCell `json:",omitempty"`
	CurrentRegisters                                                           *[100]int32          `json:"-"`
	Groups                                                                     []CurrentGroupPolicy
	GroupCarrier, MotionCarrier, PlaneCarrier, StructureCarrier, ObjectCarrier bool
	SpellGraphCarrier, EffectDriverCarrier                                     *bool                `json:",omitempty"`
	SpellPolicies                                                              []CurrentSpellPolicy `json:",omitempty"`
	Formations                                                                 [relationSlots]uint8
	Scorched                                                                   []uint16
	ROM2                                                                       *CurrentROM2Policy `json:",omitempty"`
}

func (p *CurrentWorldPolicy) KeepSpellDeviations(base []SpellRule) {
	for i := range p.SpellPolicies {
		v := &p.SpellPolicies[i]
		for _, r := range base {
			if r.ID == v.ID {
				if v.EffectDuration != nil && *v.EffectDuration == r.EffectDuration {
					v.EffectDuration = nil
				}
				break
			}
		}
	}
	p.SpellPolicies = slices.DeleteFunc(p.SpellPolicies, func(v CurrentSpellPolicy) bool {
		for _, r := range base {
			if r.ID == v.ID {
				return r.Delivery == v.Delivery && r.EffectSpeed == v.EffectSpeed && v.EffectDuration == nil
			}
		}
		return false
	})
}

func (p *CurrentWorldPolicy) KeepTerrainDeviations(base Terrain, bounds Bounds) {
	p.TerrainCells = nil
	for at := range p.Terrain.Cost {
		row := CurrentTerrainCell{Cell: uint16(at/int(bounds.Width))<<8 | uint16(at%int(bounds.Width))}
		if at >= len(base.Cost) || p.Terrain.Cost[at] != base.Cost[at] {
			v := p.Terrain.Cost[at]
			row.Cost = &v
		}
		if at >= len(base.Height) || p.Terrain.Height[at] != base.Height[at] {
			v := p.Terrain.Height[at]
			row.Height = &v
		}
		if row.Cost != nil || row.Height != nil {
			p.TerrainCells = append(p.TerrainCells, row)
		}
	}
}

func (w *World) CurrentPolicy() CurrentWorldPolicy {
	graph, drivers := w.savedSpellGraph != nil, w.savedWorldEffects != nil
	floor, ghost := w.entityIDFloor, w.ghost
	weights := w.ItemWeights()
	p := CurrentWorldPolicy{TickHigh: uint32(w.tick >> 32), ClockKnown: w.hasSessionClock, Mode: w.mode,
		EntityIDFloor: &floor, Ghost: &ghost,
		ItemWeights:  &weights,
		Terrain:      Terrain{Block: slices.Clone(w.grid), Cost: slices.Clone(w.cost), Height: slices.Clone(w.height)},
		GroupCarrier: !w.groupDialectNative(), MotionCarrier: w.savedMotion != nil, PlaneCarrier: w.savedCellPlanes != nil,
		StructureCarrier: w.hasSavedStructures, ObjectCarrier: w.savedObjects != nil, Formations: w.formations,
		Scorched: slices.Clone(w.scorchedCells), SpellGraphCarrier: &graph, EffectDriverCarrier: &drivers}
	if w.savedCellPlanes != nil {
		// The planes own Static bit 2; the grid's bit 3 only mirrors it
		// (syncNativeSavedPlaneCell). The Terrain a World is built from
		// carries no mirror: a World built with planes derives it again.
		for i := range p.Terrain.Block {
			p.Terrain.Block[i] &^= blockStaticObject
		}
	}
	p.ROM2 = w.currentROM2Policy()
	for _, g := range w.groups {
		p.Groups = append(p.Groups, CurrentGroupPolicy{g.owner, g.group, g.base, g.order, g.commandedX, g.commandedY, g.roamCounter})
	}
	for _, r := range w.spells {
		duration := r.EffectDuration
		p.SpellPolicies = append(p.SpellPolicies, CurrentSpellPolicy{ID: r.ID, Delivery: r.Delivery, EffectSpeed: r.EffectSpeed, EffectDuration: &duration})
	}
	return p
}

func (w *World) RestoreCurrentSpellDurations(rows []CurrentSpellPolicy) error {
	if len(rows) == 0 {
		return nil
	}
	if len(rows) > len(w.spells) {
		return fmt.Errorf("sim: current spell policy exceeds spell table")
	}
	spells := slices.Clone(w.spells)
	seen := map[uint16]bool{}
	for _, v := range rows {
		at := slices.IndexFunc(spells, func(r SpellRule) bool { return r.ID == v.ID })
		if at < 0 || seen[v.ID] || v.Delivery < 0 || v.EffectSpeed < 0 {
			return fmt.Errorf("sim: invalid current spell policy")
		}
		seen[v.ID] = true
		if v.EffectDuration != nil {
			spells[at].EffectDuration = *v.EffectDuration
		}
	}
	w.spells = spells
	return nil
}

// RestoreCurrentClock completes the ordinary session tick before pending
// deliveries validate their creation time. It does not advance the simulation.
// A first-game World keeps the session clock its Document import gave it; an
// older engine SAV that marked the clock unknown keeps it too, so one cadence
// runs after every LOAD. A second-game World takes the saved native tick.
func (w *World) RestoreCurrentClock(high uint32, known bool) error {
	if known && high != 0 {
		return fmt.Errorf("sim: invalid current clock")
	}
	if !w.SecondGame() {
		w.constructSessionClock()
		return nil
	}
	w.tick, w.hasSessionClock = uint64(high)<<32|uint64(uint32(w.tick)), known
	if !known {
		w.fullTick = 0
	}
	return nil
}

func (w *World) restoreCurrentPolicy(p CurrentWorldPolicy) error {
	if err := w.restoreCurrentROM2Policy(p.ROM2); err != nil {
		return err
	}
	if !p.Mode.defined() {
		return fmt.Errorf("sim: invalid current world policy")
	}
	if p.Ghost != nil {
		if err := p.Ghost.fault(); err != nil {
			return fmt.Errorf("sim: invalid current Ghost: %w", err)
		}
	}
	if err := w.RestoreCurrentClock(p.TickHigh, p.ClockKnown); err != nil {
		return err
	}
	w.spells = slices.Clone(w.spells)
	spellIDs := map[uint16]bool{}
	for _, v := range p.SpellPolicies {
		at := slices.IndexFunc(w.spells, func(r SpellRule) bool { return r.ID == v.ID })
		if at < 0 || spellIDs[v.ID] || v.Delivery < 0 || v.EffectSpeed < 0 {
			return fmt.Errorf("sim: invalid current spell policy")
		}
		spellIDs[v.ID] = true
		w.spells[at].Delivery, w.spells[at].EffectSpeed = v.Delivery, v.EffectSpeed
		if v.EffectDuration != nil {
			w.spells[at].EffectDuration = *v.EffectDuration
		}
	}
	w.mode = p.Mode
	if p.Ghost != nil {
		w.ghost = *p.Ghost
	}
	if p.CurrentRegisters != nil {
		w.registers = *p.CurrentRegisters
	}
	if p.ItemWeights != nil {
		if len(*p.ItemWeights) > 65535 {
			return fmt.Errorf("sim: current item-weight table exceeds its code population")
		}
		weights, err := normaliseItemWeights(*p.ItemWeights)
		if err != nil || !slices.Equal(weights, *p.ItemWeights) {
			return fmt.Errorf("sim: current item-weight table is not canonical: %v", err)
		}
		w.itemWeights = weights
	}
	w.cost, w.height = slices.Clone(w.cost), slices.Clone(w.height)
	seen := map[uint16]bool{}
	for _, c := range p.TerrainCells {
		at, ok := w.cellIndex(int32(c.Cell&255), int32(c.Cell>>8))
		if !ok || seen[c.Cell] || c.Cost == nil && c.Height == nil {
			return fmt.Errorf("sim: invalid current terrain deviation")
		}
		seen[c.Cell] = true
		if c.Cost != nil {
			w.cost[at] = *c.Cost
		}
		if c.Height != nil {
			w.height[at] = *c.Height
		}
	}
	w.formations = p.Formations
	if !p.GroupCarrier && !w.groupDialectNative() && w.savedGroups.FormationsPresent {
		for _, f := range w.savedGroups.Formations {
			if f.TriggerID < relationSlots {
				w.formations[f.TriggerID] = f.Mode
			}
		}
	}
	w.scorchedCells = slices.Clone(p.Scorched)
	w.groups = nil
	for _, g := range p.Groups {
		w.groups = append(w.groups, groupAI{owner: g.Owner, group: g.Selector, base: g.Base, order: g.Order, commandedX: g.CommandedX, commandedY: g.CommandedY, roamCounter: g.RoamCounter})
	}
	if !p.GroupCarrier {
		// An older engine SAV without the Group carrier takes the registry
		// constructed from its loaded native state.
		w.savedGroups = nil
		w.constructGroups(true)
	}
	if !p.MotionCarrier {
		w.savedMotion = nil
	}
	// A first-game World keeps the cell planes its LOAD import built even when
	// an older engine SAV did not mark them: every first-game World holds them.
	dropPlanes := !p.PlaneCarrier && (w.SecondGame() || w.savedCellPlanes == nil)
	if !p.MotionCarrier && dropPlanes {
		w.cellRecords = nil
	}
	if dropPlanes {
		w.savedCellPlanes = nil
	}
	// The magic wall bits are the native grid's own; a LOAD's planes carry
	// none, so every restore marks the cells of each standing wall cloud.
	w.grid = slices.Clone(w.grid)
	for _, e := range w.effects {
		if w.spellArm(e.Spell) == 19 && e.Mode == areaModeCloud {
			w.setWallCells(e.Cells, true)
		}
	}
	if !p.StructureCarrier {
		// The imported static plane keeps its structure bindings and source rows.
		// A World with cell planes mirrors Static bit 2 (border, scenery and
		// structure cells alike) into the grid, so there the bit names no
		// binding: it keeps the imported roster when that names a Building,
		// because its cell records name that roster's keys.
		if w.savedCellPlanes != nil {
			w.hasSavedStructures = len(w.savedStructures) != 0
		} else {
			w.hasSavedStructures = slices.ContainsFunc(w.grid, func(b byte) bool { return b&blockStaticObject != 0 })
		}
		if !w.hasSavedStructures {
			w.savedStructures = nil
			w.savedStructureCells = nil
		}
	}
	w.rebuildStructureSlots()
	if !p.ObjectCarrier {
		sackIDs := w.SackRuntimeIDs()
		w.savedObjects = nil
		w.carried = slices.Clone(w.carried)
		for i := range w.carried {
			w.carried[i] = cloneStacks(w.carried[i])
		}
		w.equipment = slices.Clone(w.equipment)
		w.sacks = cloneGroundSacks(w.sacks)
		for i := range w.carried {
			for j := range w.carried[i] {
				w.carried[i][j].ObjectID = 0
			}
		}
		for i := range w.equipment {
			for j := range w.equipment[i] {
				w.equipment[i][j].ObjectID = 0
			}
		}
		for i := range w.sacks {
			w.sacks[i].ObjectID, w.sacks[i].runtimeID = 0, sackIDs[i]
			for j := range w.sacks[i].ItemInstances {
				w.sacks[i].ItemInstances[j].ObjectID = 0
			}
		}
	}
	order := w.CurrentWorldEffectOrder()
	// An older engine SAV that marked the driver table absent held no
	// projectile row, and its areas continue as the native areas: a
	// first-game World takes the empty table it is constructed with.
	if p.EffectDriverCarrier != nil && !*p.EffectDriverCarrier {
		w.savedWorldEffects = nil
		w.constructEffectDriver()
	}
	if p.SpellGraphCarrier != nil && !*p.SpellGraphCarrier {
		w.savedSpellGraph = nil
		for i, r := range order {
			if r.Kind == EffectSavedGraph {
				order[i].Kind = EffectSavedArea
			}
		}
	}
	active := map[WorldEffectRef]bool{}
	for _, r := range w.currentEffectPopulation() {
		active[r] = true
	}
	w.effectOrder = slices.DeleteFunc(order, func(r WorldEffectRef) bool { return !active[r] })
	w.compactEffectOrder()
	return nil
}
