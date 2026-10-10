package sim

import "sort"

// AREA EFFECTS STANDING ON CELLS.
//
// An area effect is what a spell whose `Distribution system` column is not 1
// leaves behind when it lands: an object standing on one cell, carrying the
// spell's own id and a remaining lifetime in ticks. The mission script both
// creates one — instant 21, through a script cast (scriptcast.go) — and
// re-times one, through instant 29.
//
// The distribution program expands this anchor into its decoded covered cells.
// A blast applies once, a ring exposes successive stages, and a cloud pulses
// every sixteen ticks. Each hit reuses the ordinary spell application owner;
// Earth Wall instead contributes cell passability while it remains present.
//
// Actor attachments have their own refresh and expiry rules. Here each
// independent area object owns its clock; six cell slots hold only the current
// spell-layer pointers (MAGIC-CLOUDOWNER-155).

// Cells contains only currently owned paint, which may be empty. The scan
// square is independent. Remaining is the raw cloud word: decrement, pulse
// at each new multiple of16 including0, then remove on the next call.
type cellEffect struct {
	Key       uint16
	Spell     uint16
	Remaining uint16
	Caster    EntityID
	HasCaster bool
	Power     uint16
	Mode      uint8
	Phase     uint8
	Direction uint8
	DamageMin int32
	DamageMax int32
	Cells     []uint16
	Current   *CurrentAreaPayload
}

// Only the unshaped, test-only mode0 predecessor uses the old anchor cap.
const cellEffectSlots = 6

// cellKey is the engine's own 16-bit cell key, computed the engine's own way.
//
// THE ADD IS IN SIXTEEN BITS AND IT IS AN ADD (`TRIG-CELLEFFECT-045`, which
// supersedes `TRIG-EFFECTTIME-034`'s OR): word 0x0c is shifted left by 8 and
// word 0x08 is added. So y is truncated to a word and shifted, leaving
// only its low byte, and x is truncated to a word and ADDED — an x above 255
// therefore carries into the y byte rather than being masked off. Both
// differences are invisible on shipped data, where every authored x is 130 or
// below, and both are what the arm does.
func cellKey(x, y int32) uint16 {
	return uint16(uint16(y)<<8) + uint16(x)
}

// keyCell is cellKey read back: the low byte is x, the high byte is y. It is the
// one place the key is taken apart, so a consumer asking where an effect stands
// and the arm deciding which effects match cannot come to disagree.
func keyCell(k uint16) (x, y int32) { return int32(k & 0xff), int32(k >> 8) }

// CellEffect is one area effect as a consumer sees it: where it stands, which
// spell it is, and how many ticks it has left.
type CellEffect struct {
	X, Y      int32
	Spell     uint16
	Remaining uint16
	Power     uint16
	Mode      uint8
	Phase     uint8
	Caster    EntityID
	HasCaster bool
	Cells     [][2]int32
}

// CellEffects returns a detached view in execution order: retained original
// roots first, then native objects in causal append order. Empty paint stays
// empty; presentation must not infer coverage from an object's anchor.
func (w *World) CellEffects() []CellEffect {
	out := make([]CellEffect, len(w.effects))
	for i, e := range w.effects {
		x, y := keyCell(e.Key)
		out[i] = CellEffect{X: x, Y: y, Spell: e.Spell, Remaining: e.Remaining,
			Power: e.Power, Mode: e.Mode, Phase: e.Phase, Caster: e.Caster, HasCaster: e.HasCaster}
		out[i].Cells = make([][2]int32, len(e.Cells))
		for k, key := range e.Cells {
			cx, cy := keyCell(key)
			out[i].Cells[k] = [2]int32{cx, cy}
		}
	}
	if s := w.savedWorldEffects; s != nil {
		for _, d := range s.Areas {
			if d.Root < 0 || d.Mode != areaModeCloud {
				continue
			}
			e := w.savedSpellEffects[d.Root]
			x, y := keyCell(d.Key)
			view := CellEffect{X: x, Y: y, Spell: d.Spell, Remaining: e.AE4C, Mode: areaModeCloud}
			for _, key := range d.Cells {
				cx, cy := keyCell(key)
				view.Cells = append(view.Cells, [2]int32{cx, cy})
			}
			out = append(out, view)
		}
	}
	out = append(out[len(w.effects):], out[:len(w.effects)]...)
	return out
}

// FireWallCount is the current layer's presence (zero or one), not a stack
// depth. Independent overwritten Fire clocks remain in CellEffects with no
// owned cells (MAGIC-CLOUDOWNER-155).
func (w *World) FireWallCount(x, y int32) int {
	if w.areaLayerPresent(cellKey(x, y), 3) {
		return 1
	}
	return 0
}

// placeCellEffect is the test-only, unshaped mode0 predecessor. Its sorted
// anchor/cap policy is retained explicitly; no production cast uses it.
func (w *World) placeCellEffect(key uint16, spell uint16, life uint16) bool {
	if spell == 0 || life == 0 {
		return false
	}
	if spell != 3 && w.anchorFull(key) {
		return false
	}
	at := len(w.effects)
	for i, e := range w.effects {
		if e.Mode == 0 && e.Key > key {
			at = i
			break
		}
	}
	w.effects = append(w.effects, cellEffect{})
	copy(w.effects[at+1:], w.effects[at:])
	w.effects[at] = cellEffect{Key: key, Spell: spell, Remaining: life}
	return true
}

// anchorFull concerns only the unshaped predecessor, never typed clouds.
func (w *World) anchorFull(key uint16) bool {
	n := 0
	for _, e := range w.effects {
		if e.Key == key && e.Mode == 0 {
			n++
		}
	}
	return n >= cellEffectSlots
}

