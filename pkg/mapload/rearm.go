package mapload

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func DerivedBlock(d data.Derived, spellID uint16, source sim.WeaponSpellSource) sim.DerivedBlock {
	if spellID == 0 && d.Combat.SpellPower == 0 {
		source = sim.WeaponSpellNone
	}
	return sim.DerivedBlock{
		MaxHP: d.HealthMax, MaxMana: d.ManaMax, Speed: d.Speed, SpeedModifier: d.SpeedModifier, ScanRange: derivedByte(d.Sight),
		Reaction: d.Reaction, Mind: d.Mind, Spirit: d.Spirit, Capacity: d.Capacity,
		Skill: d.Skill, SkillSet: true,
		Body: d.Body, BodyPresent: true,
		HealthRegeneration: d.HealthRegeneration, ManaRegeneration: d.ManaRegeneration,
		RotationSpeed: d.RotationSpeed,
		Combat: sim.CombatBlock{
			DamageBase: d.Combat.DamageBase, DamageSpread: d.Combat.DamageSpread,
			SecondBase: d.Combat.SecondBase, SecondSpread: d.Combat.SecondSpread,
			ToHit: d.Combat.ToHit, Defence: d.Combat.Defence, Absorption: d.Combat.Absorption,
			AttackCharge: d.Combat.AttackChargeTime, AttackRelax: d.Combat.AttackRelaxTime,
			AlwaysHits: d.Combat.AlwaysHits, Reach: derivedByte(d.Combat.Reach),
			XPSlot: uint8(CreditedSlot(d.Combat.SkillSlot)), WeaponSpell: spellID,
			WeaponSpellLevel: d.Combat.SpellPower, WeaponSpellSource: source, Protection: d.Protection,
			Resistance: data.DamageKindResistance(d.Resistance),
			SecondaryDamage: sim.SecondaryDamage{Base: d.SecondaryDamage.Base,
				Spread: d.SecondaryDamage.Spread, Selector: d.SecondaryDamage.Selector},
		},
	}
}

func derivedByte(v int32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v)
}

func EquipTarget(c data.ItemCode, t *Table) (int, bool) {
	if t == nil || t.Shapes == nil || t.Materials == nil || t.Weapons == nil || t.Armors == nil {
		return 0, false
	}
	if _, err := data.WeaponFromCode(c, t.Shapes, t.Materials, t.Weapons); err == nil {
		return data.EquipSlotFor(c)
	}
	if t.Shields != nil {
		if _, err := data.ShieldFromCode(c, t.Shapes, t.Materials, t.Shields); err == nil {
			return data.EquipSlotFor(c)
		}
	}
	if piece, err := data.ArmorFromCode(c, t.Shapes, t.Materials, t.Armors); err == nil {
		return int(piece.Slot), true
	}
	return 0, false
}

type InitialEquipment struct {
	Placement              alm.Unit
	HealthDelta, ManaDelta int32
}

func RearmWithLayers(w *sim.World, id sim.EntityID, h data.Hero, profile data.Profile, fallback *data.Weapon, everEquipped bool, t *Table, rotationBase int32, layers []uint16, placement ...InitialEquipment) (*data.Weapon, bool) {
	items, _ := w.EquippedItems(id)
	for _, entity := range w.Entities() {
		if entity.ID == id && entity.ActorLoad.Source.Class != 0 {
			if !items[0].Empty() || everEquipped || fallback == nil {
				return CurrentItemWeapon(items[0], t), true
			}
			weapon := *fallback
			return &weapon, true
		}
	}
	loadout, ok := ResolveItemLoadout(items, fallback, everEquipped, t)
	if !ok {
		return nil, false
	}
	loadout.RotationSpeed = rotationBase
	AddLayers(&loadout, layers, t)
	if len(placement) > 0 {
		actor, _ := w.Entity(id)
		if actor.PotionStats == ([4]int32{}) {
			p := placement[0]
			b, err := blockFor(p.Placement, t, DifficultyNormal, items)
			if err != nil {
				return nil, false
			}
			definition := withTailWords(b.actorDefinition(t), p.Placement, Resolve(p.Placement, t).Arm)
			definition.NativeTraining = sim.NativeTraining{Present: true, Levels: h.Skill}
			if p.HealthDelta != 0 {
				definition.HP = min(definition.MaxHP, definition.HP+p.HealthDelta)
			}
			if p.ManaDelta != 0 {
				definition.Mana = min(definition.MaxMana, definition.Mana+p.ManaDelta)
			}
			return ApplyRearmLoadout(w, id, h, profile, loadout, items, t, definition)
		}
	}
	return ApplyRearmLoadout(w, id, h, profile, loadout, items, t)
}

func ApplyRearmLoadout(w *sim.World, id sim.EntityID, h data.Hero, profile data.Profile,
	loadout data.Loadout, items [sim.EquipSlots]sim.ItemInstance, t *Table, initial ...sim.ActorDefinition) (*data.Weapon, bool) {
	var before sim.Entity
	for _, e := range w.Entities() {
		if e.ID == id {
			before = e
			h = PotionHero(h, e.PotionStats)
			break
		}
	}
	weapon := loadout.Weapon
	if before.ActorLoad.Source.Class != 0 {
		return weapon, true
	}
	if len(initial) > 0 {
		block := DerivedBlock(data.Derived{}, 0, sim.WeaponSpellNone)
		block.InitialPlacement = &initial[0]
		return weapon, w.SetDerived(id, block)
	}
	ApplyItemEffects(&loadout, items, profile.Fighter)
	derived := h.Recompute(profile, loadout)
	if before.Humanoid {
		derived.RotationSpeed = HumanTurnRate(derived.Speed)
	}
	// Resolve the derived weapon spell through the installed table.
	spellID, _ := SpellIDByToken(t, derived.Combat.SpellName)
	source := sim.WeaponSpellNone
	var authoritativeSpell uint16
	var authoritativePower int32
	for _, entity := range w.Entities() {
		if entity.ID == id {
			// Same explicit retention policy as the post-award recompute.
			derived.Combat.SecondBase, derived.Combat.SecondSpread = entity.SecondBase, entity.SecondSpread
			source = entity.WeaponSpellSource
			authoritativeSpell = entity.WeaponSpell
			authoritativePower = entity.WeaponSpellLevel
			break
		}
	}
	if source == sim.WeaponSpellInnate || source == sim.WeaponSpellLegacy {
		spellID = authoritativeSpell
		derived.Combat.SpellPower = authoritativePower
	}
	if spellID != 0 || derived.Combat.SpellPower != 0 {
		if source == sim.WeaponSpellNone || source == sim.WeaponSpellItem {
			source = sim.WeaponSpellLegacy
		}
	}
	wrote := w.SetDerived(id, DerivedBlock(derived, spellID, source))
	if wrote {
		w.SetPotionHeadroom(id, PotionHeadroom(h, derived))
	}
	return weapon, wrote
}

func CreditedSlot(slot int32) int32 {
	if slot <= data.SkillGeneral || slot >= data.SkillSlots {
		return data.SkillGeneral
	}
	return slot
}
