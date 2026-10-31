package sim

import (
	"math/bits"

	"againrom/pkg/rules"
)

// A spell's range and duration are computed here only. Each surface names
// its projection: the record word the book popup states (TEXT-096) or the
// value a cast uses. They differ: the record range is a byte and grows by
// power/3 for arm 26 on a zero column; an area's record duration omits the
// removal tick and the staged cadence. Which the popup should state is open.

// rangeProjection selects which range a surface reads.
type rangeProjection uint8

const (
	// rangeRecord is the spell record's byte +9: the book popup's range.
	rangeRecord rangeProjection = iota
	// rangeCast is a cast's reach, which the book slot stores.
	rangeCast
)

// durationProjection selects which duration a surface reads.
type durationProjection uint8

const (
	// durationRecord is the spell record's word +0x10: the popup's duration.
	durationRecord durationProjection = iota
	// durationArea is an area's life, counting its removal tick.
	durationArea
	// durationEffect is an attached effect's duration before resistance.
	durationEffect
)

// spellRangeOf is the one range function. Rules{} reads no formula table.
func spellRangeOf(r Rules, rule SpellRule, power int32, p rangeProjection) int64 {
	if p == rangeRecord {
		return spellRecordRangeUnder(r, rule, power)
	}
	return spellRangeUnder(r, rule, power)
}

// spellDurationOf is the one duration function.
func spellDurationOf(r Rules, rule SpellRule, power int32, p durationProjection) uint16 {
	switch p {
	case durationRecord:
		return spellRecordDurationUnder(r, rule, power)
	case durationArea:
		return areaLife(rule, uint16(power))
	}
	return spellPointDurationUnder(r, rule, power)
}

// lastingTicks is the duration word of a power-scaled effect: base sixteenths
// of a unit times 1.025^power (durationSlow) or 1.05^power (durationFast),
// truncated, saturating at durationTickCeiling.
func lastingTicks(power int32, base int32, scale durationScale) uint16 {
	if base <= 0 {
		return 0
	}
	if power < 0 {
		power = 0
	} else if power > spellPowerMax {
		power = spellPowerMax
	}
	m := uint64(base) * 16
	var v uint64
	if power <= 100 {
		factor := durationSlowQ56[power]
		if scale == durationFast {
			factor = durationFastQ56[power]
		}
		hi, lo := bits.Mul64(m, factor)
		if hi>>56 != 0 {
			return durationTickCeiling
		}
		v = hi<<8 | lo>>56
	} else {
		f := durationSlowAbove[power-101]
		if scale == durationFast {
			f = durationFastAbove[power-101]
		}
		carry, _ := bits.Mul64(f.Frac, m)
		v = f.Whole*m + carry
	}
	if v > durationTickCeiling {
		return durationTickCeiling
	}
	return uint16(v)
}

// segmentedTicks is the duration word of Invisibility and Stone Curse. Up to
// power 100 it is lastingTicks. Above, the duration is k whole durations at
// power 100 plus the duration at the remaining power, where
// k = floor((power-1)/100), so power 200 holds two durations at 100 and power
// 250 holds two and the duration at 50.
func segmentedTicks(power int32, base int32, scale durationScale) uint16 {
	if power > spellPowerMax {
		power = spellPowerMax
	}
	if power <= 100 {
		return lastingTicks(power, base, scale)
	}
	k := (power - 1) / 100
	total := uint32(k)*uint32(lastingTicks(100, base, scale)) + uint32(lastingTicks(power-100*k, base, scale))
	if total > durationTickCeiling {
		return durationTickCeiling
	}
	return uint16(total)
}

// spellLastingTicks is the power-scaled duration word of a row whose effect
// duration follows its Spell Duration column: Invisibility on the fast law and
// a base of 3, Stone Curse segmented, every other row on the slow law.
func spellLastingTicks(rule SpellRule, power int32) uint16 {
	switch rule.arm() {
	case 15:
		return segmentedTicks(power, 3, durationFast)
	case 20:
		return segmentedTicks(power, rule.SpellDuration, durationSlow)
	}
	return lastingTicks(power, rule.SpellDuration, durationSlow)
}

