package terrain

import (
	"math"
	"math/big"
)

// THE EFFECT MARK RECORDS a lasting effect puts on its actor (1002; claims
// MAGIC-MARK-059, MAGIC-MARK-061, MAGIC-PROT-062, MAGIC-BLESS-064,
// MAGIC-CLOUD-065, MAGIC-089, MAGIC-090, MAGIC-091 and MAGIC-092).
//
// This file is the GEOMETRY ALONE. It takes a kind, a countdown and the actor's
// footprint scale and returns records; it resolves no art, reads no world and
// holds no state. The tier above turns a record index into a sheet and a record
// into a placement.
//
// IT IS CLIENT GEOMETRY AND IT USES FLOATS. The original's builders call FSIN,
// FCOS and ftol, and MAGIC-MARK-060 establishes that the record set is not
// stored: it is re-derived every rebuild from kind-and-countdown pairs. Nothing
// here reaches the simulation's state, so nothing here is behind the
// determinism wall — which is what makes the float arithmetic lawful in this
// package and unlawful one tier down.

// EffectMark is one 8-byte mark record (MAGIC-MARK-059): `+0x00 i16 dx`,
// `+0x02 i16 dy`, `+0x04 i16 depth`, `+0x06 u8 record index`, `+0x07 u8 phase`.
//
// Depth IS SIGNED AND IT IS BOTH A POSITION TERM AND THE PASS SELECTOR. The
// unit draw walks the array twice — the first pass drawing a record only when
// `depth > 0`, the second only when `depth <= 0` — and both passes compute the
// same position, in which `depth` is SUBTRACTED from y. So a positive depth
// lifts the record up the screen and puts it BEHIND the actor's own sprite, and
// that one field is what splits the Shield's ring and the Bless ring around the
// actor rather than stacking either in front of it.
//
// Record is the record index at `+0x06`, which is the picture id
// `2*spellId + 8` (MAGIC-PIC-026) and is the same value as the kind that built
// the record. Phase is the frame index at `+0x07`; it is per record, not per
// actor, which is what lets one builder show five different frames at once.
type EffectMark struct {
	DX, DY, Depth int
	Record        int
	Phase         int
}

// The ten kinds that reach a builder, plus the two that reach the unit draw's
// own by-kind arms instead (MAGIC-MARK-061, MAGIC-ACTOR-066). A kind is
// `2*spellId + 8` and is stated here as the literal the original's tables carry.
const (
	MarkProtectionFire  = 0x12 // spell 5
	MarkHeal            = 0x14 // spell 6
	MarkPoisonCloud     = 0x18 // spell 8
	MarkProtectionWater = 0x1c // spell 10
	MarkDrainLife       = 0x1e // spell 11
	MarkInvisibility    = 0x26 // spell 15 — no mark; the actor's own sprite arm
	MarkProtectionAir   = 0x28 // spell 16
	MarkShield          = 0x2c // spell 18
	MarkStoneCurse      = 0x30 // spell 20 — no mark; the animation-hold arm
	MarkProtectionEarth = 0x34 // spell 22
	MarkBless           = 0x36 // spell 23
	MarkCurse           = 0x3e // spell 27
)

const (
	markKindLow  = 0x12
	markKindHigh = 0x3e
)

// MarkKind is the mark kind a spell id computes: `2*spellId + 8`
// (MAGIC-MARK-061), the even half of MAGIC-PIC-026's pair and the picture id
// the record's art resolves through.
//
// It restates the arithmetic rather than calling pkg/data's CastPicture: this
// package is a render leaf and may not import pkg/data. The two are the same
// expression and neither derives from the other.
func MarkKind(spell int) int { return 2*spell + 8 }

// EffectMarkRecords is the whole dispatch (MAGIC-MARK-061): the records kind
// contributes for an actor whose class TileSize is tileSize, at the client
// element's current countdown.
//
// IT ANSWERS NIL FOR EVERY NON-MARKING KIND, and that is half the contract. The
// original's dispatch sends 35 of the 45 kinds in range to an arm that appends
// nothing and only runs the countdown, and rejects five more before the range
// test. Stone Curse and Invisibility are among the 35: MAGIC-ACTOR-066
// establishes that they change the actor's own draw instead, which the tier
// above owns and this function must not confuse with a mark.
//
// Heal and Drain Life answer nil here too, and that is a TIER decision rather
// than the original's answer. MAGIC-089 and MAGIC-090 decode their stateful
// carry-before-spawn producer: 25 cohorts of 3–5 particles, phases 0 through 7
// and a constant tile-scaled vertical step. pkg/game's healSpriteDraws owns
// that cast-observation lifetime; this stateless attached-mark rebuild cannot
// carry the previous cohort array without inventing a second owner. DIV-076.
//
// tileSize BELOW ONE IS TREATED AS ONE.
func EffectMarkRecords(kind int, countdown uint16, tileSize int) []EffectMark {
	if kind < markKindLow || kind > markKindHigh {
		return nil
	}
	if tileSize < 1 {
		tileSize = 1
	}
	switch kind {
	case MarkProtectionFire, MarkProtectionWater, MarkProtectionAir, MarkProtectionEarth:
		return protectionMarks(kind, countdown, tileSize)
	case MarkPoisonCloud:
		return poisonCloudMarks(countdown, tileSize)
	case MarkShield:
		return shieldMarks(countdown, tileSize)
	case MarkBless, MarkCurse:
		return blessCurseMarks(kind, countdown, tileSize)
	}
	return nil
}

