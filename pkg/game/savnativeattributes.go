package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// projectCurrentPartyActorFields writes the fields a party actor's record takes
// from its current party or roster member.
//
// A party hero's T0E is the member's own type through the party mint's builder,
// not a type the loaded document or an earlier world carried. A SAV holds the
// server type id (HERO-191) and the original draws a hero's class and sex from
// it (HERO-APPEAR-041). A hired member, every actor outside the party and a
// game without the rule keep their projected T0E.
func projectCurrentPartyActorFields(state *SnapshotSAVDocument, world *sim.World, s Snapshot, table *mapload.Table) error {
	party := make(map[sim.EntityID]mapload.PartyMember, len(s.CurrentRoster)+len(s.Party))
	heroes := make(map[sim.EntityID]bool, len(s.Party))
	heroType := campaignOf(s.game).files().heroSavedType
	for id, member := range s.CurrentRoster {
		party[id] = member
	}
	for i, id := range s.CurrentPartyIDs {
		if i < len(s.Party) {
			party[id] = s.Party[i]
			heroes[id] = heroType && s.Party[i].PlayerCharacter && !s.Party[i].Hired()
		}
	}
	for _, binding := range state.Actors {
		e, present := world.Entity(binding.EntityID)
		if binding.Retired || !present || !e.Humanoid {
			continue
		}
		member, found := party[e.ID]
		if !found {
			continue
		}
		record := &state.Document.Objects[binding.ObjectIndex-1]
		mustSetText(record, "Name", ordinaryActorName(member, table))
		if heroes[e.ID] && record.Class == "Human" {
			if err := savedActorSetValue(record, "T0E", uint32(member.ServerTypeID())); err != nil {
				return err
			}
		}
		if e.ActorLoad.Source.Class != 0 {
			continue
		}
		// Native Body lives in the captured roster, worn items and earned gains.
		items, ok := world.EquippedItems(e.ID)
		if !ok {
			continue
		}
		loadout, ok := mapload.ResolveItemLoadout(items, member.Weapon, member.WeaponMaterialized, table)
		if !ok {
			continue
		}
		mapload.ApplyItemEffects(&loadout, items, member.Profile.Fighter)
		mapload.AddLayers(&loadout, member.Layers, table)
		hero := mapload.PotionHero(member.Hero, e.PotionStats)
		derived := hero.RecomputeWithSkillXP(member.Profile, loadout, e.SkillXP)
		if err := savedActorSetValue(&state.Document.Objects[binding.ObjectIndex-1], "Body", uint32(uint16(derived.Body))); err != nil {
			return err
		}
	}
	return nil
}
