package sim

// fireBallSpell is the spell id whose area effect ends in one picture-13 burst
// record (ANIM-111, ANIM-112: the burst picture is 2*spell+9).
const fireBallSpell = 2

// fireBallBurstSegments is the segment count the client arm that builds the
// burst gives its record: with the start actionphase of -1 the record is
// driven on 22 ticks and collected on the next (SAV-1144, SAV-1147).
const fireBallBurstSegments = 22

// burstState is the World's burst bookkeeping. phases is the installed phase
// count of the burst picture's sheet; due holds the cells of blasts that
// landed during the current effect walk and wait for their record.
//
// raises is the end-of-pass queue of weapon and item releases of Control Spirit
// (controlspirit.go). It lives here because it is per-pass scratch like due:
// empty at every tick boundary, outside the byte form and the digest.
type burstState struct {
	phases uint16
	due    []burstCell
	raises []controlSpiritRaise
}

type burstCell struct {
	x, y  int32
	owner uint32
}

// SetBurstPhases names the phase count of the Fire_Ball burst picture's
// installed sheet. The caller owns what the projectile registry says; the
// World owns the record. Zero leaves the burst on phase 0.
func (w *World) SetBurstPhases(phases uint16) { w.burst.phases = phases }

// fireBallBurstPicture is the picture the burst record carries.
func fireBallBurstPicture() int32 { return 2*fireBallSpell + 9 }

// releaseFireBallBurst builds the record client arm 0x86 builds when a
// Fire_Ball area effect blasts at cell (cx, cy) (SAV-1144): the picture at the
// cell centre, aimed at the same point, no target, started at actionphase -1.
// The record takes its id from the counter every projectile shares (SAV-1146).
// owner is the caster's owner slot, kept on the driver row for the draw only;
// the record has no field for it and an area with no caster passes zero.
//
// Its first driver call is made on its birth tick, so no record rests at
// actionphase -1 and the corpus' leaves hold from the record's second tick
// (SAV-1144 for the leaves, ANIM-103 for the frame). A blast that lands during
// the effect walk is built after the walk, in front of the same tick's driver
// pass, which makes that call; one landing elsewhere is built and driven here.
func (w *World) releaseFireBallBurst(cx, cy int32, owner uint32) {
	if w.effectWalking {
		w.burst.due = append(w.burst.due, burstCell{cx, cy, owner})
		return
	}
	i := w.insertBurst(cx, cy, owner)
	w.stepSavedProjectile(&w.savedWorldEffects.Projectiles[i])
}

// burstTile is the side in cells of the area one burst covers.
const burstTile = 3

// releaseFireBallBursts builds one burst per burstTile square of the blast.
func (w *World) releaseFireBallBursts(cx, cy, r int32, owner uint32) {
	if r <= 1 || cx < 0 || cy < 0 || cx >= w.bounds.Width || cy >= w.bounds.Height {
		w.releaseFireBallBurst(cx, cy, owner)
		return
	}
	xs := burstCentres(cx, r, w.bounds.Width)
	ys := burstCentres(cy, r, w.bounds.Height)
	for _, x := range xs {
		for _, y := range ys {
			w.releaseFireBallBurst(x, y, owner)
		}
	}
}

func burstCentres(c, r, size int32) []int32 {
	lo, hi := max(0, c-r), min(size-1, c+r)
	m := (r + 1) / burstTile
	var out []int32
	for i := -m; i <= m; i++ {
		t := c + burstTile*i
		if t+1 < 0 || t-1 > size-1 {
			continue
		}
		t = max(lo, min(hi, t))
		if len(out) == 0 || out[len(out)-1] != t {
			out = append(out, t)
		}
	}
	return out
}

// releaseDueBursts builds the records the walk that just ended queued.
func (w *World) releaseDueBursts() {
	for _, c := range w.burst.due {
		w.insertBurst(c.x, c.y, c.owner)
	}
	w.burst.due = w.burst.due[:0]
}

func (w *World) insertBurst(cx, cy int32, owner uint32) int {
	x, y := cx*256+128, cy*256+128
	return w.insertProjectile(SavedProjectile{
		X: x, Y: y, Picture: fireBallBurstPicture(), Action: 1,
		ActionX: x, ActionY: y, ActionPhase: -1, ActionSegments: fireBallBurstSegments,
	}, SavedProjectileDriver{Phases: w.burst.phases, Owner: owner})
}

// WindUpElapsed is how many ticks have passed since id's current wind-up
// began, read from the action clock the wind-up loaded: the clock ends one
// charge and one relax after the run's first tick. It answers false when id
// is not winding up, has no known clock, or the clock is out of range.
func (w *World) WindUpElapsed(id EntityID) (int, bool) {
	e, ok := w.Entity(id)
	if !ok || (e.AttackPhase != AttackCharging && e.AttackPhase != AttackCasting) || !e.ActionClock.Known {
		return 0, false
	}
	total := chargeTicks(e) + relaxTicks(e)
	left := int64(int32(e.ActionClock.End - uint32(w.tick)))
	n := total - left
	if n < 0 || n > total {
		return 0, false
	}
	return int(n), true
}

// casterOwner is the owner slot of a cast's caster, or zero without one.
func (w *World) casterOwner(caster EntityID, has bool) uint32 {
	if has {
		if i := indexOfEntity(w.entities, caster); i >= 0 {
			return w.entities[i].Owner
		}
	}
	return 0
}