// decayCellEffects visits independent native objects in creation order.
// MAGIC-CLOUDCLOCK-154 includes the final zero pulse and following cleanup.
func (w *World) decayCellEffects(obs *castObs) {
	kept := w.effects[:0]
	for i := range w.effects {
		if w.tickCellEffect(i, obs) {
			kept = append(kept, w.effects[i])
		}
	}
	w.effects = kept
}

func (w *World) tickCellEffect(index int, obs *castObs) bool {
	e := w.effects[index]
	if e.Current != nil && e.Current.Stopped || e.Mode == areaModeCloud && e.Remaining == 0 || e.Mode != areaModeCloud && e.Remaining <= 1 {
		w.effects[index].Cells = nil
		w.clearAreaCells(e)
		return false
	}
	e.Remaining--
	if rule, ok := w.findSpell(uint32(e.Spell)); ok {
		if e.Current != nil {
			rule.Radius = e.Current.Radius
		}
		switch e.Mode {
		case areaModeCloud:
			e.Phase++
			if e.Remaining%16 == 0 {
				w.applyAreaCells(e, rule, w.cloudPulseCells(e.Key, w.spellArm(e.Spell), int32(rule.Radius)))
			}
		case areaModeRing:
			e.Phase++
			if e.Phase%3 == 0 {
				stage := int(e.Phase / 3)
				cells := w.ringStageCells(e, stage)
				e.Cells = canonicalCells(cells)
				obs.recordPaint(w, e, cells)
				w.applyAreaCells(e, rule, cells)
				if stage+1 >= ringStageCount(w.spellArm(e.Spell)) {
					w.clearAreaCells(e)
					return false
				}
			}
		}
	}
	w.effects[index] = e
	return true
}

const (
	areaModeBlast uint8 = iota + 1
	areaModeRing
	areaModeCloud
)

// The three tick modes, as a consumer of CellEffect.Mode names them
// (`MAGIC-AREADRAW-049`). They are exported because the three create three
// DIFFERENT drawables and a client cannot draw a record without knowing which:
// a CLOUD is the retained per-cell overlay and is drawn from the record's own
// cell set; a RING creates one transient object per accepted cell and its
// record must draw nothing at all; a BLAST retains no record.
const (
	AreaModeBlast = areaModeBlast
	AreaModeRing  = areaModeRing
	AreaModeCloud = areaModeCloud
)

// The `Distribution system` column values this build reads. 1 is a point
// effect and is `SpellRule.Area`'s own test; the three below are the area
// programs (`MAGIC-AREACELL-039`, `MAGIC-AREATICK-036`). No other value is
// shipped: the installed table is 18 rows of 1, 5 of 3, 2 of 4 and 3 of 5 on
// both roots.
const (
	distributionDiamond uint8 = 3
	distributionWall    uint8 = 4
	distributionStaged  uint8 = 5
)

// areaModeFor is the per-tick mode selector, and it reads the ROW rather than
// the spell id (`MAGIC-AREATICK-036`).
//
// The builder writes `effect+0x08` three times: the constructor sets 0, the
// duration arm sets 1 when `+0x4c > 0` after `Area Effect Duaration` is loaded,
// and the `Distribution system == 5` arm sets 2. THE DISTRIBUTION ARM RUNS
// SECOND and overwrites both fields, so it wins where a row carries both
// columns — `meteor_storm`'s shipped `Area Effect Duaration = 10` is
// overwritten to 0 and has no effect.
//
// NO SPELL ID ENTERS THIS. On the shipped table the two spellings agree — ids
// 4, 9 and 21 are exactly the `Distribution system = 5` rows, and Fire Ball is
// the one area row with no duration column — but an EDITED row's own columns
// now select its mode, which is what `Distribution` is parsed and serialized
// for. An id switch made both columns dead weight.
func areaModeFor(rule SpellRule) uint8 {
	if rule.Distribution == distributionStaged {
		return areaModeRing
	}
	if rule.AreaDuration > 0 {
		return areaModeCloud
	}
	return areaModeBlast
}

// areaLife is how many ticks an area effect stands, counting the tick it is
// removed on. It is the player-facing duration projection, not necessarily the
// word retained in cellEffect.Remaining.
//
// For a cloud it is `V0 + 1` where `V0 = (AreaDuration << 4) + (power << 4)/10`
// is `effect+0x4c`'s initial value (`MAGIC-AREAPULSE-037`). The extra tick is
// the removal pass that sees the raw word already at zero.
//
// A STAGED ROW'S DURATION COLUMN IS NOT READ, because the builder zeroes
// `+0x4c` on the same arm that selects staged mode. Its life is its own
// cadence: stage ticks 0, 3, 6, ... and the terminal stage removes the record
// on the tick it runs (`MAGIC-RING-048`), so the whole life is
// `(stages - 1) * 3 + 1`. That is one formula for both the record's lifetime
// and the spellbook popup's duration, where the popup previously read an
// invented `(Radius + 2) * 2` off a hard-coded id 21.
func areaLife(rule SpellRule, power uint16) uint16 {
	if areaModeFor(rule) == areaModeRing {
		return ringLife(rule.arm())
	}
	v := (int64(rule.AreaDuration) << 4)
	if v > 0 {
		v += (int64(power) << 4) / 10
	}
	return uint16(v + 1)
}

// ringLife is a staged effect's whole life in ticks: stage 0 runs at once and
// every later stage three ticks after the one before it, with the terminal
// stage removing the record on the same tick it runs. A row with no stage
// program has no life.
func ringLife(id uint16) uint16 {
	n := ringStageCount(id)
	if n == 0 {
		return 0
	}
	return uint16((n-1)*3 + 1)
}

