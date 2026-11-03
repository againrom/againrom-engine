package sim

const consumedCorpseStage uint8 = 5

type corpseRaiseKind struct {
	stage      DecayStage
	name       string
	body       uint8
	fullHealth bool
}

type corpseRaiseProfile struct {
	kinds                 []corpseRaiseKind
	prefer, excludeUndead bool
	healthFloor           int32
	creation              actorCreationProfile
	placement             func(int32, int32, int32) actorSearch
}

var firstCorpseRaise = corpseRaiseProfile{
	kinds: []corpseRaiseKind{{stage: DecayBones}}, prefer: true, healthFloor: 1,
	creation: actorCreationProfile{inherited: true, transient: true},
}

var secondCorpseRaise = corpseRaiseProfile{
	kinds:         []corpseRaiseKind{{2, "F_Zombie.1", 1, true}, {3, "F_Skeleton.1", 2, false}, {4, "Ghost", 0, false}},
	excludeUndead: true, placement: secondPlacement,
	creation: actorCreationProfile{transient: true, suppressRewards: true},
}

func RaisedCreatureNames() []string {
	var names []string
	for _, k := range secondCorpseRaise.kinds {
		names = append(names, k.name)
	}
	return names
}

func (w *World) corpseRaiseProfile() corpseRaiseProfile {
	for _, r := range w.spells {
		if r.runsArm(spiritArm) && r.secondArm().raise != nil {
			return *r.secondArm().raise
		}
	}
	return firstCorpseRaise
}

func (p corpseRaiseProfile) kind(e Entity) (corpseRaiseKind, bool) {
	if e.Alive() || p.excludeUndead && e.TypeID >= 82 && e.TypeID <= 98 {
		return corpseRaiseKind{}, false
	}
	for _, k := range p.kinds {
		if e.Decay == k.stage {
			return k, true
		}
	}
	return corpseRaiseKind{}, false
}

func (w *World) raiseTemplate(e Entity) (SpellCreature, bool) {
	p := w.corpseRaiseProfile()
	if p.prefer {
		return SpellCreature{Actor: NewActor(ActorDefinition(w.ghost), ActorPlacement{})}, w.ghost.Raisable()
	}
	k, ok := p.kind(e)
	if !ok {
		return SpellCreature{}, false
	}
	return w.spellCreature(k.name)
}

type controlSpiritRaise struct {
	Caster EntityID
	X, Y   int32
}

func bonesCorpse(e Entity) bool { return !e.Alive() && e.Decay == DecayBones }

// controlSpiritCorpseAt applies kind priority, then entity order at the cell
// (MAGIC-251, R2-ENGINE-780); -1 when none.
func (w *World) controlSpiritCorpseAt(x, y int32, prefer int) int {
	p := w.corpseRaiseProfile()
	if p.prefer && prefer >= 0 && prefer < len(w.entities) {
		if e := w.entities[prefer]; bonesCorpse(e) && e.X == x && e.Y == y {
			return prefer
		}
	}
	for _, k := range p.kinds {
		for i, e := range w.entities {
			if _, ok := p.kind(e); ok && e.Decay == k.stage && e.X == x && e.Y == y {
				return i
			}
		}
	}
	return -1
}

// queueControlSpirit admits a weapon or item release; its caller keeps entity
// indices, so drainControlSpirit consumes the corpse later. Only the target's
// cell is read.
func (w *World) queueControlSpirit(ai, ti int) bool {
	if ai < 0 || ai >= len(w.entities) || ti < 0 || ti >= len(w.entities) {
		return false
	}
	if _, ok := w.NextEntityID(); !ok {
		return false
	}
	x, y := w.entities[ti].X, w.entities[ti].Y
	corpse := w.controlSpiritCorpseAt(x, y, ti)
	if corpse < 0 {
		return false
	}
	if _, ok := w.raiseTemplate(w.entities[corpse]); !ok {
		return false
	}
	corpses := 0
	for _, e := range w.entities {
		if _, ok := w.corpseRaiseProfile().kind(e); ok && e.X == x && e.Y == y {
			corpses++
		}
	}
	claimed := 0
	for _, r := range w.burst.raises {
		if r.X == x && r.Y == y {
			claimed++
		}
	}
	if claimed >= corpses {
		return false
	}
	w.burst.raises = append(w.burst.raises, controlSpiritRaise{Caster: w.entities[ai].ID, X: x, Y: y})
	return true
}

func (w *World) drainControlSpirit() {
	queue := w.burst.raises
	w.burst.raises = nil
	for _, r := range queue {
		ci := indexOfEntity(w.entities, r.Caster)
		if ci < 0 {
			continue
		}
		ti := w.controlSpiritCorpseAt(r.X, r.Y, -1)
		if ti < 0 {
			continue
		}
		if _, raised := w.raiseControlSpirit(ci, ti); raised != 0 {
			w.markSpellEffect(indexOfEntity(w.entities, r.Caster), w.controlSpiritSpell())
			w.markSpellEffect(indexOfEntity(w.entities, raised), w.controlSpiritSpell())
		}
	}
}

// raiseControlSpirit applies the corpse profile and the shared actor builder.
// The caller's indices are stale afterwards (MAGIC-251, R2-ENGINE-780).
func (w *World) raiseControlSpirit(ci, ti int) (consumed bool, raised EntityID) {
	p := w.corpseRaiseProfile()
	ti = w.controlSpiritCorpseAt(w.entities[ti].X, w.entities[ti].Y, ti)
	if ti < 0 {
		return false, 0
	}
	next, available := w.NextEntityID()
	if !available {
		return false, 0
	}
	ghost, ok := w.raisedGhost(ci, ti, next)
	if !ok {
		return false, 0
	}
	corpse, caster := w.entities[ti], w.entities[ci]
	w.entities[ti].setCurrentHealth(controlSpiritCorpseHP)
	w.entities[ti].Decay = DecayStage(consumedCorpseStage)
	if !w.remove([]EntityID{corpse.ID}) {
		w.entities[ti] = corpse
		return false, 0
	}
	search := actorSearch(func(*World, int) (int32, int32, bool) { return corpse.X, corpse.Y, true })
	if p.placement != nil {
		search = p.placement(corpse.X, corpse.Y, 0)
	}
	c, _ := w.raiseTemplate(corpse)
	id, err := w.createActor(ghost, c.Pack, c.Worn, caster.X, caster.Y, search, true, p.creation)
	if err != nil {
		return true, 0
	}
	if c.Name != "" {
		w.castObservation.recordCreatedActor(id, c.Name)
	}
	return true, id
}
