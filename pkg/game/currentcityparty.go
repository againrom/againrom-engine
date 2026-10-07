package game

import (
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type currentCityPartyPolicy struct {
	ClassWire          uint16
	Class              int32
	SuppressCorpseLoot bool
	BookMode           uint8
	LegacyBookWire     *[32]byte `json:",omitempty"`
	ExtraSpells        uint32
	OrderedStacks      bool
	Load               *currentCityLoadPolicy `json:",omitempty"`
	Potion             *sim.ActiveEffect      `json:",omitempty"`
}

type currentCityLoadPolicy struct {
	SourceClass             uint8
	EquipmentRuntimePresent bool
	MovementPresent         bool
	SpeedWire               int16
	SpeedLift               int64
}

type currentPartyHoldingLocation struct {
	Slot      uint8
	Inventory uint32
}

func projectCurrentCityParty(doc *sav.DocumentData, party []mapload.PartyMember, actors []uint16, a *currentActionData, table *mapload.Table) error {
	if len(party) != len(actors) {
		return fmt.Errorf("current city party lacks complete bindings")
	}
	records, err := sav.ReadDocumentCharacters(*doc, actors)
	if err != nil {
		return err
	}
	for i, member := range party {
		c := records[i]
		p, err := captureOrdinaryPartyMember(sim.EntityID(i), member, c, table)
		if err != nil {
			return err
		}
		a.Party = append(a.Party, p)
		a.Bindings = append(a.Bindings, currentActionBinding{ID: p.Entity, Object: actors[i]})
	}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, b)
}

func captureOrdinaryPartyMember(id sim.EntityID, member mapload.PartyMember, c sav.DocumentCharacter, table *mapload.Table) (currentPartyMember, error) {
	p := captureCurrentParty(id, member)
	p.ordinary = &c
	p.Policy = capturePartyPolicy(member, c)
	p.City = &currentCityPartyPolicy{ClassWire: c.Character.Basis.Human.TypeID, Class: member.Class,
		SuppressCorpseLoot: member.SuppressCorpseLoot, BookMode: uint8(member.Book.State),
		ExtraSpells: member.KnownSpells & ^uint32(0x1ffffffe), Potion: member.PotionEffect}
	if member.Book.State == sim.BookLegacy {
		anchor := sim.BookValueAnchor(importedSpellbook(c.Character.HasSpellbook, c.Character.Spells), c.Character.KnownSpells())
		p.City.LegacyBookWire = &anchor
	}
	native := true
	if _, ok := member.OriginalHumanState(); ok {
		native = false
	}
	if member.Carry != nil {
		p.City.OrderedStacks = member.Carry.OrderedStacks != nil
		if load := member.Carry.LiveLoad; load != nil {
			native = load.Inventory.Source.Class != 2
			p.City.Load = &currentCityLoadPolicy{SourceClass: load.Inventory.Source.Class,
				EquipmentRuntimePresent: load.Inventory.Source.EquipmentRuntimePresent, MovementPresent: load.Movement.Present,
				SpeedWire: c.Character.LoadState.Speed, SpeedLift: int64(load.Speed) - int64(c.Character.LoadState.Speed)}
		}
	}
	var err error
	p.Base, err = capturePartyBase(member.Hero, c.Character, native, [4]int32{})
	if err != nil {
		return currentPartyMember{}, err
	}
	p.Weapon = captureNamedPartyWeapon(member, cityMemberEquipment(member, table)[0])
	if member.OriginalHuman != nil {
		p.Human = &currentPartyHuman{Retired: member.OriginalHuman.Retired}
	}
	if c.Character.Class != "Unit" {
		p.Name = nil
	}
	p.Member, p.WeaponName, p.OriginalWeaponName = nil, nil, nil
	return p, nil
}

