package sim

// creatureAim is where a slot-drawn spell is aimed. Each of the 28 spell ids
// has one arm in the original's selector, and the arm fixes the target
// (MAGIC-237, MAGIC-238).
type creatureAim uint8

const (
	creatureAimNone       creatureAim = iota // Fire Sacrifice, Control Spirit: no order
	creatureAimVictim                        // a cast at the engage victim
	creatureAimCaster                        // a cast at the caster itself
	creatureAimVictimCell                    // a cast at the victim's cell
	creatureAimStepCell                      // a cast at the cell one step toward the victim
)

// creatureAimOf is the arm table: seven ids at the victim, nine at the caster,
// eight at the victim's cell, Acid Stream and Teleport at the cell next to the
// caster, and two at nothing. None of the 28 is aimed at an ally.
func creatureAimOf(id uint32) creatureAim {
	switch id {
	case 1, 11, 13, 14, 20, 27, 28:
		return creatureAimVictim
	case 5, 6, 10, 15, 16, 18, 22, 23, 24:
		return creatureAimCaster
	case 2, 3, 7, 8, 12, 17, 19, 21:
		return creatureAimVictimCell
	case 9, 26:
		return creatureAimStepCell
	}
	return creatureAimNone
}

// creatureAim is the aim of the table row id. A shared row takes its arm's
// aim. A row with no singular arm is aimed by its own columns: a unit-target
// row at the victim, an area row at the victim's cell (DIV-2619).
func (w *World) creatureAim(id uint32) creatureAim {
	rule, ok := w.findSpell(id)
	if !ok {
		return creatureAimOf(id)
	}
	switch arm := rule.arm(); {
	case arm != ArmNone:
		return creatureAimOf(uint32(arm))
	case rule.TargetsUnit:
		return creatureAimVictim
	case rule.Area:
		return creatureAimVictimCell
	}
	return creatureAimNone
}

// creatureStepCell is the cell one step from the caster toward the victim,
// along the eight-way heading the original's direction helper names for the
// two positions (MAGIC-238). The original adds the step with no clamp; the
// engine keeps the cell inside the map, one axis at a time.
func (w *World) creatureStepCell(caster, victim Entity) (int32, int32) {
	step := stepOf[w.headingBetween(caster, victim)>>5]
	x, y := caster.X+step[0], caster.Y+step[1]
	x = max(0, min(x, w.bounds.Width-1))
	y = max(0, min(y, w.bounds.Height-1))
	return x, y
}

// creatureAimOperands resolves a drawn spell's arm against the caster and the
// engage victim: the target actor, or the cell for a cast at a cell.
func (w *World) creatureAimOperands(i int, victim EntityID, id uint32) (target EntityID, x, y int32, atCell, ok bool) {
	caster := w.entities[i]
	vi := indexOfEntity(w.entities, victim)
	if vi < 0 {
		return 0, 0, 0, false, false
	}
	v := w.entities[vi]
	switch w.creatureAim(id) {
	case creatureAimVictim:
		return victim, v.X, v.Y, false, true
	case creatureAimCaster:
		return caster.ID, caster.X, caster.Y, false, true
	case creatureAimVictimCell:
		return 0, v.X, v.Y, true, true
	case creatureAimStepCell:
		x, y = w.creatureStepCell(caster, v)
		return 0, x, y, true, true
	}
	return 0, 0, 0, false, false
}

func isRangeRefusal(refusal string) bool {
	return refusal == refusalTargetOutOfRange || refusal == refusalCellOutOfRange
}

