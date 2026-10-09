package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// THE BOLT'S OWN FIGURE, AND THE TRAIL BEHIND A FLYING ONE.
//
// Two picture ids draw a path rather than a sprite at their own position, 34
// (Lightning) and 36 (Prismatic Spray). The figure is a bounded random walk
// spanning the WHOLE caster-to-target segment from its first frame, regenerated
// whole on each of the object's 13 ticks; the object's position is never
// written, so nothing travels and nothing is created at arrival
// (`MAGIC-BOLTGATE-069`, `MAGIC-BOLTSHAPE-070`, `MAGIC-BOLTLIST-071`,
// `MAGIC-BOLTSTILL-072`, `MAGIC-BOLTEND-074`).
//
// THE ACCUMULATING TRAIL IS A DIFFERENT MECHANISM AND BELONGS TO DIFFERENT
// PICTURES — 10 (Fire Arrow) and 12 (Fire Ball), where it is a queue of at most
// six PAST positions whose oldest entry is dropped one at a time
// (`MAGIC-TRAIL-073`). A build that gave the bolt a growing trail would be
// visibly wrong in both directions at once.
//
// NOTHING HERE REACHES pkg/sim. A bolt's shape is client presentation: it is
// drawn from a generator seeded by the cast observation and the object's age,
// it is not stored, and no simulation state, tick or digest reads it.

// boltRNG is the C runtime generator the shape routine calls: MSVC's linear
// congruential generator with RAND_MAX 0x7fff (`AI-RAND-058`).
//
// IT IS NOT math/rand AND NOT A CLOCK. The figure has to reproduce from the
// cast observation alone, so a headless test drawing the same object at the
// same age gets the same points, and two machines drawing one replay draw one
// picture.
type boltRNG struct{ state uint32 }

func (r *boltRNG) next() int {
	r.state = r.state*214013 + 2531011
	return int((r.state >> 16) & 0x7fff)
}

// The shape generator's own constants, each an `.rdata` double or an immediate
// in the routine that consumes it (`MAGIC-BOLTSHAPE-070`). Every fraction is
// carried in hundredths, which is the unit the abscissa's own `0.01` scale puts
// them in, so the whole walk is integer arithmetic.
const (
	// boltAbscissaStep is the modulus of the abscissa's own draw, `rand()%50`,
	// scaled by 0.01 — so one step advances by 0 to 0.49.
	boltAbscissaStep = 50
	// boltAbscissaEnd is the bound the walk ends past, 0.7 in hundredths, and
	// the abscissa of the figure's last point: the walk is stretched onto the
	// whole segment, so an abscissa of boltAbscissaEnd is the target.
	boltAbscissaEnd = 70
	// boltOrdinateStep is the modulus of the perpendicular draw, `rand()%7`,
	// taken with a random sign.
	boltOrdinateStep = 7
	// boltOrdinateClamp is the `.rdata` pair 3.0 and -3.0 the running ordinate
	// is held between.
	boltOrdinateClamp = 3
	// boltOrdinateScale is the `.rdata` 0.03 the running ordinate is turned into
	// a distance by, in hundredths OF THE SEGMENT LENGTH: the generator
	// multiplies it by that length and keeps the product as its own scale.
	boltOrdinateScale = 3
	// The rejection bound: 0.15 of the segment length. A figure with a point
	// further than this from the straight line is thrown away whole and the
	// walk re-entered, which is what the generator's self-call does.
	boltRejectNumer = 15
	// boltHundredths is the denominator both fractions above are carried in.
	boltHundredths = 100
)

// THE PERPENDICULAR IS A FRACTION OF THE SEGMENT AND NOT A PIXEL COUNT. It
// is one: 6 px over a cast of 250.
//
// The three constants are calibrated to each other only under this reading: the
// clamp puts a raw point at 3 × 0.03 = 9% of the length, the smoothing pass
// carries a sharpened one to 13.5%, and the rejection bound sits just above both
// at 15% — so the generator's reject-and-rerun is reachable but rare, which is
// what a reject-and-rerun is for. Under the pixel reading the bound is 15% of a
// length against a 3 px excursion and can never be reached at any real distance.

// boltUnitsPerPixel converts screen pixels into the ShotScale units every point
// on this seam is stated in. The stamp width is the one pixel quantity left.
const boltUnitsPerPixel = ui.ShotScale / terrain.CellSize

