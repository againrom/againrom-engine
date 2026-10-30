package game

import (
	"image"
	"math"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// spellBolt is one path figure's or shower's input: the two cells it runs
// between, the picture, the driver calls made and the figure seed. The flying
// object itself is a World record (flight.go); a path record is read into
// one spellBolt per link for the path producer.
type spellBolt struct {
	from, to image.Point
	picture  int
	owner    uint32
	// age is the record's actionphase, which reseeds the figure each call,
	// and phase the record's own phase (MAGIC-281).
	age, phase int

	// seed is the figure's own generator seed and tag the record tag the
	// second path picture adds its per-point phase from.
	seed uint32
	tag  int

	// launch is the object's start relative to the from cell's centre, in
	// ShotScale units (castOffset, MAGIC-261).
	launch image.Point
}

// healBurst is the client lifetime of one Heal or Drain shower. Its anchor is
// the target's application cell, deliberately fixed for the short effect: it
// survives target movement/removal without retaining a simulation object.
type healBurst struct {
	at      image.Point
	picture int
	owner   uint32
	seed    uint32
	tile    int
	drain   bool
	age     int
}

const (
	healBurstLife    = 32
	healSpawnTicks   = 25 // countdown 32 through 8, inclusive
	healParticleLife = 8  // visible phases 0 through 7
)

type showerParticle struct {
	dx, depth int
}

// showerCohorts replays the retained deterministic particle policy from age
// zero. The target-mark claims do not establish its native recurrence.
func showerCohorts(b healBurst, through int) [][]showerParticle {
	tile := b.tile
	if tile < 1 {
		tile = 1
	}
	rng := boltRNG{state: b.seed}
	out := make([][]showerParticle, through+1)
	for tick := 0; tick <= through; tick++ {
		n := rng.next()%3 + 3
		cohort := make([]showerParticle, n)
		for i := range cohort {
			a := float64(rng.next()%360) * math.Pi / 180
			cohort[i] = showerParticle{
				dx:    int(math.Trunc(math.Cos(a) * float64(16*tile))),
				depth: int(math.Trunc(math.Sin(a) * float64(8*tile))),
			}
		}
		out[tick] = cohort
	}
	return out
}

// observeCasts gives released records their appearance and plays cast cues.
//
// A BOOK CAST STARTS A CAST RUN ON ITS CASTER: the caster plays its own
// Attack run, once, one frame per tick, for the expanded Attack timeline's
// own length. A weapon release starts no run: its carrier already plays its
// attack run.
//
// A book cast and a staff's cast-diverted release each leave one cast record
// (SAV-1129, ANIM-147). A siege or weapon-spell rider leaves none: it builds
// its area effect and transport only (ANIM-115). Only Fire_Ball's blast and
// the staged stages build a burst record (ANIM-148).
func (mw *mapWorld) observeCasts(events []sim.CastEvent) (marks []sim.ProjectileMark) {
	for i, ev := range events {
		from := image.Point{X: int(ev.FromX), Y: int(ev.FromY)}
		to := image.Point{X: int(ev.ToX), Y: int(ev.ToY)}
		// The direct client-message selector belongs to the applied cast, not
		// command admission: a refused cast therefore has no event from which a
		// sound could be produced. It is separate from the cast animation's
		// AttackDelay hook and both may lawfully select the same sample.
		source := "book-cast-direct"
		if ev.Weapon {
			source = "weapon-cast-direct"
		}
		mw.playSpellSound(castSpellSoundSlot(mw.spells(), ev.Spell), ev.Owner, from, source)
		if !ev.Weapon {
			span := mw.startBookCastRun(ev)
			mw.scheduleBookCastSound(ev, span)
		}
		if !ev.Produced {
			marks = append(marks, mw.releaseObservedCasts(events[i:i+1])...)
			ev = events[i]
		}
		if ev.Produced {
			mw.keepCastLook(ev, i)
		}
		mw.scheduleBlastSound(ev.Spell, ev.Owner, from, to, ev.Weapon && !ev.Rider)
		target, targetPresent := mw.entity(ev.Target)
		if arm := mw.world.SpellArm(ev.Spell); arm == 6 || arm == 11 {
			if mw.files().targetMarkOnRelease && !ev.AtCell && targetPresent {
				marks = append(marks, sim.ProjectileMark{Target: target.ID,
					Picture: int32(data.CastPicture(int(ev.Spell))), Binding: target.SourceBinding})
			} else {
				mw.healBursts = append(mw.healBursts, healBurst{at: to,
					picture: data.CastPicture(int(ev.Spell)), owner: ev.TargetOwner,
					seed: mw.visualCastSeed(ev.Caster, ev.Target, int(ev.Spell), ev.AtCell), tile: mw.effectTileSize(ev.Target), drain: arm == 11})
			}
		}
	}
	return marks
}

func (mw *mapWorld) effectTileSize(id sim.EntityID) int {
	e, ok := mw.entity(id)
	if !ok || mw.units == nil {
		return 1
	}
	return markTileSize(mw.units.Classes[e.Class])
}

// chainTagCount is how many record tags the second path picture cycles through.
// The engine tags each generated set with its victim's loop index modulo seven,
// and that sheet holds seven blocks of five phases.
//
// A CAST THAT REPORTS A VICTIM LIST TAGS BY VICTIM INDEX (recordPaths). What
// fills the caster's victim array in the original is not decoded
// (`MAGIC-BOLTLIST-071`'s Unknown); this build fills it from the one arm that
// reaches more than one actor and falls back to the cast's own index inside the
// tick for every row that reaches exactly one.
const chainTagCount = 7

// castSeed is one cast object's figure seed. It is a function of the caster,
// the target and the spell, so the same replay draws the same bolt and two
// casts in one tick do not draw one figure twice. A book cast takes the three
// off its observation and a weapon-borne release off the attacking entity.
func castSeed(caster, target sim.EntityID, spell int) uint32 {
	return uint32(caster)*2654435761 + uint32(target)*2246822519 + uint32(spell) + 1
}

// fireBallSpell is the spell whose burst the World builds as a record.
const fireBallSpell = 2

// fireBallBurstPicture is the stationary explosion behind spell 2. Its
// decoded 22-tick life over an 11-frame half clock would otherwise wrap its
// last visible tick from frame 10 back to frame 0 (ANIM-103).
const fireBallBurstPicture = 13

// seedRestoredRuns gives every entity restored mid-wind-up, and carrying no
// saved animation clock of its own, the run clock a
// fresh run would have reached: the phase memory advanceSwings compares
// against, and the swing count the action clock implies. Without it the first
// walk reads the restored run as one that has just begun and re-releases a
// shot already in flight. A run whose clock is unknown keeps the count one,
// past a release at tick zero.
func (mw *mapWorld) seedRestoredRuns() {
	for _, e := range mw.world.EntityView() {
		if _, held := mw.phase[e.ID]; held || !windUp(e.AttackPhase) || !e.Alive() {
			continue
		}
		mw.phase[e.ID] = e.AttackPhase
		// Every first-game World stamps its action clock from the session
		// sub-tick the command drain reads (SESS-TICK-006), so the elapsed
		// count is the swing count a fresh run reached.
		n, ok := mw.world.WindUpElapsed(e.ID)
		if !ok {
			n = 1
		}
		mw.swing[e.ID] = n
		mw.shots.begin(e.ID, e.AttackPhase)
	}
}

// castDistance is the caster-to-target distance in the engine's own position
// units, where one cell spans 256.
//
// IT IS AN INTEGER SQUARE ROOT and not a float. The engine converts a distance
// term with ftol, whose rounding is not published, so any form here is ours; an
// integer one is chosen so the one arithmetic a player can see reproduces on
// every machine. It truncates, so a diagonal crossing is at most one tick
// shorter than a real-valued term would give.
func castDistance(from, to image.Point) int {
	dx, dy := to.X-from.X, to.Y-from.Y
	return data.PictureCellUnits * isqrt(dx*dx+dy*dy)
}

// isqrt is the integer square root: the largest n with n*n <= v, and 0 for any
// v at or below 0. Newton's iteration from a power-of-two seed, so it terminates
// in a fixed number of steps at every input and divides by nothing that can be
// zero.
func isqrt(v int) int {
	if v <= 0 {
		return 0
	}
	x := v
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + v/x) / 2
	}
	return x
}

