package terrain

// Water animation, decoded from the game (research TERR-ANIM-006…010).
//
// A water cell keeps the blend column and sub-cell its tile word stores and
// varies only the strip group, which the renderers replace with 8 + phase. Since
// slot = g*4 + b and the tile3 variant is V = (g&3)*4 + b, that walks
// tile3-{b, b+4, b+8, b+12} as the phase steps 0→1→2→3→0. The 16 shipped tile3
// files are exactly four phases by four blend columns, and expanding every water
// cell of the 38-map corpus over all four phases never leaves that set
// (TERR-ANIM-010).
const (
	// WaterPhases is the length of the cycle: four tile3 variants per blend
	// column.
	WaterPhases = 4

	// TicksPerVariant is how many logic ticks each variant holds. The phase uses
	// animCtr>>2, so the image changes every fourth tick (TERR-ANIM-007).
	TicksPerVariant = 4

	// WaterCycleTicks is a full four-variant cycle in logic ticks.
	WaterCycleTicks = WaterPhases * TicksPerVariant

	// counterShift is the >>2 that turns the logic-tick counter into a variant
	// counter.
	counterShift = 2

	// phaseMask reduces the phase sum to 0..3. Masking (rather than a modulo)
	// keeps the function total: any int, including a negative one, lands in
	// range.
	phaseMask = WaterPhases - 1
)

// IsWaterGroup reports whether a strip group is water — groups 8..11, the tile3
// family (TERR-ANIM-006).
func IsWaterGroup(group int) bool {
	return group >= waterGroupLo && group <= waterGroupHi
}

// WaterPhase is the animation phase of the water cell at world position
// (col, row) with the given tick counter:
//
//	phase = (g + (col+1)·row + (animCtr>>2)) & 3
//
// g is the cell's *stored* group (8..11), not the overridden one — the authored
// group is the formula's first term, so folding the override in first would drop
// it. The (col+1)·row term offsets neighbouring cells against each other into a
// diagonal ripple; because it uses world rather than screen coordinates the
// pattern is stable under scrolling.
//
// The function is total: the mask leaves every input, including negative
// coordinates and any counter, in 0..3.
func WaterPhase(group, col, row int, animCtr uint32) int {
	return (group + (col+1)*row + int(animCtr>>counterShift)) & phaseMask
}

// ResolveAnimated maps a type1 tile word to the graphic it draws at world
// position (col, row) on tick animCtr. For water it selects the strip group
// 8 + phase while preserving the word's blend column and sub-cell; for every
// other word it is exactly Resolve, so position and counter change nothing.
//
// Like Resolve it is total over all 65536 words and never rejects. The slot
// it names stays inside 0..127 for every water input, since the group it
// substitutes is at most 11.
func ResolveAnimated(word uint16, col, row int, animCtr uint32) TileRef {
	g, b, sub := splitTileWord(word)
	if !IsWaterGroup(g) {
		return TileRef{Slot: g*blendColumns + b, Sub: sub}
	}
	phase := WaterPhase(g, col, row, animCtr)
	return TileRef{Slot: (waterGroupBase+phase)*blendColumns + b, Sub: sub, Water: true}
}

// Tick cadence (TERR-ANIM-008). The game's own logic tick fires every
// TickMillis ms, and the counter that drives WaterPhase advances by one per
// tick.
const (
	// SpeedIndexMin and SpeedIndexMax bound the game's speed index; values
	// outside are clamped, as the game clamps them.
	SpeedIndexMin = 0
	SpeedIndexMax = len(speedTPS) - 1

	// DefaultSpeedIndex is the index map load pushes: 16 tps, so a 62 ms tick,
	// a new water variant every ~248 ms and a full cycle every ~992 ms.
	DefaultSpeedIndex = 4

	// millisPerSecond is the dividend of the game's dtMs = 1000/tps.
	millisPerSecond = 1000

	// microsPerMilli widens a whole-millisecond tick to the unit a Ticker
	// holds. The index path MULTIPLIES the game's shipped quotient by it and
	// never re-divides: index 4 is 62 ms and therefore 62000 us, NOT the 62500
	// that dividing a microsecond dividend by 16 would give. The truncation is
	// the game's own arithmetic and belongs to the speeds that are the game's.
	microsPerMilli = 1000

	// microsPerSecond is the dividend of OUR rate model's period.
	microsPerSecond = millisPerSecond * microsPerMilli

	// minPeriodUS is the shortest period a Ticker will hold. Both period
	// functions below are positive over their whole domains, so this only keeps
	// the type total for a caller computing one some other way — and a zero
	// period is a division panic rather than a wrong cadence.
	minPeriodUS = 1
)