// boltMaxWalkPoints bounds the raw walk. The abscissa's draw can be 0, so the
// walk can fail to advance; the engine's own bound is its recursion and its
// heap, and this build states one.
const boltMaxWalkPoints = 24

// boltMaxAttempts bounds the rejection loop. The engine re-enters the generator
// with the same four arguments and states no bound at all, which is safe there
// only because the clamp and the bound are calibrated to each other: with a
// 32-pixel cell, 0.15 of the shortest real cast is 4.8 pixels against a clamp of
// 3, so no figure over a real distance is ever rejected. This build stops after
// four and keeps the last figure rather than recursing without a floor.
const boltMaxAttempts = 4

// castOrigin is a cast object's start point: the from cell's centre plus its
// launch offset (castLaunch), in ShotScale units.
func castOrigin(from, launch image.Point) image.Point {
	return from.Mul(ui.ShotScale).Add(launch)
}

// castShotPoint interpolates a cast object from its launch point toward the
// target cell, so a flying picture, its trail and a path figure leave one
// point. A same-cell object (a burst, an area-paint cell, a Teleport object)
// stands at its launch point.
func castShotPoint(from, to image.Point, num, den int, launch image.Point) image.Point {
	origin := castOrigin(from, launch)
	if from == to {
		return origin
	}
	toPt := image.Point{X: to.X * ui.ShotScale, Y: to.Y * ui.ShotScale}
	return image.Point{
		X: origin.X + (toPt.X-origin.X)*num/den,
		Y: origin.Y + (toPt.Y-origin.Y)*num/den,
	}
}

// boltPath is one tick's whole figure, in ShotScale units, from the object's
// launch point to the target's.
//
// THE FIRST POINT IS THE LAUNCH POINT AND THE LAST IS THE TARGET'S,
// UNCHANGED. The walk's abscissa is a fraction of the segment and
// its termination bound is the far end, so the figure spans the whole segment
// on every tick and is never partial. Which of the two endpoints the engine's
// own list starts from is not established (`MAGIC-BOLTSHAPE-070`'s Unknown);
// the caster's is chosen because that is the object's own point.
//
// stamp IS THE SPACING the stamped sprites are placed at, in ShotScale units,
// and it is the one argument here the engine does not have. The decoded walk
// ends after three to five points, which at a long cast leaves the stamped
// sprites detached from each other; the figure is subdivided until no gap
// exceeds it, so a bolt is a bolt at every distance. Subdivision moves no
// existing point, so the walk's own shape is unchanged by it.
//
// A SAME-CELL CAST ANSWERS ONE POINT, at the launch point, so a cast onto
// the caster's own cell still has nothing to rotate onto and nothing to
// divide by — checked on the CELLS rather than on the launch-to-target
// vector, because the launch offset alone would otherwise put a false
// direction between a caster and himself.
func boltPath(from, to image.Point, seed uint32, stamp int, launch image.Point) []image.Point {
	origin := castOrigin(from, launch)
	if from == to {
		return []image.Point{origin}
	}
	return boltPathFrom(origin, to, seed, stamp)
}

func boltPathFrom(origin, to image.Point, seed uint32, stamp int) []image.Point {
	toX, toY := to.X*ui.ShotScale, to.Y*ui.ShotScale
	dx, dy := toX-origin.X, toY-origin.Y
	length := isqrt(dx*dx + dy*dy)
	if length == 0 {
		return []image.Point{origin}
	}

	rng := boltRNG{state: seed}
	var absc, ord []int
	for attempt := 0; attempt < boltMaxAttempts; attempt++ {
		absc, ord = boltWalk(&rng)
		if boltWithin(ord) {
			break
		}
	}

	// The canonical horizontal frame rotated onto the segment: the cosine is
	// dx/length and the sine dy/length, and every generated point is mapped
	// through that rotation. Written with the division last so the two
	// quotients are taken once each, on the whole term.
	out := make([]image.Point, 0, len(absc))
	for i := range absc {
		along := absc[i] * length / boltAbscissaEnd
		perp := ord[i] * length * boltOrdinateScale / boltHundredths
		out = append(out, image.Point{
			X: origin.X + (along*dx-perp*dy)/length,
			Y: origin.Y + (along*dy+perp*dx)/length,
		})
	}
	return boltDensify(boltSmooth(out), stamp)
}

