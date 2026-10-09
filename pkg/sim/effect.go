package sim

import "sort"

// attachedEffect is one lasting effect on an actor. The list is canonical by
// target then spell; therefore a spell id has at most one record per actor.
type attachedEffect struct {
	Target    EntityID
	Caster    EntityID
	HasCaster bool
	Spell     uint16
	Kind      EffectKind
	Mode      EffectMode
	Magnitude int32
	Remaining uint16
}

// ActiveEffect is the read-only public shape used by presentation and tools.
type ActiveEffect struct {
	Target    EntityID
	Caster    EntityID
	HasCaster bool
	Spell     uint16
	Kind      EffectKind
	Mode      EffectMode
	Magnitude int32
	Remaining uint16
}

func (w *World) ActiveEffects() []ActiveEffect {
	out := make([]ActiveEffect, len(w.attached))
	for i, e := range w.attached {
		out[i] = ActiveEffect(e)
	}
	return out
}

func (w *World) setNativeAttachedEffectMask(target EntityID, spell uint16, present bool) {
	if spell >= 32 {
		return
	}
	i := indexOfEntity(w.entities, target)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 || !w.entities[i].NativeBasis.ScalarIsKnown(ScalarU144) {
		return
	}
	mask := &w.entities[i].NativeBasis.Scalars[ScalarU144]
	if present {
		*mask |= uint32(1) << spell
	} else {
		*mask &^= uint32(1) << spell
	}
}

// setAttachedEffectDuration is instant 30's canonical store
// (TRIG-EFFECTTIME-034). The original effect id is a byte, so both the stored
// id and the authored parameter are compared after byte narrowing. Every
// target-and-spell match is written; a missing match creates nothing.
func (w *World) setAttachedEffectDuration(target EntityID, spell uint8, duration uint16) {
	for i := range w.attached {
		e := &w.attached[i]
		if e.Target == target && uint8(e.Spell) == spell {
			e.Remaining = duration
		}
	}
}

// HasEffectSpell reports whether target currently carries the lasting effect of
// spell. It is the exported form of the same canonical lookup the simulation's
// own arms use, for a presentation tier that must ask about one spell rather
// than walk the whole set (1002).
func (w *World) HasEffectSpell(target EntityID, spell uint16) bool {
	return w.hasAttachedSpell(target, spell)
}

// stoneCursed is the central action gate for spell 20. The attached record is
// already canonical, hashed and persisted; movement, attack and cast paths all
// read this one lookup rather than carrying a second immobilisation flag that
// could outlive the effect.
func (w *World) stoneCursed(i int) bool {
	return i >= 0 && i < len(w.entities) && w.hasAttachedSpell(w.entities[i].ID, 20)
}

// InvisibleTo reports whether target is hidden from the participant in roster
// slot owner (1002; MAGIC-ACTOR-066).
//
// MAGIC-ACTOR-066 establishes that the unit draw gates the whole actor sprite
// on a per-player bit whenever the actor carries the `invisibility` kind, and
// skips the sprite when the bit is clear. THE BIT'S OWNING TABLE WAS NOT
// IDENTIFIED, so what decides it is authored here: the participant sees the
// actor when the actor is its own, or when one of its own actors stands within
// its own decoded SeeInvisible detector radius of it. That is the same test
// groupDetectsInvisible already applies for acquisition, so what a participant
// may see and what its units may attack cannot come apart. DIV-074.
//
// IT IS OBSERVATION ONLY. Nothing here is read by the simulation, nothing is
// stored, and nothing enters the byte form or the digest — ActiveEffects' own
// terms, one function up.
func (w *World) InvisibleTo(target EntityID, owner uint32) bool {
	ti := indexOfEntity(w.entities, target)
	if ti < 0 || !w.hasAttachedSpell(target, 15) {
		return false
	}
	at := cellOf(w.entities[ti])
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner != owner {
			continue
		}
		if e.ID == target {
			return false
		}
		if cellOf(e).chebyshevTo(at) <= int64(e.SeeInvisible) {
			return false
		}
	}
	return true
}