// creatureEngageCast is the spell arm of the engage routine for a creature
// whose class carries a spellbook. The draw runs on every pass, whether or not
// the creature's own cast is pending (AI-376). A draw that selects a spell
// replaces the creature's order with a cast aimed by the spell's arm and kept
// armed (MAGIC-240); a draw that selects none lets the ordinary engage replace
// it. A cast out of range walks toward the victim and stays armed until it is
// in range (MAGIC-239); a cast the engine refuses for any other reason leaves
// the pass to the ordinary engage (DIV-1725). The result reports whether the
// selection took the pass.
func (w *World) creatureEngageCast(i int, victim EntityID) bool {
	e := w.entities[i]
	if e.Owner == 0 || w.suppressedAcquirer(e) || !e.Alive() {
		return false
	}
	if !e.hasCreatureSpells() {
		return w.mageEngageCast(i, victim)
	}
	w.engageDrew = append(w.engageDrew, e.ID)
	id := w.creatureSpellPick(e)
	if w.actorCastBusy(i) {
		w.creatureReplaceCastOrder(i, id, victim)
		return false
	}
	if id == 0 {
		w.dropCastApproach(i)
		return w.mageEngageCast(i, victim)
	}
	if !knowsSpell(e, id) || w.creatureAim(id) == creatureAimNone {
		return true
	}
	target, x, y, atCell, ok := w.creatureAimOperands(i, victim, id)
	if !ok {
		return false
	}
	w.dropCastApproach(i)
	refusal := w.creatureCastRefusal(i, target, x, y, atCell, id, false)
	if refusal == "" {
		w.standForCreatureCast(i)
		if atCell {
			return w.beginBookSpellAtMode(i, x, y, id, true)
		}
		return w.beginBookSpellMode(i, target, id, true)
	}
	if !isRangeRefusal(refusal) || w.creatureCastRefusal(i, target, x, y, atCell, id, true) != "" {
		return false
	}
	return w.armCreatureCast(i, victim, target, x, y, atCell, id)
}

// creatureCastRefusal is the engine's admission verdict for a drawn cast, with
// or without the range test.
func (w *World) creatureCastRefusal(i int, target EntityID, x, y int32, atCell bool, id uint32, skipRange bool) string {
	if atCell {
		return w.bookSpellCellAdmissionRange(i, x, y, id, false, true, true, skipRange)
	}
	return w.bookSpellAdmissionRange(i, target, id, false, true, true, skipRange)
}

// standForCreatureCast ends the creature's walk and its attack order: the cast
// order replaces them, and the creature casts in place.
func (w *World) standForCreatureCast(i int) {
	w.entities[i].clearActiveAttack()
	w.clearOrder(i)
}

// armCreatureCast stores a retained cast order the creature cannot yet fire
// and sends the creature toward its victim.
func (w *World) armCreatureCast(i int, victim, target EntityID, x, y int32, atCell bool, id uint32) bool {
	c := bookCast{Caster: w.entities[i].ID, Target: target, Spell: uint16(id), X: x, Y: y,
		AtCell: atCell, Retained: true}
	return w.armCastApproach(i, &c, victim, true) && w.queueBookCast(c)
}

// dropCastApproach ends an armed approach: another order replaced it.
func (w *World) dropCastApproach(i int) {
	if s := w.orderStateAt(i); s.approaching() {
		w.bookCasts = append(w.bookCasts[:s.bookAt], w.bookCasts[s.bookAt+1:]...)
	}
}

// castApproachPursues reports how a caster walks its armed cast into reach. A
// computer caster pursues its victim with the attack's pursuit (MAGIC-239: the
// actor approach passes the target to the pursuit step), and so does a human
// participant's creature on its own class-slot cast, which holds that attack.
// The human participant's book order walks to the cell: the cast's cell, or
// its target's cell.
func (w *World) castApproachPursues(e Entity) bool {
	return !w.humanParticipantUnit(e) || e.hasCreatureSpells() && e.HasAttackTarget
}

// armCastApproach is the one builder of an armed approach (the original's
// pending kind 8 or 9 whose reach gate failed, MAGIC-340). It points the
// caster's walk into reach and reports whether the cast stays armed. victim
// is the unit a pursuing caster walks toward; pursue picks the pursuit over
// the walk to the cell.
func (w *World) armCastApproach(i int, c *bookCast, victim EntityID, pursue bool) bool {
	e := w.entities[i]
	if !c.Retained {
		return false
	}
	if pursue {
		if !(e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit && e.AttackTarget == victim) &&
			!w.attachAttack(i, victim, AttackTargetUnit, false) {
			return false
		}
	} else {
		w.attachMoveOrder(i, c.X, c.Y, false)
	}
	c.Phase, c.Remaining, c.Progress, c.Complete = bookApproach, 0, 0, false
	return true
}