// castRunIdentity is the identity of one particular visible book-cast run.
// It deliberately occupies one byte: Go may coalesce pointers to distinct
// zero-sized values, while a delayed sound must distinguish a replacement run
// from the one it cancelled. The identity is presentation-only and is neither
// serialized nor hashed.
type castRunIdentity struct{ _ byte }

// castRun is one book caster's run: how many ticks of it are left, how many
// it stands for in total, and which exact run it is. The span is kept
// because the drawn frame is SCALED to it and the remaining count alone
// cannot say what it started from. identity lets a delayed AttackDelay hook
// prove that the run which scheduled it is still the run the caster is
// visibly playing.
type castRun struct {
	left, span int
	identity   *castRunIdentity
}

const castRunFallbackTicks = 8

// startCastRun puts a caster into its own attack run and answers the span
// that run stands for.
//
// THE SPAN IS THE CASTER'S OWN ATTACK CHARGE, the wind-up a weapon-borne cast by
// this same actor resolves on. So a fast caster swings fast and a slow one
// swings slowly, and a book cast and a staff cast by one actor take the same
// time on screen. A caster naming no charge takes castRunFallbackTicks.
//
// IT NO LONGER SIZES THE PROJECTILE. 0154 gave the bolt this same span so
// that one swing covered one projectile; the picture's own flight length is
// decoded and this is not, so the object takes its own and the swing keeps
// this one.
//
// THE SWING CLOCK IS SEEDED ONE BELOW ZERO because advanceSwings' own increment
// runs later in the same tick: the run then lands on its first frame through the
// same increment every other frame of it comes from.
//
// A CAST ARRIVING INSIDE A RUN RESTARTS IT: the previous swing is cut off
// and a fresh one begins, so no cast is ever drawn without a swing of its
// own starting with it.
func (mw *mapWorld) startCastRun(id sim.EntityID) int {
	e, ok := mw.entity(id)
	if !ok {
		return castRunFallbackTicks
	}
	span := int(e.AttackCharge)
	if span < castRunFallbackTicks {
		span = castRunFallbackTicks
	}
	mw.castRun[id] = castRun{left: span, span: span, identity: &castRunIdentity{}}
	mw.swing[id] = -1
	return span
}

