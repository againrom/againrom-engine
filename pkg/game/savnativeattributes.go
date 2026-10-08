package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func projectCurrentPartyActorFields(state *SnapshotSAVDocument, world *sim.World, s Snapshot, table *mapload.Table) error {
	party := make(map[sim.EntityID]mapload.PartyMember, len(s.CurrentRoster)+len(s.Party))
	for id, member := range s.CurrentRoster {
		party[id] = member
	}
	for i, id := range s.CurrentPartyIDs {
		if i < len(s.Party) {
			party[id] = s.Party[i]
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
		mustSetText(&state.Document.Objects[binding.ObjectIndex-1], "Name", ordinaryActorName(member, table))
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