// rearmCastApproach turns a retained cast refused for range at a later
// admission back into an approach. Only a pursuing caster holds such a cast.
func (w *World) rearmCastApproach(i int, c *bookCast, refusal string) bool {
	victim, ok := w.approachVictim(i, *c)
	return ok && isRangeRefusal(refusal) && w.castApproachPursues(w.entities[i]) && w.armCastApproach(i, c, victim, true)
}

// approachVictim is the unit a retained cast that went out of range walks
// toward, and whether it has one.
func (w *World) approachVictim(i int, c bookCast) (EntityID, bool) {
	if !c.AtCell {
		return c.Target, true
	}
	e := w.entities[i]
	return e.AttackTarget, e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit
}

// startApproachedCast begins the wind-up of an armed cast that is now in
// range, facing its target as admission does. A walking caster stands, as the
// install stands the member (MAGIC-340).
func (w *World) startApproachedCast(i int, c *bookCast) {
	e := w.entities[i]
	pursues := w.castApproachPursues(e)
	retained := c.Retained && pursues && e.HasAttackTarget && e.AttackTargetKind == AttackTargetUnit
	w.standForCreatureCast(i)
	if !pursues {
		w.standCaster(i)
	}
	caster := w.entities[i]
	rule, _ := w.bookSpell(caster, uint32(c.Spell))
	c.Phase, c.Remaining, c.Progress, c.Complete = bookCharging, castWindupTicks(caster), 0, false
	c.Retained = retained
	if c.AtCell {
		w.turnToCell(i, c.X, c.Y)
	} else if ti := indexOfEntity(w.entities, c.Target); ti >= 0 {
		w.turnToActor(i, ti)
	}
	w.chargeBookCast(i, rule)
}

// approachTargetGone reports an armed unit cast whose target left the map
// (script removal keeps the body off map): the approach ends unpaid
// (DIV-3682).
func (w *World) approachTargetGone(c bookCast) bool {
	ti := indexOfEntity(w.entities, c.Target)
	return !c.AtCell && ti >= 0 && w.entities[ti].OffMap
}

// walkApproach keeps a walking caster's approach on a unit aimed at the
// unit's cell. A cast at a cell keeps its cell.
func (w *World) walkApproach(i int, c *bookCast) {
	if c.AtCell || w.castApproachPursues(w.entities[i]) {
		return
	}
	ti := indexOfEntity(w.entities, c.Target)
	if ti < 0 {
		return
	}
	if t := w.entities[ti]; t.X != c.X || t.Y != c.Y {
		c.X, c.Y = t.X, t.Y
		w.attachMoveOrder(i, t.X, t.Y, false)
	}
}

// creatureReplaceCastOrder is what a draw does to the retained cast order of a
// creature whose cast is pending. The cast in flight is the actor's own and
// runs on; a draw that selects no spell replaces the retained order with the
// ordinary engage, so no further cast follows, and a draw that selects a spell
// replaces the retained operands that the next cast takes. A draw that lands
// during the wind-up leaves the retained operands as they are.
func (w *World) creatureReplaceCastOrder(i int, id uint32, victim EntityID) {
	k, ok := w.bookCastIndex(w.entities[i].ID)
	if !ok || !w.bookCasts[k].Retained {
		return
	}
	c := &w.bookCasts[k]
	if id == 0 {
		switch c.Phase {
		case bookCharging:
			c.Retained = false
		case bookRelaxing:
			w.entities[i].CastWait = c.Remaining
			w.bookCasts = append(w.bookCasts[:k], w.bookCasts[k+1:]...)
		default:
			w.bookCasts = append(w.bookCasts[:k], w.bookCasts[k+1:]...)
		}
		return
	}
	if c.Phase == bookCharging || !knowsSpell(w.entities[i], id) {
		return
	}
	target, x, y, atCell, ok := w.creatureAimOperands(i, victim, id)
	if !ok {
		return
	}
	c.Spell, c.Target, c.X, c.Y, c.AtCell = uint16(id), target, x, y, atCell
}
