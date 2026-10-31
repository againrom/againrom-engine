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
	creatureAimScanCell                      // a cast at the first open cell around the victim
)

// creatureAimProfile is a game's creature slot-cast aim: the aim of each key,
// what the key is, and the planner's gates around the aim.
type creatureAimProfile struct {
	aims []creatureAim
	// byArm: the key is the drawn row's first-game arm; else the drawn id
	// itself, with no Spells column read (R2-ENGINE-653).
	byArm bool
	// scanBeyond: a scan aim casts only when caster and victim stand more
	// than this many cells apart (R2-ENGINE-651, R2-ENGINE-654).
	scanBeyond int64
	// offMapVictimStops: a victim off the map stops the planner before the
	// draw, with no attack (R2-ENGINE-655, DIV-4385).
	offMapVictimStops bool
}

var firstGameCreatureAim = creatureAimProfile{aims: creatureAims[:], byArm: true}

var secondGameCreatureAim = creatureAimProfile{aims: secondCreatureAims[:], scanBeyond: 2, offMapVictimStops: true}

func (w *World) creatureAimProfile() *creatureAimProfile {
	if w.SecondGame() {
		return &secondGameCreatureAim
	}
	return &firstGameCreatureAim
}

func (p *creatureAimProfile) aimOf(key uint32) creatureAim {
	if key < uint32(len(p.aims)) {
		return p.aims[key]
	}
	return creatureAimNone
}

// creatureAimOf is the first game's arm table: seven ids at the victim, nine
// at the caster, eight at the victim's cell, Acid Stream and Teleport at the
// cell next to the caster, and two at nothing. None is aimed at an ally.
func creatureAimOf(id uint32) creatureAim { return firstGameCreatureAim.aimOf(id) }

// secondCreatureAims is the second game's 29 cells by id; 22 aims at nothing
// (R2-ENGINE-653).
var secondCreatureAims = [...]creatureAim{
	1: creatureAimVictim, 5: creatureAimVictim, 10: creatureAimVictim, 11: creatureAimVictim, 16: creatureAimVictim,
	18: creatureAimVictim, 26: creatureAimVictim, 28: creatureAimVictim, 29: creatureAimVictim,
	4: creatureAimCaster, 8: creatureAimCaster, 12: creatureAimCaster, 13: creatureAimCaster, 19: creatureAimCaster,
	20: creatureAimCaster, 21: creatureAimCaster, 24: creatureAimCaster, 25: creatureAimCaster, 27: creatureAimCaster,
	2: creatureAimVictimCell, 3: creatureAimVictimCell, 6: creatureAimVictimCell, 7: creatureAimVictimCell,
	14: creatureAimVictimCell, 15: creatureAimVictimCell, 17: creatureAimVictimCell,
	9: creatureAimStepCell, 23: creatureAimScanCell,
}

var creatureAims = [...]creatureAim{
	1: creatureAimVictim, 11: creatureAimVictim, 13: creatureAimVictim, 14: creatureAimVictim,
	20: creatureAimVictim, 27: creatureAimVictim, 28: creatureAimVictim,
	5: creatureAimCaster, 6: creatureAimCaster, 10: creatureAimCaster, 15: creatureAimCaster, 16: creatureAimCaster,
	18: creatureAimCaster, 22: creatureAimCaster, 23: creatureAimCaster, 24: creatureAimCaster,
	2: creatureAimVictimCell, 3: creatureAimVictimCell, 7: creatureAimVictimCell, 8: creatureAimVictimCell,
	12: creatureAimVictimCell, 17: creatureAimVictimCell, 19: creatureAimVictimCell, 21: creatureAimVictimCell,
	9: creatureAimStepCell, 26: creatureAimStepCell,
}

// creatureAim is the aim of a drawn id in the world's game. By arm, a row
// takes its arm's aim and an id with no row its own.
func (w *World) creatureAim(id uint32) creatureAim {
	p := w.creatureAimProfile()
	if !p.byArm {
		return p.aimOf(id)
	}
	if rule, ok := w.findSpell(id); ok {
		return p.aimOf(uint32(rule.arm()))
	}
	return p.aimOf(id)
}

// creatureScanCell is the first cell of the 3 by 3 around the victim, rows
// outer and columns inner, where the caster's whole footprint is open terrain,
// and whether the scan found one. The packed cell 0 is no cell
// (R2-ENGINE-654, DIV-4386).
func (w *World) creatureScanCell(caster, victim Entity) (int32, int32, bool) {
	if (cell{x: caster.X, y: caster.Y}).chebyshevTo(cell{x: victim.X, y: victim.Y}) <= w.creatureAimProfile().scanBeyond {
		return 0, 0, false
	}
	for y := victim.Y - 1; y <= victim.Y+1; y++ {
		for x := victim.X - 1; x <= victim.X+1; x++ {
			if w.terrainOpenFootprint(caster, x, y) {
				return x, y, x != 0 || y != 0
			}
		}
	}
	return 0, 0, false
}

