package sim

// CombatBlock is the complete combat part of a live recompute. It carries
// the damage, hit, defence, absorption, cadence, always-hit, reach, experience
// slot and weapon-spell values. It also carries the two five-value families,
// elemental Protection and weapon-kind Resistance, and one secondary-damage
// triple. Every member describes the same actor under the same loadout and
// crosses the determinism wall together.
//
// XPSlot has two consumers: it selects the fighter skill paid by a blow and,
// for values 1..5, the target Resistance byte that reduces that blow (1039).
// Protection is the recomputed base onto which SetCombat reapplies live effects;
// Resistance is already narrowed to its five canonical bytes by the caller.
// WeaponSpell and WeaponSpellLevel keep the existing attached-cast route.
//
// Every member uses Entity's own name and width: signed combat and Protection
// values stay int32, Reach and XPSlot stay uint8, WeaponSpell stays uint16, and
// each Resistance value stays uint8. SetCombat therefore needs no width-changing
// conversion at the storage boundary.
type CombatBlock struct {
	DamageBase, DamageSpread   int32
	SecondBase, SecondSpread   uint8
	ToHit, Defence, Absorption int32
	AttackCharge, AttackRelax  int32
	AlwaysHits                 bool
	Reach                      uint8
	XPSlot                     uint8
	WeaponSpell                uint16
	WeaponSpellLevel           int32
	WeaponSpellSource          WeaponSpellSource
	Protection                 [5]int32
	Resistance                 [5]uint8
	SecondaryDamage            SecondaryDamage
}

// DerivedBlock is the complete live sheet that pkg/data owns and pkg/sim
// consumes. It crosses the determinism wall as finished integers: the caller
// computes once at the fixed post-award point, and this world stores the result
// before the next decision, hash, save, or UI projection can read it.
type DerivedBlock struct {
	Combat                 CombatBlock
	MaxHP, MaxMana         int32
	Speed                  int32
	ScanRange              uint8
	Reaction, Mind, Spirit int32
	// Capacity is the actor's carrying capacity, `Body x 10 + 1`
	// (HERO-SIGHT-007, High). It rides on this block for Speed's own reason: it
	// comes off the SAME recompute over the SAME loadout, it is read by the
	// same law -- the overload penalty compares the load against it and nothing
	// else does -- and a door that carried a re-armed actor's new speed and not
	// his new capacity would leave the penalty measuring against the Body he
	// had before.
	Capacity                                            int32
	HealthRegeneration, ManaRegeneration, RotationSpeed int32
	// Skill is the six effective levels of the same recompute: the trained
	// base plus the bonus of the worn items. SkillSet says the block carries
	// them; a block without them leaves the entity's levels as they are.
	Skill       [skillSlots]int32
	SkillSet    bool
	Body        int32
	BodyPresent bool
}

// SetDerived replaces one actor's complete derived sheet without refilling its
// pools. Current health and mana are preserved when a maximum rises and clamped
// only when it falls. Active spell deltas are re-applied to the fresh base so a
// level-up cannot erase Shield, Haste, Slow, Stone Curse, or protections.
func (w *World) SetDerived(id EntityID, d DerivedBlock) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || d.Combat.XPSlot >= skillSlots || secondaryDamageFault(d.Combat.SecondaryDamage) != nil {
		return false
	}
	// Source-backed actors use their full live/base/modifier producer. A
	// native sheet cannot replace that basis through a cached roster path.
	if w.entities[i].ActorLoad.Source.Class != 0 {
		return false
	}
	e := &w.entities[i]
	if d.BodyPresent {
		e.NativeBasis = e.NativeBasis.WithBody(uint16(d.Body))
	}
	priorSpeed := w.effectDelta(id, EffectSpeed)
	e.MaxHP, e.MaxMana = d.MaxHP, d.MaxMana
	if e.HP > e.MaxHP {
		e.HP = e.MaxHP
	}
	if e.Mana > e.MaxMana {
		e.Mana = e.MaxMana
	}
	// Reconcile attached magnitudes when the fresh base cannot sustain them.
	// The raw modifier changes by that history delta, not by a sheet inverse.
	speed := d.Speed + w.effectDelta(id, EffectSpeed)
	if speed < minEffectSpeed {
		w.redistributeShortfall(id, EffectSpeed, -1, minEffectSpeed-speed)
		speed = minEffectSpeed
	}
	e.Speed = speed
	nativeModifierEffectDelta(e, EffectSpeed, w.effectDelta(id, EffectSpeed)-priorSpeed)
	e.HumanMovement = HumanMovement{}
	e.Reaction, e.Mind, e.Spirit = d.Reaction, d.Mind, d.Spirit
	e.HealthRegeneration = d.HealthRegeneration + w.effectDelta(id, EffectHealthRegeneration)
	e.ManaRegeneration = d.ManaRegeneration + w.effectDelta(id, EffectManaRegeneration)
	nativeModifierEffectDelta(e, EffectHealthRegeneration, 0)
	nativeModifierEffectDelta(e, EffectManaRegeneration, 0)
	e.RotationSpeed = d.RotationSpeed
	if e.RotationSpeed <= 0 && e.Turning() {
		e.Facing = e.DesiredFacing
		e.clearTurn()
	}
	// The capacity is assigned flat: no effect delta touches it, no clamp
	// applies, and unlike Speed it is not a base something else adds to.
	// The LOAD is not touched here at all -- it is a function of what the
	// actor holds, recomputeLoad (weight.go) is its only writer, and a
	// re-arm moves neither container.
	e.Capacity = d.Capacity
	if d.SkillSet {
		e.Skill = d.Skill
	}
	if e.ActorLoad.Present {
		e.Load = e.ActorLoad.CurrentLoad()
	}
	sight := int32(d.ScanRange) + w.effectDelta(id, EffectScanRange)
	if sight < 0 {
		sight = 0
	} else if sight > 255 {
		sight = 255
	}
	e.ScanRange = uint8(sight)
	combat := d.Combat
	// A body that already took its death transition stores half of the current
	// derived defence. A post-equipment recompute must not restore the full value.
	// A newly felled actor still receives the ordinary one-time shift below.
	if !e.Alive() && e.Decay != DecayNone {
		combat.Defence >>= 1
	}
	if !w.SetCombat(id, combat) {
		return false
	}
	w.clearFelled(i)
	RefreshBook(w.rules, e, w.spells)
	w.refreshSavedBookRoots(i)
	return true
}