func restoreCurrentCityParty(doc *sav.DocumentData, a *currentActionData, table *mapload.Table, topology ...**cityObjectTopology) ([]mapload.PartyMember, []sav.Character, bool, error) {
	if a == nil || a.Party == nil {
		return nil, nil, false, nil
	}
	if doc == nil || doc.World != nil || len(doc.Objects) > 0x7fff || len(a.Party) > len(doc.Objects) || len(a.Party) == 0 || len(a.Roster) != 0 {
		return nil, nil, false, fmt.Errorf("invalid current city member population")
	}
	if err := bindCurrentPartyRecords(a, doc); err != nil {
		return nil, nil, false, err
	}
	var actors []cityActorObjectBinding
	for _, member := range a.Party {
		for _, binding := range a.Bindings {
			if !binding.Structure && !binding.Missing && binding.ID == member.Entity {
				actors = append(actors, cityActorObjectBinding{Object: binding.Object, PartyID: string(member.ID)})
			}
		}
	}
	graph, err := captureCurrentCityTopology(doc, a, actors)
	if err != nil {
		return nil, nil, false, err
	}
	items := map[uint16]sim.ItemStack{}
	for _, row := range a.Ownership {
		if row.Object == 0 || row.Kind != 1 || row.Item != nil || row.Effect != nil || row.Spell != nil || row.Sack != nil {
			return nil, nil, false, fmt.Errorf("current city item has no ordinary binding")
		}
		if _, duplicate := items[row.Object]; duplicate {
			return nil, nil, false, fmt.Errorf("current city item repeats its binding")
		}
		item, err := readCurrentItem(doc, row, table)
		if err != nil {
			return nil, nil, false, err
		}
		items[row.Object] = item.Value
	}
	var party []mapload.PartyMember
	var chars []sav.Character
	seen := map[string]bool{}
	for _, p := range a.Party {
		if p.Member != nil || p.Policy == nil || p.Base == nil || p.City == nil || p.ordinary == nil || p.City.BookMode > uint8(sim.BookNativeAbsent) {
			return nil, nil, false, fmt.Errorf("incomplete current city member policy")
		}
		state, err := currentCityPartyState(doc, a, p, items)
		if err != nil {
			return nil, nil, false, err
		}
		member, err := p.restoreFromState(state, table)
		if err != nil {
			return nil, nil, false, err
		}
		if member.ID == "" || seen[member.ID] {
			return nil, nil, false, fmt.Errorf("current city member has absent or repeated identity")
		}
		seen[member.ID] = true
		party, chars = append(party, member), append(chars, p.ordinary.Character)
	}
	if len(topology) != 0 && topology[0] != nil {
		*topology[0] = graph
	}
	return party, chars, true, nil
}

func currentCityPartyState(doc *sav.DocumentData, a *currentActionData, p currentPartyMember, items map[uint16]sim.ItemStack) (*currentPartyState, error) {
	var actor uint16
	for _, binding := range a.Bindings {
		if !binding.Structure && binding.ID == p.Entity && !binding.Missing {
			actor = binding.Object
		}
	}
	return currentOrdinaryPartyState(doc, actor, p, p.City, items)
}