func effectIndex(a []attachedEffect, target EntityID, spell uint16) (int, bool) {
	i := sort.Search(len(a), func(i int) bool {
		return a[i].Target > target || a[i].Target == target && a[i].Spell >= spell
	})
	return i, i < len(a) && a[i].Target == target && a[i].Spell == spell
}

// minEffectSpeed is the least speed an effect may leave on an actor.
//
// IT IS AUTHORED, NOT DECODED. HERO-SPEED-008 establishes only that the
// effective speed reaches the mover as a byte at [actor+0x154]+0xa; what the
// original does when an arm drives that number below zero is not decoded, so
// this build states a floor rather than holding the arm. One is chosen and not
// zero because zero is the value rated (world.go) reads as "this actor has no
// rate at all", which puts an unrated mover on the cell-per-tick cadence — the
// FASTEST the engine has. Without the floor, Slow at power 60 on a speed-10
// actor produced a unit about 21x faster than the same actor unslowed.
const minEffectSpeed int32 = 1

// applyEffectDelta applies amount of kind to entity i.
//
// IT RETURNS WHAT ACTUALLY LANDED, and whether the kind is one it moves at
// all. A clamp means less than amount lands, and the caller stores what landed
// so that the expiry — which subtracts the stored number — is the exact
// inverse of the apply. Before that, an effect that clamped on the way in
// subtracted its whole nominal magnitude on the way out and permanently
// rewrote the actor's base value: a power-45 Protection effect on a base of 80
// clamped to 100 and expired to 70.
//
// The second result is false for a kind this function does not move. Bless,
// Curse and Invisibility carry a magnitude that is a parameter other code
// reads (a probability, MAGIC-SING-019 (e)) rather than a state delta, and
// their stored magnitude must not be replaced by what this function did.
//
// ONE CLAMP PER CALL IS NOT ONE CLAMP PER ACTOR. A single call landing
// exactly what it stores is correct only in isolation; the moment a SECOND
// effect of the same kind stands on the same target, a clamp engaged by
// their COMBINED total can strand one record's own reversal short of what it
// promised (round3-review.md R3-A1: three EffectSpeed records, removed in
// duration order, left a base-10 actor at 15). removeAttachedAt's own caller
// covers that case by calling redistributeShortfall on whatever this
// function could not deliver — this function stays a pure single-record
// apply, and does not call it itself, so that its return value keeps meaning
// exactly what landed FOR THIS CALL, which is what both the direct caller
// here and the redistribution math need it to mean.
func (w *World) applyEffectDelta(i int, kind EffectKind, amount int32) (int32, bool) {
	if i < 0 || i >= len(w.entities) {
		return 0, false
	}
	if w.entities[i].ActorLoad.Source.Class != 0 {
		return w.sourceEffect(i, kind, amount)
	}
	before := w.entities[i]
	next, landed, moves := effectLanding(before, kind, amount)
	if !moves {
		return 0, false
	}
	nativeModifierEffectDelta(&next, kind, landed)
	w.entities[i] = next
	if kind != EffectHealth || amount <= 0 {
		w.reportHealthLoss(before, i)
	}
	return landed, true
}

