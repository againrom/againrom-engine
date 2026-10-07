package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type currentPartyState struct {
	Class              int32
	SuppressCorpseLoot bool
	KnownSpells        uint32
	Book               sim.Spellbook
	Equipment          [sim.EquipSlots]sim.ItemInstance
	Items              []sim.ItemInstance
	Stacks             []sim.ItemStack
	SkillXP            [data.SkillSlots]int32
	Load               *sim.ActorLoadSnapshot
	Saved              *mapload.Saved
	Potion             *sim.ActiveEffect
	PotionStats        [4]int32
	Native             bool
}

func bindCurrentPartyRecords(a *currentActionData, doc *sav.DocumentData) error {
	bindings := map[sim.EntityID]currentActionBinding{}
	for _, b := range a.Bindings {
		if !b.Structure {
			if _, exists := bindings[b.ID]; exists {
				return fmt.Errorf("current party has repeated actor binding %d", b.ID)
			}
			bindings[b.ID] = b
		}
	}
	var indices []uint16
	var members []*currentPartyMember
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		seen := map[sim.EntityID]bool{}
		for i := range rows {
			p := &rows[i]
			if seen[p.Entity] {
				return fmt.Errorf("current party repeats entity %d", p.Entity)
			}
			seen[p.Entity] = true
			b, ok := bindings[p.Entity]
			if !ok {
				return fmt.Errorf("current party entity %d has no actor binding", p.Entity)
			}
			if p.Template != nil {
				if err := p.bindTemplate(); err != nil {
					return err
				}
				continue
			}
			if b.Missing {
				continue
			}
			indices = append(indices, b.Object)
			members = append(members, p)
		}
	}
	if len(indices) == 0 {
		return nil
	}
	if doc == nil {
		return fmt.Errorf("current party has no ordinary document")
	}
	records, err := sav.ReadDocumentCharacters(*doc, indices)
	if err != nil {
		return err
	}
	for i := range members {
		members[i].ordinary = &records[i]
	}
	return nil
}

func stripCurrentPartyValues(a *currentActionData, w *sim.World, tables ...*mapload.Table) error {
	var table *mapload.Table
	if len(tables) != 0 {
		table = tables[0]
	}
	var hero mapload.PartyMember
	for _, p := range a.Party {
		if member := p.restore(); member.StartingHero {
			hero = member
			break
		}
	}
	live := map[sim.EntityID]sim.Entity{}
	for _, e := range w.Entities() {
		live[e.ID] = e
	}
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		for i := range rows {
			p := &rows[i]
			e, present := live[p.Entity]
			if !present {
				next, err := captureCurrentPartyTemplate(p.Entity, p.restore(), hero, table)
				if err != nil {
					return err
				}
				*p = next
				continue
			}
			if p.ordinary == nil {
				return fmt.Errorf("current party actor %d has no ordinary record", p.Entity)
			}
			member := p.restore()
			if e.ActorLoad.Source.Class == 0 && e.NativeTraining.Present {
				member.Hero.Skill = e.NativeTraining.Levels
			}
			base, err := capturePartyBase(member.Hero, p.ordinary.Character, e.ActorLoad.Source.Class != 2, e.PotionStats)
			if err != nil {
				return err
			}
			p.Base = base
			p.Base.LegacyTraining = e.ActorLoad.Source.Class == 0 && !e.NativeTraining.Present
			p.Base.LegacyClass = e.ActorLoad.Source.Class == 0 && !e.NativeClass.Present
			equipment, ok := w.EquippedItems(p.Entity)
			if !ok {
				return fmt.Errorf("current party actor %d has no current equipment", p.Entity)
			}
			p.Weapon = captureNamedPartyWeapon(member, equipment[0])
			p.Policy = capturePartyPolicy(member, *p.ordinary)
			if e.Book.State != sim.BookLegacy {
				p.Policy.LegacySpellbookPresent = false
			}
			if member.OriginalHuman != nil {
				p.Human = &currentPartyHuman{Retired: member.OriginalHuman.Retired}
			}
			if p.ordinary.Character.Class != "Unit" {
				p.Name = nil
			}
			p.Member, p.WeaponName, p.OriginalWeaponName = nil, nil, nil
		}
	}
	return nil
}