// markBase is the vertical base every builder but the Shield's is passed:
// `TileSize*32` (MAGIC-PROT-062, MAGIC-BLESS-064, MAGIC-CLOUD-065). It is
// subtracted from y, so it puts the mark that many pixels ABOVE the actor
// anchor.
func markBase(tileSize int) int { return tileSize * 32 }

// protectionMarks is the four Protections' shared builder (MAGIC-PROT-062).
//
// ONE RECORD, AND THE FOUR DIFFER ONLY BY A PLACEMENT TABLE. All four sit
// `TileSize*32` above the actor anchor and carry `depth = 0`, so all four draw
// AFTER the actor's sprite; the four offsets put them 6 pixels left, right, up
// and down of one another. The diamond is what four simultaneous Protections
// produce and not a case the original recognises — nothing in the arm consults
// the other effects, and nothing here does either.
//
// The phase is `countdown mod 6`.
func protectionMarks(kind int, countdown uint16, tileSize int) []EffectMark {
	var dx, dy int
	switch kind {
	case MarkProtectionFire:
		dx = -6
	case MarkProtectionWater:
		dx = 6
	case MarkProtectionAir:
		dy = -6
	case MarkProtectionEarth:
		dy = 6
	}
	return []EffectMark{{
		DX:     dx,
		DY:     dy - markBase(tileSize),
		Depth:  0,
		Record: kind,
		Phase:  int(countdown % 6),
	}}
}

func poisonCloudMarks(countdown uint16, tileSize int) []EffectMark {
	return []EffectMark{{
		DY:     -markBase(tileSize),
		Record: MarkPoisonCloud,
		Phase:  int(countdown % 6),
	}}
}

// The Bless and Curse trail (MAGIC-BLESS-064): five steps, four records each,
// 18 degrees apart, on a circle of radius 20 at `TileSize*32` above the anchor.
// Bless steps down from 89 and Curse steps up from 0, so the two sweep opposite
// ways as the countdown falls.
const (
	blessRadius     = 20.0
	blessStepDegree = 18
	blessFirstStep  = 89
	blessSteps      = 5
)

// blessCurseMarks builds one of the two twenty-mark rings.
//
// THE GEOMETRY'S FRAME INDEX IS `4 - step/18`, which establishes the five
// available frames but is not used as a shared static frame for the four stars
// at that step. Each star takes a deterministic presentation-only starting
// phase from blessInitialPhase. The complemented countdown advances by one when
// the element countdown falls by one, so every star advances on the same tick
// cadence while retaining its own phase offset. No random source or simulation
// state reaches this builder.
//
// THE SINE GOES INTO depth AND NOT INTO dy, which is what splits the ring: the
// upper half of the circle carries positive depth and draws behind the actor,
// the lower half carries negative depth and draws in front (MAGIC-MARK-059's
// two passes).
//
// THE FOUR SIGN COMBINATIONS PER STEP ARE AUTHORED. MAGIC-BLESS-064 states
// that a step's four records take `+cos`, `-cos`, `+sin` and `-sin` terms and
// that the assignment was read from the store order rather than proved. This
// build takes the four sign combinations of (cos, sin), which completes the
// first-quadrant sweep of the five steps into a full circle of twenty marks —
// the reading that agrees with the claim's own "twenty marks on a rotating
// circle". DIV-075.
func blessCurseMarks(kind int, countdown uint16, tileSize int) []EffectMark {
	base := markBase(tileSize)
	out := make([]EffectMark, 0, blessSteps*4)
	clock := int(^countdown) % blessSteps
	for i := 0; i < blessSteps; i++ {
		step := blessFirstStep - i*blessStepDegree
		if kind == MarkCurse {
			step = i * blessStepDegree
		}
		rad := float64(step) * math.Pi / 180
		c := ftol(math.Cos(rad) * blessRadius)
		s := ftol(math.Sin(rad) * blessRadius)
		for _, sign := range [4][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			index := len(out)
			out = append(out, EffectMark{
				DX:     sign[0] * c,
				DY:     -base,
				Depth:  sign[1] * s,
				Record: kind,
				Phase:  (blessInitialPhase(kind, index) + clock) % blessSteps,
			})
		}
	}
	return out
}

// blessInitialPhase is a deterministic pseudo-random phase per star. Its input
// is presentation geometry only. The integer mix is stable across machines and
// does not import a clock or random generator into simulation state.
func blessInitialPhase(kind, index int) int {
	x := uint32(index+1)*1664525 + uint32(kind)*1013904223
	x ^= x >> 16
	return int(x % blessSteps)
}