// boltMaxStampsPerGap bounds one gap's subdivision, so a figure over any
// distance holds a bounded number of stamps. It is three times what it was
// before boltStampsPerFrame divided the spacing by three, so the distance one
// gap can still be filled across is the same.
const boltMaxStampsPerGap = 72

// boltStampsPerFrame is how many stamps are placed per frame width, and it
// is what makes the figure a LINE rather than a row of separate glyphs
// (owner).
//
// A FRAME IS NOT ITS ART. Both path sheets are 16 by 16 and the glyph inside is
// a four-pointed star: measured on the shipped `lightnin\sprites` frame 0, 99 of
// the 256 pixels are painted, the fully opaque core is 8 by 6, and the rest is
// the star's thin arms. Spacing the stamps one frame width apart therefore left
// 8 px of arm between each pair of cores, which is what the owner reported as a
// broken line. Three per width spaces them between 2.7 and 5.3 px, under the
// core in both axes, so consecutive cores overlap whatever the figure's
// direction.
const boltStampsPerFrame = 3

// boltDensify spaces the stamps at no more than one stamp width apart, gap by
// gap. It moves no existing point and it inserts evenly, so a short gap of the
// walk does not end up carrying as many sprites as a long one.
func boltDensify(points []image.Point, stamp int) []image.Point {
	if stamp <= 0 || len(points) < 2 {
		return points
	}
	out := make([]image.Point, 0, 2*len(points))
	for i := 0; i < len(points)-1; i++ {
		a, d := points[i], points[i+1].Sub(points[i])
		n := 1
		for n*n*stamp*stamp < d.X*d.X+d.Y*d.Y && n < boltMaxStampsPerGap {
			n++
		}
		for k := 0; k < n; k++ {
			out = append(out, image.Pt(a.X+d.X*k/n, a.Y+d.Y*k/n))
		}
	}
	return append(out, points[len(points)-1])
}

// boltWalk is the raw random walk in the canonical frame: an abscissa in
// hundredths and a perpendicular in pixels, one pair per point.
//
// The abscissa accumulates `rand()%50` and the walk ends once it passes 0.7. The
// perpendicular accumulates `rand()%7` with a random sign into a value clamped
// to 3.0 and -3.0 — an increment reaching 6 against a clamp of 3, which is what
// makes the figure a zigzag rather than a drift.
func boltWalk(rng *boltRNG) (absc, ord []int) {
	absc = append(absc, 0)
	ord = append(ord, 0)
	a, o := 0, 0
	for a < boltAbscissaEnd && len(absc) < boltMaxWalkPoints {
		a += rng.next() % boltAbscissaStep
		step := rng.next() % boltOrdinateStep
		if rng.next()&1 == 1 {
			step = -step
		}
		o += step
		o = min(max(o, -boltOrdinateClamp), boltOrdinateClamp)
		if a >= boltAbscissaEnd {
			a, o = boltAbscissaEnd, 0
		}
		absc = append(absc, a)
		ord = append(ord, o)
	}
	if absc[len(absc)-1] != boltAbscissaEnd {
		absc = append(absc, boltAbscissaEnd)
		ord = append(ord, 0)
	}
	return absc, ord
}

// boltSmoothPasses is how many times boltCutCorners runs. One pass still
// shows a shallow break at every cut point; two reads as a smoothed curve
// rather than a bevelled polyline, at a walk of the three-to-five points the
// decoded generator typically produces (owner).
const boltSmoothPasses = 2

// boltSmooth rounds every interior corner of the ROTATED figure by CUTTING
// it, replacing the SHARPENING the earlier build ran on the walk's own
// (absc, ord) pairs.
//
// IT RUNS ON THE ROTATED WORLD POINTS, in ShotScale units, RATHER THAN ON THE
// WALK'S OWN (absc, ord) PAIRS. Rotation is affine — a linear map plus the
// origin's own translation — and an affine map preserves a weighted average
// exactly, so cutting a corner before or after rotation cuts the same corner.
// It is done after because the walk's own ordinate is clamped to plus or
// minus three (boltOrdinateClamp): a quarter-cut of two values that close
// together is 0 under Go's own truncating integer division at every pair
// this generator can produce, which would have made the pass a silent no-op
// on the one axis a kink actually turns on. The rotated points carry
// magnitudes in the tens to hundreds of ShotScale units, where the same
// quarter-cut divides to something real.
//
// THE TWO ENDS ARE NEVER MOVED, so the figure still spans the whole segment
// from the caster's own departure point to the target's.
//
// DIV-080
func boltSmooth(points []image.Point) []image.Point {
	for pass := 0; pass < boltSmoothPasses; pass++ {
		points = boltCutCorners(points)
	}
	return points
}

