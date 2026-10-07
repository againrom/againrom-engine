package data

// FoldWeapon is the equip step's own arithmetic: what a bearer's combat
// block becomes once a weapon is placed in his hand, on top of what he
// already has.
//
// THE FOLD HAS TWO ARMS, ONE INSIDE THE OTHER. Every weapon — melee or
// ranged — ASSIGNS its bearer's reach from its own range, assigns each
// half of the cadence pair from its own half when that half is not the
// format's empty cell, and ASSIGNS its own spell pair whole; only a MELEE
// weapon then goes on to ADD its physical damage pair, its own to-hit and
// its defence onto the bearer's own. Those four additions are additions,
// never a replacement: a bearer already carrying a nonzero pair, from his
// own row or an earlier fold, keeps every bit of it and the weapon's numbers
// land on top of it.
//
// THE SPELL PAIR IS AN ASSIGNMENT, NOT A FOURTH ADDITION, and it runs on
// BOTH arms, above the `if !w.Ranged()` branch beside Reach and the cadence
// pair — a ranged weapon carrying a spell must carry it, and two weapons
// folded in sequence cannot sum their spells the way two damage pairs do
// (plan D-3). A weapon carrying no spell therefore CLEARS whatever spell an
// earlier fold left standing, the same way an empty cadence cell does NOT
// clear the cadence: SpellName and SpellPower have no format-empty sentinel
// of their own to test, so every fold states the bearer's spell fresh,
// whole, from this weapon alone.
//
// A RANGED WEAPON USES TWO DIFFERENT SOURCES. Its accuracy contribution is
// the bearer's General level supplied by the caller: Weapon::Equip copies that
// word into the to-hit modifier, and the ordinary modifier fold adds it to the
// live to-hit. Its own damage does not join the physical pair. Exact attack
// types 11 and 12 add the weapon's byte pair to SecondaryDamage and assign the
// encoded kind 1 or 2. The resolver's published permutation makes those Fire
// and Earth respectively, represented here by Protection indices 0 and 3.
// Other ranged attack types still take General accuracy but write no weapon
// damage triple. Every ranged arm contributes no Defence and clears SkillSlot
// to General, so the credited slot stays 0.
//
// w == nil returns c UNCHANGED: a bare bearer folds nothing, which is a
// state the caller already holds and not a case this function has to name.
func FoldWeapon(c Combat, w *Weapon, general int32) Combat {
	if w == nil {
		return c
	}

	c.Reach = w.Range
	c.AttackChargeTime = cellOr(w.ChargeTime, c.AttackChargeTime)
	c.AttackRelaxTime = cellOr(w.RelaxTime, c.AttackRelaxTime)
	c.SpellName = w.SpellName
	c.SpellPower = w.SpellPower
	// The active slot is an assignment on both arms. A supported melee kind
	// selects its own slot; the ranged arm clears it to General. UnitDef has no
	// recompute before this fold, so this write is also what lets a creature's
	// equipped weapon name the resistance byte its blows consult.
	c.SkillSlot = activeSkill(w)

	if w.Ranged() {
		// HERO-GENERAL-090: the ranged arm copies General, not the weapon's
		// own ToHit and not three times a class-specific skill.
		c.ToHit += general
		switch w.AttackType {
		case 0xb:
			c.SecondaryDamage = foldRangedDamage(c.SecondaryDamage, *w, 0)
		case 0xc:
			c.SecondaryDamage = foldRangedDamage(c.SecondaryDamage, *w, 3)
		}
		return c
	}

	// Only the melee arm: its four ordinary additions. Any SecondaryDamage
	// already present stands unchanged; Recompute applies the ordered item-
	// effect replacement after this fold.
	c.DamageBase += w.DamageBase
	c.DamageSpread += w.DamageSpread
	c.ToHit += w.ToHit
	c.Defence += w.Defence
	return c
}

// foldRangedDamage reproduces Weapon::Equip's two byte ADDs and one selector
// assignment. uint8 addition deliberately wraps at the field width.
func foldRangedDamage(d SecondaryDamage, w Weapon, selector uint8) SecondaryDamage {
	d.Base += uint8(w.DamageBase)
	d.Spread += uint8(w.DamageSpread)
	d.Selector = selector
	return d
}