// RateMin and RateMax bound a RATE: a whole number of ticks a second, OURS by
// choice and not one of the game's nine speeds. A rate outside the range is
// brought to the nearest one inside it and never refused, exactly as the
// shipped index clamp treats an index outside the table.
//
// The ceiling is four times the highest rate asked for: unbounded, the period
// falls to zero and the accumulator has nothing to divide by, while at 1024 the
// truncated microsecond period is still within 0.1% of the rate it names.
const (
	RateMin = 1
	RateMax = 1024
)

// speedTPS maps a clamped speed index to logic ticks per second.
var speedTPS = [...]int{8, 10, 12, 14, 16, 20, 24, 28, 32}

// ClampSpeedIndex brings any index into range rather than rejecting it, keeping
// the cadence functions total. It is exported because the index is REPORTED as
// well as consumed — a viewer holding the index it was given for its own
// summary has to hold the same clamped value the cadence functions read, and a
// second copy of the rule is how the two would come to disagree.
func ClampSpeedIndex(speedIndex int) int {
	if speedIndex < SpeedIndexMin {
		return SpeedIndexMin
	}
	if speedIndex > SpeedIndexMax {
		return SpeedIndexMax
	}
	return speedIndex
}

// TicksPerSecond is the logic-tick rate for a speed index.
func TicksPerSecond(speedIndex int) int { return speedTPS[ClampSpeedIndex(speedIndex)] }

// TickMillis is one logic tick in milliseconds: 1000/tps by integer division,
// as the game computes it. The truncation is the game's own — index 4 gives 62,
// not 62.5, so a water cycle takes 992 ms rather than a round second.
func TickMillis(speedIndex int) int { return millisPerSecond / TicksPerSecond(speedIndex) }

// ClampRate brings any rate into RateMin..RateMax rather than rejecting it,
// keeping the rate model total the way ClampSpeedIndex keeps the index model
// total. It is the ONE spelling of that clamp: the ladder that doubles and
// halves a rate needs the same bounds the period does, and two copies is how a
// control comes to stop one step short of what the clock accepts.
func ClampRate(rate int) int {
	if rate < RateMin {
		return RateMin
	}
	if rate > RateMax {
		return RateMax
	}
	return rate
}

// RatePeriod is one tick of OUR rate model in microseconds: 1000000/rate over a
// clamped rate, truncated.
//
// A rate of ours is not a number the game ever held, so it inherits none of the
// game's millisecond truncation and is carried to the microsecond instead:
// 1000000/256 is 3906 us and runs at 256.0, where 1000/256 is 3 ms and runs at
// 333. Over the whole range the truncated period is at worst 0.0962% fast.
func RatePeriod(rate int) int { return microsPerSecond / ClampRate(rate) }