// boltCutCorners is one corner-cutting pass: every interior point is replaced
// by two points, one a quarter of the way back toward its earlier neighbour
// and one a quarter of the way forward toward its later neighbour, and the
// straight chord between those two stands in for the sharp turn the original
// point made. The first and last points pass through unchanged.
//
// A WALK OF FEWER THAN THREE POINTS HAS NO INTERIOR POINT TO CUT, and is
// returned as it stands — this is the zero-length-segment and the
// two-point-walk case both.
func boltCutCorners(points []image.Point) []image.Point {
	n := len(points)
	if n < 3 {
		return points
	}
	out := make([]image.Point, 0, 2*n)
	out = append(out, points[0])
	for i := 1; i < n-1; i++ {
		out = append(out, boltCut(points[i-1], points[i]), boltCut(points[i+1], points[i]))
	}
	out = append(out, points[n-1])
	return out
}

// boltCut is the point one quarter of the way from b toward a — integer
// arithmetic throughout, so the figure keeps reproducing bit for bit from its
// seed (the walk is client presentation and outside the determinism wall, but
// tests depend on the figure regenerating identically).
func boltCut(a, b image.Point) image.Point {
	return image.Point{X: b.X + (a.X-b.X)/4, Y: b.Y + (a.Y-b.Y)/4}
}

// boltWithin is the rejection test: no point's distance from the straight line
// may exceed 0.15 of the segment. Both sides are fractions of the length, so the
// length itself cancels and the test is on the walk's OWN, UNSMOOTHED ordinate
// — the same value the engine's own reject-and-rerun tests, on this build's
// reading of `MAGIC-BOLTSHAPE-070`.
//
// SMOOTHING NO LONGER RUNS BEFORE THIS TEST, because it no longer runs on the
// (absc, ord) pairs at all (boltSmooth's own doc). The walk's own clamp holds
// every raw ordinate at [-3, 3] (boltOrdinateClamp) against a bound of 5, so
// this test does not reject any walk this generator can produce; it is kept
// running rather than removed, so a walk whose own clamp or bound changes
// underneath it is still checked.
func boltWithin(ord []int) bool {
	const bound = boltRejectNumer / boltOrdinateScale
	for _, v := range ord {
		if v > bound || -v > bound {
			return false
		}
	}
	return true
}

// boltSeed is one cast object's own generator seed. It is a function of the
// observation and of the object's age, so the figure is regenerated whole on
// every tick — the engine replaces the whole point list on each of the 13, and
// a seed that did not move with the age would hold the kinks still.
func boltSeed(b spellBolt) uint32 {
	s := b.seed*2654435761 + uint32(b.age)*2246822519 + uint32(b.picture)
	return s*214013 + 2531011
}

// boltRampPhase is the sheet frame a path picture shows at age ticks of its
// own life.
//
// THE THIRTEEN VALUES ARE OURS. The engine's arm for these two pictures is a
// thirteen-entry jump table whose entries write the frame and nothing else, and
// the table's values are not published — only that they exist and that they are
// a ramp. Both sheets carry five phases behind the per-point offset, so this
// walks those five once across the object's thirteen ticks.
func boltRampPhase(age, life, phases int) int {
	if phases <= 0 || life <= 0 {
		return 0
	}
	return min(max(age*phases/life, 0), phases-1)
}

// chainPhaseStride is the per-point phase picture 36 adds: the record's own tag
// times five. Its sheet holds 35 frames, which is seven tags of five phases —
// so the stride and the frame count fix each other.
const chainPhaseStride = 5