// creatureKeepsPick is the planner's distance gate: a pick aimed by the scan
// is dropped at or within the profile's distance, and the attack follows.
func (w *World) creatureKeepsPick(i int, victim EntityID, id uint32) bool {
	vi := indexOfEntity(w.entities, victim)
	if id == 0 || vi < 0 || w.creatureAim(id) != creatureAimScanCell {
		return id != 0
	}
	c, v := w.entities[i], w.entities[vi]
	return (cell{x: c.X, y: c.Y}).chebyshevTo(cell{x: v.X, y: v.Y}) > w.creatureAimProfile().scanBeyond
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
	case creatureAimScanCell:
		x, y, ok = w.creatureScanCell(caster, v)
		return 0, x, y, true, ok
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
// in range (MAGIC-239), and one drawn during a loaded strike waits for the
// strike to end; a cast the engine refuses for any other reason leaves
// the pass to the ordinary engage (DIV-1725). The profile's gates stop it in
// the second game's order (R2-ENGINE-655). The result reports whether the
// selection took the pass.
func (w *World) creatureEngageCast(i int, victim EntityID) bool {
	e := w.entities[i]
	if e.Owner == 0 || w.suppressedAcquirer(e) || !e.Alive() {
		return false
	}
	if !e.hasCreatureSpells() {
		return w.mageEngageCast(i, victim)
	}
	if vi := indexOfEntity(w.entities, victim); vi >= 0 && w.entities[vi].OffMap && w.creatureAimProfile().offMapVictimStops {
		return true
	}
	w.engageDrew = append(w.engageDrew, e.ID)
	id := w.creatureSpellPick(e)
	if !w.creatureKeepsPick(i, victim, id) {
		id = 0
	}
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
		// A scan with no cell ends the call with no attack (R2-ENGINE-655).
		return indexOfEntity(w.entities, victim) >= 0
	}
	w.dropCastApproach(i)
	refusal := w.creatureCastRefusal(i, target, x, y, atCell, id, false)
	if refusal == "" {
		return w.startEngageCast(i, victim, target, x, y, atCell, id)
	}
	if w.strikeHoldsCast(i, refusal) {
		return w.armCreatureCast(i, victim, target, x, y, atCell, id)
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

// crossingDefersCast reports a caster still crossing into a cell: both games'
// executors run a step in progress to the next cell centre before they read
// the cast order (AI-350, AI-352, MAGIC-239, R2-ENGINE-546, R2-ENGINE-560).
func (w *World) crossingDefersCast(i int) bool {
	return w.entities[i].Transit > 0
}

// startEngageCast begins an engage-drawn cast that admission accepts. A caster
// whose crossing defers the cast arms it instead and walks the crossing out;
// stepBookCasts starts the armed cast at the cell centre.
func (w *World) startEngageCast(i int, victim, target EntityID, x, y int32, atCell bool, id uint32) bool {
	if w.crossingDefersCast(i) {
		return w.armCreatureCast(i, victim, target, x, y, atCell, id)
	}
	w.standForCreatureCast(i)
	if atCell {
		return w.beginBookSpellAtMode(i, x, y, id, true)
	}
	return w.beginBookSpellMode(i, target, id, true)
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
	w.replaceRetainedCast(i, id, victim, w.creatureAimOperands)
}

// victimAimOperands aims a cast at the victim itself, as the full-book draw
// does for every id (AI-554).
func (w *World) victimAimOperands(_ int, victim EntityID, _ uint32) (EntityID, int32, int32, bool, bool) {
	vi := indexOfEntity(w.entities, victim)
	if vi < 0 {
		return 0, 0, 0, false, false
	}
	return victim, w.entities[vi].X, w.entities[vi].Y, false, true
}

// replaceRetainedCast is creatureReplaceCastOrder with the draw's own aim.
func (w *World) replaceRetainedCast(i int, id uint32, victim EntityID,
	aim func(int, EntityID, uint32) (EntityID, int32, int32, bool, bool)) {
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
	target, x, y, atCell, ok := aim(i, victim, id)
	if !ok {
		return
	}
	c.Spell, c.Target, c.X, c.Y, c.AtCell = uint16(id), target, x, y, atCell
	// The actor's selected spell follows the order, as LOAD restores it.
	w.entities[i].AdmittedBookSpell = uint16(id)
}
