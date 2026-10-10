package mapload

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func CheatItem(name string, t *Table) (sim.ItemInstance, bool) {
	if t == nil || name == "" || len(name) > 4096 {
		return sim.ItemInstance{}, false
	}
	if t.MagicItems != nil {
		for row := 1; row < t.MagicItems.Len() && row <= 255; row++ {
			if t.MagicItems.EntryName(row) == name {
				return ItemInstanceFromCode(uint16(0x0e00|row), t), true
			}
		}
	}
	for code := 1; code <= 65535; code++ {
		if display, ok := t.Names.NameFor(data.ItemCode(code)); ok && display == name {
			return SourceConstructedItem(ItemInstanceFromCode(uint16(code), t), t), true
		}
	}
	if _, _, rejected, err := ParseItemCell(name, t); err != nil || rejected != 0 {
		return sim.ItemInstance{}, false
	}
	if _, item, err := resolveWeaponCell(name, t); err == nil {
		return item, true
	}
	if _, item, err := resolveShieldCell(name, t); err == nil {
		return item, true
	}
	if _, item, err := resolveArmorCell(name, t); err == nil {
		return item, true
	}
	return sim.ItemInstance{}, false
}

type cheatDefinition struct {
	data.Collection
	row int
}

func (c cheatDefinition) EntryName(row int) string {
	if row != c.row {
		return ""
	}
	return c.Collection.EntryName(row)
}

func cheatDefinitionRow(name string, c data.Collection) int {
	if c != nil {
		for row := 1; row < c.Len() && row <= 255; row++ {
			if c.EntryName(row) == name {
				return row
			}
		}
	}
	return -1
}

func CheatActor(name string, hero bool, t *Table, diff Difficulty) (sim.Entity, []sim.ItemStack, [sim.EquipSlots]sim.ItemInstance, error) {
	var worn [sim.EquipSlots]sim.ItemInstance
	if t == nil || name == "" || len(name) > 4096 {
		return sim.Entity{}, nil, worn, fmt.Errorf("actor name is unavailable")
	}
	serverKeys := t.Game.Edition().SecondUnitKeys
	local := *t
	placement := alm.Unit{X: 12<<8 | 128, Y: 12<<8 | 128}
	row := cheatDefinitionRow(name, t.Units)
	if serverKeys {
		row, placement = secondCheatPlacement(name, t, &local, placement)
		if row < 0 {
			return sim.Entity{}, nil, worn, fmt.Errorf("actor name is unknown")
		}
	} else if row >= 0 {
		def, err := data.NewUnitDef(name, t.Units.EntryParams(row))
		if err != nil {
			return sim.Entity{}, nil, worn, fmt.Errorf("actor definition is unavailable")
		}
		if def.TypeID == 0 {
			row = -1
		} else {
			if def.TypeID < unitsKeyFloor || def.TypeID > 255 || def.Face < 0 || def.Face > 255 {
				return sim.Entity{}, nil, worn, fmt.Errorf("actor definition is unavailable")
			}
			local.Units = cheatDefinition{Collection: t.Units, row: row}
			placement.ClassID, placement.ClassSubID = int16(def.TypeID), uint16(def.Face)
		}
	}
	if row < 0 && !serverKeys {
		row = cheatDefinitionRow(name, t.Humans)
		if row < 0 {
			return sim.Entity{}, nil, worn, fmt.Errorf("actor name is unknown")
		}
		local.NPC = nil
		local.composedNPC = map[int32]int{1: row}
		placement.ClassID, placement.ClassSubID, placement.Flags = 1, 1, 1
	}
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{placement}}
	w, roster, err := FromALMRoster(m, &local, diff)
	if err != nil {
		return sim.Entity{}, nil, worn, err
	}
	e, found := w.Entity(0)
	if !found || Resolve(placement, &local).Index != row {
		return sim.Entity{}, nil, worn, fmt.Errorf("actor definition did not construct its exact row")
	}
	pack, _ := w.CarriedStacks(e.ID)
	worn, _ = w.EquippedItems(e.ID)
	var human *data.HumanState
	if e.Humanoid {
		h := cheatHumanState(e)
		human = &h
	}
	initial := e.NativeBasis
	e, _, err = ConstructActorBasis(e, roster[e.ID], &placement, &local, 1, 1, human, 0)
	if err != nil {
		return sim.Entity{}, nil, worn, err
	}
	if initial.BasePresent {
		e.ActorLoad.Source.Base = initial.Base
	}
	if initial.ModifierPresent {
		e.ActorLoad.Source.Modifier = initial.Modifier
	}
	e = ConstructActorLoad(e, worn, pack)
	e.NativeClass = sim.NativeClass{}
	e.NativeTraining = sim.NativeTraining{}
	return e, pack, worn, nil
}