// startBookCastRun starts the run a book cast event plays. A first-game
// Teleport is reported at its admission and its relocation follows on the
// countdown, so its run is the caster class's own Attack timeline, one phase
// per tick, drawn across the move (ANIM-TELEPORT-172). Every other cast keeps
// the run scaled to its charge.
func (mw *mapWorld) startBookCastRun(ev sim.CastEvent) int {
	if mw.files().teleportCastRun && mw.world.SpellArm(ev.Spell) == teleportArm {
		if c := mw.casterClass(ev.Caster); c != nil && len(c.Anim.AttackTrack) > 0 {
			// advanceCastRuns takes a tick off in the admitting tick itself,
			// so a run drawn on span ticks starts one above its span.
			span := len(c.Anim.AttackTrack)
			mw.castRun[ev.Caster] = castRun{left: span + 1, span: span, identity: &castRunIdentity{}}
			mw.swing[ev.Caster] = -1
			return span
		}
	}
	return mw.startCastRun(ev.Caster)
}

// teleportArm is Teleport's spell arm.
const teleportArm = 26

// casting reports that id is inside a cast run this tier is playing. It is
// the entity's OWN half of the swing gate: an entity resolving an attack
// order swings because it holds a victim, and a book caster holds none.
func (mw *mapWorld) casting(id sim.EntityID) bool { return mw.castRun[id].left > 0 }