func (w *World) landArea(rule SpellRule, power uint16, caster EntityID, hasCaster bool, fromX, fromY, x, y int32, obs *castObs) bool {
	return w.landAreaFacing(rule, power, caster, hasCaster, fromX, fromY, x, y, 0, false, obs)
}

// areaLandingRefusal is why an area row would refuse to land at (x, y), or ""
// when it would land. It is the CELL form's counterpart of pointEffectRefusal
// (spell.go) and it is read-only in the same sense: no roll, no queue, no
// field, so the same function answers for the admission predicate, for the
// pre-cost gate inside the two cast arms, and for the lawful save instrument.
//
// IT EXISTS BECAUSE ADMISSION AND APPLY MUST AGREE. spec.md states that a
// refusal, a release-time cancellation and an effect that cannot apply spend no
// mana, start no recovery and award no training. castBookAt paid the row's mana
// and THEN returned on landArea's false; castSpell's area arm paid, marked both
// actors and started recovery for a landing that never happened. Fire Sacrifice
// was the worst of the three: its arm sets the caster's health to 1 and its mana
// to 0 before the record is inserted, so a refused landing took everything the
// caster had.
//
// Read-only admission checks the supported program, lifetime and current-cell
// authority before any source resource or existing owner changes.
func (w *World) areaLandingRefusal(rule SpellRule, power uint16, x, y int32) string {
	if !rule.Area {
		return ""
	}
	mode := areaModeFor(rule)
	if mode == areaModeBlast {
		// A blast retains no object but can clear an existing layer. Resolve
		// its current-cell authority before changing that layer or spending.
		return w.areaPaintIssue(rule, x, y)
	}
	if mode == areaModeRing && ringStageCount(rule.arm()) == 0 {
		return "row selects a staged program its spell id has none of"
	}
	if areaLife(rule, power) == 0 {
		return "area lifetime is zero"
	}
	if issue := w.areaPaintIssue(rule, x, y); issue != "" {
		return issue
	}
	return ""
}

// A book cast and a weapon-borne release reach this function on the same
// terms. Neither walk pays a cast award (DIV-238): castSpell and castBookAt do.
func (w *World) landAreaFacing(rule SpellRule, power uint16, caster EntityID, hasCaster bool, fromX, fromY, x, y int32, facing uint8, fixedFacing bool, obs *castObs, prepared ...*AreaSaveState) bool {
	return w.landAreaAimed(areaAim{}, rule, power, caster, hasCaster, fromX, fromY, x, y, facing, fixedFacing, obs, prepared...)
}

// landAreaAimed is landAreaFacing for a cast aimed at an actor: the area
// lands on the actor's cell and a delayed delivery times itself to the
// actor's position.
func (w *World) landAreaAimed(aim areaAim, rule SpellRule, power uint16, caster EntityID, hasCaster bool, fromX, fromY, x, y int32, facing uint8, fixedFacing bool, obs *castObs, prepared ...*AreaSaveState) bool {
	if rule.Delivery == 2 {
		if w.areaLandingRefusal(rule, power, x, y) != "" {
			return false
		}
		w.queueSpellDelivery(spellDelivery{Caster: caster, HasCaster: hasCaster,
			FromX: fromX, FromY: fromY, X: x, Y: y, Rule: rule,
			Power: int32(power), AtCell: true, Facing: facing, FixedFacing: fixedFacing, aim: aim})
		return true
	}
	mode := areaModeFor(rule)
	var current *AreaSaveState
	if len(prepared) != 0 {
		current = prepared[0]
		if current != nil {
			mode, rule.Radius = current.Mode, current.Radius
		}
	}
	// Direction and radius are relative to the aimed anchor (MAGIC-RING-048).
	anchorX, anchorY := x, y
	// Refuse before Fire Sacrifice spends its caster or a layer is cleared.
	if r := w.areaLandingRefusal(rule, power, anchorX, anchorY); r != "" && current == nil {
		return false
	}
	e := cellEffect{Key: cellKey(anchorX, anchorY), Spell: rule.ID, Caster: caster, HasCaster: hasCaster, Power: power, Mode: mode,
		Direction: areaDirection(x-fromX, y-fromY)}
	if fixedFacing {
		e.Direction = uint8(FacingDir(facing))
	}
	if current != nil {
		e.Direction = current.Direction
		e.Current = &CurrentAreaPayload{Radius: current.Radius, Payload: current.Payload}
	}
	if rule.arm() == 4 && hasCaster {
		if ci := indexOfEntity(w.entities, caster); ci >= 0 {
			base, spread := sacrificeDamage(w.entities[ci], power)
			e.DamageMin, e.DamageMax = base, base+spread
			before := w.entities[ci]
			w.entities[ci].setCurrentHealth(1)
			w.reportHealthLoss(before, ci)
			w.entities[ci].setCurrentMana(0)
		}
	}
	if mode == areaModeBlast {
		cells := w.blastCellsInMap(x, y, int32(rule.Radius))
		e.Cells = canonicalCells(cells)
		w.resolveLayerConflicts(&e)
		release := w.holdLayerCosts()
		for _, key := range e.Cells {
			w.recomputeSavedCell(key)
			w.deleteEmptyAreaCell(key)
		}
		release()
		w.refreshSavedPlaneBlocks()
		w.applyAreaCells(e, rule, cells)
		if rule.ID == fireBallSpell {
			w.releaseFireBallBursts(anchorX, anchorY, int32(rule.Radius), w.casterOwner(caster, hasCaster))
		}
		return true
	}
	if mode == areaModeRing {
		// Stage zero executes now; the last stage executes three ticks per
		// stage later. The extra one keeps the generic lifetime guard from
		// removing the record before that terminal execution. A row with no
		// stage program was refused above.
		e.Remaining = areaLife(rule, power)
		e.Phase = 0
		cells := w.ringStageCells(e, 0)
		e.Cells = canonicalCells(cells)
		obs.recordPaint(w, e, cells)
		w.applyAreaCells(e, rule, cells)
		return w.addAreaEffect(e)
	}
	// A cloud stores the raw counter V0, while areaLife reports the V0+1
	// ticks over which the record is observed and removed. areaLandingRefusal
	// has already rejected a wrapped zero life, so the subtraction is defined.
	e.Remaining = areaLife(rule, power) - 1
	if current != nil {
		e.Remaining = current.Remaining
	}
	// THE PAINTED SHAPE IS THE ROW'S `Distribution system` COLUMN
	// (`MAGIC-AREACELL-039`), not its spell id: 4 goes through the two wall
	// tables and 3 through the diamond walk. Both shipped distribution-4 rows
	// are walls — `wall_of_fire` (3) and `wall_of_earth` (19) — and forking on
	// id 19 gave Wall of Fire a 25-cell diamond.
	if rule.Distribution == distributionWall {
		e.Cells = w.wallCells(x, y, e.Direction)
		if rule.arm() == 19 {
			e.Cells = w.skipOccupiedGround(e.Cells)
		}
	} else {
		e.Cells = w.diamondCells(x, y, int32(rule.Radius))
	}
	w.resolveLayerConflicts(&e)
	w.scorchCells(rule.arm(), e.Cells)
	return w.addAreaEffect(e)
}