// effectLanding is applyEffectDelta's own arithmetic without the write: the
// actor amount of kind would produce, what landed on it, and whether kind is
// one that moves a field at all. The apply above is this function plus one
// assignment, so the two cannot come to disagree about a clamp.
//
// IT EXISTS BECAUSE THE AUTOCAST USEFULNESS GATE MUST ASK THE QUESTION WITHOUT
// ANSWERING IT (autoCastImproves, spell.go). A buff whose whole magnitude is
// already clamped away lands nothing, and a gate comparing nominal magnitudes
// would call that cast an improvement and recast it whenever the wait expired,
// which is the abuse the gate was written against in a second form.
//
// IT TAKES AND RETURNS e BY VALUE. Nothing here writes through a pointer, and
// that is what makes the gate's speculative call safe on a live actor.
func effectLanding(e Entity, kind EffectKind, amount int32) (Entity, int32, bool) {
	switch kind {
	case EffectHealthRegeneration:
		before := e.HealthRegeneration
		e.HealthRegeneration = potionPool(before, 1<<31-1, amount)
		return e, e.HealthRegeneration - before, true
	case EffectManaRegeneration:
		if !isMage(e) {
			return e, 0, true
		}
		before := e.ManaRegeneration
		e.ManaRegeneration = potionPool(before, 1<<31-1, amount)
		return e, e.ManaRegeneration - before, true
	case EffectHealth:
		// A positive ordinary effect is not a resurrection mechanic. It may
		// have been attached while the actor was alive and tick, or reverse a
		// temporary penalty, after a later blow felled it. Refuse that positive
		// move here as the last health-effect gate so neither path can create a
		// positive-HP entity that still carries corpse decay state.
		if amount > 0 && !e.Alive() {
			return e, 0, true
		}
		before := e.HP
		e.setCurrentHealth(e.HP + amount)
		if e.HP > e.MaxHP {
			e.setCurrentHealth(e.MaxHP)
		}
		return e, e.HP - before, true
	case EffectSpeed:
		e.HumanMovement = HumanMovement{}
		before := e.Speed
		v := e.Speed + amount
		if v < minEffectSpeed {
			v = minEffectSpeed
		}
		e.Speed = v
		if e.Humanoid {
			e.refreshHumanTurnRate()
		}
		return e, e.Speed - before, true
	case EffectScanRange:
		before := int32(e.ScanRange)
		v := before + amount
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		e.ScanRange = uint8(v)
		return e, v - before, true
	case EffectAbsorption:
		e.Absorption += amount
		return e, amount, true
	case EffectProtectionFire, EffectProtectionWater, EffectProtectionAir, EffectProtectionEarth:
		k := int(kind - EffectProtectionFire)
		before := e.Protection[k]
		v := before + amount
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		e.Protection[k] = v
		return e, v - before, true
	}
	return e, 0, false
}

func (w *World) removeAttachedAt(i int) bool {
	e := w.attached[i]
	if ti := indexOfEntity(w.entities, e.Target); ti >= 0 && w.entities[ti].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(ti)
		if e.Mode&EffectContinuous == 0 {
			if _, ok := n.sourceEffect(ti, e.Kind, -e.Magnitude); !ok {
				return false
			}
		}
		n.attached = append(n.attached[:i], n.attached[i+1:]...)
		*w = n
		w.clearFelled(ti)
		return true
	}
	if ti := indexOfEntity(w.entities, e.Target); ti >= 0 && e.Mode&EffectContinuous == 0 {
		landed, moved := w.applyEffectDelta(ti, e.Kind, -e.Magnitude)
		// A SECOND clamp, engaged by whatever ELSE is still active on the
		// target, can deliver less reversal than this record's own landing
		// promised — R3-A1: three speed effects standing together, removed
		// in duration order, left a base-10 actor at 15 because the discarded
		// remainder of the first removal was never accounted for. What this
		// call could not deliver is redistributed onto a still-active sibling
		// of the same kind (see redistributeShortfall), EXCEPT for Health:
		// its clamp is the MaxHP ceiling, reachable by ordinary healing that
		// has nothing to do with any effect, so there is no effect-free base
		// to redistribute against the way Speed, ScanRange and Protection
		// each have one.
		if moved && e.Kind != EffectHealth {
			if shortfall := landed - (-e.Magnitude); shortfall != 0 {
				w.redistributeShortfall(e.Target, e.Kind, i, shortfall)
			}
		}
		// A reversal can fell its target: a duration-mode health effect that
		// raised health gives it back on expiry. The residue an actor leaves
		// when it stops being alive is clearFelled's one rule (step.go), and
		// it is called on every other path that can drive health down —
		// attachEffect below, the continuous tick, and every blow. Without it
		// the byte form refuses the world it produces, on binary.go's own
		// "the five refusals are the states a tick cannot produce".
		w.clearFelled(ti)
	}
	w.attached = append(w.attached[:i], w.attached[i+1:]...)
	w.setNativeAttachedEffectMask(e.Target, e.Spell, false)
	return true
}

