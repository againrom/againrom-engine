package terrain

// TERR-LIGHT-120

// ramp is q(k) = k*m/120, the one ramp helper every twilight arm calls with
// its own k. Division truncates toward zero, matching every other integer
// division this package's transforms use.
func ramp(k, m int) int {
	return k * m / 120
}

// SkyLight returns the sky tint, the two intensity bytes and the two shroud
// indices at an in-game minute, on the schedule's own seven arms. With cycle
// false it returns the cycle-off arm at every minute -- the same five values
// the day arm below carries, and the same shroud pair DefaultDaytime holds.
// Theta is not this function's; the Light it returns always carries the zero
// angle, left at its type's zero value.
//
// With cycle true, the hour selects one of six bands (the seventh arm, cycle
// off, has no hour -- it is selected by the switch's argument alone). The
// hour is computed exactly as SunAngle computes it, and the ramp phase is
// the already-1440-reduced minute modulo 120: every two-hour half-band spans
// exactly 120 minutes and opens at phase 0, so a ramp is never entered
// part-way (spec, "The schedule").
func SkyLight(minute uint64, cycle bool) Light {
	if !cycle {
		return Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}
	}

	m := minute % MinutesPerDay
	hour := ((m + ClockOriginMinute) / minutesPerHour) % hoursPerDay
	p := int(m % 120)

	switch {
	case hour >= 6 && hour <= 17: // day -- the same five values as cycle off, no ramp.
		return Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}

	case hour == 18 || hour == 19: // dusk 1
		return Light{
			Ambient:      uint8(14 + ramp(10, p)),
			Range:        uint8(32 - ramp(12, p)),
			SkyTint:      [3]uint8{uint8(ramp(24, p)), 0, uint8(ramp(8, p))},
			ShroudObject: 6,
			ShroudUnit:   3,
		}

	case hour == 20 || hour == 21: // dusk 2
		return Light{
			Ambient:      uint8(24 + ramp(8, p)),
			Range:        uint8(20 - ramp(12, p)),
			SkyTint:      [3]uint8{uint8(24 - ramp(24, p)), uint8(ramp(12, p)), uint8(8 + ramp(40, p))},
			ShroudObject: 6,
			ShroudUnit:   3,
		}

	case hour == 2 || hour == 3: // dawn 1
		return Light{
			Ambient:      uint8(32 - ramp(4, p)),
			Range:        uint8(8 + ramp(12, p)),
			SkyTint:      [3]uint8{uint8(ramp(24, p)), 12, uint8(48 - ramp(48, p))},
			ShroudObject: 6,
			ShroudUnit:   3,
		}

	case hour == 4 || hour == 5: // dawn 2
		return Light{
			Ambient:      uint8(28 - ramp(14, p)),
			Range:        uint8(20 + ramp(12, p)),
			SkyTint:      [3]uint8{uint8(24 - ramp(24, p)), uint8(12 - ramp(12, p)), 0},
			ShroudObject: 6,
			ShroudUnit:   3,
		}

	default: // night: 22, 23, 0, 1 -- constant, no ramp.
		return Light{Ambient: 32, Range: 8, SkyTint: [3]uint8{0, 12, 48}, ShroudObject: 8, ShroudUnit: 4}
	}
}