// The Shield's decoded scales and 90-step phase (MAGIC-091, MAGIC-092).
const (
	shieldBaseScale     = 11
	shieldEnvelopeScale = 28
	shieldHeightScale   = 16
	shieldCycle         = 90
)

// shieldMarks emits the two decoded components in their production order.
// Component A samples the fixed radius 16*T circle every fourth generated
// point and emits both rotated records for each sample. Component B follows
// after every A record; its PC53 envelope determines its frame, radius and
// sampling interval, then it emits the upper/lower pair for every sample.
// Axis duplicates are retained because the original producer retains them.
func shieldMarks(countdown uint16, tileSize int) []EffectMark {
	phase := int(countdown % shieldCycle)
	theta := float64(4*phase) * math.Pi / 180
	sin, cos := math.Sin(theta), math.Cos(theta)
	fixedRadius := tileSize * shieldHeightScale
	out := make([]EffectMark, 0, fixedRadius*4)
	for _, p := range sampledMidpointCircle(fixedRadius, 4) {
		q := ftol(float64(p[1]) * float64(tileSize*shieldEnvelopeScale) / float64(fixedRadius))
		frame := abs(4 - ftol(float64(abs(p[1])*5)/float64(fixedRadius)))
		xcos := ftol(float64(p[0]) * cos)
		xsin := ftol(float64(p[0]) * sin)
		out = append(out,
			EffectMark{DX: xcos, DY: -q, Depth: tileSize*shieldBaseScale + xsin, Record: MarkShield, Phase: frame},
			EffectMark{DX: -xsin, DY: -q, Depth: tileSize*shieldBaseScale + xcos, Record: MarkShield, Phase: frame},
		)
	}

	// Component B is independently rasterised after every Component A record.
	// Its envelope preserves the original PC53, round-to-nearest operation
	// boundaries; real-number division by 45 disagrees on shipped phases.
	envelope := shieldEnvelopePC53(tileSize, phase)
	frame := 4 - abs(phase-45)/9
	ratio := float64(envelope) / float64(tileSize*shieldEnvelopeScale)
	radius := ftol(math.Sin(math.Acos(ratio)) * float64(fixedRadius))
	for _, p := range sampledMidpointCircle(radius, 2*frame) {
		out = append(out,
			EffectMark{DX: p[0], DY: -(envelope + tileSize*shieldBaseScale), Depth: p[1], Record: MarkShield, Phase: frame},
			EffectMark{DX: p[0], DY: envelope - tileSize*shieldBaseScale, Depth: p[1], Record: MarkShield, Phase: frame},
		)
	}
	return out
}

var (
	shieldPhaseScaleNum = big.NewInt(6405119470038039)
	shieldPhaseScaleDen = new(big.Int).Lsh(big.NewInt(1), 58)
)

func shieldPC53() *big.Float {
	return new(big.Float).SetPrec(53).SetMode(big.ToNearestEven)
}

func shieldEnvelopePC53(tileSize, phase int) int {
	scale := shieldPC53().SetRat(new(big.Rat).SetFrac(
		new(big.Int).Set(shieldPhaseScaleNum), new(big.Int).Set(shieldPhaseScaleDen)))
	product := shieldPC53().Mul(shieldPC53().SetInt64(int64(phase)), scale)
	shifted := shieldPC53().Sub(product, shieldPC53().SetInt64(1))
	absolute := shieldPC53().Abs(shifted)
	scaled := shieldPC53().Mul(absolute, shieldPC53().SetInt64(int64(tileSize*shieldEnvelopeScale)))
	integer, _ := scaled.Int(nil)
	return int(integer.Int64())
}

func sampledMidpointCircle(radius, interval int) [][2]int {
	if radius <= 0 {
		return nil
	}
	var out [][2]int
	x, y, errv := 0, radius, 2-2*radius
	counter := interval
	for y > 0 {
		if counter >= interval {
			out = append(out, [2]int{x, y}, [2]int{-x, y}, [2]int{x, -y}, [2]int{-x, -y})
			counter = 0
		}
		counter++
		if errv < 0 {
			if 2*(errv+y)-1 <= 0 {
				x++
				errv += 1 + 2*x
			} else {
				x++
				y--
				errv += 2 + 2*(x-y)
			}
		} else if errv == 0 {
			x++
			y--
			errv = 2 + 2*(x-y)
		} else if 2*(errv-x)-1 <= 0 {
			x++
			y--
			errv += 2 + 2*(x-y)
		} else {
			y--
			errv += 1 - 2*y
		}
	}
	return out
}

// ftol is the original's float-to-long conversion: truncation toward zero, the
// x87 `ftol` helper's own rounding as the claims cite it. It is written once
// here so that every builder rounds the same way and a builder cannot come to
// round differently from its neighbour.
func ftol(v float64) int { return int(v) }

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