func currentOrdinaryPartyState(doc *sav.DocumentData, actor uint16, p currentPartyMember, policy *currentCityPartyPolicy, items map[uint16]sim.ItemStack, locations ...[]currentPartyHoldingLocation) (*currentPartyState, error) {
	if p.ordinary == nil || p.Base == nil || policy == nil || policy.BookMode > uint8(sim.BookNativeAbsent) {
		return nil, fmt.Errorf("incomplete ordinary party state")
	}
	c := p.ordinary.Character
	state := &currentPartyState{Class: int32(c.Basis.Human.TypeID), SuppressCorpseLoot: policy.SuppressCorpseLoot,
		KnownSpells: c.KnownSpells(), Book: importedSpellbook(c.HasSpellbook, c.Spells),
		Native: p.Base.Native, Saved: restoredSaved(c, &RestoredParty{}), Potion: policy.Potion}
	if policy.ClassWire == c.Basis.Human.TypeID {
		state.Class = policy.Class
	}
	var err error
	state.Book, state.KnownSpells, err = sim.RestoreBookMode(state.Book, state.KnownSpells, sim.BookState(policy.BookMode), policy.LegacyBookWire, policy.ExtraSpells)
	if err != nil {
		return nil, err
	}
	for i, xp := range c.SkillXP {
		state.SkillXP[i] = int32(xp)
	}
	if policy.Load != nil {
		state.Load = originalActorLoad(c.LoadState, int32(c.LoadState.Speed))
		if state.Load == nil || policy.Load.SourceClass > 2 {
			return nil, fmt.Errorf("invalid current city load policy")
		}
		if policy.Load.SourceClass != 0 {
			state.Load.Inventory.Source = originalActorBasis(c.Basis)
			if policy.Load.SourceClass != state.Load.Inventory.Source.Class {
				return nil, fmt.Errorf("current city load class differs from ordinary actor")
			}
			state.Load.Inventory.Source.EquipmentRuntimePresent = policy.Load.EquipmentRuntimePresent
		}
		if policy.Load.SpeedLift < -1<<31-int64(policy.Load.SpeedWire) || policy.Load.SpeedLift > 1<<31-1-int64(policy.Load.SpeedWire) {
			return nil, fmt.Errorf("current city speed operand exceeds native width")
		}
		if policy.Load.SpeedWire == c.LoadState.Speed {
			state.Load.Speed = int32(int64(c.LoadState.Speed) + policy.Load.SpeedLift)
		}
		if policy.Load.MovementPresent {
			state.Load.Movement.NativeSpeed = state.Load.Speed
		} else {
			state.Load.Movement = sim.HumanMovement{}
		}
	}
	if actor == 0 || int(actor) > len(doc.Objects) {
		return nil, fmt.Errorf("current city member lost ordinary actor")
	}
	r := &doc.Objects[actor-1]
	item := func(ref uint16) (sim.ItemStack, error) {
		if ref == 0 {
			return sim.ItemStack{}, nil
		}
		if v, ok := items[ref]; ok {
			return v.Clone(), nil
		}
		return sim.ItemStack{}, fmt.Errorf("current city holding lacks bound item %d", ref)
	}
	for slot := range state.Equipment {
		field, ordinal := "Worn", slot
		if slot < 2 {
			field, ordinal = []string{"HeldWeapon", "HeldShield"}[slot], 0
		}
		refs, _ := savedObjectRefs(r, field)
		if slot < 2 && len(refs) == 1 && refs[0] == 0 {
			if worn, _ := savedObjectRefs(r, "Worn"); slot < len(worn) && worn[slot] != 0 {
				refs, ordinal = worn, slot
			}
		}
		if ordinal >= len(refs) {
			continue
		}
		v, err := item(refs[ordinal])
		if err != nil {
			return nil, err
		}
		state.Equipment[slot] = v.Instance()
	}
	refs, _ := savedObjectRefs(r, "Inventory")
	equipped := map[uint32]bool{}
	if len(locations) != 0 {
		seen := map[uint8]bool{}
		for _, location := range locations[0] {
			if r.Class != "Unit" || location.Slot < 2 || int(location.Slot) >= sim.EquipSlots || uint64(location.Inventory) >= uint64(len(refs)) || seen[location.Slot] || equipped[location.Inventory] {
				return nil, fmt.Errorf("invalid current Unit equipment location")
			}
			seen[location.Slot], equipped[location.Inventory] = true, true
			v, err := item(refs[location.Inventory])
			if err != nil || v.Count != 1 {
				return nil, fmt.Errorf("invalid current Unit equipment item: %v", err)
			}
			state.Equipment[location.Slot] = v.Instance()
		}
	}
	if policy.OrderedStacks {
		state.Stacks = make([]sim.ItemStack, 0, len(refs))
	}
	for ordinal, ref := range refs {
		if equipped[uint32(ordinal)] {
			continue
		}
		v, err := item(ref)
		if err != nil {
			return nil, err
		}
		if uint64(len(state.Items))+uint64(v.Count) > 65536 {
			return nil, fmt.Errorf("current city pack exceeds native item bound")
		}
		for count := v.Count; count > 0; count-- {
			state.Items = append(state.Items, v.Instance())
		}
		if policy.OrderedStacks {
			state.Stacks = append(state.Stacks, v)
		}
	}
	return state, nil
}