// New objects append in causal order. Six is the number of spell layers,
// never a limit on independently ticking clouds (MAGIC-CLOUDOWNER-155).
func (w *World) addAreaEffect(e cellEffect) bool {
	if e.Spell == 0 || e.Remaining == 0 && e.Mode != areaModeCloud {
		return false
	}
	ordered := w.effectOrder != nil || w.savedSpellGraph != nil || len(w.deliveries) > 0
	if ordered {
		w.prepareEffectAppend()
	}
	w.effects = append(w.effects, e)
	if ordered {
		w.noteEffectAppend(WorldEffectRef{Kind: EffectNativeArea, Index: uint32(len(w.effects) - 1)})
	}
	if e.Mode == areaModeCloud {
		release := w.holdLayerCosts()
		for _, key := range e.Cells {
			if w.savedCellPlanes != nil {
				w.createSavedCell(key)
				w.recomputeSavedCell(key)
			}
		}
		release()
		w.refreshSavedPlaneBlocks()
	}
	if w.spellArm(e.Spell) == 19 {
		w.setWallCells(e.Cells, true)
	}
	return true
}

func (w *World) clearAreaCells(e cellEffect) {
	defer w.holdLayerCosts()()
	if e.Mode != areaModeCloud {
		return
	}
	if w.spellArm(e.Spell) == 19 {
		w.setWallCells(e.Cells, false)
	}
	for _, key := range e.Cells {
		w.recomputeSavedCell(key)
		w.deleteEmptyAreaCell(key)
	}
	w.refreshSavedPlaneBlocks()
}

func (w *World) setWallCells(cells []uint16, set bool) {
	for _, k := range cells {
		x, y := keyCell(k)
		i, ok := cellIndexIn(w.bounds, x, y)
		if !ok {
			continue
		}
		if set {
			w.grid[i] |= blockMagicWall
		} else {
			w.grid[i] &^= blockMagicWall
		}
	}
}

// areaDirection is `effect+0x4a`, the one direction byte every area program
// reads.
//
// THERE IS ONE CONVENTION AND NOT TWO. The builder writes the field once, at
// `L05428`, as `(bearing(caster, targetCell) & 0xff) >> 5` — the bearing byte
// truncated to one of eight (`MAGIC-AREACELL-039`), which is the same 0..7
// index `MAGIC-RING-048` reads as Acid Stream's orientation and the same one
// the wall switch at `L05427` indexes with. This build's own facing byte is
// `d << 5` for direction d (facing.go), so the shift recovers d exactly:
// 0 north, 2 east, 4 south, 6 west.
//
// A zero delta names no direction and answers 0, which is what facingToward
// refuses to invent; a cast at the caster's own cell has no bearing.
func areaDirection(dx, dy int32) uint8 {
	f, ok := facingToward(dx, dy)
	if !ok {
		return 0
	}
	return f >> 5
}