// RateOf is RatePeriod READ BACKWARDS: the whole number of ticks a second a
// clock holding this period is running at, clamped into the same range.
//
// It lives HERE, on the line under its forward twin, and that position is the
// whole mechanism: a period and a rate are two readings of one quantity, and an
// inverse written anywhere else is a second arithmetic to keep in step with this
// file's. Nothing is added to Ticker for it — that type holds ONE quantity and
// no notion of where it came from, deliberately, so that it cannot come to
// disagree with itself about what it is doing, and a rate field on it would be
// exactly the second nominal identity that doc refuses.
//
// It is TOTAL over every int, by the same clamps its twin uses: a period below
// one microsecond is the shortest a Ticker will hold, and the quotient is
// brought into range rather than refused.
//
// THE ROUND TRIP IS EXACT, and it is checked exhaustively rather than argued:
// RateOf(RatePeriod(r)) == ClampRate(r) for every one of the 1024 rates in
// range and for inputs past both ends. That is not a general property of
// truncating division composed with itself — it holds because the range's top
// is far below the dividend's square root — so it is pinned by a test that
// walks the whole domain, not by this paragraph.
//
// A period that is NOT one of RatePeriod's outputs still answers: the game's own
// speed-index periods reach this function whenever a map is freshly opened, and
// the map-load period of 62000 us reads back as 16 — the same 16 the speed table
// names. The truncation that made 62000 out of 62500 is the game's own and is
// visible in the period, which is why anything reporting a cadence reports both.
func RateOf(periodUS int) int {
	if periodUS < minPeriodUS {
		periodUS = minPeriodUS
	}
	return ClampRate(microsPerSecond / periodUS)
}

// SpeedIndexPeriod is one tick of the GAME's speed index in microseconds: the
// shipped whole-millisecond quotient WIDENED, never recomputed. Index 4 is
// 62000 us, so a water cycle is 992000 — the numbers this project holds because
// the game computes them that way, unchanged by the unit the clock now runs in.
func SpeedIndexPeriod(speedIndex int) int { return TickMillis(speedIndex) * microsPerMilli }

// The CADENCE LADDER: the ONE place a cadence key's meaning lives.
//
// A rung is an ordered position on it, slowest first, and the two cadence keys
// move that position by ONE. The MIDDLE of the ladder is the game's own speed
// table, rung for rung and period for period — the original's `+`/`-` step its
// speed index by one (TERR-ANIM-008), and so do ours while they are on it. That
// is what puts the cadence a map opens at ON the ladder, and therefore back
// within reach after any number of presses.
//
// Below and above the table the ladder is OURS, and it is reached only PAST the
// ends of the shipped set: the halvings from the table's slowest speed down to
// RateMin and the doublings from its fastest up to RateMax. The range is 0041's
// unchanged; what moved is that it now extends the shipped set instead of
// standing in for it.
//
// THAT SHAPE IS THE SEAM. The limit the decode implies is nine settings from 8
// to 32 ticks a second; lifting it is these two spans, named, bounded, and
// reported apart from the shipped ones by SpeedIndexOf below. What was here
// before did not extend the shipped set, it REPLACED it: a ladder that doubled
// a rate of our model could not land on 62000 us at all, so one press of `+`
// moved a freshly opened map off the game's own cadence and no sequence of
// presses brought it back.
//
// A step of one across the extension would not be a control — 0041's own
// objection to stepping a rate by one, and it still holds at 1024 — so the
// extension keeps that ladder's doubling while the shipped span keeps the
// game's own table.
const (
	// CadenceRungMin and CadenceRungMax bound the ladder. A rung outside is
	// brought to the nearest one inside and never refused, exactly as an index
	// outside the speed table and a rate outside the range are.
	CadenceRungMin = 0

	// CadenceShippedLo and CadenceShippedHi are the rungs the GAME's OWN nine
	// speeds occupy: rung CadenceShippedLo+i is speed index i. Outside them is
	// our extension, and how far outside is the rung's distance from whichever
	// end it left.
	//
	// The three rungs below are 8 -> 4 -> 2 -> 1 ticks a second and the five
	// above are 32 -> 64 -> ... -> 1024, which is what makes the ladder's two
	// ends exactly RateMin and RateMax. Those two identities are pinned by a
	// test against speedTPS and the rate bounds, not trusted here.
	CadenceShippedLo = 3
	CadenceShippedHi = CadenceShippedLo + SpeedIndexMax

	CadenceRungMax = CadenceShippedHi + 5

	// DefaultCadenceRung is the normal ladder position corresponding exactly
	// to the game's shipped default speed index. UI startup and the local
	// GameSpeed preference share this identity rather than restating the offset.
	DefaultCadenceRung = CadenceShippedLo + DefaultSpeedIndex
)

