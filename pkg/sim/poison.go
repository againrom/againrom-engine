package sim

// ANIM-074: Token8 uses its own direct HP writer. The signed water protection
// is not clamped, and zero protection bypasses the rounding conversion.
func poisonDamage(magnitude, protection int32) int64 {
	damage := -int64(magnitude)
	if protection != 0 {
		damage = (damage*(100-int64(protection)) + 50) / 100
	}
	return damage
}

// applyPoisonAttachment puts one poison tick on the target. A unit that has
// died takes no harm and shows no number; it reports false and the caller
// ends the record. Signed healing poison still reaches a fallen unit.
func (w *World) applyPoisonAttachment(index, target int) bool {
	damage := poisonDamage(w.attached[index].Magnitude, w.entities[target].Protection[1])
	if damage > 0 && w.entities[target].Dead() {
		return false
	}
	if damage == 0 {
		return true
	}
	// This branch does not invoke the potion/derive dispatcher or cap HP at
	// MaxHP. The ordinary presentation delta sees no hit for a computed zero.
	before := w.entities[target].HP
	previous := w.entities[target]
	w.entities[target].setCurrentHealth(w.entities[target].HP - int32(damage))
	w.reportHealthMessage(previous, target)
	w.awardPoisonTick(index, target, damage)
	// Native zero-crossing policy is shared with Heal/property writes: signed
	// healing can make this still-present actor living again. Clear its death
	// state once; do not reconstruct prior actions, group membership or loot.
	// This is not a claim about original Poison revival scheduling.
	w.restoreAfterHealthGain(target, before)
	w.clearFelled(target)
	return true
}