// castSwingSpan is the interval one cast's swing must cover, and whether
// this entity is casting at all.
//
// TWO PRODUCERS, ONE ANSWER: a weapon-borne cast is winding up, so the interval
// is the countdown the release resolves on; a book cast is inside a run this
// tier started, so it is that run's own span. A MELEE SWING ANSWERS FALSE and is
// drawn exactly as it was before this story — the owner's rule is about magic
// projectiles, and widening it to the blow would change a picture already
// accepted.
func (mw *mapWorld) castSwingSpan(e sim.Entity) (int, bool) {
	if e.HasAttackTarget && e.AttackPhase == sim.AttackCasting {
		span := int(e.AttackCharge)
		if span < 1 {
			span = 1
		}
		return span, true
	}
	if r := mw.castRun[e.ID]; r.left > 0 {
		return r.span, true
	}
	return 0, false
}

func scaleRun(elapsed, span, track int) int {
	if track <= 0 || elapsed < 0 {
		return 0
	}
	if span < 1 {
		span = 1
	}
	if i := elapsed * track / span; i < track {
		return i
	}
	return track - 1
}

// advanceCastRuns takes one tick off every live cast run and drops the ones that
// have played out. It rides tick's own post-step window beside advanceSwings,
// which is where the frame counter those runs index is advanced.
func (mw *mapWorld) advanceCastRuns() {
	for id, r := range mw.castRun {
		if mw.world.HasEffectArm(id, 20) {
			continue
		}
		if r.left <= 1 {
			delete(mw.castRun, id)
			continue
		}
		r.left--
		mw.castRun[id] = r
	}
}

func (mw *mapWorld) advanceHealBursts() {
	live := mw.healBursts[:0]
	for _, b := range mw.healBursts {
		b.age++
		if b.age < healBurstLife {
			live = append(live, b)
		}
	}
	mw.healBursts = live
}

// showerMarkRecords is the shared Heal/Drain geometry builder. The engine
// retains 25 cohorts and eight particle phases as a bounded presentation policy.
func showerMarkRecords(b healBurst) []terrain.EffectMark {
	tile := max(b.tile, 1)
	lastSpawn := min(b.age, healSpawnTicks-1)
	if b.age < 0 || b.age >= healBurstLife {
		return nil
	}
	first := max(lastSpawn-healParticleLife+1, 0)
	cohorts := showerCohorts(b, lastSpawn)
	step := int(math.Trunc(1 + float64(32*tile)/7))
	var out []terrain.EffectMark
	for tick := first; tick <= lastSpawn; tick++ {
		particleAge := b.age - tick
		if particleAge < 0 || particleAge >= healParticleLife {
			continue
		}
		dy := -step * particleAge
		if b.drain {
			dy = -32*tile + step*particleAge
		}
		for _, particle := range cohorts[tick] {
			out = append(out, terrain.EffectMark{DX: particle.dx, DY: dy, Depth: particle.depth, Record: b.picture, Phase: particleAge})
		}
	}
	return out
}

func (mw *mapWorld) healSpriteDraws() []ui.HealSprite {
	var out []ui.HealSprite
	for _, b := range mw.healBursts {
		sheet := mw.projectiles.Sheet(b.picture)
		if sheet == nil || sheet.Phases <= 0 {
			continue
		}
		base := image.Pt(b.at.X*ui.ShotScale, b.at.Y*ui.ShotScale)
		for _, m := range showerMarkRecords(b) {
			frame, _, ok := terrain.SelectEffectFrame(sheet, 0, m.Phase)
			if !ok {
				continue
			}
			pos := base.Add(image.Pt(m.DX, m.DY-m.Depth).Mul(boltUnitsPerPixel))
			out = append(out, ui.HealSprite{Cell: b.at, Pos: pos, Sheet: sheet, Frame: frame, Owner: b.owner})
		}
	}
	return out
}