// canonicalCells is a cell list in the byte form's own order: ascending key,
// each key once.
//
// A STAGED EFFECT'S VISIT ORDER IS SIGNIFICANT AT APPLICATION AND NOT IN THE
// RECORD. `MAGIC-RING-048` has a ring paint one cell and walk that cell's
// occupant slots before it produces the next, and applyAreaCells is handed the
// generator's own list in that order. What the record keeps afterwards is the
// SET of painted cells: the layer overlap test, the passability restore and the
// client's draw all read it without order.
//
// The byte form refuses a cell list that is not strictly ascending
// (castbinary.go), which is what makes a decoded list a set rather than a
// sequence two different worlds could spell two ways. Storing the visit order
// instead made a save taken while Fire Sacrifice or Acid Stream stood
// unloadable: MarshalBinary wrote it and UnmarshalBinary refused it.
func canonicalCells(cells []uint16) []uint16 {
	if len(cells) == 0 {
		return nil
	}
	out := append([]uint16(nil), cells...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	k := 0
	for i := range out {
		if i > 0 && out[i] == out[k-1] {
			continue
		}
		out[k] = out[i]
		k++
	}
	return out[:k]
}

func (w *World) cellsWhere(points [][2]int32) []uint16 {
	out := make([]uint16, 0, len(points))
	seen := map[uint16]bool{}
	for _, p := range points {
		if _, ok := cellIndexIn(w.bounds, p[0], p[1]); !ok {
			continue
		}
		k := cellKey(p[0], p[1])
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (w *World) diamondCells(cx, cy, r int32) []uint16 {
	var p [][2]int32
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if cellEffectAbs(dx)+cellEffectAbs(dy) <= r+1 {
				p = append(p, [2]int32{cx + dx, cy + dy})
			}
		}
	}
	return w.cellsWhere(p)
}

// blastCells preserves the x-outer/y-inner visit order and independently
// byte-truncated accessor coordinates (UNIT-AREAVISIT-071). Unlike a ring it
// has no inset gate, and unlike a painted cell set it must not sort or dedup.
func blastCells(cx, cy, r int32) []uint16 {
	var cells []uint16
	for dx := -r; dx <= r; dx++ {
		for dy := -r; dy <= r; dy++ {
			cells = append(cells, cellKey(int32(uint8(cx+dx)), int32(uint8(cy+dy))))
		}
	}
	return cells
}

func (w *World) blastCellsInMap(cx, cy, r int32) []uint16 {
	if 2*r+1 <= 256 {
		return blastCells(cx, cy, r)
	}
	x0, x1 := max(0, cx-r), min(cx+r, min(w.bounds.Width, 256)-1)
	y0, y1 := max(0, cy-r), min(cy+r, min(w.bounds.Height, 256)-1)
	if x1 < x0 || y1 < y0 {
		return nil
	}
	cells := make([]uint16, 0, int(x1-x0+1)*int(y1-y0+1))
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			cells = append(cells, cellKey(x, y))
		}
	}
	return cells
}

func ringStageCount(id uint16) int {
	switch id {
	case 4:
		return 2
	case 9:
		return 6
	case 21:
		return 32
	default:
		return 0
	}
}

// ringStageCells is the three ring programs' exact cell producer. Unlike the
// static cloud and wall helpers, order is significant here: an accepted cell is
// painted and its occupant slots are applied before the next offset is read.
func (w *World) ringStageCells(e cellEffect, stage int) []uint16 {
	cx, cy := keyCell(e.Key)
	var offsets [][2]int32
	switch w.spellArm(e.Spell) {
	case 4: // Fire Sacrifice: two fixed shells; orientation is ignored.
		if stage == 0 {
			offsets = [][2]int32{{-1, 1}, {-1, 0}, {-1, -1}, {0, 1}, {0, -1}, {1, 1}, {1, 0}, {1, -1}}
		} else if stage == 1 {
			offsets = [][2]int32{{-2, 1}, {-2, 0}, {-2, -1}, {-1, 2}, {0, 2}, {1, 2}, {-1, -2}, {0, -2}, {1, -2}, {2, 1}, {2, 0}, {2, -1}}
		}
	case 9: // Acid Stream: alternating straight and diagonal wedges.
		if stage < 0 || stage >= 6 {
			break
		}
		if e.Direction&1 == 0 {
			if stage < 5 {
				for x := -stage; x <= stage; x++ {
					offsets = append(offsets, [2]int32{int32(x), int32(stage)})
				}
			}
		} else {
			for i := 0; i <= stage; i++ {
				offsets = append(offsets, [2]int32{int32(stage - i), int32(i)})
			}
		}
		for i := range offsets {
			dx, dy := offsets[i][0], offsets[i][1]
			switch e.Direction & 7 {
			case 0:
				offsets[i] = [2]int32{dx, -dy}
			case 1:
				offsets[i] = [2]int32{dx, -dy}
			case 2:
				offsets[i] = [2]int32{dy, dx}
			case 3:
				offsets[i] = [2]int32{dx, dy}
			case 4:
				offsets[i] = [2]int32{-dx, dy}
			case 5:
				offsets[i] = [2]int32{-dx, dy}
			case 6:
				offsets[i] = [2]int32{-dy, dx}
			case 7:
				offsets[i] = [2]int32{-dx, -dy}
			}
		}
	case 21: // Meteor Storm: one independently sampled cell per stage.
		if stage >= 0 && stage < 32 {
			offsets = append(offsets, [2]int32{w.rng.uniform(5) - 2, w.rng.uniform(5) - 2})
		}
	}
	out := make([]uint16, 0, len(offsets))
	for _, d := range offsets {
		// The coordinate arithmetic is byte arithmetic before the inset test.
		x, y := int32(uint8(cx+d[0])), int32(uint8(cy+d[1]))
		if x < 8 || y < 8 || x > w.bounds.Width-9 || y > w.bounds.Height-9 {
			continue
		}
		out = append(out, cellKey(x, y))
	}
	return out
}
func cellEffectAbs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

var (
	wallTableA = [10][2]int32{{-2, 1}, {-1, 1}, {0, 1}, {1, 1}, {2, 1}, {-2, 0}, {-1, 0}, {0, 0}, {1, 0}, {2, 0}}
	wallTableB = [9][2]int32{{-2, 2}, {-1, 1}, {0, 0}, {1, -1}, {2, -2}, {-1, 2}, {0, 1}, {1, 0}, {2, -1}}
)

var wallDirs = [8]struct {
	diagonal     bool
	swap         bool
	signX, signY int32
}{
	0: {false, false, 1, -1}, //
	1: {true, false, 1, -1},  //
	2: {false, true, 1, 1},   //
	3: {true, false, 1, 1},   //
	4: {false, false, -1, 1}, //
	5: {true, false, -1, 1},  //
	6: {false, true, -1, 1},  //
	7: {true, false, -1, -1}, //
}

