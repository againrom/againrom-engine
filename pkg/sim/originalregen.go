package sim

// regenerateOriginalCurrent consumes the restored words, not a re-derived
// sheet. SAV-REGENWIDTH-528 through SAV-REGENWIRE-532 establish the local
// arithmetic and health-before-mana order. Original post-load clock origin,
// state-16 lifetime and callbacks remain explicit DIV-738/1016 limits.
func (w *World) regenerateOriginalCurrent(i int, health bool) {
	e := &w.entities[i]
	if int16(e.HP) <= 0 {
		return
	}
	rate := e.regenerationRate(w.tick)
	if health && int16(e.HealthRegenPeriod) != 0 {
		hp := e.HP
		regenerateCurrentWord(&hp, e.MaxHP, e.HealthRegenPeriod, &e.HealthHundredths, 2, e.HealthRegeneration, rate)
		e.setCurrentHealth(hp)
		// The original callback's effects are Unknown. Use the existing native
		// death transition so a negative gain cannot leave a dead actor with a
		// live order, cast or invalid native save. This is not damage/XP.
		w.clearFelled(i)
	}
	// The entry health gate is not rechecked after the health stores/callback.
	mana := e.Mana
	regenerateCurrentWord(&mana, e.MaxMana, e.ManaRegenPeriod, &e.ManaHundredths, 1, e.ManaRegeneration, rate)
	e.setCurrentMana(mana)
}

// regenerateCurrentWord has signed-word reads, wrapping dword products and
// accumulator, truncating division, then byte/word stores before the signed
// upper bound. In particular it neither normalizes negative remainders nor
// clamps below zero. Conversion happens at reads even while an untouched SAV
// pool still carries its original u16 magnitude (DIV-634).
func regenerateCurrentWord(cur *int32, maximum, period int32, rest *uint8, factor, modifier, rate int32) {
	c, m, p := int32(int16(*cur)), int32(int16(maximum)), int32(int16(period))
	if c >= m {
		return
	}
	// LOAD and native decode refuse a source-current zero mana divisor with a
	// nonzero maximum. Health zero is gated by the caller. Keep the low-level
	// helper total for invalid hand-built state; this is not ROM1 fault handling.
	if p == 0 {
		return
	}
	product := m * factor * (int32(int16(modifier)) + 100) * rate
	acc := c*100 + int32(*rest) + product/p
	*rest = uint8(acc % 100)
	*cur = int32(int16(acc / 100))
	if *cur > m {
		*cur = m
	}
}