// boltDraws is what the viewer is handed this frame: every World record in
// flight with its trail, then burning scenery and the standing clouds.
//
// A REFUSED SELECTION DROPS THE OBJECT AND NOT THE FRAME. A picture with no
// sheet, a sheet with no frames and an index past the sheet each produce no
// entry at all, which is the engine's own answer to the same three.
func (mw *mapWorld) boltDraws(ents []sim.Entity) []ui.SpellBolt {
	out := mw.savedProjectileScene(true)
	out = append(out, mw.burningSceneryDraws()...)
	return append(out, mw.areaEffectDraws(mw.world.CellEffects(), ents)...)
}

func (mw *mapWorld) burningSceneryDraws() []ui.SpellBolt {
	if mw.view == nil {
		return nil
	}
	var out []ui.SpellBolt
	// Reuse the installed stationary fire sheet and its established cell phase.
	sheet := mw.projectiles.Sheet(15)
	if sheet == nil {
		return nil
	}
	for _, cell := range mw.view.BurningScenery() {
		frame, ok := terrain.OverlayFrame(3, sheet.Phases, int(mw.world.Tick()), cell.X, cell.Y)
		if ok && sheet.Frame(frame) != nil {
			out = append(out, ui.SpellBolt{Cell: cell, To: cell,
				Pos:   image.Pt(cell.X*ui.ShotScale, cell.Y*ui.ShotScale),
				Sheet: sheet, Frame: frame, Pass: ui.SpellOverlayA})
		}
	}
	return out
}

// areaEffectDraws is the RETAINED per-cell overlay: the cells a standing
// cloud currently holds, each drawn from the four baked overlay sheets.
//
// ONLY A CLOUD IS DRAWN HERE, and that is the whole shape of the three tick
// modes (`MAGIC-AREADRAW-049`). A cloud creates no object at all: it registers
// its cells in the map layer and the map draw re-reads that mask every frame,
// which is what this loop is. A STAGED effect registers no map layer — it sends
// one message per accepted cell and each builds a transient object, which
// arrives here as an AreaPaint and becomes a burst record. A BLAST retains
// no record at all; its single centre object comes through the cast event.
// Drawing a staged record's cells here as well drew each stage for the three
// ticks until the next replaced it, so nothing ever overlapped and the effect
// did not spread.
//
// FOUR SPELLS HAVE OVERLAY ART AND SIX REACH THE CLOUD MODE
// (`MAGIC-OVERLAYART-051`, data.OverlayPicture). The other two paint their
// cells, damage through them and draw nothing.
//
// THE PHASE IS NOT THE RECORD'S AGE. What a cloud puts on the client carries no
// object, no counter and no phase per cell (`MAGIC-OVERLAY-050`), so a cell
// cannot remember when it started burning and its frame is a function of the
// free-running tick and of the cell alone — terrain.OverlayFrame. Passing the
// record's own age through the projectile clock is what made a burning cell
// cycle its birth and fade frames for as long as it burned.
func (mw *mapWorld) areaEffectDraws(effects []sim.CellEffect, ents []sim.Entity) []ui.SpellBolt {
	var out []ui.SpellBolt
	counter := int(mw.world.Tick())
	// Cells is the current owned coverage. An independent area clock can keep
	// running with no Cells; neither its anchor nor its damage scan supplies a
	// fallback footprint. MAGIC-OVERLAY-050's one bit per spell per cell still
	// projects one retained sprite if multiple input records name the same cell.
	// The free-running cell frame law discards no record-local phase here.
	type overlayCell struct {
		mask, drawable uint8
		draws          [4]ui.SpellBolt
	}
	// Keep first-seen cell order without iterating a map. Within each cell,
	// MAGIC-OVERLAYART-051 runs Fire then Earth in A, and Freezing else Poison
	// in B. These are mask arms, not the canonical records' insertion order.
	var cells []overlayCell
	indices := make(map[image.Point]int)
	for _, e := range effects {
		if e.Mode != sim.AreaModeCloud {
			continue
		}
		// The picture is the row's own cell identifier in the World's game;
		// the slot and pass follow the row's arm.
		picture, ok := mw.spells().OverlayPicture(int(e.Spell))
		if !ok {
			continue
		}
		arm, pass := 0, ui.SpellOverlayA // MAGIC-OVERLAYART-051
		switch mw.world.SpellArm(e.Spell) {
		case 3:
		case 19:
			arm = 1
		case 7:
			arm, pass = 2, ui.SpellOverlayB
		case 8:
			arm, pass = 3, ui.SpellOverlayB
		default:
			continue
		}
		sheet := mw.projectiles.Sheet(picture)
		owner := uint32(0)
		if e.HasCaster {
			if caster, ok := entityIn(ents, e.Caster); ok {
				owner = caster.Owner
			}
		}
		for _, c := range e.Cells {
			at := image.Pt(int(c[0]), int(c[1]))
			i, seen := indices[at]
			if !seen {
				i = len(cells)
				indices[at] = i
				cells = append(cells, overlayCell{})
			}
			cell := &cells[i]
			bit := uint8(1 << arm)
			cell.mask |= bit
			if cell.drawable&bit != 0 || sheet == nil {
				continue
			}
			frame, ok := terrain.OverlayFrame(int(e.Spell), sheet.Phases, counter, at.X, at.Y)
			if !ok || sheet.Frame(frame) == nil {
				continue
			}
			cell.drawable |= bit
			cell.draws[arm] = ui.SpellBolt{
				Cell:  at,
				To:    at,
				Pos:   image.Pt(at.X*ui.ShotScale, at.Y*ui.ShotScale),
				Sheet: sheet, Frame: frame, Owner: owner, Pass: pass,
			}
		}
	}
	for _, cell := range cells {
		for arm, draw := range cell.draws {
			if cell.drawable&(1<<arm) == 0 || arm == 3 && cell.mask&(1<<2) != 0 {
				continue
			}
			out = append(out, draw)
		}
	}
	return out
}