func (w *World) wallCells(cx, cy int32, d uint8) []uint16 {
	s := wallDirs[d&7]
	table := wallTableA[:]
	if s.diagonal {
		table = wallTableB[:]
	}
	p := make([][2]int32, 0, len(table))
	for _, o := range table {
		dx, dy := o[0], o[1]
		if s.swap {
			dx, dy = dy, dx
		}
		p = append(p, [2]int32{cx + s.signX*dx, cy + s.signY*dy})
	}
	return w.cellsWhere(p)
}

// skipOccupiedGround drops the cells Wall of Earth refuses.
//
// `MAGIC-WALLEARTH-042`: the per-cell add tests the cell record's `+0x4` slot
// for id 0x13 only and skips the cell when it is non-null. That slot takes an
// actor of movement domain 1 or 2 (`TERR-CELLREC-146`), which is this build's
// layer 0 — ground and ghost — so a flyer overhead does not refuse a cell and a
// body lying on one does. Refusing on "any ground-layer occupant present" is
// unaffected by R3-A2 below: whether the slot decodes as one actor or this
// build widens it to several, an occupied cell is occupied either way.
//
// IT IS THE FOOTPRINT AND NOT THE ANCHOR. An actor's pointer is stored in every
// cell of its n x n footprint and the entry is all-or-nothing
// (`TERR-FOOTPRINT-147`), so a cell inside a large unit holds that unit. Testing
// `e.X == x && e.Y == y` painted six wall cells inside a live 3x3 unit.
func (w *World) skipOccupiedGround(cells []uint16) []uint16 {
	out := cells[:0]
	for _, k := range cells {
		x, y := keyCell(k)
		if ground, _ := w.cellLayerOccupants(x, y); len(ground) == 0 {
			out = append(out, k)
		}
	}
	return out
}

// cellLayerOccupants is every actor covering (x, y) that would reach a cell
// record's occupant slot, split into the two movement layers: ground and
// ghost together, the flyer alone.
//
// THE DECODED RECORD CAPS EACH LAYER'S SLOT AT ONE ACTOR
// (`TERR-CELLREC-146`, High): `+0x4` takes an actor of domain 1 or 2, `+0x8`
// an actor of domain 3, and both adds (at L02040 and
// L02041) test their slot for occupancy and return without storing when the
// slot is already taken. So the cap is a consequence of a
// footprint that ROM1's own movement never lets overlap another one — the
// branch above exists for a state validated movement does not produce.
//
// Reading one occupant per layer for that reachable state — fix round 2,
// superseded here — made the actor that lost the collision immune to every
// blast, ring and cloud on the shared cell (round 3 review, R3-A2; the since
// closed DIV-054 and the still-open DIV-055). This function returns every
// covering actor of a layer instead of
// the first one found, which is what applyAreaCells now walks. It keeps the
// one part of TERR-CELLREC-146 this build's own dwell ladder can maintain
// unilaterally — `cellRecordHolds` — and gives up only the per-layer cap that
// depends on a footprint invariant. Runtime movement and placement now enforce
// it, but constructors, decoded forms and explicit headless force-actions can
// still present an already-overlapping state. DIV-055 therefore remains open
// and this function stays total over every state the World type can hold.
//
// `+0xc` is the structure slot and `+0x10` the sack slot. Neither is an entity
// layer, so this helper still returns only ground and air actors;
// applyAreaCells performs the ring/blast structure read separately, after both
// actor groups and before advancing to the next cell.
//
// MOVE-REFRESH-012, MOVE-AREA-038
func (w *World) cellLayerOccupants(x, y int32) (ground, air []int) {
	if x >= 0 && x < 256 && y >= 0 && y < 256 {
		if c := w.motionCell(uint16(y)<<8 | uint16(x)); c != nil {
			for layer, slot := range []SavedActorSlot{c.Ground, c.Air} {
				if slot.Bound && w.currentMotionSlot(slot) {
					if i := indexOfEntity(w.entities, slot.Entity); i >= 0 {
						if layer == 0 {
							ground = append(ground, i)
						} else {
							air = append(air, i)
						}
					}
				}
			}
		}
	}
	for i := range w.entities {
		e := &w.entities[i]
		if !entityCoversCell(e, x, y) {
			continue
		}
		if m := w.motionFor(e.ID); m != nil && m.Current {
			continue
		}
		if !cellRecordHolds(e) {
			continue
		}
		if e.Domain.layer() == 0 {
			ground = append(ground, i)
		} else {
			air = append(air, i)
		}
	}
	return ground, air
}

// cellRecordHolds reports whether e stands in a cell record's actor slot.
//
// It is counted()'s life ladder without counted()'s flyer rule (route.go). An
// off-map actor holds no ground at all; a body holds its slot while it dwells or
// remains restorative-targetable, exactly as the route plane does. The flyer
// rule is deliberately NOT here: a flyer with an order contends with no mover,
// but it is still over the cell, and borrowing that rule would let a moving
// flyer fly out of a Fire Ball.
//
// Whether the original clears an actor's slot on death is not decoded. What is
// decoded is that the slot holds an actor and that the removal clears it; this
// build ties terminal removal to Dwell and keeps a restorable body's slot until
// revival or the finished-body floor.
func cellRecordHolds(e *Entity) bool {
	if e.OffMap {
		return false
	}
	if !e.Alive() {
		return e.Dying() || e.restorativeTargetable()
	}
	return true
}

func layerSpell(id uint16) bool {
	switch id {
	case 3, 7, 8, 12, 17, 19:
		return true
	}
	return false
}
func containsKey(a []uint16, k uint16) bool {
	for _, candidate := range a {
		if candidate == k {
			return true
		}
	}
	return false
}

