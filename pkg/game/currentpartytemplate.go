package game

import (
	"encoding/binary"
	"fmt"
	"math"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// An absent member owns a detached ordinary graph, never a World population.
// Owner and primary-character context normally come from its enclosing Player.
type currentPartyTemplate struct {
	Records            sav.DocumentFragment
	Items              []currentOwnedObject
	Policy             *currentCityPartyPolicy
	StartingHero       bool
	HasOwner           bool
	ManaReservePercent uint32
	Equipment          []currentPartyHoldingLocation `json:",omitempty"`
	PackAbsent         bool
	UnitSkillXP        *[6]int32                  `json:",omitempty"`
	SavedLifts         []currentTemplateSavedLift `json:",omitempty"`
}

type currentTemplateSavedLift struct {
	Field uint8
	Wire  uint16
	Lift  int64
}

func templateSavedFields(saved *mapload.Saved) []*int32 {
	return []*int32{&saved.Cell.X, &saved.Cell.Y, &saved.HP, &saved.MaxHP,
		&saved.HealthRegenPeriod, &saved.Mana, &saved.MaxMana, &saved.ManaRegenPeriod}
}

func captureTemplateSavedLifts(saved *mapload.Saved, c sav.Character, sourcePools bool) []currentTemplateSavedLift {
	if saved == nil {
		return nil
	}
	wire := templateSavedFields(restoredSaved(c, &RestoredParty{}))
	var lifts []currentTemplateSavedLift
	for i, value := range templateSavedFields(saved) {
		if i >= 2 && sourcePools && uint16(*value) != uint16(*wire[i]) {
			continue
		}
		if *value != *wire[i] {
			lifts = append(lifts, currentTemplateSavedLift{Field: uint8(i), Wire: uint16(*wire[i]), Lift: int64(*value) - int64(*wire[i])})
		}
	}
	return lifts
}

func (p currentPartyMember) restoreTemplateSavedLifts(saved *mapload.Saved) error {
	fields, seen := templateSavedFields(saved), map[uint8]bool{}
	for _, lift := range p.Template.SavedLifts {
		if !p.Policy.Saved || int(lift.Field) >= len(fields) || seen[lift.Field] || lift.Lift == 0 ||
			lift.Field < 2 && lift.Wire > 255 || lift.Lift < math.MinInt32-int64(lift.Wire) || lift.Lift > math.MaxInt32-int64(lift.Wire) {
			return fmt.Errorf("invalid absent template saved width operand")
		}
		seen[lift.Field] = true
		if value := fields[lift.Field]; *value == int32(lift.Wire) {
			*value = int32(int64(*value) + lift.Lift)
		}
	}
	return nil
}

func currentMemberActorRecord(member, hero mapload.PartyMember, table *mapload.Table, identity uint32) (sav.DocumentRecordData, error) {
	unit := nativeCityUnitData(identity, 0, member, hero, table, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
	unit.ContainerFlag = 1
	unit.Equipment = make([]uint16, 13)
	if member.Book.WirePresent(member.KnownSpells) {
		unit.SpellbookFlag, unit.SpellbookCount = 1, 1
	}
	if err := nativeCityInitializeHumanMovement(&unit, table); err != nil {
		return sav.DocumentRecordData{}, err
	}
	class := "Human"
	sourcePools := member.Carry != nil && member.Carry.LiveLoad != nil && member.Carry.LiveLoad.Inventory.Source.Class != 0
	if sourcePools {
		load := member.Carry.LiveLoad
		if err := load.Validate(); err != nil {
			return sav.DocumentRecordData{}, err
		}
		source := load.Inventory.Source
		if source.Class == 1 {
			class = "Unit"
		}
		h := mapload.SourceHumanState(source, load.Inventory.Accumulator)
		if err := nativeCityApplyHumanState(&unit, member, h); err != nil {
			return sav.DocumentRecordData{}, err
		}
		unit.Scalar2[28], unit.Scalar2[29] = load.HealthHundredths, load.ManaHundredths
		unit.Scalar2[34], unit.Scalar2[39], unit.Scalar2[40] = source.Reach, source.AttackCharge, source.AttackRelax
		for i, value := range []uint16{uint16(load.Speed), uint16(load.Inventory.OwnWeight), uint16(load.Load), uint16(load.Capacity)} {
			if i == 0 && load.Movement.Present {
				value = uint16(load.Movement.RawSpeed)
			}
			binary.LittleEndian.PutUint16(unit.Scalar2[(4+i)*2:], value)
		}
	} else if err := applyCurrentCityHuman(&unit, member, table); err != nil {
		return sav.DocumentRecordData{}, err
	}
	if saved := member.Saved; saved != nil {
		copy(unit.Token, constructedPositionBlock(int32(uint8(saved.Cell.X)), int32(uint8(saved.Cell.Y)), 0))
		binary.LittleEndian.PutUint32(unit.Token[19:], uint32(saved.MapUnitID))
		if !sourcePools {
			for i, value := range []int32{saved.HP, saved.MaxHP, saved.HealthRegenPeriod, saved.Mana, saved.MaxMana, saved.ManaRegenPeriod} {
				binary.LittleEndian.PutUint16(unit.Scalar2[(8+i)*2:], uint16(value))
			}
		}
	}
	return sav.DocumentActorFromCityUnit(class, unit)
}

func captureCurrentPartyTemplate(id sim.EntityID, member, hero mapload.PartyMember, table *mapload.Table) (currentPartyMember, error) {
	b := generatedDocumentBuilder{table: table}
	b.reserveCurrentParty([]mapload.PartyMember{member})
	r, err := currentMemberActorRecord(member, hero, table, b.identity())
	if err != nil {
		return currentPartyMember{}, fmt.Errorf("current absent member %d: %w", id, err)
	}
	actor, err := b.append(r)
	if err != nil {
		return currentPartyMember{}, err
	}
	a := currentActionData{}
	var equipment []currentPartyHoldingLocation
	if err := b.currentCityHoldings(actor, member, &a, &equipment); err != nil {
		return currentPartyMember{}, err
	}
	fragment, permutation, err := sav.ReindexDocumentFragment(sav.DocumentFragment{Actor: actor, Objects: b.doc.Objects})
	if err != nil {
		return currentPartyMember{}, err
	}
	for i := range a.Ownership {
		a.Ownership[i].Object = permutation[a.Ownership[i].Object]
		if err := captureCurrentItemAbsence(&sav.DocumentData{Objects: fragment.Objects}, &a.Ownership[i]); err != nil {
			return currentPartyMember{}, err
		}
	}
	characters, err := sav.ReadDocumentCharacters(sav.DocumentData{Objects: fragment.Objects}, []uint16{fragment.Actor})
	if err != nil {
		return currentPartyMember{}, err
	}
	p, err := captureOrdinaryPartyMember(id, member, characters[0], table)
	if err != nil {
		return currentPartyMember{}, err
	}
	p.Template = &currentPartyTemplate{Records: fragment, Items: a.Ownership, Policy: p.City, StartingHero: member.StartingHero, Equipment: equipment}
	sourcePools := member.Carry != nil && member.Carry.LiveLoad != nil && member.Carry.LiveLoad.Inventory.Source.Class != 0
	p.Template.SavedLifts = captureTemplateSavedLifts(member.Saved, characters[0].Character, sourcePools)
	p.City = nil
	p.Name = nil
	if r.Class == "Unit" && member.Carry != nil {
		xp := member.Carry.SkillXP
		p.Template.UnitSkillXP = &xp
	}
	if member.Carry != nil && member.Carry.LiveLoad != nil && member.Carry.LiveLoad.Inventory.Source.Class != 0 {
		source := member.Carry.LiveLoad.Inventory.Source
		p.Template.HasOwner, p.Template.ManaReservePercent = source.HasOwner, source.ManaReservePercent
	} else if member.OriginalHuman != nil {
		p.Template.HasOwner, p.Template.ManaReservePercent = member.OriginalHuman.State.HasOwner, member.OriginalHuman.State.ManaReservePercent
	}
	if len(equipment) != 0 && member.Carry != nil && member.Carry.LiveLoad != nil {
		p.Template.PackAbsent = !member.Carry.LiveLoad.Inventory.ContainerPresent
	}
	if err := p.bindTemplate(); err != nil {
		return currentPartyMember{}, err
	}
	return p, nil
}

func (p *currentPartyMember) bindTemplate() error {
	if p.Template == nil || p.Template.Policy == nil || p.Member != nil || p.City != nil || p.Policy == nil || p.Base == nil || p.Weapon == nil {
		return fmt.Errorf("current absent member has incomplete or conflicting policy")
	}
	fragment, err := sav.CloneDocumentFragment(p.Template.Records)
	if err != nil {
		return err
	}
	characters, err := sav.ReadDocumentCharacters(sav.DocumentData{Objects: fragment.Objects}, []uint16{fragment.Actor})
	if err != nil {
		return err
	}
	p.ordinary = &characters[0]
	p.ordinary.Character.Hero = p.Template.StartingHero
	p.ordinary.Character.Basis.Human.HasOwner = p.Template.HasOwner
	p.ordinary.Character.Basis.Human.ManaReservePercent = p.Template.ManaReservePercent
	if (p.Template.UnitSkillXP != nil) != (p.ordinary.Character.Class == "Unit" && p.Policy.Carry) {
		return fmt.Errorf("current template has missing or conflicting native skill experience")
	}
	if xp := p.Template.UnitSkillXP; xp != nil {
		for i, value := range xp {
			p.ordinary.Character.SkillXP[i] = uint32(value)
			p.ordinary.Character.Basis.SkillXP[i] = uint32(value)
		}
	}
	return nil
}

func (p currentPartyMember) templateState(table *mapload.Table) (*currentPartyState, error) {
	if err := p.bindTemplate(); err != nil {
		return nil, err
	}
	doc := sav.DocumentData{Objects: p.Template.Records.Objects}
	items := map[uint16]sim.ItemStack{}
	ids := map[sim.SavedObjectID]bool{}
	for _, row := range p.Template.Items {
		if err := validateCurrentItemAbsence(row); err != nil {
			return nil, err
		}
		if row.Kind != 1 || row.Object == 0 || int(row.Object) > len(doc.Objects) || row.Item != nil || row.Effect != nil || row.Spell != nil || row.Sack != nil || row.ID != 0 && ids[row.ID] {
			return nil, fmt.Errorf("invalid absent member item policy")
		}
		if _, exists := items[row.Object]; exists {
			return nil, fmt.Errorf("repeated absent member item policy")
		}
		ids[row.ID] = true
		item, err := readCurrentItem(&doc, row, table)
		if err != nil {
			return nil, err
		}
		items[row.Object] = item.Value
	}
	for i, r := range doc.Objects {
		if savedItemClass(r.Class) {
			if _, exists := items[uint16(i+1)]; !exists {
				return nil, fmt.Errorf("absent member item lacks policy")
			}
		}
	}
	state, err := currentOrdinaryPartyState(&doc, p.Template.Records.Actor, p, p.Template.Policy, items, p.Template.Equipment)
	if err != nil {
		return nil, err
	}
	if p.Template.PackAbsent {
		if len(p.Template.Equipment) == 0 || state.Load == nil || len(state.Items) != 0 || state.Load.Inventory.InsertIndex != 0 || state.Load.Inventory.Accumulator != 0 {
			return nil, fmt.Errorf("invalid absent template pack policy")
		}
		state.Load.Inventory.ContainerPresent = false
	}
	if state.Load != nil {
		if err := state.Load.Validate(); err != nil {
			return nil, err
		}
	}
	if err := p.restoreTemplateSavedLifts(state.Saved); err != nil {
		return nil, err
	}
	return state, nil
}
