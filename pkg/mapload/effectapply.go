package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// ApplyItemEffects folds every equipped instance in slot order and every
// effect in stored order into one effect-free loadout. fighter is the class
// gate used by the two skill families and the mana-only arms.
func ApplyItemEffects(loadout *data.Loadout, equipped [sim.EquipSlots]sim.ItemInstance, fighter bool) {
	if loadout == nil {
		return
	}
	for _, item := range equipped {
		for _, effect := range item.Effects {
			scalar := itemEffectScalar(effect)
			switch effect.Kind {
			case 2:
				loadout.Mod.Body += scalar
			case 3:
				loadout.Mod.Mind += scalar
			case 4:
				loadout.Mod.Reaction += scalar
			case 5:
				loadout.Mod.Spirit += scalar
			case 7:
				loadout.Mod.HealthMax += scalar
			case 8:
				loadout.Mod.HealthRegeneration += scalar
			case 10:
				if !fighter {
					loadout.Mod.ManaMax += scalar
				}
			case 11:
				if !fighter {
					loadout.Mod.ManaRegeneration += scalar
				}
			case 12:
				loadout.Mod.ToHit += scalar
			case 13:
				loadout.Mod.DamageBase += scalar
			case 14:
				loadout.Mod.DamageSpread += scalar
			case 15:
				loadout.Mod.Defence += scalar
			case 16:
				loadout.Mod.Absorption += scalar
			case 17:
				loadout.Mod.Speed += scalar
			case 18:
				loadout.Mod.RotationSpeed += scalar
			case 19:
				loadout.Mod.Sight += scalar
			case 20:
				// The current combat model has no physical Protection slot;
				// this kind is outside shipped and generated populations.
			case 21, 22, 23, 24, 25:
				loadout.Mod.Protection[effect.Kind-21] += scalar
			case 26:
				if fighter {
					loadout.Mod.SkillBonus[data.SkillGeneral] += scalar
				}
			case 27, 28, 29, 30, 31:
				if fighter {
					loadout.Mod.SkillBonus[effect.Kind-26] += scalar
				}
			case 32:
				if !fighter {
					loadout.Mod.SkillBonus[data.SkillGeneral] += scalar
				}
			case 33, 34, 35, 36, 37:
				if !fighter {
					loadout.Mod.SkillBonus[effect.Kind-32] += scalar
				}
			case 43:
				// The authored range stays packed in Operand. The original
				// dispatcher sends that complete state-0 scalar through the
				// same damage-base arm as kinds 13 and 49; it does not install
				// a base/spread pair as the five elemental arms below do.
				loadout.Mod.DamageBase += scalar
			case 44, 45, 46, 47, 48:
				loadout.Mod.SecondaryDamage = data.SecondaryDamage{
					Base: uint8(effect.Operand), Spread: uint8(effect.Operand >> 8),
					Selector: effect.Kind - 44,
				}
				loadout.Mod.HasSecondaryDamage = true
			case 49:
				loadout.Mod.DamageBase += scalar
			}
		}
	}
}

// EquippedSkillBonus is the per-slot skill bonus the worn items give a member
// of the given class.
func EquippedSkillBonus(equipped [sim.EquipSlots]sim.ItemInstance, fighter bool) [data.SkillSlots]int32 {
	var loadout data.Loadout
	ApplyItemEffects(&loadout, equipped, fighter)
	return loadout.Mod.SkillBonus
}

// TrainedSkills is the trained base inside a hero's effective levels: the
// effective levels less the bonus included in them. A trained level is never
// negative; slot 0 is not floored, as the restore does not floor it.
func TrainedSkills(effective, bonus [data.SkillSlots]int32) [data.SkillSlots]int32 {
	base := effective
	for i := range base {
		base[i] -= bonus[i]
		if i != int(data.SkillGeneral) && base[i] < data.SkillFloor {
			base[i] = data.SkillFloor
		}
	}
	return base
}

func itemEffectScalar(effect sim.ItemEffect) int32 {
	if effect.Mode == 0 || effect.Mode == 8 {
		return int32(effect.Operand)
	}
	return int32(int16(effect.Operand))
}

func simSecondaryDamage(d data.SecondaryDamage) sim.SecondaryDamage {
	return sim.SecondaryDamage{Base: d.Base, Spread: d.Spread, Selector: d.Selector}
}

// EquippedPoolEffects are state-0 effects whose act is not part of the
// effect-free data recompute.
func EquippedPoolEffects(equipped [sim.EquipSlots]sim.ItemInstance) (health, mana int32, taught uint32) {
	for _, item := range equipped {
		for _, effect := range item.Effects {
			switch effect.Kind {
			case 6:
				health += itemEffectScalar(effect)
			case 9:
				mana += itemEffectScalar(effect)
			case 42:
				id := uint16(effect.Operand)
				if id < 32 {
					taught |= uint32(1) << id
				}
			}
		}
	}
	return health, mana, taught
}
