package sim

const consumedCorpseStage uint8 = 5

type controlSpiritRaise struct {
	Caster EntityID
	X, Y   int32
}

func bonesCorpse(e Entity) bool { return !e.Alive() && e.Decay == DecayBones }

// controlSpiritCorpseAt is the bones corpse in cell (x, y), prefer first, else
// the first in entity order; -1 when none (MAGIC-251).
func (w *World) controlSpiritCorpseAt(x, y int32, prefer int) int {
	if prefer >= 0 && prefer < len(w.entities) {
		if e := w.entities[prefer]; bonesCorpse(e) && e.X == x && e.Y == y {
			return prefer
		}
	}
	for i, e := range w.entities {
		if bonesCorpse(e) && e.X == x && e.Y == y {
			return i
		}
	}
	return -1
}

// queueControlSpirit admits a weapon or item release; its caller keeps entity
// indices, so drainControlSpirit consumes the corpse later. Only the target's
// cell is read.
func (w *World) queueControlSpirit(ai, ti int) bool {
	if ai < 0 || ai >= len(w.entities) || ti < 0 || ti >= len(w.entities) || !w.ghost.Raisable() {
		return false
	}
	if _, ok := w.NextEntityID(); !ok {
		return false
	}
	x, y := w.entities[ti].X, w.entities[ti].Y
	corpses := 0
	for _, e := range w.entities {
		if bonesCorpse(e) && e.X == x && e.Y == y {
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
			w.markSpellEffect(indexOfEntity(w.entities, r.Caster), controlSpiritSpellID)
			w.markSpellEffect(indexOfEntity(w.entities, raised), controlSpiritSpellID)
		}
	}
}

// raiseControlSpirit consumes the corpse at ti and places the raised actor in
// the corpse's cell. Placement never fails: the original's blockers are Unknown
// (MAGIC-251). The caller's indices are stale afterwards.
func (w *World) raiseControlSpirit(ci, ti int) (consumed bool, raised EntityID) {
	next, available := w.NextEntityID()
	if !available {
		return false, 0
	}
	ghost, ok := w.raisedGhost(ci, ti, next)
	if !ok {
		return false, 0
	}
	corpse := w.entities[ti]
	w.entities[ti].setCurrentHealth(controlSpiritCorpseHP)
	w.entities[ti].Decay = DecayStage(consumedCorpseStage)
	if !w.remove([]EntityID{corpse.ID}) {
		w.entities[ti] = corpse
		return false, 0
	}
	w.entities = append(w.entities, ghost)
	w.appendActorTraversal(ghost.ID)
	w.routes = append(w.routes, nil)
	w.carried = append(w.carried, nil)
	w.equipment = append(w.equipment, [EquipSlots]ItemInstance{})
	return true, ghost.ID
}
