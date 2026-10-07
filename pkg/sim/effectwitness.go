package sim

import "fmt"

// SpellEffectWitness is the compact result of ControlledSpellEffectWitness.
// It contains no asset bytes and no private world state; a developer tool can
// print it after loading a real campaign mission to prove that the installed
// terrain and spell rows reach the ordinary simulation owners.
type SpellEffectWitness struct {
	Bounds Bounds
	X, Y   int32

	PointBefore, PointAfter int32
	PointEffects            int

	AreaFriendBefore, AreaEnemyBefore int32
	AreaFriendAt15, AreaEnemyAt15     int32
	AreaFriendAt16, AreaEnemyAt16     int32

	// AreaOutside is the actor two cells west of the target cell. The
	// installed Wall of Fire row is `Distribution system = 4`, so it paints
	// the 10-cell wall table and that actor stands outside it; the
	// distribution-3 diamond the row does not ask for reaches three cells and
	// would take it. It is the one cell that tells the two shapes apart on
	// installed data.
	AreaOutsideBefore, AreaOutsideAfter int32

	WallBefore [3]bool
	WallDuring [3]bool
	WallAfter  [3]bool

	FireWallOverlap           int
	LightCells, DarknessCells int
}

// ControlledSpellEffectWitness reuses a loaded mission's exact terrain planes
// and installed spell table in three small controlled worlds. It is deliberately
// a developer witness, not gameplay: the actors are synthetic and their cells
// are selected from an open patch in the real map. Commands, wind-up, ordinary
// apply, area cadence, passability derivation and removal are all the production
// paths. This separation keeps game assets out of tests while still proving the
// real mission wiring which asset-free tests cannot see.
func ControlledSpellEffectWitness(w *World) (SpellEffectWitness, error) {
	if w == nil {
		return SpellEffectWitness{}, fmt.Errorf("sim: no mission world for spell-effect witness")
	}
	for _, id := range []uint16{3, 19, 24} {
		if _, ok := w.findSpell(uint32(id)); !ok {
			return SpellEffectWitness{}, fmt.Errorf("sim: mission spell table has no row %d", id)
		}
	}
	x, y, ok := w.witnessPatch(9)
	if !ok {
		return SpellEffectWitness{}, fmt.Errorf("sim: mission has no 9x9 patch open to all movement domains")
	}
	out := SpellEffectWitness{Bounds: w.bounds, X: x, Y: y}

	point, err := w.witnessWorld([]Entity{
		witnessMage(1, x-4, y, 24),
		witnessUnit(2, x-2, y, DomainGround, SelfSlot),
	})
	if err != nil {
		return out, err
	}
	out.PointBefore = point.entities[1].Speed
	if _, err := witnessCast(point, Cast(1, 2, 24)); err != nil {
		return out, fmt.Errorf("sim: point witness: %w", err)
	}
	out.PointAfter = point.entities[1].Speed
	out.PointEffects = len(point.ActiveEffects())

	area, err := w.witnessWorld([]Entity{
		witnessMage(1, x-4, y, 3),
		witnessUnit(2, x, y, DomainGround, SelfSlot),
		witnessUnit(3, x+1, y, DomainGround, 2),
		witnessUnit(4, x-2, y, DomainGround, 2),
	})
	if err != nil {
		return out, err
	}
	if _, err := witnessCast(area, CastAt(1, 3, CellPoint{X: x, Y: y})); err != nil {
		return out, fmt.Errorf("sim: area witness: %w", err)
	}
	out.AreaFriendBefore, out.AreaEnemyBefore = area.entities[1].HP, area.entities[2].HP
	out.AreaOutsideBefore = area.entities[3].HP
	for range 15 {
		Step(area, nil)
	}
	out.AreaFriendAt15, out.AreaEnemyAt15 = area.entities[1].HP, area.entities[2].HP
	Step(area, nil)
	out.AreaFriendAt16, out.AreaEnemyAt16 = area.entities[1].HP, area.entities[2].HP
	out.AreaOutsideAfter = area.entities[3].HP

	for domain := DomainGround; domain <= DomainAir; domain++ {
		wall, err := w.witnessWorld([]Entity{
			witnessMage(1, x-4, y, 19),
			witnessUnit(2, x-2, y, domain, SelfSlot),
		})
		if err != nil {
			return out, err
		}
		out.WallBefore[domain] = witnessRouteOpen(wall, x, y)
		if _, err := witnessCast(wall, CastAt(1, 19, CellPoint{X: x, Y: y})); err != nil {
			return out, fmt.Errorf("sim: wall witness domain %d: %w", domain, err)
		}
		out.WallDuring[domain] = witnessRouteOpen(wall, x, y)
		wall.setCellEffectTime(x, y, 19, 1)
		// Instant 29 writes the raw counter. The first tick spends its one and
		// leaves the record canonically at zero; the next sees zero and removes
		// the wall before route admission is measured.
		Step(wall, nil)
		Step(wall, nil)
		out.WallAfter[domain] = witnessRouteOpen(wall, x, y)
	}

	// The owner-directed overlap is exercised separately so its second cast
	// cannot change the damage-cadence witness above. Both walls travel through
	// the ordinary command and apply paths on this mission's installed row.
	overlap, err := w.witnessWorld([]Entity{
		witnessMage(1, x-4, y-1, 3),
		witnessMage(2, x-4, y+1, 3),
	})
	if err != nil {
		return out, err
	}
	if _, err := witnessCast(overlap, CastAt(1, 3, CellPoint{X: x, Y: y})); err != nil {
		return out, fmt.Errorf("sim: first overlapping fire wall: %w", err)
	}
	if _, err := witnessCast(overlap, CastAt(2, 3, CellPoint{X: x, Y: y})); err != nil {
		return out, fmt.Errorf("sim: second overlapping fire wall: %w", err)
	}
	out.FireWallOverlap = overlap.FireWallCount(x, y)

	for spell, dst := range map[uint16]*int{12: &out.LightCells, 17: &out.DarknessCells} {
		if _, ok := w.findSpell(uint32(spell)); !ok {
			continue
		}
		lighting, err := w.witnessWorld([]Entity{witnessMage(1, x-4, y, spell)})
		if err != nil {
			return out, err
		}
		if _, err := witnessCast(lighting, CastAt(1, SpellID(spell), CellPoint{X: x, Y: y})); err != nil {
			return out, fmt.Errorf("sim: lighting spell %d: %w", spell, err)
		}
		for _, effect := range lighting.CellEffects() {
			if effect.Spell == spell {
				*dst += len(effect.Cells)
			}
		}
	}
	return out, nil
}