// boltPhaseBlock is how the ramp and the tag divide one path sheet: how many
// phases the ramp walks, and how many blocks of that many the tag selects
// between.
//
// PICTURE 36's REGISTRY Phases IS THE WHOLE SHEET AND NOT ONE BLOCK. The record
// states 35, which is the frame count, while the ramp is over the five phases of
// one block and the tag picks the block. Reading the registry value as the
// ramp's own range walked all 35 frames across the object's thirteen ticks — so
// the figure cycled every colour instead of holding one, and then ran off the
// end of the sheet for every tag above zero and drew nothing at all from the
// tick the sum passed 34. Picture 34 states 5 and holds one block, so it takes
// the same arithmetic with a block count of one.
func boltPhaseBlock(picture, phases int) (ramp, blocks int) {
	if picture != data.PicturePathSecond || phases < 2*chainPhaseStride {
		return phases, 1
	}
	return chainPhaseStride, phases / chainPhaseStride
}

// pathDraws is one path object's whole tick: the figure regenerated, and the
// sheet stamped at every point of it.
//
// THE OBJECT'S OWN POSITION IS NEVER DRAWN. The engine's two path arms ignore
// it, and this build gives these two pictures no interpolated sprite at all.
func (mw *mapWorld) pathDraws(b spellBolt) []ui.SpellBolt {
	sheet, frame, points := mw.pathFigure(b)
	out := make([]ui.SpellBolt, 0, len(points))
	for _, p := range points {
		out = append(out, ui.SpellBolt{
			Cell: b.from, To: b.to, Pos: p, Sheet: sheet, Frame: frame, Owner: b.owner,
		})
	}
	return out
}

// pathFigure is one path object's sheet, frame and figure this tick. The
// figure is empty when the sheet or its frame is absent. The spell light
// stamps the same points the draw stamps (objectLightStamps).
func (mw *mapWorld) pathFigure(b spellBolt) (*terrain.EffectSheet, int, []image.Point) {
	sheet := mw.projectiles.Sheet(b.picture)
	if sheet == nil {
		return nil, 0, nil
	}
	ramp, blocks := boltPhaseBlock(b.picture, sheet.Phases)
	base := boltRampPhase(b.age, b.life, ramp)
	if blocks > 1 {
		base += (b.tag % blocks) * chainPhaseStride
	}
	frame, _, ok := terrain.SelectEffectFrame(sheet, 0, base)
	if !ok {
		return nil, 0, nil
	}
	stamp := 0
	if f := sheet.Frame(frame); f != nil {
		stamp = f.Width * boltUnitsPerPixel / boltStampsPerFrame
	}
	points := boltPath(b.from, b.to, boltSeed(b), stamp, b.launch)
	if b.centered {
		points = boltPathFrom(b.from.Mul(ui.ShotScale), b.to, boltSeed(b), stamp)
	}
	return sheet, frame, points
}

// trailDraws is the smoke behind a travelling object: at most six PAST
// positions, oldest first, each stamped with the trail sheet its picture
// names.
//
// THE POSITIONS ARE THE ONES THE OBJECT HELD, not a decoration. The engine saves
// the object's position at the driver's entry, BEFORE any arm has moved it, and
// appends that saved value after the switch — so the newest entry is where the
// object stood one tick ago and the oldest is where it stood six ticks ago. This
// build recomputes them from the object's own two ends rather than keeping a
// queue: the interpolation is a pure function of the age, so the queue and the
// recomputation cannot come apart.
//
// THE FRAME IS THE ENTRY'S OWN AGE. Which frame the engine stamps is not
// published; each trail sheet holds exactly six frames against a queue bounded
// at exactly six, and the art runs from a small dense puff at frame 0 to a large
// pale one at frame 5, so the entry appended this tick takes frame 0 and the one
// about to expire takes frame 5.
func (mw *mapWorld) trailDraws(b spellBolt) []ui.SpellBolt {
	sheet := mw.projectiles.SmokeSheet(data.CastTrailSlot(b.picture))
	if sheet == nil {
		return nil
	}
	out := make([]ui.SpellBolt, 0, data.CastTrailLength)
	for k := 0; k < data.CastTrailLength; k++ {
		num := b.age - k
		if num < 0 {
			break
		}
		frame, _, ok := terrain.SelectEffectFrame(sheet, 0, k)
		if !ok {
			continue
		}
		out = append(out, ui.SpellBolt{
			Cell: b.from, To: b.to, Pos: castShotPoint(b.from, b.to, num, b.life, b.launch),
			Sheet: sheet, Frame: frame, Owner: b.owner,
		})
	}
	return out
}