// Current inventory, books and pools are views of the restored World. The
// member retains its base inputs and presence modes, not another live copy.
func (p currentPartyMember) restoreFromCurrent(w *sim.World, t *mapload.Table) (mapload.PartyMember, error) {
	var state *currentPartyState
	for _, e := range w.Entities() {
		if e.ID != p.Entity {
			continue
		}
		if p.Template != nil {
			return mapload.PartyMember{}, fmt.Errorf("current template actor %d is present in World", p.Entity)
		}
		if state != nil {
			return mapload.PartyMember{}, fmt.Errorf("current party actor %d is ambiguous", p.Entity)
		}
		equipment, equipmentOK := w.EquippedItems(p.Entity)
		items, itemsOK := w.CarriedItems(p.Entity)
		stacks, stacksOK := w.CarriedStacks(p.Entity)
		if !equipmentOK || !itemsOK || !stacksOK {
			return mapload.PartyMember{}, fmt.Errorf("current party actor %d has no current holdings", p.Entity)
		}
		state = &currentPartyState{Class: e.Class, SuppressCorpseLoot: e.SuppressCorpseLoot,
			KnownSpells: e.KnownSpells, Book: e.Book, Equipment: equipment, Items: items, Stacks: stacks,
			SkillXP: e.SkillXP, Load: e.CurrentActorLoad(), PotionStats: e.PotionStats, Native: e.ActorLoad.Source.Class != 2,
			Saved: &mapload.Saved{Cell: mapload.Cell{X: e.X, Y: e.Y}, MapUnitID: e.MapUnitID,
				HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana,
				HealthRegenPeriod: e.HealthRegenPeriod, ManaRegenPeriod: e.ManaRegenPeriod}}
		for _, effect := range w.ActiveEffects() {
			if effect.Target == p.Entity && effect.Spell == 0 {
				state.Potion = &effect
				break
			}
		}
	}
	if p.Template != nil {
		var err error
		if err := p.bindTemplate(); err != nil {
			return mapload.PartyMember{}, err
		}
		state, err = p.templateState(t)
		if err != nil {
			return mapload.PartyMember{}, err
		}
	}
	return p.restoreFromState(state, t)
}

func (p currentPartyMember) restoreFromState(state *currentPartyState, t *mapload.Table) (mapload.PartyMember, error) {
	if (p.Member == nil) == (p.Policy == nil) {
		return mapload.PartyMember{}, fmt.Errorf("current party has absent or conflicting member policy")
	}
	if p.Policy != nil && (p.Base == nil || p.Weapon == nil) {
		return mapload.PartyMember{}, fmt.Errorf("current member has incomplete base or weapon policy")
	}
	out := p.restore()
	if p.ordinary == nil {
		if p.Policy != nil {
			return mapload.PartyMember{}, fmt.Errorf("current member policy lacks ordinary actor")
		}
		return out, nil
	}
	c := p.ordinary.Character
	if p.Template != nil {
		out.Name = c.Name
	}
	if c.Class != "Unit" {
		out.Name = c.Name
		out.Profile.Fighter = c.Basis.Human.Fighter
		out.Profile.Rider = data.RiderTypeID(int32(c.Basis.Human.TypeID))
		if !nativeCitySiegeMember(out) {
			typ, hired := originalMercenaryType(c, t)
			out.MercenaryType = 0
			if hired {
				out.MercenaryType = uint8(typ)
				out.Temporary, out.PlayerCharacter = true, false
			}
		}
		if out.Hired() && !nativeCitySiegeMember(out) {
			out.DefinitionRow = c.DefRow
			if out.Name == "" && t != nil && t.Humans != nil && int(c.DefRow) < t.Humans.Len() {
				out.Name = t.Humans.EntryName(int(c.DefRow))
			}
		}
	}
	out.StartingHero = c.Hero
	if p.Policy != nil {
		out.FigureFace = p.Policy.face(*p.ordinary)
	}
	if state == nil {
		if p.Policy != nil {
			return mapload.PartyMember{}, fmt.Errorf("current member policy lacks restored actor state")
		}
		// An entry template for an actor already removed from the World is
		// separate from the current body retained by the ordinary dead list.
		return out, nil
	}
	equipment, items, stacks := state.Equipment, state.Items, state.Stacks
	if nativeCitySiegeMember(out) {
		// A siege engine's weapon belongs to its Units row, which the mission
		// start re-derives. The member holds no worn items, as a fresh hire.
		equipment = [sim.EquipSlots]sim.ItemInstance{}
	}
	out.PotionEffect = state.Potion
	base := p.Base
	if base == nil && !state.Native {
		base = &currentPartyBase{}
	}
	if base != nil {
		if base.Native != state.Native {
			return mapload.PartyMember{}, fmt.Errorf("current party base arithmetic mode differs from actor state")
		}
		var err error
		out.Hero, err = base.restore(c, state.PotionStats)
		if err != nil {
			return mapload.PartyMember{}, err
		}
		repairHeroSkills(&out.Hero.Skill, c.SkillLevels, c.SkillXP)
	}
	out.Class, out.SuppressCorpseLoot = state.Class, state.SuppressCorpseLoot
	out.KnownSpells, out.Book = state.KnownSpells, state.Book
	switch state.Book.State {
	case sim.BookPresent, sim.BookNativePresent:
		out.SpellbookPresent = true
		out.SpellbookRestored = true
	case sim.BookAbsent, sim.BookNativeAbsent:
		out.SpellbookPresent = false
		out.SpellbookRestored = true
	case sim.BookLegacy:
	default:
		return mapload.PartyMember{}, fmt.Errorf("current party actor %d has invalid book mode", p.Entity)
	}
	out.WornItems = equipment
	for i, item := range equipment {
		out.Worn[i] = item.Code
	}
	out.CarriedItems, out.Carried = items, nil
	if items != nil {
		out.Carried = make([]uint16, len(items))
	}
	for i, item := range items {
		out.Carried[i] = item.Code
	}
	weapon := p.Weapon
	if weapon == nil {
		weapon = captureNamedPartyWeapon(out, equipment[0])
	}
	out.Weapon = weapon.restore(equipment[0], out.WeaponMaterialized, t)
	out.Weapon = canonicalMageWeapon(out.Weapon, out.Mage, t)
	if out.Carry != nil {
		out.Carry = &mapload.Carry{SkillXP: state.SkillXP, Items: out.Carried, ItemInstances: items,
			Equipped: out.Worn, EquippedItems: equipment, OrderedStacks: stacks, LiveLoad: state.Load}
	}
	if out.Saved != nil {
		out.Saved = state.Saved
	}
	human := p.Human
	if human == nil && out.OriginalHuman != nil {
		human = &currentPartyHuman{Retired: out.OriginalHuman.Retired}
	}
	if state.Native && p.Human == nil {
		out.OriginalHuman = nil
	} else if human != nil {
		h := cityHumanState(sav.CityCharacter{Stats: c.Stats, SkillXP: c.SkillXP, Experience: c.Experience}, c.Basis.Human)
		out.OriginalHuman = mapload.BindOriginalHuman(out, h)
		out.OriginalHuman.Retired = human.Retired || state.Native
	}
	return mapload.CloneParty([]mapload.PartyMember{out})[0], nil
}

