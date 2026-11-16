package sim

import (
	"math"
	"math/big"
)

// Native regeneration (0109) retains its widened arithmetic policy. Imported
// source-current actors dispatch to originalregen.go's signed-word consumer
// instead (1107, DIV-738); native-retired actors use this legacy branch.
//
// regenPhase is the sixteen-sub-tick slot the pass dispatches on. It shares
// decayPhase's value and not its reason, so it is no alias of it.
// manaRegenCycle is scriptCycle: the mana arm fires on every full tick.
// regenScale is the fixed-point scale of a pool while a gain accumulates, a
// hundredth of a point, which the remainder byte holds; regenPercent is the
// modifier's base.
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
// maxima, on the dispatch slot regenPhase names. It is the second game's
// regeneration (nativeRegen), called every sub-tick, exactly as decayPass is.
//
// Mana moves on every pass. Health moves on the pass whose count mod 4
// equals the actor's runtime id mod 4 (R2-ENGINE-1205), so the actors of one
// map heal on four staggered passes. The pass count is the sub-tick over
// sixteen; the original counts passes in a counter of its own.
func (w *World) regenPass() {
	const phases = healthRegenCycle / manaRegenCycle
	if w.smoothRegen {
		mana := windowPart(w.tick%manaRegenCycle, regenPhase, manaRegenCycle)
		w.regenerateEach(mana, func(e *Entity) (regenPart, bool) {
			end := uint64(e.SourceBinding.RuntimeID)%phases*manaRegenCycle + regenPhase
			return windowPart(w.tick%healthRegenCycle, end, healthRegenCycle), true
		})
		return
	}
	if w.tick%manaRegenCycle != regenPhase {
		return
	}
	pass := w.tick / manaRegenCycle
	w.regenerateEach(regenPart{}, func(e *Entity) (regenPart, bool) {
		return regenPart{}, pass%phases == uint64(e.SourceBinding.RuntimeID)%phases
	})
}

// regenerateActors is the first game's pool pass: health moves for every
// actor or for none.
func (w *World) regenerateActors(health bool) {
	w.regenerateEach(regenPart{}, func(*Entity) (regenPart, bool) { return regenPart{}, health })
}

// regenerateActorsSmooth is the first game's pool pass under the smooth option.
// FullTick turns over after phase 14, so a cycle position counts from phase 15.
func (w *World) regenerateActorsSmooth(c SessionClock) {
	phase := uint64((c.phase() + 16) % 16)
	cycle := uint64(c.FullTick&3)*manaRegenCycle + (phase+1)%manaRegenCycle
	health := windowPart(cycle, regenPhase+1, healthRegenCycle)
	w.regenerateEach(windowPart(phase, regenPhase, manaRegenCycle), func(*Entity) (regenPart, bool) { return health, true })
}

// regenPart is part j of k of a window's gain; the zero value is the whole.
type regenPart struct{ j, k int64 }

// windowPart is the part at position pos of a cycle of k closing at end.
func windowPart(pos, end, k uint64) regenPart {
	return regenPart{j: int64((pos + 2*k - end - 1) % k), k: int64(k)}
}

func (p regenPart) applies(negative bool) bool {
	return p.k == 0 || !negative || p.j == p.k-1
}

// step is the part of gain for a pool holding base; negatives go whole on the
// closing part.
func (p regenPart) step(gain, base int64) (part int64, apply bool) {
	negative := gain < 0 || base < 0
	if !p.applies(negative) {
		return 0, false
	}
	if p.k == 0 || negative {
		return gain, true
	}
	return gain/p.k + partShare(p, gain%p.k), true
}

// partShare is what part p adds to q for a nonnegative gain q*k+r, so no
// product passes r*k.
func partShare(p regenPart, r int64) int64 {
	return r*(p.j+1)/p.k - r*p.j/p.k
}

// regenerateEach advances each on-map actor's pools by the mana part and by
// the part health names for the actor, if the health arm runs.
func (w *World) regenerateEach(mana regenPart, health func(*Entity) (regenPart, bool)) {
	for i := range w.entities {
		e := &w.entities[i]
		// The pass walks the on-map list, which an actor a script took off the
		// map has left, so it gains neither health nor mana until it returns
		// (UNIT-REGEN-130).
		if e.OffMap {
			continue
		}
		healthPart, healthDue := health(e)
		if e.CurrentProfileBasis == ProfileOriginalCurrent {
			w.regenerateOriginalCurrent(i, healthDue, mana, healthPart)
			continue
		}
		if !e.InLife(w.game.regenLife) {
			continue
		}
		rate := e.regenerationRate(w.tick)
		m := e.Mana
		regenerate(&m, e.MaxMana, e.ManaRegenPeriod, &e.ManaHundredths, 1, e.ManaRegeneration, rate, mana)
		e.setCurrentMana(m)
		if healthDue {
			hp := e.HP
			regenerate(&hp, e.MaxHP, e.HealthRegenPeriod, &e.HealthHundredths, 2, e.HealthRegeneration, rate, healthPart)
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
func regenerate(cur *int32, max, period int32, rest *uint8, double int64, modifier, rate int32, part regenPart) {
	if max <= 0 || *cur >= max || period <= 0 {
		return
	}
	product := int64(max) * double * int64(regenPercent+modifier)
	if rate > 1 && (product > math.MaxInt64/int64(rate) || product < math.MinInt64/int64(rate)) {
		regenerateWide(cur, max, period, rest, product, rate, part)
		return
	}
	base := int64(*cur)*regenScale + int64(*rest)
	gain, apply := part.step(product*int64(rate)/int64(period), base)
	if !apply {
		return
	}
	if (base > 0 && gain > math.MaxInt64-base) || (base < 0 && gain < math.MinInt64-base) {
		regenerateWide(cur, max, period, rest, product, rate, part)
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
func regenerateWide(cur *int32, maximum, period int32, rest *uint8, product int64, rate int32, part regenPart) {
	var acc, gain, q, rem big.Int
	gain.SetInt64(product)
	gain.Mul(&gain, big.NewInt(int64(rate)))
	gain.Quo(&gain, big.NewInt(int64(period)))
	base := int64(*cur)*regenScale + int64(*rest)
	negative := gain.Sign() < 0 || base < 0
	if !part.applies(negative) {
		return
	}
	if part.k != 0 && !negative {
		var r big.Int
		gain.QuoRem(&gain, big.NewInt(part.k), &r)
		gain.Add(&gain, big.NewInt(partShare(part, r.Int64())))
	}
	acc.SetInt64(base)
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
