package mapload

import (
	"encoding/binary"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// SourceConstructedItem fills a newly generated item through the established
// definition constructors (ITEM-SCALE-017, ITEM-LADDER-019, ITEM-ARMFILL-032).
// This is not a LOAD rebind: explicit saved weights, concrete classes, saved
// rows and current blocks always win, including a saved BaseItem's class zero.
func SourceConstructedItem(item sim.ItemInstance, t *Table) sim.ItemInstance {
	if item.WeightPresent || item.SourceEquipment.Class != 0 || t == nil {
		return item
	}
	c := data.ItemCode(item.Code)
	// Source producers must not use the native partial-table identity factor.
	if t.Shapes == nil || t.Materials == nil || c.C() >= t.Shapes.Len() || c.A() >= t.Materials.Len() ||
		len(t.Shapes.EntryDoubles(c.C())) < 9 || len(t.Materials.EntryDoubles(c.A())) < 9 {
		return item
	}
	s := sim.SourceEquipment{DefinitionRow: uint8(c.D())}
	var weight int32
	switch c.B() {
	case 1:
		v, err := data.WeaponFromCode(c, t.Shapes, t.Materials, t.Weapons)
		if err != nil || v.Code != c {
			return item
		}
		return sourceWeaponItem(item, v, t)
	case 2:
		v, err := data.ShieldFromCode(c, t.Shapes, t.Materials, t.Shields)
		if err != nil {
			return item
		}
		s.Class, weight = sim.SourceShield, v.Weight
		binary.LittleEndian.PutUint16(s.Defence[:], uint16(v.Defence))
		binary.LittleEndian.PutUint16(s.Defence[2:], uint16(v.Absorption))
	case 3, 4, 5, 6, 7, 8, 9, 10, 11, 12:
		v, err := data.ArmorFromCode(c, t.Shapes, t.Materials, t.Armors)
		if err != nil {
			return item
		}
		s.Class, s.OwnKind, weight = sim.SourceArmor, uint8(v.Slot), v.Weight
		binary.LittleEndian.PutUint16(s.Defence[:], uint16(v.Defence))
		binary.LittleEndian.PutUint16(s.Defence[2:], uint16(v.Absorption))
	default:
		return item
	}
	item.SourceEquipment, item.WeightPresent, item.Weight = s, true, int16(weight)
	item.Kind = 1
	if s.Class == sim.SourceWeapon {
		item.Kind = 2
	}
	return item
}

func sourceWeaponItem(item sim.ItemInstance, weapon data.Weapon, t *Table) sim.ItemInstance {
	s := sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: uint8(weapon.Row), OwnKind: uint8(weapon.Range)}
	binary.LittleEndian.PutUint16(s.Attack[:], uint16(weapon.ToHit))
	s.Attack[14], s.Attack[15] = uint8(weapon.DamageBase), uint8(weapon.DamageSpread)
	binary.LittleEndian.PutUint16(s.Defence[:], uint16(weapon.Defence))
	item.SourceEquipment = BindSourceItemDefinition(sim.ItemInstance{Code: item.Code, SourceEquipment: s}, t).SourceEquipment
	if !item.SourceEquipment.Definition.Present {
		definition := sim.SourceWeaponDefinition{Present: true, AttackType: weapon.AttackType,
			Charge: weapon.ChargeTime, Relax: weapon.RelaxTime, Suitable: 1}
		if hands, ok := weaponHands(&weapon, t); ok {
			definition.Hands = hands
		}
		item.SourceEquipment.Definition = definition
	}
	item.WeightPresent, item.Weight, item.Kind = true, int16(weapon.Weight), 2
	return item
}

// SourceEquippedItem materialises the owned Spell child of a freshly
// equipped weapon from the same first kind-41 effect used by combat.
func SourceEquippedItem(item sim.ItemInstance, t *Table) sim.ItemInstance {
	item = SourceConstructedItem(item, t)
	if item.SourceEquipment.Class != sim.SourceWeapon || item.SourceEquipment.Spell.Present || t == nil {
		return item
	}
	for _, effect := range item.Effects {
		if effect.Kind != 41 {
			continue
		}
		id := uint8(effect.Operand)
		rules := SpellRules(t)
		if id == 0 || int(id) > len(rules) || rules[id-1].ID != uint16(id) {
			return item
		}
		rule := rules[id-1]
		mana := max(int32(0), min(rule.ManaCost, int32(0xffff)))
		item.SourceEquipment.Spell = sim.SourceItemSpell{Present: true, ID: id, Range: rule.MaxRange, ManaCost: uint16(mana)}
		if rule.Defensive {
			item.SourceEquipment.Spell.Defensive = 1
		}
		return item
	}
	return item
}

// DeclareSourceConstructors adds integer constructor operands to the world's
// existing producer-code population. A fresh native decode needs no table or
// captured App to equip a generated item later. Missing/partial rows stay
// unavailable; failed equipment commands remain atomic rather than native.
func DeclareSourceConstructors(w *sim.World, t *Table) {
	if w == nil || t == nil {
		return
	}
	hasSource := false
	for _, e := range w.Entities() {
		hasSource = hasSource || e.ActorLoad.Source.Class != 0
	}
	if !hasSource {
		return
	}
	weights := w.ItemWeights()
	for i := range weights {
		if weights[i].Constructor.Class != 0 {
			continue
		}
		weights[i].Constructor = SourceConstructedItem(sim.PlainItem(weights[i].Code), t).SourceEquipment
	}
	_ = w.DeclareItemWeights(weights)
}