// SetCombat validates the complete block before writing it. Attached effects
// retain their magnitudes, with shortfall reconciliation at protection clamps.
// Equipment callers own when the recompute runs; this stores its current result.
func (w *World) SetCombat(id EntityID, c CombatBlock) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return false
	}
	// The entity this call would leave behind, asked before it is committed —
	// a COPY, so a refusal has written nothing to the world's own storage.
	next := w.entities[i]
	next.XPSlot = c.XPSlot
	next.WeaponSpell, next.WeaponSpellLevel = c.WeaponSpell, c.WeaponSpellLevel
	next.WeaponSpellSource = c.WeaponSpellSource
	next.SecondaryDamage = c.SecondaryDamage
	weapon := w.equipment[i][slotWeapon]
	if spell, power, ok := weapon.CastSpell(); ok {
		next.WeaponSpell, next.WeaponSpellLevel = spell, power
		next.WeaponSpellSource = WeaponSpellItem
	} else if weaponPairPresent(next) && next.WeaponSpellSource == WeaponSpellNone {
		next.WeaponSpellSource = WeaponSpellLegacy
	}
	if experienceFault(next) != nil {
		return false
	}
	if secondaryDamageFault(next.SecondaryDamage) != nil {
		return false
	}
	if err := validateWeaponSource(next, weapon); err != nil {
		return false
	}
	e := &w.entities[i]
	e.DamageBase, e.DamageSpread = c.DamageBase, c.DamageSpread
	e.retireCurrentProfile()
	e.SecondBase, e.SecondSpread = c.SecondBase, c.SecondSpread
	e.ToHit, e.Defence, e.Absorption = c.ToHit, c.Defence, c.Absorption+w.effectDelta(id, EffectAbsorption)
	// THE PROTECTION SLOTS ARE FIVE AND THE PROTECTION EFFECT KINDS ARE FOUR.
	// Protection[4] is Astral (pkg/ui/panel.go's own row order) and this
	// vocabulary has no EffectProtectionAstral: EffectProtectionFire + 4 is
	// EffectBless, whose magnitude is a probability in [20,100]
	// (MAGIC-SING-019 (e)) and not a protection at all. Adding the kind
	// constants without that bound wrote a blessed actor's Bless magnitude
	// into its Astral protection on every recompute, where it was hashed,
	// serialized and displayed, and where Bless's own removal could not
	// reach it.
	// round3-review.md's minor findings name this loop the same family as
	// R3-A1: a Protection effect that clamped at attach, then a level rise
	// that reaches this recompute while it is still active, then its own
	// expiry, could leave the actor below its base — this loop's own clamp
	// silently discarded what an active effect's stored magnitude assumed
	// still had headroom. hasEffect gates the redistribution exactly where
	// the doc above already gates the effectDelta read: Astral (k=4) has no
	// EffectKind at all, so there is no active-effect sibling to correct
	// against and none is attempted.
	for k := range e.Protection {
		prior := w.effectDelta(id, EffectProtectionFire+EffectKind(k))
		e.Protection[k] = c.Protection[k]
		hasEffect := EffectKind(k) <= EffectProtectionEarth-EffectProtectionFire
		if hasEffect {
			e.Protection[k] += w.effectDelta(id, EffectProtectionFire+EffectKind(k))
		}
		raw := e.Protection[k]
		if e.Protection[k] < 0 {
			e.Protection[k] = 0
		}
		if e.Protection[k] > 100 {
			e.Protection[k] = 100
		}
		if hasEffect {
			if shortfall := e.Protection[k] - raw; shortfall != 0 {
				w.redistributeShortfall(id, EffectProtectionFire+EffectKind(k), -1, shortfall)
			}
			nativeModifierEffectDelta(e, EffectProtectionFire+EffectKind(k), w.effectDelta(id, EffectProtectionFire+EffectKind(k))-prior)
		}
	}
	// Weapon-kind resistances have no timed effect delta and no [0,100]
	// clamp. Their byte already embodies the original's modulo-256 store.
	e.Resistance = c.Resistance
	if e.Humanoid && e.ActorLoad.Source.Class == 0 {
		e.NativeBasis.DefenceKnown &^= uint32(1) << 16
		if e.NativeBasis.ModifierByteKnown(58) {
			e.NativeBasis.DefencePresent = true
			e.NativeBasis.DefenceKnown |= uint32(1) << 16
			e.NativeBasis.Defence[16] = e.NativeBasis.Modifier[58]
		}
	}
	e.SecondaryDamage = c.SecondaryDamage
	e.AttackCharge, e.AttackRelax = c.AttackCharge, c.AttackRelax
	e.AlwaysHits = c.AlwaysHits
	e.Reach = c.Reach
	e.XPSlot = c.XPSlot
	e.WeaponSpell, e.WeaponSpellLevel = next.WeaponSpell, next.WeaponSpellLevel
	e.WeaponSpellSource = next.WeaponSpellSource
	return true
}