// cadencePeriods is the ladder itself, built once from the two period functions
// above so that neither the shipped speeds nor our own rates gain a second
// spelling in this package. It is strictly decreasing, which is what lets
// CadenceRung invert it.
var cadencePeriods = buildCadenceLadder()

func buildCadenceLadder() [CadenceRungMax + 1]int {
	var l [CadenceRungMax + 1]int
	for r := CadenceRungMin; r < CadenceShippedLo; r++ {
		l[r] = RatePeriod(speedTPS[SpeedIndexMin] >> uint(CadenceShippedLo-r))
	}
	for i := SpeedIndexMin; i <= SpeedIndexMax; i++ {
		l[CadenceShippedLo+i] = SpeedIndexPeriod(i)
	}
	for r := CadenceShippedHi + 1; r <= CadenceRungMax; r++ {
		l[r] = RatePeriod(speedTPS[SpeedIndexMax] << uint(r-CadenceShippedHi))
	}
	return l
}

// ClampCadenceRung brings any rung into range rather than rejecting it, keeping
// the ladder total the way ClampSpeedIndex and ClampRate keep the index and the
// rate models total. It is the ONE spelling of that clamp, for ClampRate's own
// reason: the control that steps a rung needs the same bounds the ladder does,
// and two copies is how a control comes to stop one step short of it.
func ClampCadenceRung(rung int) int {
	if rung < CadenceRungMin {
		return CadenceRungMin
	}
	if rung > CadenceRungMax {
		return CadenceRungMax
	}
	return rung
}

// CadencePeriod is the tick length of ladder rung n, in microseconds.
//
// It is what a cadence key WRITES, and it is written once per change and handed
// to both consumers of it, so there is one period value and two readers of it
// rather than one number two call sites each divide for themselves.
func CadencePeriod(rung int) int { return cadencePeriods[ClampCadenceRung(rung)] }

// CadenceRung is CadencePeriod READ BACKWARDS: the rung a clock holding this
// period stands on. It lives on the line under its forward twin for RateOf's
// reason — an inverse written anywhere else is a second arithmetic to keep in
// step with this file's.
//
// THE ROUND TRIP IS EXACT: CadenceRung(CadencePeriod(n)) == ClampCadenceRung(n)
// for every rung, checked exhaustively rather than argued. That follows from the
// ladder being strictly decreasing, which is itself pinned by a test.
//
// It is TOTAL over every int. A period no rung produced answers with the first
// rung at least as fast as it — the slowest rung that would not run the world
// slower than the period asks — and a period faster than the whole ladder
// answers with its top. A caller that needs to know whether the period was on
// the ladder at all compares CadencePeriod of the answer against it, which is
// what SpeedIndexOf does; nothing here pretends an off-ladder period is a rung.
func CadenceRung(periodUS int) int {
	for r := CadenceRungMin; r <= CadenceRungMax; r++ {
		if cadencePeriods[r] <= periodUS {
			return r
		}
	}
	return CadenceRungMax
}

// SpeedIndexOf is SpeedIndexPeriod read backwards: which of the game's nine
// speed settings a clock holding this period is running at, and whether ANY
// of them is.
//
// IT REPORTS ABSENCE WHERE RateOf CLAMPS, and the difference is deliberate.
// RatePeriod is a division over a continuous range and its inverse can bring
// any period into that range without asserting anything false. SpeedIndexPeriod
// names exactly nine periods; snapping a tenth to the nearest of them would
// state that the game offers a setting the game does not offer, which is the
// substitution this story exists to remove. So the match is EXACT: a period our
// extension produced, and a period nothing on the ladder produced, both answer
// false.
func SpeedIndexOf(periodUS int) (int, bool) {
	r := CadenceRung(periodUS)
	if cadencePeriods[r] != periodUS || r < CadenceShippedLo || r > CadenceShippedHi {
		return 0, false
	}
	return r - CadenceShippedLo, true
}