// redistributeShortfall keeps a derived value's invariant intact when a clamp
// truncates what one write delivers: the entity's current value must stay
// exactly reconstructible as an effect-free base plus the sum of every active
// effect's own stored magnitude, so the NEXT writer of the same kind — an
// expiry, an attach, SetDerived's fresh base, SetCombat's protection loop —
// still sees the correct total rather than one degraded by a clamp two
// writers back.
//
// shortfall is delivered minus intended: positive when the clamp gave back
// MORE than the record on its own called for (a floor bit under a
// still-negative remainder), negative when it gave back LESS (a ceiling). It
// is folded onto ONE still-active record of the same kind and target — the
// first in canonical (target, spell) order, skipping index skip (the record
// whose own operation produced the shortfall; pass -1 when none applies, as
// SetDerived and SetCombat's protection loop do, since neither is removing a
// record). No sibling to fold onto (kind not active at all, or this is the
// last one and the base itself is why the clamp fired) leaves nothing to
// correct, and the call is a no-op.
//
// THIS IS A CHOICE, NOT THE ONLY CORRECT ONE, and it has a KNOWN COST:
// SetDerived's own doc says active spell deltas are "re-applied to the fresh
// base," which reads naturally as the RULE's own nominal magnitude, and a
// record that has absorbed another record's shortfall no longer holds that
// nominal number — a level-up recompute that lands while a redistributed
// sibling is still active combines the fresh base with an adjusted, not
// nominal, total. The alternative — leave every record's stored magnitude
// alone and let a clamped removal simply discard what it could not deliver —
// is what R3-A1 measured: a base-10 actor left at 15 with no record of the
// affected actor's own further recompute at all. Reversal exactness for the
// SAME base is the invariant spec.md states and round3-review.md measured;
// exactness of a LATER, DIFFERENT base's recompute against a SIBLING this
// clamp had to touch is the narrower, second-order case this trades away.
// DIV-053 records it.
func (w *World) redistributeShortfall(target EntityID, kind EffectKind, skip int, shortfall int32) {
	if shortfall == 0 {
		return
	}
	for i := range w.attached {
		if i == skip {
			continue
		}
		e := &w.attached[i]
		if e.Target == target && e.Kind == kind && e.Mode&EffectContinuous == 0 {
			e.Magnitude += shortfall
			return
		}
	}
}

func (w *World) removeAttachedSpell(target EntityID, spell uint16) bool {
	i, ok := effectIndex(w.attached, target, spell)
	if !ok {
		return false
	}
	return w.removeAttachedAt(i)
}

func (w *World) attachedMagnitude(target EntityID, spell uint16) (int32, bool) {
	i, ok := effectIndex(w.attached, target, spell)
	if !ok {
		return 0, false
	}
	return w.attached[i].Magnitude, true
}

func (w *World) hasAttachedSpell(target EntityID, spell uint16) bool {
	_, ok := effectIndex(w.attached, target, spell)
	return ok
}

func (w *World) effectDelta(target EntityID, kind EffectKind) int32 {
	var out int32
	for _, e := range w.attached {
		if e.Target == target && e.Kind == kind && e.Mode&EffectContinuous == 0 {
			out += e.Magnitude
		}
	}
	return out
}