// secondCheatPlacement keys a second-game placement to the creature row named
// name, else the human row: the placement resolves a row by its server id
// (R2-ENGINE-301 tries a creature by name, then a human). local sees only that
// row, so the server id cannot reach another row carrying the same id.
func secondCheatPlacement(name string, t *Table, local *Table, placement alm.Unit) (int, alm.Unit) {
	if row := cheatDefinitionRow(name, t.Units); row >= 0 {
		if id, ok := data.UnitServerID(t.Units.EntryParams(row)); ok && id > 0 {
			local.Units = cheatDefinition{Collection: t.Units, row: row}
			placement.ServerID = uint32(id)
			return row, placement
		}
	}
	if row := cheatDefinitionRow(name, t.Humans); row >= 0 {
		if id, ok := data.HumanServerID(t.Humans.EntryParams(row)); ok && id > 0 {
			local.Humans = cheatDefinition{Collection: t.Humans, row: row}
			placement.ServerID, placement.Flags = uint32(id), rom2PersonFlag
			return row, placement
		}
	}
	return -1, placement
}

func cheatHumanState(e sim.Entity) data.HumanState {
	h := data.HumanState{Body: e.NativeBasis.Body, Reaction: uint16(e.Reaction), Mind: uint16(e.Mind), Spirit: uint16(e.Spirit),
		Speed: uint16(e.Speed), Capacity: uint16(e.Capacity), Health: uint16(e.HP), HealthMax: uint16(e.MaxHP), HealthPeriod: uint16(e.HealthRegenPeriod),
		Mana: uint16(e.Mana), ManaMax: uint16(e.MaxMana), ManaPeriod: uint16(e.ManaRegenPeriod), Sight: uint16(e.ScanRange) << 8,
		TypeID: uint16(e.TypeID), MoverSpeed: uint8(e.RotationSpeed), Fighter: e.NativeClass.Fighter,
		HasSpellbook: e.Book.WirePresent(e.KnownSpells), HasOwner: e.Owner != 0, ManaReservePercent: 95}
	h.Base = sourceAttack(e.NativeBasis.Base)
	h.Modifier = SourceHumanState(sim.SourceActor{Modifier: e.NativeBasis.Modifier}, 0).Modifier
	for i, level := range e.Skill {
		h.Attack.Skill[i], h.SkillXP[i] = uint16(level), uint32(e.SkillXP[i])
		h.Experience += h.SkillXP[i]
	}
	h.Attack.ToHit, h.Attack.DamageBase, h.Attack.DamageSpread, h.Attack.Active = uint16(e.ToHit), uint8(e.DamageBase), uint8(e.DamageSpread), e.XPSlot
	h.Attack.SecondBase, h.Attack.SecondSpread = e.SecondBase, e.SecondSpread
	h.Attack.ElementalBase, h.Attack.ElementalSpread = e.SecondaryDamage.Base, e.SecondaryDamage.Spread
	if e.SecondaryDamage.Base != 0 || e.SecondaryDamage.Spread != 0 {
		h.Attack.ElementalKind = e.SecondaryDamage.Selector + 1
	}
	h.Defence.Defence, h.Defence.Absorption = uint16(e.Defence), uint16(e.Absorption)
	h.Defence.Resistance[0] = e.NativeBasis.Defence[16]
	for i := range e.Protection {
		h.Defence.Protection[i+1], h.Defence.Resistance[i+1] = uint16(e.Protection[i]), e.Resistance[i]
	}
	return h
}