// spellRange is the decoded cast reach after power. Teleport is the sole
// faster-growing arm; every other spell adds power/30 to its installed base.
func spellRange(rule SpellRule, power int32) int64 {
	if rule.bookInstance {
		return int64(rule.MaxRange)
	}
	// MAGIC-ARM-014's first gate is exact: a zero maximum means no range
	// bonus is computed at all. Shield and Fire Sacrifice ship on this arm.
	if rule.MaxRange == 0 {
		return 0
	}
	divisor := int32(30)
	if rule.arm() == 26 {
		divisor = 3
	}
	return int64(rule.MaxRange) + int64(power/divisor)
}

// spellRecordRange is the record's byte +9: the row's range column plus
// power/30 when that column is non-zero, power/3 for Teleport (TEXT-096).
func spellRecordRange(rule SpellRule, power int32) int64 {
	if rule.bookInstance {
		return int64(rule.MaxRange)
	}
	base := int32(rule.MaxRange)
	switch {
	case rule.arm() == 26:
		base += power / 3
	case base != 0:
		base += power / 30
	}
	return int64(uint8(base))
}

// spellRecordDuration is the record's word +0x10 in sixteenths of the duration
// unit: the Spell Duration column through the power ramp when it is positive,
// else the Area Effect Duration column plus power*16/10 when that is positive,
// else 0 (TEXT-096). Spell 15 takes the fast ramp on a base of 3.
func spellRecordDuration(rule SpellRule, power int32) uint16 {
	switch {
	case rule.SpellDuration > 0:
		return spellLastingTicks(rule, power)
	case rule.AreaDuration > 0:
		return uint16(int64(rule.AreaDuration)<<4 + (int64(power)<<4)/10)
	}
	return 0
}

func rangeBonusFor(r Rules, rule SpellRule, power int32) (int64, bool) {
	v, ok := r.SpellFormulaAt(rules.FormulaRange, rule.ID, power)
	return int64(v), ok
}

func spellRangeUnder(r Rules, rule SpellRule, power int32) int64 {
	if rule.bookInstance || rule.MaxRange == 0 {
		return spellRange(rule, power)
	}
	if bonus, ok := rangeBonusFor(r, rule, power); ok {
		return min(int64(rule.MaxRange)+bonus, 255)
	}
	return spellRange(rule, power)
}

func spellRecordRangeUnder(r Rules, rule SpellRule, power int32) int64 {
	if !rule.bookInstance && (rule.arm() == 26 || rule.MaxRange != 0) {
		if bonus, ok := rangeBonusFor(r, rule, power); ok {
			return min(int64(rule.MaxRange)+bonus, 255)
		}
	}
	return spellRecordRange(rule, power)
}

func spellLastingTicksUnder(r Rules, rule SpellRule, power int32) uint16 {
	if rule.Second && rule.arm() == 15 {
		// The second game's Invisibility lasts P << 4 (R2-ENGINE-022), at
		// application and in the spell record alike.
		return uint16(min(int64(power)<<4, durationTickCeiling))
	}
	f, ok := r.SpellFormulaAt(rules.FormulaDuration, rule.ID, power)
	if !ok {
		return spellLastingTicks(rule, power)
	}
	base := int64(rule.SpellDuration)
	if rule.arm() == 15 {
		base = 3
	}
	if base <= 0 {
		return 0
	}
	return uint16(min(base*16*int64(f)/1000, durationTickCeiling))
}

func spellPointDurationUnder(r Rules, rule SpellRule, power int32) uint16 {
	switch rule.arm() {
	case 15, 5, 10, 16, 18, 20, 22, 23, 24, 27, 28:
		return spellLastingTicksUnder(r, rule, power)
	}
	if rule.Second && areaRow(rule.arm()).secondLasting {
		return spellLastingTicksUnder(r, rule, power)
	}
	return rule.EffectDuration
}

func spellRecordDurationUnder(r Rules, rule SpellRule, power int32) uint16 {
	if rule.SpellDuration > 0 {
		return spellLastingTicksUnder(r, rule, power)
	}
	return spellRecordDuration(rule, power)
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
// `(stages - 1) * 3 + 1`. The book popup states the record word instead
// (spellRecordDuration).
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
func ringLife(arm uint16) uint16 {
	n := ringStageCount(arm)
	if n == 0 {
		return 0
	}
	return uint16((n-1)*3 + 1)
}