func restoreCurrentPartyMembers(ms *Mission, a *currentActionData, t *mapload.Table) error {
	var party []mapload.PartyMember
	var ids []sim.EntityID
	var roster map[sim.EntityID]mapload.PartyMember
	for _, p := range a.Party {
		member, err := p.restoreFromCurrent(ms.World, t)
		if err != nil {
			return err
		}
		party, ids = append(party, member), append(ids, p.Entity)
	}
	if a.Roster != nil {
		roster = make(map[sim.EntityID]mapload.PartyMember, len(a.Roster))
	}
	for _, p := range a.Roster {
		member, err := p.restoreFromCurrent(ms.World, t)
		if err != nil {
			return err
		}
		roster[p.Entity] = member
	}
	// The saved party records carry no clothing layers; the mod mark's layers
	// were placed on the members the mission started with.
	for i := range party {
		keepLayers(&party[i], ms.Party)
	}
	for id, member := range roster {
		for _, old := range ms.Start.Roster {
			keepLayers(&member, []mapload.PartyMember{old})
		}
		roster[id] = member
	}
	if a.Party != nil {
		ms.Party, ms.Start.IDs = party, ids
	}
	if a.Roster != nil {
		ms.Start.Roster = roster
	}
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		for _, p := range rows {
			if p.Base == nil || !p.Base.Native || p.Template != nil {
				continue
			}
			member := roster[p.Entity]
			for i, id := range ids {
				if id == p.Entity {
					member = party[i]
				}
			}
			training := sim.NativeTraining{}
			if !p.Base.LegacyTraining {
				training = sim.NativeTraining{Present: true, Levels: member.Hero.Skill}
			}
			ms.World.SetNativeTraining(p.Entity, training)
			class := sim.NativeClass{}
			if !p.Base.LegacyClass && p.ordinary != nil && p.ordinary.Character.Class == "Human" {
				class = sim.NativeClass{Present: true, Fighter: member.Profile.Fighter}
			}
			ms.World.SetNativeClass(p.Entity, class)
		}
	}
	return nil
}

// keepLayers copies the clothing layers of the member of the same identity in
// from.
func keepLayers(member *mapload.PartyMember, from []mapload.PartyMember) {
	for _, old := range from {
		if old.ID == member.ID && len(old.Layers) != 0 {
			member.Layers = append([]uint16(nil), old.Layers...)
			return
		}
	}
}
