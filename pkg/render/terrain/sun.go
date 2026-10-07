package terrain

// The clock, in the units the sun model runs on.
const (
	// SubTicksPerMinute is the tick cycle the whole engine runs on: sixteen
	// sub-ticks are one full tick, and ONE FULL TICK IS ONE IN-GAME MINUTE.
	// pkg/sim's own tick is that sub-tick — it is the unit its script pass and
	// its engagement decision take modulo 16 — so no second clock is needed.
	SubTicksPerMinute = 16

	// MinutesPerDay is 24 x 60. Every clock is reduced modulo this before
	// anything else, which is the whole of why the model is total over uint64:
	// the reduction is exact, because 720, 240 and 1440 all divide it, so a
	// reduced clock and a raw one select the same band and the same phase.
	MinutesPerDay = 1440

	// ClockOriginMinute is the 0x168 the sun model's argument carries: the
	// argument is fullTicks + 360, so a run beginning at sub-tick 0 begins at
	// in-game 06:00 — the first minute of the daylight band.
	ClockOriginMinute = 360

	// RelightPeriodMinutes is how often the light is rebuilt when nothing
	// forces it: one relight per 20 in-game minutes, 72 per in-game day.
	RelightPeriodMinutes = 20

	minutesPerHour = 60
	hoursPerDay    = 24
)

// The angle constants, read from the image's own bytes rather than from a
// disassembler's labels.
const (
	// sunHigh is +0.78539815, the double 0x3fe921fb4d12d84a: a quarter of a
	// TRUNCATED pi (3.1415926/4), and NOT math.Pi/4, whose double is
	// 0x3fe921fb54442d18. It is what the cycle-off arm and the dusk arm store,
	// and it is the night arm's subtrahend.
	sunHigh = 0.78539815

	// sunLow is -0.78539815, the same double with its sign bit set. It is what
	// the dawn arm stores and it is the day arm's subtrahend.
	sunLow = -0.78539815

	// daySunStep is the day band's per-minute increment. The image holds it
	// NEGATED, as the double 0xbf61df469d353918, and reaches this sign through
	// a reverse subtract; the sign is folded into the expression here rather
	// than into the constant, so the constant reads as the step it is.
	//
	// 720 x this is 1.5707963 — the whole daylight sweep, and a truncated pi/2.
	daySunStep = 0.0021816615277777777

	// nightSunStep is the night band's per-minute decrement: the same arc
	// traversed in 240 minutes instead of 720, so exactly three times the day
	// step. It is written as its own literal and not as 3*daySunStep, so that
	// the relation stays a check.
	nightSunStep = 0.0065449845833333332
)

// ClockMinute is the in-game minute a sub-tick clock stands at.
func ClockMinute(subTicks uint64) uint64 { return subTicks / SubTicksPerMinute }

// RelightDue reports whether an unforced relight fires on this sub-tick.
//
// Both halves are the engine's own gate. The first selects the sub-tick on which
// the full tick rolls, so the light is never rebuilt part-way through a minute;
// the second selects one full tick in twenty, measured on the sun model's own
// argument rather than on the raw minute. The +360 is therefore kept even though
// 360 is a multiple of 20 and cancels: the gate is written on the argument, and
// hiding that would make the day's phase and the relight's phase look like two
// unrelated numbers.
//
// It is TOTAL over uint64 and needs no reduction: both operations are moduli.
func RelightDue(subTicks uint64) bool {
	if subTicks%SubTicksPerMinute != 0 {
		return false
	}
	return (ClockMinute(subTicks)+ClockOriginMinute)%RelightPeriodMinutes == 0
}

// SunAngle is the sun's angle in radians at an in-game minute, with the
// cycle on or off.
//
// With the cycle OFF it is the fixed literal, at every minute — the value this
// tree drew before the cycle existed.
//
// With it ON the hour selects one of four bands that partition the day:
//
//	 6..17  day    computed, m = minute mod 720,  theta = sunLow  + m*daySunStep
//	18..21  dusk   the literal sunHigh
//	22,23,0,1 night computed, m = minute mod 240, theta = sunHigh - m*nightSunStep
//	 2..5   dawn   the literal sunLow
//
// so 960 of a day's 1440 minutes carry a computed angle and 480 a literal one,
// and the angle never leaves [sunLow, sunHigh].
//
// THE TWO PHASES LOOK SIMPLER HERE THAN IN THE IMAGE, and the simplification is
// arithmetic rather than a reading. The routine's argument is minute+360 and it
// adds 0x168 a SECOND time internally, so the day band's phase is
// (minute+720) mod 720, which is minute mod 720; the night band's is
// (minute+480) mod 240, which is minute mod 240. Both are written in the reduced
// form because the unreduced one invites a reader to believe the phase is offset
// from the minute when it is not — the band BOUNDARIES carry the whole of the
// +360, and they are computed from it below.
//
// The hour is computed from the argument and not from the minute, which is where
// the 06:00 start comes from: minute 0 gives argument 360 gives hour 6.
func SunAngle(minute uint64, cycle bool) float64 {
	if !cycle {
		return sunHigh
	}
	// Reduced first, so the function is total over uint64. 1440 is a multiple
	// of 720, of 240 and of 60, so the reduction moves neither phase below nor
	// the hour.
	m := minute % MinutesPerDay
	hour := ((m + ClockOriginMinute) / minutesPerHour) % hoursPerDay
	switch {
	case hour >= 6 && hour <= 17:
		return sunLow + float64(m%720)*daySunStep
	case hour >= 18 && hour <= 21:
		return sunHigh
	case hour >= 2 && hour <= 5:
		return sunLow
	default: // 22, 23, 0, 1
		return sunHigh - float64(m%240)*nightSunStep
	}
}

// SunAt is the whole Light at an in-game minute: skylight.go's SkyLight for
// the sky tint, the two intensity bytes and the two shroud indices, with
// Theta set to SunAngle's own answer. It no longer reads DefaultDaytime.
//
// THE TWO HALVES ARE INDEPENDENTLY DECODED FUNCTIONS, joined here and
// nowhere else: SkyLight carries no angle (its Theta is always the type's
// zero value) and SunAngle carries none of the other four. Composing them,
// rather than re-spelling either one inside this function, is what turns
// "with the cycle off this is exactly the light the tree drew before" into
// an AGREEMENT BETWEEN TWO READINGS -- SkyLight's cycle-off arm is asserted
// equal to DefaultDaytime (AC-3), it is no longer copied from it here --
// which is the stronger form. That closes 0092's D-1 (0100 spec D-1):
// docs/0092-day-night/ itself is unamended and stands as the record of what
// that story shipped.
//
// THE SCHEDULE NOW MOVES EVERYTHING A SUN CAN MOVE. The angle changes the
// relief's contrast and reverses its lateral shear at noon, exactly as before;
// the ambient, range and sky tint now darken the ground and every sprite
// through the night as well, where until this story they held the daytime
// value at every band. skylight.go carries the whole per-band table; this
// function owns only where the two meet.
func SunAt(minute uint64, cycle bool) Light {
	lt := SkyLight(minute, cycle)
	lt.Theta = SunAngle(minute, cycle)
	return lt
}