// observeAreaPaints voices staged cells and hands the World the ones its step
// built no records for.
func (mw *mapWorld) observeAreaPaints(paints []sim.AreaPaint) {
	for i, p := range paints {
		// Picture 51 sounds from its common driver phase; other staged cells
		// retain their construction sound, including ROM2's delayed cell.
		slot, delay := mw.spells().CellSound(int(p.Spell))
		if mw.files().commonBurstSound && data.BurstPicture(int(p.Spell)) == 51 {
			slot = 0
		}
		source := "spell-effect"
		if delay > 0 {
			source = "storm-phase"
		}
		for _, c := range p.Cells {
			at := image.Pt(int(c.X), int(c.Y))
			mw.queueSpellSound(slot, p.Owner, at, delay, source)
		}
		mw.releaseObservedPaints(paints[i : i+1])
	}
}

func (mw *mapWorld) observeProjectilePhases(events []sim.ProjectilePhase) {
	for _, ev := range events {
		row := uint16((ev.Picture - 9) / 2)
		mw.queueSpellSound(effectSpellSoundSlot(mw.spells(), row), ev.Owner,
			image.Pt(int(ev.CellX), int(ev.CellY)), 0, "storm-phase")
	}
}

// entityIn is one entity of a snapshot by id, on indexOfEntity's own linear
// shape one package down: the snapshots this reads are a mission's entity list
// and want no index.
func entityIn(ents []sim.Entity, id sim.EntityID) (sim.Entity, bool) {
	for _, e := range ents {
		if e.ID == id {
			return e, true
		}
	}
	return sim.Entity{}, false
}

// savedProjectileFacing is the sheet facing a record's `dir` draws at: the
// draw subtracts 8 and keeps four bits (ANIM-PROJ-026).
func savedProjectileFacing(dir int32) int {
	return int(dir-8) & 0xf
}