func (w *World) resolveLayerConflicts(e *cellEffect) {
	if !layerSpell(w.spellArm(e.Spell)) && w.spellArm(e.Spell) != 2 {
		return
	}
	if w.spellArm(e.Spell) == 8 {
		kept := e.Cells[:0]
		for _, key := range e.Cells {
			if !w.areaLayerPresent(key, 3) {
				kept = append(kept, key)
			}
		}
		e.Cells = kept
	}
	for i := range w.effects {
		old := &w.effects[i]
		if old.Mode != areaModeCloud || !areaLayerConflict(w.spellArm(e.Spell), w.spellArm(old.Spell)) {
			continue
		}
		kept := old.Cells[:0]
		for _, key := range old.Cells {
			if !containsKey(e.Cells, key) {
				kept = append(kept, key)
			}
		}
		old.Cells = kept // the independent clock survives even when this is empty
	}
	for _, key := range e.Cells {
		for layer, spell := range areaLayerSpells {
			if areaLayerConflict(w.spellArm(e.Spell), spell) {
				w.clearSavedAreaLayer(key, layer)
			}
		}
	}
}

func sacrificeDamage(caster Entity, power uint16) (int32, int32) {
	total := int64(caster.HP) + int64(caster.Mana)
	base := max(0, min(255, total))
	spread := max(0, min(255, min(512, total+int64(power))-base))
	return int32(base), int32(spread)
}

func (w *World) applyAreaCells(e cellEffect, rule SpellRule, cells []uint16) {
	if e.Current != nil {
		d := SavedAreaDriver{Key: e.Key, Spell: e.Spell, Mode: e.Mode}
		var caster []EntityID
		if e.HasCaster {
			caster = []EntityID{e.Caster}
		}
		w.applySavedAreaPayload(d, SavedSpellEffect{AE44: &e.Current.Payload}, cells, caster...)
		return
	}
	w.scorchCells(rule.arm(), cells)
	power := int32(e.Power)
	if e.DamageMin != 0 || e.DamageMax != 0 {
		rule.DamageMin, rule.DamageMax, rule.Damaging = e.DamageMin, e.DamageMax, true
		// Fire Sacrifice's stored pair is the constructed damage object,
		// already derived from HP/mana/power, not a spell-table column pair.
		power = 0
	}
	ci := -1
	if e.HasCaster {
		ci = indexOfEntity(w.entities, e.Caster)
	}
	// Cell order is observable: every ring program paints one cell and walks
	// that cell's occupant slots before producing the next. Keep the cell list's
	// decoded order outermost and the world's stable entity-id order inside it.
	for _, k := range cells {
		x, y := keyCell(k)
		// WHICH LAYERS ARE READ DIFFERS BY MODE (`MAGIC-AREACELL-039`): the
		// cloud pulse reads only `+0x4` (`L05435`), while ring (`L05436`,
		// `L05437`, `L05438`) and blast (`L05439`, `L05440`,
		// `L05441`) read all three in `+0x4`, `+0x8`, `+0xc` order. The third
		// is the registered structure slot; clouds stop after the ground slot.
		//
		// EACH LAYER IS EVERY COVERING ACTOR AND NOT ITS FIRST ONE
		// (R3-A2; cellLayerOccupants above). The decoded record caps a layer's
		// slot at one, but this build does not maintain the footprint
		// invariant that cap depends on, so a layer here can hold more than
		// one actor — every one of them is walked, in the world's stable
		// entity-id order within the layer.
		ground, air := w.cellLayerOccupants(x, y)
		groups := [...][]int{ground, air}
		read := len(groups)
		if e.Mode == areaModeCloud {
			read = 1
		}
		for _, group := range groups[:read] {
			for _, i := range group {
				a := w.entities[i]
				if !spellTargetable(a, rule) || !w.areaHitAllowed(ci, rule, a) {
					continue
				}
				if rule.arm() == 19 {
					continue
				}
				// No walker-level object deduplication. Damage objects apply on
				// each reference; ordinary effects keep attachEffect's own timed
				// replacement/refresh rules (MAGIC-AREAAPPLY-038, UNIT-AREADIRECT-072).

				applied := false
				if rule.arm() == 2 {
					// MAGIC-FIREDIV-047 divides the copied base and spread bytes
					// independently by the actor footprint area. Reconstruct the
					// endpoint only afterwards, so their remainders cannot carry.
					// Each covered cell still performs one ordinary application.
					side := int64(a.TokenSize)
					if side < 1 {
						side = 1
					}
					adjusted := rule
					base, spread := spellDamageUnder(w.rules, rule, power)
					adjusted.DamageMin = int32(int64(uint8(base)) / (side * side))
					adjusted.DamageMax = adjusted.DamageMin + int32(int64(uint8(spread))/(side*side))
					applied = w.ordinaryAreaEffect(ci, i, adjusted, 0)
				} else {
					applied = w.ordinaryAreaEffect(ci, i, rule, power)
				}
				if !applied {
					continue
				}
				w.markSpellEffect(i, rule.ID)
				// NO CAST AWARD IS PAID HERE (DIV-238). One was, per application,
				// and it paid nothing for the six cloud and wall rows, which reach
				// this walk only on a later pulse whose call passed the award over,
				// and nothing for any area row whose landing reached no unit.
				// castSpell and castBookAt pay the cast's own single award.
				if rule.arm() == 17 && ci >= 0 && ci != i {
					w.orderAttack(i, w.entities[ci].ID)
				}
			}
		}
		if e.Mode != areaModeCloud && rule.arm() != 19 && rule.Damaging {
			w.applyStructureSpellAt(x, y, rule, power)
		}
	}
}