// attachEffect applies, refreshes, or replaces one effect using the decoded
// same-id and Bless/Curse annihilation rules.
func (w *World) attachEffect(target, caster EntityID, rule SpellRule, kind EffectKind, magnitude int32, duration uint16, mode EffectMode) bool {
	ti := indexOfEntity(w.entities, target)
	if ti < 0 || duration == 0 || kind == EffectNone {
		return false
	}
	poison := rule.ID == 8 && mode&EffectContinuous != 0
	if w.entities[ti].ActorLoad.Source.Class != 0 && !poison {
		return w.attachSourceEffect(ti, caster, rule, kind, magnitude, duration, mode)
	}
	if rule.ID == 23 || rule.ID == 27 {
		opposite := uint16(23)
		if rule.ID == 23 {
			opposite = 27
		}
		if oi, ok := effectIndex(w.attached, target, opposite); ok {
			w.removeAttachedAt(oi)
			return true
		}
	}
	i, exists := effectIndex(w.attached, target, rule.ID)
	if exists {
		if mode&EffectContinuous != 0 {
			w.attached[i].Remaining = duration
			w.setNativeAttachedEffectMask(target, rule.ID, true)
			return true
		}
		w.removeAttachedAt(i)
		i, _ = effectIndex(w.attached, target, rule.ID)
	}
	e := attachedEffect{Target: target, Caster: caster, HasCaster: indexOfEntity(w.entities, caster) >= 0, Spell: rule.ID, Kind: kind, Mode: mode, Magnitude: magnitude, Remaining: duration}
	if !e.HasCaster {
		// An absent source is not an entity reference. Keep the same canonical
		// zero payload used by imported attachments and detached casters.
		e.Caster = 0
	}
	w.attached = append(w.attached, attachedEffect{})
	copy(w.attached[i+1:], w.attached[i:])
	w.attached[i] = e
	w.setNativeAttachedEffectMask(target, rule.ID, true)
	// Every effect is applied at attachment time. Continuous means the same
	// ordinary apply is repeated each eighth tick; it does not postpone the
	// first application until the countdown happens to reach a multiple of 8.
	//
	// THE STORED MAGNITUDE BECOMES WHAT LANDED, for every kind whose magnitude
	// IS a state delta and every mode whose expiry reverses it. That is what
	// makes apply and expire exact inverses under the clamps. A CONTINUOUS
	// effect is excluded because its stored magnitude is re-applied on every
	// eighth tick rather than reversed: a continuous heal that clamped at full
	// health once would otherwise be re-applied as zero for the rest of its
	// life.
	if poison {
		w.applyPoisonAttachment(i, ti)
	} else if applied, moved := w.applyEffectDelta(ti, kind, magnitude); moved && mode&EffectContinuous == 0 {
		w.attached[i].Magnitude = applied
	}
	w.clearFelled(ti)
	w.markSpellEffect(ti, rule.ID)
	w.entities[ti].SpellFX = duration
	return true
}

func (w *World) awardPoisonTick(effectIndex, targetIndex int, damage int64) {
	e := &w.attached[effectIndex]
	if !e.HasCaster || damage == 0 {
		return
	}
	ci := indexOfEntity(w.entities, e.Caster)
	if ci < 0 || w.entities[ci].HP < 0 {
		e.HasCaster = false
		e.Caster = 0
		return
	}
	rule, ok := w.findSpell(uint32(e.Spell))
	if !ok {
		return
	}
	w.flipOnBlow(ci, targetIndex)
	w.awardSpellDamage(ci, targetIndex, rule, damage)
}

func (w *World) stepAttachedEffects() {
	for i := 0; i < len(w.attached); {
		e := &w.attached[i]
		if m := w.motionFor(e.Target); m != nil && m.Current && m.ActorAction == 0x10 {
			i++
			continue
		}
		if e.Spell == 0 {
			if ti := indexOfEntity(w.entities, e.Target); ti < 0 || !w.entities[ti].OrdinaryTargetable() {
				i++
				continue
			}
		}
		if e.Remaining > 9600 || e.Mode&(EffectDuration|EffectContinuous) == 0 {
			i++
			continue
		}
		beforeRemaining := e.Remaining
		if e.Mode&EffectContinuous != 0 && beforeRemaining > 0 && beforeRemaining%8 == 0 {
			if ti := indexOfEntity(w.entities, e.Target); ti >= 0 {
				if e.Spell == 8 {
					w.applyPoisonAttachment(i, ti)
				} else if _, ok := w.applyEffectDelta(ti, e.Kind, e.Magnitude); !ok && w.entities[ti].ActorLoad.Source.Class != 0 {
					i++
					continue
				}
				e = &w.attached[i]
				w.clearFelled(ti)
			}
		}
		if e.Remaining > 0 {
			e.Remaining--
		}
		if e.Remaining == 0 {
			if !w.removeAttachedAt(i) {
				w.attached[i].Remaining = beforeRemaining
				i++
			}
			continue
		}
		i++
	}
	// A removed newer mark must reveal any older lasting effect still attached
	// to the actor. Instant feedback keeps its short mark until it expires; an
	// actor with no such mark takes the first canonical attached record and its
	// real remaining duration.
	for _, e := range w.attached {
		if ti := indexOfEntity(w.entities, e.Target); ti >= 0 && w.entities[ti].SpellFX == 0 {
			w.entities[ti].SpellFX = e.Remaining
			w.entities[ti].SpellFXSpell = uint8(e.Spell)
		}
	}
}