// EffectFacing is the sheet facing of a flight running (dx, dy): the one
// projectile direction helper (ANIM-138) in the sheet's wheel.
func EffectFacing(dx, dy int) int {
	return savedProjectileFacing(sim.ProjectileDirection(int32(dx), int32(dy)))
}

// savedProjectileDraws is the records' draws without their trails.
func (mw *mapWorld) savedProjectileDraws() []ui.SpellBolt { return mw.savedProjectileScene(false) }

// armedRecords is the World's records with a live driver row, in store order,
// and each one's draw owner.
func (mw *mapWorld) armedRecords() ([]sim.SavedProjectile, map[uint16]uint32) {
	d := mw.world.SavedWorldEffectDrivers()
	if d == nil {
		return nil, nil
	}
	owners := map[uint16]uint32{}
	for _, row := range d.Projectiles {
		if !row.Retired {
			owners[row.ID] = row.Owner
		}
	}
	var out []sim.SavedProjectile
	for _, p := range mw.world.SavedProjectiles().Items {
		if _, armed := owners[p.ID]; armed {
			out = append(out, p)
		}
	}
	return out, owners
}

// savedProjectileScene is the one draw producer of objects in flight: each
// armed record from its own position, direction and phase, a path record as
// its figures, each followed by its smoke trail when trails is set.
func (mw *mapWorld) savedProjectileScene(trails bool) []ui.SpellBolt {
	records, owners := mw.armedRecords()
	set := mw.spells()
	var out []ui.SpellBolt
	for _, p := range records {
		if set.Effect(p.Picture).Hidden {
			continue
		}
		if set.DrawsPath(p.Picture) {
			for _, b := range mw.recordPaths(p, owners[p.ID]) {
				out = append(out, mw.pathDraws(b)...)
			}
			continue
		}
		cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
		if p.Picture == unitShotDeformationPicture {
			if !mw.projectiles.HasPicture(int(p.Picture)) {
				continue
			}
			out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: image.Pt(int(p.X), int(p.Y)), AbsolutePosition: true,
				Effect: ui.SpellBackgroundDeformation, Phase: int(p.Phase)})
			continue
		}
		sheet := mw.projectiles.Sheet(int(p.Picture))
		drawn, raise := set.DrawnFrame(p.Picture, int(p.Phase))
		frame, mirror, ok := terrain.SelectEffectFrame(sheet, savedProjectileFacing(p.Dir), drawn)
		if !ok {
			continue
		}
		out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: image.Pt(int(p.X), int(p.Y)), AbsolutePosition: true, Sheet: sheet, Frame: frame, Mirror: mirror, Owner: owners[p.ID], Raise: raise})
		if trails {
			out = append(out, mw.savedProjectileTrail(p)...)
		}
	}
	return out
}

// savedProjectileTrail is the smoke behind a record of a smoke-leaving
// picture: one stamp per trail point, oldest first (ANIM-140), point i drawing
// frame i, so the newest of six draws frame 5, and only for a row whose
// Palette is not 0 (ANIM-142).
func (mw *mapWorld) savedProjectileTrail(p sim.SavedProjectile) []ui.SpellBolt {
	slot := mw.spells().TrailSlot(p.Picture)
	if slot < 0 || !mw.projectiles.Smokes(int(p.Picture)) {
		return nil
	}
	sheet := mw.projectiles.SmokeSheet(slot)
	if sheet == nil {
		return nil
	}
	var out []ui.SpellBolt
	trail := mw.shots.trail[p.ID]
	for k, at := range trail {
		frame, _, ok := terrain.SelectEffectFrame(sheet, 0, k)
		if !ok {
			continue
		}
		cell := image.Pt(at.X/ui.ShotScale, at.Y/ui.ShotScale)
		out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: at, AbsolutePosition: true, Sheet: sheet, Frame: frame})
	}
	return out
}
