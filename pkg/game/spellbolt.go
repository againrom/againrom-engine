package game

import (
	"image"
	"math"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// spellBolt is one cast's map object: the two cells it runs between, the picture
// it draws, how many ticks it lives and how many of them have run.
//
// THE CELLS ARE THE OBSERVATION'S AND ARE NEVER RE-READ. sim.CastEvent carries
// where both actors stood when the cast landed, so a caster that walks away, and
// a target that is felled and decays, leave the object running between the cells
// the cast actually crossed.
//
// A BURST IS THE SAME STRUCT WITH BOTH ENDS EQUAL. A burst does not move —
// the engine clears its target and copies its own position into the fields
// the driver interpolates toward, so every one of the driver's divisions
// works on zero — and "both ends are the same cell" is exactly that, in
// the one memory that already ages, compacts and pushes.
type spellBolt struct {
	from, to image.Point
	picture  int
	owner    uint32
	age      int

	// life is how many ticks this object is drawn for: the picture's own flight
	// length for a flying object and the sender's chosen lifetime for a burst.
	// It is NOT the caster's attack charge, which is what 0154 gave a bolt: the
	// picture's length is decoded and the swing's is not, so the two are
	// allowed to come apart and a long cast outlives its swing.
	life int

	// delay is how many of those ticks pass before the object is drawn at all,
	// and it is a BURST's alone. This build's simulation applies a cast at the
	// release tick and carries no flight time, so a burst spawned at the
	// observation would begin while the cast object it belongs to is still
	// crossing. Holding it back by that object's own drawn length puts the
	// explosion at the arrival, which is where it is seen.
	delay int

	// seed and tag belong to a PATH object. seed is the figure's own generator
	// seed and tag is the record tag the second path picture adds its per-point
	// phase from. Both are zero for every other picture.
	seed uint32
	tag  int

	// facing is the caster's own facing at the moment of the cast. It is zero
	// for a burst or an area-paint object.
	facing uint8
	// A runtime-id-zero source is a cell, not a person's hand (1084).
	centered bool
	// launch is the object's start relative to the from cell's centre, in
	// ShotScale units (castLaunch, MAGIC-261). Zero for a burst, an area-paint
	// object and a cell source.
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

// showerCohorts reproduces MAGIC-090's RNG consumption from tick zero so a
// redraw is stable: one draw chooses 3..5 records, then one angle draw supplies
// both coordinates of every record. The original seeds its process RNG from
// time; this client seeds from the cast observation so replay presentation is
// deterministic while retaining the decoded recurrence, order and geometry.
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

// observeCasts turns one tick's cast observations into what the front end
// draws.
//
// A BOOK CAST STARTS A CAST RUN ON ITS CASTER: the caster plays its own
// Attack run, once, one frame per tick, for the expanded Attack timeline's
// own length. It also spawns whatever objects its two pictures name.
//
// A CASTER'S REPLACEMENT RELEASE STARTS NO RUN. Its caster is already resolving
// an attack order, so it is already playing that run and already drawing its
// picture off the attack cycle (weaponBoltDraws); a second run would draw the
// same cast twice. It still spawns a BURST, because the burst is a separate
// object at the target and the wind-up draws only the thing that flies.
//
// A RIDER RELEASE SPAWNS ONE LIKE A BOOK CAST. It is the other weapon arm: a
// fighter carrying a weapon whose spell rides its blows. Its carrier is not
// a mage, so advanceAttack never sets AttackCasting for it and there is no
// wind-up for a live draw to hang off; the release applies at the blow.
// Before this the arm was skipped with the caster's, drew nothing at all,
// and left the burst appearing at the target with nothing having flown to
// it. Three shipped Boulder Throwers carry Fire Ball on this arm.
//
// A CASTER'S RELEASE THAT REPORTS A VICTIM LIST SPAWNS ITS SET TOO. The live
// draw has one figure, attacker to attack target, and Prismatic Spray owes one
// per selected victim. The two never meet on one frame: the release tick
// leaves AttackCasting, so the wind-up figure's last frame is the tick before
// and the set's first is the release tick itself.
func (mw *mapWorld) observeCasts(events []sim.CastEvent) {
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
		mw.playSpellSound(castSpellSoundSlot(ev.Spell), ev.Owner, from, source)
		// THE RUN AND THE OBJECT ARE SEPARATE QUESTIONS HERE. A cast run is the
		// caster's own animation, and a rider's carrier is swinging a weapon,
		// not casting: it is already playing its attack run and replacing that
		// with a cast run would animate a blow as a spell. So a rider takes the
		// object and not the run.
		if !ev.Weapon {
			span := mw.startCastRun(ev.Caster)
			mw.scheduleBookCastSound(ev, span)
		}
		if !ev.Weapon || ev.Rider || len(ev.Victims) > 0 {
			mw.spawnCastSet(from, to, ev, i)
		}
		// A Fire_Ball burst is a World record built when the area blasts, and
		// is drawn from the record like a loaded one.
		if int(ev.Spell) != fireBallSpell {
			delay := mw.burstDelay(ev, from, to)
			mw.spawnBurst(to, int(ev.Spell), ev.Owner, delay)
		}
		mw.scheduleBlastSound(ev.Spell, ev.Owner, from, to, ev.Weapon && !ev.Rider)
		if arm := mw.world.SpellArm(ev.Spell); arm == 6 || arm == 11 {
			mw.healBursts = append(mw.healBursts, healBurst{at: to,
				picture: data.CastPicture(int(arm)), owner: ev.TargetOwner,
				seed: mw.visualCastSeed(ev.Caster, ev.Target, int(ev.Spell), ev.AtCell), tile: mw.effectTileSize(ev.Target), drain: arm == 11})
		}
	}
}

func (mw *mapWorld) effectTileSize(id sim.EntityID) int {
	e, ok := mw.entity(id)
	if !ok || mw.units == nil {
		return 1
	}
	return markTileSize(mw.units.Classes[e.Class])
}

// spawnCastSet puts one observation's cast objects on the map: ONE PER
// VICTIM the cast reached, each tagged with that victim's own index (owner).
//
// ONE SHIPPED ROW REACHES MORE THAN ONE ACTOR — Prismatic Spray, whose apply
// walks every hostile actor within a power-scaled radius of the target and
// reports them (`applyPrismatic`, pkg/sim). Its picture's sheet holds seven
// five-phase blocks and each block is a different colour: measured on the
// shipped `chain\sprites`, blocks 0 to 6 are cyan, green, yellow, red, magenta,
// purple and blue. The tag selects the block, so the cast draws one figure per
// victim and each figure is a different colour, which is the spell the owner
// named. Before this the whole spray drew one figure at the directed target.
//
// THE TAG IS THE VICTIM'S LOOP INDEX and not the cast's index inside the tick.
// That is `MAGIC-BOLTLIST-071`'s own rule, and it was unreachable while this
// build had one victim per observation; the cast index stays as the fallback
// for a row that reports no list.
//
// THE SEED MOVES WITH THE VICTIM so two figures of one spray are two different
// walks. Nothing here reaches pkg/sim: the list is observation, the seed is
// arithmetic on it, and neither is read back.
func (mw *mapWorld) spawnCastSet(from, to image.Point, ev sim.CastEvent, index int) {
	spell, owner := int(ev.Spell), ev.Owner
	seed := mw.visualCastSeed(ev.Caster, ev.Target, spell, ev.AtCell)
	class := mw.casterClass(ev.Caster)
	if len(ev.Victims) == 0 {
		mw.spawnCast(from, to, spell, owner, seed, index%chainTagCount, ev.Facing, class)
		return
	}
	for k, v := range ev.Victims {
		mw.spawnCast(from, image.Pt(int(v.X), int(v.Y)), spell, owner,
			seed+uint32(k)*2654435761, k%chainTagCount, ev.Facing, class)
	}
}

// spawnCast puts the flying object on the map, if this spell's picture flies
// at all.
//
// TWENTY-ONE OF THE TWENTY-EIGHT SPELLS SPAWN NOTHING HERE, and that is the
// rule and not a fallback: the picture id a spell computes indexes a hard-coded
// table of flight lengths whose value is zero for all but seven pictures, and a
// zero-length object never executes a driver arm.
//
// EVERY OBJECT STARTS AT ITS CASTER CLASS'S LAUNCH POINT (castLaunch,
// MAGIC-261): the staff tip, the hand or the Selection fallback the class
// and facing select.
//
// TELEPORT SPAWNS TWO stationary objects, both on the Selection fallback. The
// first stands at the caster's cell plus that delta; the second's raw point is
// the destination plus the same delta (MAGIC-265; DIV-2658).
func (mw *mapWorld) spawnCast(from, to image.Point, spell int, owner uint32, seed uint32, tag int, facing uint8, class *terrain.UnitClass) {
	picture := data.CastPicture(spell)
	life := data.CastFlight(picture, castDistance(from, to))
	if life <= 0 {
		return
	}
	launch := castLaunch(class, facing, picture)
	if picture == teleportPicture {
		arrive := image.Point{}
		if class != nil {
			arrive = castSelectionDelta(class)
		}
		mw.bolts = append(mw.bolts,
			spellBolt{from: from, to: from, picture: picture, owner: owner, life: life, seed: seed, tag: tag, facing: facing, launch: launch},
			spellBolt{from: to, to: to, picture: picture, owner: owner, life: life, seed: seed, tag: tag, facing: facing, launch: arrive})
		return
	}
	mw.bolts = append(mw.bolts, spellBolt{from: from, to: to, picture: picture, owner: owner,
		life: life, seed: seed, tag: tag, facing: facing, launch: launch})
}

// chainTagCount is how many record tags the second path picture cycles through.
// The engine tags each generated set with its victim's loop index modulo seven,
// and that sheet holds seven blocks of five phases.
//
// A CAST THAT REPORTS A VICTIM LIST TAGS BY VICTIM INDEX (spawnCastSet). What
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

// burstDelay is how long a burst waits for the cast object it belongs to.
//
// ONLY A TRAVELLING PICTURE DELAYS ONE. A path picture's object stands still for
// its whole life, and a spell whose cast picture puts nothing on the map has
// nothing to wait for. On the shipped book exactly one spell has both a
// travelling cast picture and a burst sheet: Fire Ball.
//
// THE TWO WEAPON ARMS DIFFER HERE. A caster's replacement release is drawn
// live off the swing and its picture is already at the victim on the release
// tick, so it waits nothing. A RIDER'S DOES WAIT: it spawns an ordinary cast
// object that travels caster to target exactly as a book cast's does, so
// without the wait a rider Fire Ball's explosion fires while its fireball is
// still in the air. That is the same defect the delay was written for, on
// the arm that reaches the three shipped Boulder Throwers.
func (mw *mapWorld) burstDelay(ev sim.CastEvent, from, to image.Point) int {
	if ev.Weapon && !ev.Rider {
		return 0
	}
	picture := data.CastPicture(int(ev.Spell))
	if !data.CastTravels(picture) {
		return 0
	}
	return data.CastFlight(picture, castDistance(from, to))
}

// teleportPicture is the one picture that puts TWO objects on the map: the
// spawner copy-constructs a second one (MAGIC-265).
const teleportPicture = 60

// fireBallSpell is the spell whose burst the World builds as a record.
const fireBallSpell = 2

// fireBallBurstPicture is the stationary explosion behind spell 2. Its
// decoded 22-tick life over an 11-frame half clock would otherwise wrap its
// last visible tick from frame 10 back to frame 0.
const fireBallBurstPicture = 13

// healingPicture has a decoded one-tick cast object, but the ordinary
// projectile draw arm deliberately submits no blit for it. The target-local
// positive-Heal consumer below is the only path that draws this sheet.
const healingPicture = 20

// spawnBurst puts the stationary burst object at the cell the cast landed
// on, if this spell's burst picture names a sheet.
//
// The sim owns area delivery and reports the semantic cast once. This client
// object is only the decoded stationary burst picture at the landing cell; it
// never performs damage or advances the area's pulse program.
//
// A SPELL WITH NO BURST SHEET SPAWNS NOTHING. It is cheaper to append an object that draws
// nothing than to ask the bundle here — but it would also age, compact and push
// every tick, and "nothing is drawn" would stop being visible in this file.
func (mw *mapWorld) spawnBurst(at image.Point, spell int, owner uint32, delay int) {
	picture := data.BurstPicture(spell)
	if mw.projectiles.Sheet(picture) == nil {
		return
	}
	mw.bolts = append(mw.bolts, spellBolt{
		from: at, to: at, picture: picture, owner: owner,
		life: delay + data.BurstLife(picture), delay: delay,
	})
}

// armBurstPhases hands the World the installed phase count of the Fire_Ball
// burst's sheet, which the World's burst record needs for its frame leaf.
func (mw *mapWorld) armBurstPhases() {
	if sheet := mw.projectiles.Sheet(fireBallBurstPicture); sheet != nil && sheet.Phases > 0 && sheet.Phases <= 65535 {
		mw.world.SetBurstPhases(uint16(sheet.Phases))
	}
}

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
		n, ok := mw.world.WindUpElapsed(e.ID)
		if !ok {
			n = 1
		} else {
			// The action clock counts from the original's swing start, the tick
			// before the clock's own zero reading.
			n--
			// The record is already in the document (SAV-1153).
			if n < 0 && mw.holdsDelayZeroRecord(e) {
				n = 0
			}
		}
		mw.swing[e.ID] = n
		mw.shots.begin(e.ID, e.AttackPhase)
	}
}