// EffectMarkWitness is one marking spell cast on a controlled actor inside a
// loaded mission's own terrain and spell table (1002): the row that was cast,
// and the effect the ordinary paths left attached to the target.
type EffectMarkWitness struct {
	Spell    uint16
	Attached bool
	Target   EntityID
}

func ControlledEffectMarkWitness(w *World) ([]EffectMarkWitness, error) {
	if w == nil {
		return nil, fmt.Errorf("sim: no mission world for effect-mark witness")
	}
	x, y, ok := w.witnessPatch(9)
	if !ok {
		return nil, fmt.Errorf("sim: mission has no 9x9 patch open to all movement domains")
	}
	var out []EffectMarkWitness
	for _, id := range []uint16{5, 8, 10, 15, 16, 18, 20, 22, 23, 27} {
		row := EffectMarkWitness{Spell: id}
		rule, ok := w.findSpell(uint32(id))
		if !ok {
			out = append(out, row)
			continue
		}
		// THE COMMAND IS THE ROW'S OWN. An area row is delivered at a cell and
		// a point row at a unit; a self-only row refuses an ally, so the ally
		// is tried first and the caster itself second. All three are the
		// ordinary command path — nothing here reaches past castSpell.
		for _, target := range []EntityID{2, 1} {
			controlled, err := w.witnessWorld([]Entity{
				witnessMage(1, x-4, y, id),
				witnessUnit(2, x-2, y, DomainGround, SelfSlot),
			})
			if err != nil {
				return out, err
			}
			command := Cast(1, target, SpellID(id))
			if rule.Area {
				command = CastAt(1, SpellID(id), CellPoint{X: x - 2, Y: y})
			}
			if _, err := witnessCast(controlled, command); err != nil {
				continue
			}
			for _, e := range controlled.ActiveEffects() {
				if e.Spell == id {
					row.Attached, row.Target = true, e.Target
				}
			}
			if row.Attached {
				break
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func (w *World) witnessPatch(width int32) (int32, int32, bool) {
	r := width / 2
	for y := r + 1; y+r < w.bounds.Height-1; y++ {
		for x := r + 1; x+r < w.bounds.Width-1; x++ {
			open := true
			for py := y - r; py <= y+r && open; py++ {
				for px := x - r; px <= x+r; px++ {
					if !w.terrainOpen(DomainGround, px, py) || !w.terrainOpen(DomainGhost, px, py) || !w.terrainOpen(DomainAir, px, py) {
						open = false
						break
					}
				}
			}
			if open {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func (w *World) witnessWorld(entities []Entity) (*World, error) {
	terrain := Terrain{
		Block:  append([]byte(nil), w.grid...),
		Cost:   append([]byte(nil), w.cost...),
		Height: append([]byte(nil), w.height...),
	}
	return NewStockedSpelledWorld(0x1001, w.bounds, w.mode, terrain, entities, nil, Relations{}, nil, nil, w.spells)
}

func witnessMage(id EntityID, x, y int32, spell uint16) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, Mana: 1000, MaxMana: 1000,
		Speed: 10, ScanRange: 20, TokenSize: 1, Owner: SelfSlot, Mind: 100,
		AttackCharge: 8, AttackRelax: 4, KnownSpells: uint32(1) << spell, GainsXP: true}
}

func witnessUnit(id EntityID, x, y int32, domain Domain, owner uint32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 1000, MaxHP: 1000, Speed: 10,
		ScanRange: 20, TokenSize: 1, Domain: domain, Owner: owner}
}

func witnessCast(w *World, command Command) (CastEvent, error) {
	for tick := 0; tick < 64; tick++ {
		var events []CastEvent
		if tick == 0 {
			events = StepObserved(w, []Command{command})
		} else {
			events = StepObserved(w, nil)
		}
		if len(events) != 0 {
			return events[0], nil
		}
	}
	return CastEvent{}, fmt.Errorf("spell %d did not apply within 64 ticks", commandSpell(command))
}

func commandSpell(command Command) uint16 {
	if command.Spell != 0 {
		return command.Spell
	}
	return uint16(command.Y)
}

func witnessRouteOpen(w *World, x, y int32) bool {
	_, ok := w.canonicalRoute(newRouteScratch(w), 1, terrainRelation, noWindow, flatBudget, exactGoal, x, y)
	return ok
}