// Ticker converts elapsed wall-clock MICROSECONDS into logic ticks, carrying
// the sub-tick remainder so no time is lost or double-counted.
//
// It holds ONE quantity — the period — and no notion of where that period came
// from. A rate and a decoded speed index are two ways to compute one
// (RatePeriod, SpeedIndexPeriod) and neither is the other; a clock carrying both
// a period and an index would carry two nominal identities that can come to
// disagree about what it is doing.
//
// The clock itself belongs to the caller: this tier sees only integer
// microseconds, which is what makes the cadence testable without a clock or a
// window.
type Ticker struct {
	periodUS int
	acc      int    // microseconds not yet consumed by a whole tick
	count    uint32 // the animCtr WaterPhase reads
}

// NewTicker builds a ticker at the given period in microseconds. Pass
// SpeedIndexPeriod(DefaultSpeedIndex) for the cadence map load selects, or
// RatePeriod(r) for a rate.
func NewTicker(periodUS int) *Ticker {
	t := &Ticker{}
	t.SetPeriod(periodUS)
	return t
}

// SetPeriod changes the tick length. The accumulated remainder is kept, so a
// re-rate mid-run neither loses the fraction of a tick already elapsed nor
// fires a spurious tick.
func (t *Ticker) SetPeriod(periodUS int) {
	if periodUS < minPeriodUS {
		periodUS = minPeriodUS
	}
	t.periodUS = periodUS
}

// Period reports the current tick length in microseconds.
func (t *Ticker) Period() int { return t.periodUS }

// Count reports the tick counter to pass to WaterPhase / ResolveAnimated.
func (t *Ticker) Count() uint32 { return t.count }

// Remainder reports the microseconds accumulated toward the next tick: how far
// through the current tick this clock stands, in the same unit Period reports.
//
// It is NOT bounded by the period. SetPeriod keeps the accumulated remainder, so
// a re-rate to a shorter period leaves a value larger than the period behind,
// and the very next AdvanceMicros consumes it as the whole ticks it is worth. A
// consumer measuring a fraction of a tick with it owns that clamp.
func (t *Ticker) Remainder() int { return t.acc }

// AdvanceMicros feeds elapsed wall-clock microseconds to the accumulator and
// returns how many whole ticks fired. A negative elapsed is ignored rather than
// rewinding the counter. The whole quotient is taken at once, so a long stall
// (a minimised window, a breakpoint) costs one division rather than a catch-up
// loop, and the time it represents is still conserved exactly.
func (t *Ticker) AdvanceMicros(elapsedUS int) int {
	if elapsedUS <= 0 {
		return 0
	}
	t.acc += elapsedUS
	ticks := t.acc / t.periodUS
	t.acc -= ticks * t.periodUS
	t.count += uint32(ticks)
	return ticks
}

// AdvanceOne advances exactly one logical tick without consulting a deadline
// or changing the fractional paced phase. It is the clock-side primitive for
// the decoded unpaced owner loop (`SESS-CLOCK-005`): one eligible idle callback
// is one tick even when no wall-clock microsecond elapsed, and a long callback
// gap is still one tick rather than a catch-up quotient.
func (t *Ticker) AdvanceOne() { t.count++ }

// ResetPhase clears the fractional progress toward the next paced tick while
// preserving the selected period and the logical counter. Ctrl+numpad minus
// resets the original's phase/epoch when it restores the paced loop; resetting
// the counter here would instead rewind visible animation.
func (t *Ticker) ResetPhase() { t.acc = 0 }

func (t *Ticker) Restore(count uint32, remainderUS int) { t.count, t.acc = count, remainderUS }