// holdsDelayZeroRecord: e's swing builds its record on the start tick.
func (mw *mapWorld) holdsDelayZeroRecord(e sim.Entity) bool {
	picture, delay, drawn := mw.classShot(mw.spellClientClass(e.ID, e.Class))
	return drawn && delay == 0 && picture <= unitShotRegistryTop &&
		e.HasAttackTarget && e.AttackTargetKind == sim.AttackTargetUnit && e.Reach > 1
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
		if r.left <= 1 {
			delete(mw.castRun, id)
			continue
		}
		r.left--
		mw.castRun[id] = r
	}
}

// advanceBolts ages every live object by one tick and drops the ones whose life
// has run out. It rides tick's own post-step window beside advanceSwings, so
// what the push selects is the age this tick produced.
//
// IT COMPACTS IN PLACE AND KEEPS THE SLICE'S OWN ORDER, which is the order the
// observations arrived in — ascending caster id inside one tick, ticks in order
// across them — so what the drawing tier receives is a function of the casts and
// not of a map walk.
func (mw *mapWorld) advanceBolts() { mw.advanceBoltsBornFrom(len(mw.bolts)) }

// advanceBoltsBornFrom ages every object except a path object at index born or
// later, spawned this tick: its first push draws driver call 1 (DIV-2686).
func (mw *mapWorld) advanceBoltsBornFrom(born int) {
	live := mw.bolts[:0]
	for i, b := range mw.bolts {
		if i < born || !data.CastDrawsPath(b.picture) {
			b.age++
		}
		if b.age < b.life {
			live = append(live, b)
		}
	}
	mw.bolts = live
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

// healSpriteDraws carries MAGIC-089/090's shower records. Countdown values 32
// through 8 append a 3..5 particle cohort. The most recent eight cohorts remain
// visible at phases 0..7. One angle supplies each particle's elliptical dx and
// depth; Heal rises from the target and Drain descends from one tile above it.
func (mw *mapWorld) healSpriteDraws() []ui.HealSprite {
	var out []ui.HealSprite
	for _, b := range mw.healBursts {
		sheet := mw.projectiles.Sheet(b.picture)
		if sheet == nil || sheet.Phases <= 0 {
			continue
		}
		base := image.Pt(b.at.X*ui.ShotScale, b.at.Y*ui.ShotScale)
		tile := b.tile
		if tile < 1 {
			tile = 1
		}
		lastSpawn := b.age
		if lastSpawn >= healSpawnTicks {
			lastSpawn = healSpawnTicks - 1
		}
		first := lastSpawn - healParticleLife + 1
		if first < 0 {
			first = 0
		}
		cohorts := showerCohorts(b, lastSpawn)
		step := int(math.Trunc(1 + float64(32*tile)/7))
		for tick := first; tick <= lastSpawn; tick++ {
			particleAge := b.age - tick
			if particleAge < 0 || particleAge >= healParticleLife {
				continue
			}
			frame, _, ok := terrain.SelectEffectFrame(sheet, 0, particleAge)
			if !ok {
				continue
			}
			dy := -step * particleAge
			if b.drain {
				dy = -32*tile + step*particleAge
			}
			for _, particle := range cohorts[tick] {
				pos := base.Add(image.Pt(particle.dx*boltUnitsPerPixel,
					(dy-particle.depth)*boltUnitsPerPixel))
				out = append(out, ui.HealSprite{Cell: b.at, Pos: pos, Sheet: sheet, Frame: frame, Owner: b.owner})
			}
		}
	}
	return out
}

// boltDraws is what the viewer is handed this frame: every live cast object,
// travelling or standing, every weapon-borne cast currently winding up, and
// every ranged swing's shot in flight (unitshot.go).
//
// AN OBJECT REACHES ITS TARGET ON ITS LAST DRAWN TICK. The engine's driver moves
// it by the remaining gap over the remaining ticks, once per tick, before the
// draw — so an object of life N stands at (age+1)/N of the way at age, and at
// the target itself at age N-1. shotPoint is the one interpolation this and the
// archer's mark share, reused rather than restated.
//
// A REFUSED SELECTION DROPS THE OBJECT AND NOT THE FRAME. A picture with no
// sheet, a sheet with no frames and an index past the sheet each produce no
// entry at all, which is the engine's own answer to the same three.
func (mw *mapWorld) boltDraws(ents []sim.Entity) []ui.SpellBolt {
	var out []ui.SpellBolt
	for _, b := range mw.bolts {
		if b.picture == healingPicture || b.age < b.delay {
			continue
		}
		// A path picture ignores the object's own position entirely: it draws the
		// figure and no interpolated sprite.
		if data.CastDrawsPath(b.picture) {
			out = append(out, mw.pathDraws(b)...)
			continue
		}
		age, life := b.age-b.delay, b.life-b.delay
		num := age + 1
		if num > life {
			num = life
		}
		if d, ok := mw.spellDraw(b.picture, b.from, b.to, castShotPoint(b.from, b.to, num, life, b.launch),
			age, b.owner); ok {
			out = append(out, d)
		}
		out = append(out, mw.trailDraws(b)...)
	}
	out = append(out, mw.savedProjectileDraws()...)
	out = append(out, mw.weaponBoltDraws(ents)...)
	out = append(out, mw.unitShotDraws()...)
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
// arrives here as an AreaPaint and becomes a burst in mw.bolts. A BLAST retains
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
		picture, ok := data.OverlayPicture(int(e.Spell))
		if !ok {
			continue
		}
		arm, pass := 0, ui.SpellOverlayA
		switch e.Spell {
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

// observeAreaPaints turns one tick's staged stages into transient client
// objects: one per accepted cell, at that cell, for the picture's own
// decoded burst life — 16 ticks, 18 for Acid Stream's picture
// (`MAGIC-AREADRAW-049`, `MAGIC-RING-048`).
//
// THEY OVERLAP BY CONSTRUCTION and that is the point. A stage runs every three
// ticks while an object lives sixteen, so five or six stages' cells are drawn at
// once: Fire Sacrifice's inner shell is still burning when its outer shell
// lights, Meteor Storm's rocks accumulate across its area over 32 stages, and
// Acid Stream's cone grows away from the caster instead of flickering one wedge
// at a time.
//
// A SPELL WITH NO BURST SHEET SPAWNS NOTHING, spawnBurst's own rule and for its
// reason.
func (mw *mapWorld) observeAreaPaints(paints []sim.AreaPaint) {
	for _, p := range paints {
		picture := data.BurstPicture(int(p.Spell))
		// A staged sender produces one logical client object per accepted cell,
		// hence one selector per cell. Picture 51 is the one decoded exception:
		// its object stays silent at construction and voices slot 551 at phase 8.
		for _, c := range p.Cells {
			at := image.Pt(int(c.X), int(c.Y))
			delay := 0
			if picture == stormBurstPicture {
				delay = stormSoundPhase
			}
			source := "spell-effect"
			if picture == stormBurstPicture {
				source = "storm-phase"
			}
			mw.queueSpellSound(effectSpellSoundSlot(p.Spell), p.Owner, at, delay, source)
		}
		if mw.projectiles.Sheet(picture) == nil {
			continue
		}
		life := data.BurstLife(picture)
		for _, c := range p.Cells {
			at := image.Pt(int(c.X), int(c.Y))
			mw.bolts = append(mw.bolts, spellBolt{
				from: at, to: at, picture: picture, owner: p.Owner, life: life,
			})
		}
	}
}

// spellDraw is one object's whole crossing of the seam: the sheet its
// picture names, the phase its age gives, the facing its flight gives, and
// the frame the three of them select.
//
// BOTH CELLS ARE CARRIED. Visibility is the CASTER'S, so the object is hidden
// wherever the caster is hidden; the relief is interpolated between the two,
// so a picture crossing from a hill to a valley arrives on its target rather
// than above it (ui.spellBoltLift). A burst's two ends are equal, so the two
// answers coincide there.
func (mw *mapWorld) spellDraw(picture int, from, to, pos image.Point, age int, owner uint32) (ui.SpellBolt, bool) {
	if picture == unitShotDeformationPicture {
		if !mw.projectiles.HasPicture(picture) {
			return ui.SpellBolt{}, false
		}
		return ui.SpellBolt{Cell: from, To: to, Pos: pos, Owner: owner,
			Effect: ui.SpellBackgroundDeformation, Phase: age}, true
	}
	sheet := mw.projectiles.Sheet(picture)
	if sheet == nil {
		return ui.SpellBolt{}, false
	}
	phase, ok := terrain.EffectPhase(sheet.Clock, age, sheet.Phases)
	if !ok {
		return ui.SpellBolt{}, false
	}
	if picture == fireBallBurstPicture && age == data.BurstLife(picture)-1 {
		phase = sheet.Phases - 1
	}
	frame, mirror, ok := terrain.SelectEffectFrame(sheet,
		EffectFacing(to.X-from.X, to.Y-from.Y), phase)
	if !ok {
		return ui.SpellBolt{}, false
	}
	return ui.SpellBolt{
		Cell: from, To: to, Pos: pos, Sheet: sheet, Frame: frame, Mirror: mirror, Owner: owner,
	}, true
}

// weaponBoltDraws is the LIVE half: one object per attacker standing in the
// casting wind-up, interpolated from its own cell toward its victim's by the
// swing clock over the charge — the SAME two numbers an archer's shot is placed
// by (entityDraws, world.go), so the two marks cannot come to measure a cycle
// differently.
//
// THE THREE GATES ARE THE CYCLE'S OWN: alive, holding a victim, and in the
// casting phase. Reach is NOT asked — a weapon-borne cast's admission distance
// is the spell row's own MaxRange and not the weapon's reach (closedOn,
// pkg/sim/combat.go) — so a staff that casts at an adjacent enemy draws its
// picture crossing one cell rather than nothing.
//
// A WEAPON WHOSE SPELL DOES NOT FLY DRAWS NOTHING. The wind-up is a picture
// of a projectile leaving, and 21 of the 28 spells put no projectile on the
// map at all; the mark this replaced was drawn for every one of them.
//
// THE PHASE IS THE SWING CLOCK'S, not a life this tier is keeping: the object
// here is not remembered between frames, so the only count it has is how far
// through the wind-up its caster stands.
func (mw *mapWorld) weaponBoltDraws(ents []sim.Entity) []ui.SpellBolt {
	var out []ui.SpellBolt
	for _, b := range mw.weaponBolts(ents) {
		if data.CastDrawsPath(b.picture) {
			out = append(out, mw.pathDraws(b)...)
			continue
		}
		if d, ok := mw.spellDraw(b.picture, b.from, b.to, castShotPoint(b.from, b.to, b.age, b.life, b.launch),
			b.age, b.owner); ok {
			out = append(out, d)
		}
		out = append(out, mw.trailDraws(b)...)
	}
	return out
}

// weaponBolts is the live weapon-borne objects weaponBoltDraws draws, one per
// attacker in the casting wind-up, with the swing clock as age and the charge
// as life. The spell light reads the same objects (objectLightStamps).
func (mw *mapWorld) weaponBolts(ents []sim.Entity) []spellBolt {
	var out []spellBolt
	for _, e := range ents {
		if !e.Alive() || !e.HasAttackTarget || e.AttackTargetKind != sim.AttackTargetUnit || e.AttackPhase != sim.AttackCasting {
			continue
		}
		if e.WeaponSpell == 0 {
			continue
		}
		picture := data.CastPicture(int(e.WeaponSpell))
		if !data.CastFlies(picture) || picture == healingPicture {
			continue
		}
		// THE VICTIM IS LOOKED UP IN THIS SAME SNAPSHOT and not asked of the
		// world again — advanceSwings' own rule for the same pair (world.go):
		// the attacker and the target must be read from the same instant, or a
		// picture could be aimed at a cell one of them no longer stands on.
		victim, ok := entityIn(ents, e.AttackTarget)
		if !ok || mw.sprayLiveFrom(picture, e.X, e.Y) {
			continue
		}
		den := int(e.AttackCharge)
		if den < 1 {
			den = 1
		}
		num := mw.swing[e.ID]
		if num < 0 {
			num = 0
		}
		if num > den {
			num = den
		}
		// THE SAME OBJECT THIS TIER DRAWS FOR A BOOK CAST, on the swing clock
		// instead of on an object's own age. A weapon-borne release is not a
		// spellBolt in mw.bolts — it is rebuilt from the attack cycle every
		// frame — so it is assembled here and handed to the same producers,
		// and a picture cannot come to be drawn one way from a book and another
		// way from a staff.
		out = append(out, spellBolt{
			from:    image.Point{X: int(e.X), Y: int(e.Y)},
			to:      image.Point{X: int(victim.X), Y: int(victim.Y)},
			picture: picture,
			owner:   e.Owner,
			age:     num,
			life:    den,
			seed:    mw.visualCastSeed(e.ID, e.AttackTarget, int(e.WeaponSpell), false),
			facing:  e.Facing,
			launch:  mw.castLaunchFor(e.ID, e.Facing, picture),
			// THE TAG IS THE ATTACKER'S OWN ID, not zero. A book cast takes the
			// observation's index inside its tick; this producer holds no
			// observation, and leaving the field unset would make every staff on
			// the map draw Prismatic Spray's phase block 0 of 7 while a book cast
			// varies. The id is stable frame to frame, so the block does not
			// flicker across one swing. Both indices are authored: what fills the
			// caster's victim array is not decoded (DIV-081).
			tag: int(mw.visualActorLabel(e.ID)) % chainTagCount,
		})
	}
	return out
}

// sprayLiveFrom reports whether a Prismatic Spray set spawned from this cell
// is still drawn. A staff's next wind-up waits for it: the set already draws
// a figure to every victim, so the wind-up would be a second one.
func (mw *mapWorld) sprayLiveFrom(picture int, x, y int32) bool {
	if picture != data.PicturePathSecond {
		return false
	}
	for _, b := range mw.bolts {
		if b.picture == picture && b.from == image.Pt(int(x), int(y)) {
			return true
		}
	}
	return false
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

// Bound original projectiles draw directly from their persisted current phase
// and position; a second cosmetic timer must not restart them after native LOAD.
func (mw *mapWorld) savedProjectileDraws() []ui.SpellBolt {
	d := mw.world.SavedWorldEffectDrivers()
	if d == nil {
		return nil
	}
	armed := map[uint16]bool{}
	owners := map[uint16]uint32{}
	for _, row := range d.Projectiles {
		if !row.Retired {
			armed[row.ID] = true
			owners[row.ID] = row.Owner
		}
	}
	var out []ui.SpellBolt
	for _, p := range mw.world.SavedProjectiles().Items {
		if !armed[p.ID] || p.Picture == 34 || p.Picture == 36 || p.Picture == healingPicture {
			continue
		}
		cell := image.Pt(int(p.X)/256, int(p.Y)/256)
		if p.Picture == unitShotDeformationPicture {
			if !mw.projectiles.HasPicture(int(p.Picture)) {
				continue
			}
			out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: image.Pt(int(p.X), int(p.Y)), AbsolutePosition: true,
				Effect: ui.SpellBackgroundDeformation, Phase: int(p.Phase)})
			continue
		}
		sheet := mw.projectiles.Sheet(int(p.Picture))
		frame, mirror, ok := terrain.SelectEffectFrame(sheet, savedProjectileFacing(p.Dir), int(p.Phase))
		if !ok {
			continue
		}
		out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: image.Pt(int(p.X), int(p.Y)), AbsolutePosition: true, Sheet: sheet, Frame: frame, Mirror: mirror, Owner: owners[p.ID]})
		out = append(out, mw.savedProjectileTrail(p)...)
	}
	return out
}

// savedProjectileTrail is the smoke behind a record of a smoke-leaving
// picture: one stamp per trail point, oldest first (ANIM-140), the frame its
// age, so the newest point takes frame 0.
func (mw *mapWorld) savedProjectileTrail(p sim.SavedProjectile) []ui.SpellBolt {
	slot := data.CastTrailSlot(int(p.Picture))
	if slot < 0 {
		return nil
	}
	sheet := mw.projectiles.SmokeSheet(slot)
	if sheet == nil {
		return nil
	}
	var out []ui.SpellBolt
	trail := mw.shots.trail[p.ID]
	for k, at := range trail {
		frame, _, ok := terrain.SelectEffectFrame(sheet, 0, len(trail)-1-k)
		if !ok {
			continue
		}
		cell := image.Pt(at.X/ui.ShotScale, at.Y/ui.ShotScale)
		out = append(out, ui.SpellBolt{Cell: cell, To: cell, Pos: at, AbsolutePosition: true, Sheet: sheet, Frame: frame})
	}
	return out
}
