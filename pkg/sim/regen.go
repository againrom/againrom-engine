package sim

import (
	"math"
	"math/big"
)

// Native regeneration (0109) retains its widened arithmetic policy. Imported
// source-current actors dispatch to originalregen.go's signed-word consumer
// instead (1107, DIV-738); native-retired actors use this legacy branch.
//
// regenPhase is the sixteen-sub-tick slot the whole pass dispatches on, the
// same slot decayPass's own walk is filtered on. It is written out here
// rather than aliased to decayPhase in step.go: the two share a VALUE and
// not a REASON — the ladder's is a free choice between two legal slots and
// this one is the dispatch slot itself — and an alias would let a later
// story's decision about the ladder move regeneration by accident.
//
// manaRegenCycle is scriptCycle itself, under its own name: the mana arm
// fires on every full tick, which is scriptCycle's own cadence.
//
// regenScale is the fixed-point scale a pool is carried at while a gain
// accumulates — a hundredth of a point, which is what the remainder byte
// holds. The two are the same number and are two constants ON PURPOSE:
// folding them into one would make the gain expression read as `max * 2 /
// period`, which truncates to zero for most units and is the one way this
// story can be built so that it compiles, stays deterministic, and heals
// nobody.
//
// HERO-REGEN-021
const (
	regenPhase       = 12
	manaRegenCycle   = scriptCycle
	healthRegenCycle = 4 * scriptCycle
	regenScale       = 100
	regenPercent     = 100
)

// regenPass advances every living entity's health and mana toward their
// maxima, on the dispatch slot regenPhase names.
//
// It is called every sub-tick, exactly as decayPass is, and phase-gates only
// itself: there is no dispatch here for it to join. healthRegenCycle is a
// multiple of manaRegenCycle, so every tick that qualifies for health
// already qualifies for mana; the separate test below is what says so rather
// than that arithmetic fact.
func (w *World) regenPass() {
	if w.tick%manaRegenCycle != regenPhase {
		return
	}
	health := w.tick%healthRegenCycle == regenPhase
	w.regenerateActors(health)
}

func (w *World) regenerateActors(health bool) {
	for i := range w.entities {
		e := &w.entities[i]
		// The pass walks the on-map list, which an actor a script took off the
		// map has left, so it gains neither health nor mana until it returns
		// (UNIT-REGEN-130).
		if e.OffMap {
			continue
		}
		if e.CurrentProfileBasis == ProfileOriginalCurrent {
			w.regenerateOriginalCurrent(i, health)
			continue
		}
		if !e.Alive() {
			continue
		}
		rate := e.regenerationRate(w.tick)
		mana := e.Mana
		regenerate(&mana, e.MaxMana, e.ManaRegenPeriod, &e.ManaHundredths, 1, e.ManaRegeneration, rate)
		e.setCurrentMana(mana)
		if health {
			hp := e.HP
			regenerate(&hp, e.MaxHP, e.HealthRegenPeriod, &e.HealthHundredths, 2, e.HealthRegeneration, rate)
			e.setCurrentHealth(hp)
		}
	}
}

// regenerate advances one pool — health or mana — by one qualifying
// tick's gain.
//
// It returns at once on three gates, and the first is NOT optional:
//
//   - max <= 0. This is a retained native-domain guard, not a ROM1 signed-word
//     rule (SAV-REGENWIDTH-528). The gain below is proportional to the
//     maximum and takes its sign, and *cur >= max never closes under a
//     negative one — so without this gate, the mana of an entity below a
//     negative maximum falls forever and wraps its own width. The health arm
//     is shielded from this by Alive() elsewhere in the package (a maximum
//     that is zero or negative there pairs only with a health of exactly
//     zero, the one value Alive() still admits); the mana arm has no such
//     shield, which is why this is one gate in the shared helper rather than
//     a clause on the health arm alone.
//   - *cur >= max. The pool has nothing left to gain.
//   - period <= 0. No shipped class pairs an absent period with a
//     pool it could fill, so this case is unreachable on the corpus — but an
//     unreachable divide is still a divide, and this is what keeps it one.
//
// Past those three it computes the gain, the accumulator, the remainder and
// the capped store, in that order, and the division is taken ONCE, on the
// whole product: a gain that truncates to zero on a given tick is the
// arithmetic's own answer, not a defect, because the remainder byte is what
// carries the rest to the next qualifying tick (AC-2).
func regenerate(cur *int32, max, period int32, rest *uint8, double int64, modifier, rate int32) {
	if max <= 0 || *cur >= max || period <= 0 {
		return
	}
	product := int64(max) * double * int64(regenPercent+modifier)
	if rate > 1 && (product > math.MaxInt64/int64(rate) || product < math.MinInt64/int64(rate)) {
		regenerateWide(cur, max, period, rest, product, rate)
		return
	}
	gain := product * int64(rate) / int64(period)
	base := int64(*cur)*regenScale + int64(*rest)
	if (base > 0 && gain > math.MaxInt64-base) || (base < 0 && gain < math.MinInt64-base) {
		regenerateWide(cur, max, period, rest, product, rate)
		return
	}
	acc := base + gain
	rem := acc % regenScale
	q := acc / regenScale
	if rem < 0 {
		rem += regenScale
		q--
	}
	if q > int64(max) {
		q = int64(max)
	}
	*cur = int32(q)
	*rest = uint8(rem)
}

// Only extreme native int32 modifiers need this path. Rate3 must not turn a
// positive widened gain into damage by overflowing the former rate1 product.
// The original-current consumer deliberately retains its separate dword wrap.
func regenerateWide(cur *int32, maximum, period int32, rest *uint8, product int64, rate int32) {
	var acc, gain, q, rem big.Int
	gain.SetInt64(product)
	gain.Mul(&gain, big.NewInt(int64(rate)))
	gain.Quo(&gain, big.NewInt(int64(period)))
	acc.SetInt64(int64(*cur)*100 + int64(*rest))
	acc.Add(&acc, &gain)
	q.QuoRem(&acc, big.NewInt(100), &rem)
	if rem.Sign() < 0 {
		rem.Add(&rem, big.NewInt(100))
		q.Sub(&q, big.NewInt(1))
	}
	if q.Cmp(big.NewInt(int64(maximum))) > 0 {
		*cur = maximum
	} else {
		*cur = int32(q.Int64())
	}
	*rest = uint8(rem.Uint64())
}