// applyStructureSpellAt walks the structure slot after ground and air, and
// applies one Building direct-damage call per current reference. Building's
// spell-2 size divisor is one, not its rectangle (UNIT-AREAHP-073).
func (w *World) applyStructureSpellAt(x, y int32, rule SpellRule, power int32) bool {
	i, ok := w.structureSlots[cellKey(x, y)]
	if !ok {
		return false
	}
	s := &w.structures[i]
	base, spread := spellDamageUnder(w.rules, rule, power)
	// The direct effect carries byte combat components. The Building resolver
	// ignores even a positive base when the spread byte is zero, consumes no
	// roll in that arm, and subtracts five after a positive-spread roll.
	damage := int32(0)
	if s.MaxHealth != 0 && uint8(spread) != 0 {
		damage = max(int32(uint8(base))+w.rng.uniform(int32(uint8(spread)))-5, 0)
	}
	// Subtract as a word, then clamp a negative signed word. Do not refuse
	// HP zero: retained aliases still reach the resolver on later visits.
	s.Field42 -= uint16(damage)
	if int16(s.Field42) < 0 {
		s.Field42 = 0
	}
	return true
}

func (w *World) areaHitAllowed(ci int, rule SpellRule, target Entity) bool {
	if rule.AreaHits == AreaHitsAll || ci < 0 || ci >= len(w.entities) {
		return true
	}
	caster := w.entities[ci]
	if target.ID == caster.ID || target.Owner != 0 && target.Owner == caster.Owner {
		return false
	}
	return rule.AreaHits != AreaHitsHostile || w.hostileTo(&caster, &target)
}

func entityCoversCell(e *Entity, x, y int32) bool {
	side := footprintSide(e.TokenSize)
	return x >= e.X && y >= e.Y && x < e.X+side && y < e.Y+side
}

// applyPrismatic is Prismatic Spray's own arm (MAGIC-SING-019 (g)). Both
// delivery arms apply prismaticVictims' list and answer each victim's cell in
// that order, read before the apply: the drawing tier stamps one figure per
// entry (MAGIC-SPRAY-136). The secondaries pass the caster group's filters;
// the primary bypasses them and is never a body (DIV-072).
func (w *World) applyPrismatic(ci, ti int, rule SpellRule, power int32) []CellPoint {
	return w.applyPrismaticItem(ci, ti, rule, power, false)
}

func (w *World) applyPrismaticItem(ci, ti int, rule SpellRule, power int32, itemCast bool) []CellPoint {
	if rule.Delivery == 2 {
		return w.preparePrismatic(ci, ti, rule, power, itemCast)
	}
	if ti < 0 || ti >= len(w.entities) || prismaticBodyPrimary(w.entities[ti], rule) {
		return nil
	}
	var victims []CellPoint
	for _, id := range w.victimIDs(w.prismaticVictims(ci, ti, rule, power)) {
		i := indexOfEntity(w.entities, id)
		if i < 0 || !spellTargetable(w.entities[i], rule) {
			continue
		}
		victims = append(victims, CellPoint{X: w.entities[i].X, Y: w.entities[i].Y})
		if w.ordinaryEffect(ci, i, rule, power) {
			w.markSpellEffect(i, rule.ID)
			if ci >= 0 && !itemCast {
				w.awardSkill(ci, int32(rule.School), (int64(rule.ManaCost)+1)/2, -1)
			}
		}
	}
	return victims
}

func (w *World) victimIDs(indices []int) []EntityID {
	ids := make([]EntityID, len(indices))
	for k, i := range indices {
		ids[k] = w.entities[i].ID
	}
	return ids
}

// setCellEffectTime is instant 29's whole arm.
//
// It addresses the six current cell layers. A matching current owner receives
// the new lifetime; overwritten independent objects at that anchor do not.
// A covered cell need not be the owner's anchor. The legacy unshaped mode0
// retains its old anchor-only behavior.
//
// THE COMPARISON IS AGAINST THE FULL 32-BIT PARAMETER (`TRIG-CELLEFFECT-045`,
// superseding `TRIG-EFFECTTIME-034`'s `(u8)p2`): the arm zero-extends the
// effect's own id BYTE into a register and compares it with the whole dword. An
// authored id of 256 or more therefore matches nothing, where a byte-truncated
// comparison would predict a match. Every shipped node authors 3 or 19.
//
// A DURATION OF ZERO IS WRITTEN AS THE ORIGINAL WRITES IT and is not refused
// here: the store is a word write with no test in front of it. The effect is
// then removed by the next decay pass, which is what an authored 0 asks for. No
// shipped node authors zero — the shipped values are 1, 30000 and 60000.
//
// IT SENDS NOTHING. `TRIG-CELLEFFECT-045` reads the whole 24-instruction body
// and finds no packet helper: instant 29 is invisible until the effect's own
// tick changes what is drawn. This build has nothing to send, so that property
// costs it nothing to keep.
func (w *World) setCellEffectTime(x, y, spell, duration int32) {
	key := cellKey(x, y)
	life := uint16(duration)
	for i := len(w.effects) - 1; i >= 0; i-- {
		e := &w.effects[i]
		if int32(e.Spell) != spell {
			continue
		}
		if e.Mode == 0 && e.Key == key {
			e.Remaining = life
			continue
		} // legacy anchor-only records
		if e.Mode == areaModeCloud && containsKey(e.Cells, key) {
			e.Remaining = life
			return
		}
	}

	if retained := w.savedWorldEffects; retained != nil {
		for _, d := range retained.Areas {
			if d.Root >= 0 && d.Mode == areaModeCloud && int32(d.Spell) == spell && containsKey(d.Cells, key) {
				w.savedSpellEffects[d.Root].AE4C = life
				w.captureSavedGraphAreas()
				return
			}
		}
	}
	// An effect written to zero is removed on the next decay pass rather than
	// here, so the arm leaves the same state whatever order its writes happen
	// in. Nothing between here and that pass reads an area effect at all.
}
