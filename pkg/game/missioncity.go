package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func (s *CampaignSession) missionCityProvenance(in townInstall, city, captured Snapshot, world *sim.World) (*sav.CityProvenance, error) {
	var provenance *sav.CityProvenance
	var err error
	// A current mission SAV is itself the producer's detached document. Its
	// generated marker proves that it has no imported source roster to retain;
	// project the live native state again at the town boundary instead of
	// treating its generated actor records as an original Human basis.
	if captured.SavedDocument != nil && !generatedCurrentMissionDocument(captured.SavedDocument) {
		state, e := snapshotSavedDocument(&Mission{World: world, savedDocument: captured.SavedDocument})
		if e != nil {
			return nil, e
		}
		if state == nil || state.Document == nil {
			return nil, originalCityUnsupportedf("mission city save has no complete source document")
		}
		seen := make(map[string]bool, len(city.Party))
		for _, kept := range city.Party {
			if kept.ID == "" || seen[kept.ID] {
				return nil, originalCityUnsupportedf("mission city roster has a missing or duplicate party ID")
			}
			seen[kept.ID] = true
		}
		var refs []uint16
		bound := make(map[string]bool, len(city.Party))
		for i, member := range s.live.mission.party {
			if !seen[member.ID] {
				continue
			}
			var ref uint16
			for _, binding := range state.Actors {
				if binding.EntityID == s.live.mission.ids[i] && !binding.Retired {
					if ref != 0 {
						return nil, originalCityUnsupportedf("mission city actor has repeated source binding")
					}
					ref = binding.ObjectIndex
				}
			}
			if ref == 0 {
				return nil, originalCityUnsupportedf("mission city actor has no exact source binding")
			}
			repairGeneratedHeroDefRow(state.Document, ref, member, world, s.live.mission.ids[i], in.table)
			refs = append(refs, ref)
			bound[member.ID] = true
		}
		provenance, err = sav.CityFromMissionDocument(*state.Document, refs)
		if err != nil {
			return nil, err
		}
		// A city.Party member the mission never carried in has no object the
		// frozen source document could ever bind. The one production route
		// here is a town-chapter companion grant (addChapterCompanions,
		// npc:22): it appends straight to s.Carried after the document was
		// already frozen at mission entry, so no amount of searching finds her
		// there. She is appended the same way the native branch below already
		// builds this exact companion when it has no live entity either: fresh,
		// from the installed NPC table alone. Any other unbound shape keeps a
		// refusal that names the member and why.
		if len(bound) != len(city.Party) {
			hero := nativeCityHeroOf(city.Party)
			data := provenance.Data()
			for _, member := range city.Party {
				if bound[member.ID] {
					continue
				}
				if !nativeCityGraftableMember(member) {
					return nil, originalCityUnsupportedf("mission city member %q has no exact source binding and no native construction path", member.ID)
				}
				var gerr error
				data, gerr = graftNativeCityMember(data, member, hero, in.table)
				if gerr != nil {
					return nil, fmt.Errorf("mission city member %q: %w", member.ID, gerr)
				}
			}
			provenance, err = sav.CityFromData(data)
		}
	} else {
		if s.Town.progress != nil {
			return nil, originalCityUnsupportedf("source campaign lacks its mission or city document")
		}
		current := make(map[string]sim.Entity)
		for i, member := range s.live.mission.party {
			for _, e := range world.Entities() {
				if e.ID == s.live.mission.ids[i] && e.Alive() {
					current[member.ID] = e
				}
			}
		}
		apply := func(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table) error {
			e, ok := current[member.ID]
			if !ok {
				if member.Carry == nil && member.CompanionNPC == 22 {
					return nativeCityApplyHuman(unit, member, table)
				}
				return originalCityUnsupportedf("native mission city lacks current member %s", member.ID)
			}
			return nativeMissionHuman(unit, member, table, e)
		}
		attach := func(objects []sav.CityObjectData, unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, owner uint32, seq *int) ([]sav.CityObjectData, error) {
			if _, exists := current[member.ID]; !exists && member.Carry == nil && member.CompanionNPC == 22 {
				return nativeCityAttachItems(objects, unit, member, table, owner, seq)
			}
			stacks, _ := world.CarriedStacks(current[member.ID].ID)
			return nativeMissionItems(objects, unit, member, table, owner, seq, stacks)
		}
		var cityData sav.CityData
		cityData, err = nativeCityDataConstruct(nativeCityRosterMembers(city.Party), nativeCityHumanHiredMembers(city.Party), in.table, city.Difficulty, attach, apply)
		if err == nil {
			err = nativeCityPlayerFormation(&cityData, captured.ApplicationState, world)
		}
		if err == nil {
			provenance, err = sav.CityFromData(cityData)
		}
	}
	return provenance, err
}

func generatedCurrentMissionDocument(state *SnapshotSAVDocument) bool {
	return state != nil && state.Document != nil && state.Document.Marker == sav.GeneratedCityMarker
}

// repairGeneratedHeroDefRow corrects a frozen mission document's own Human
// row for the starting hero when that row was never his: a generated first
// mission has no preceding town, so its walked-in hero is built by the same
// generic, many-to-one TypeID search every other party member uses
// (mapload.ConstructActorBasis), landing on a real but generic sibling row
// instead of his own chargen one. The corresponding live actor carries this
// same generic row as its own committed SourceBinding for the whole mission
// already played, so it is left alone; only the frozen document object this
// export reads back out is repaired, using the same data.ChargenBase lookup
// chargen itself used to seed him, off the StartingHero/FigureDir this party
// member has carried untouched since. A document object that already names
// a different row, or an actor that is not both StartingHero and Generated,
// is untouched: this is a targeted repair of one known gap, not a general
// rewrite of the document's own reported identity.
func repairGeneratedHeroDefRow(doc *sav.DocumentData, ref uint16, member mapload.PartyMember, world *sim.World, id sim.EntityID, t *mapload.Table) {
	if !member.StartingHero || doc == nil || ref == 0 || int(ref) > len(doc.Objects) || t == nil || t.Humans == nil {
		return
	}
	var generated bool
	for _, e := range world.Entities() {
		if e.ID == id {
			// Native current actors have no source binding at all. They are the
			// same source-free construction authority as an explicit generated
			// binding for this export repair; imported class-1/2 actors remain
			// protected from a chargen-row rewrite.
			generated = e.SourceBinding.Class == 0 || e.SourceBinding.Generated()
			break
		}
	}
	if !generated {
		return
	}
	fig := data.FigureDir(member.FigureDir)
	_, row, ok := data.ChargenBase(t.Humans, fig.Mage(), fig.Female())
	if !ok || row <= 0 || row > 255 {
		return
	}
	obj := &doc.Objects[ref-1]
	for i := range obj.Values {
		if obj.Values[i].Name == "T0C" {
			obj.Values[i].Value = uint32(row)
			return
		}
	}
}

func (f *FrontEnd) exportMissionCity(city, captured Snapshot, world *sim.World, label string) ([]byte, error) {
	if _, err := fameForOriginal(city); err != nil {
		return nil, err
	}
	provenance, err := f.missionCityProvenance(f.townInstall(), city, captured, world)
	if err != nil {
		return nil, err
	}
	shortcuts, err := quickSpellsToOriginalIndices(city.QuickSpells)
	if err != nil {
		return nil, err
	}
	if city.Gold < 0 || uint64(city.Gold) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("city purse is outside uint32")
	}
	update := sav.CityUpdate{Label: []byte(label), Money: uint32(city.Gold), Shortcuts: &shortcuts}
	for _, actor := range provenance.Roster() {
		u := originalCityBaselineUpdate(actor)
		u.Returned = true
		update.Characters = append(update.Characters, u)
	}
	var campaign sav.CampaignProjection
	if city.CampaignState {
		campaign = f.cityCampaignMarkerPaths(city.Campaign)
		campaign = f.campaignMapObjects(campaign)
		campaign.FirstMapPoint = true
	} else {
		campaign, err = nativeCampaignProjection(f, city)
		if err != nil {
			return nil, err
		}
	}
	update.Campaign = &campaign
	return provenance.Marshal(update)
}

func nativeMissionItems(objects []sav.CityObjectData, unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, owner uint32, seq *int, ordered []sim.ItemStack) ([]sav.CityObjectData, error) {
	if member.OriginalHuman != nil || member.Carry == nil || member.PotionEffect != nil {
		return nil, originalCityUnsupportedf("native mission city requires a normalized native member")
	}
	load := member.Carry.LiveLoad
	if load != nil {
		if err := load.Validate(); err != nil || load.Inventory.Source.Class != 0 {
			return nil, originalCityUnsupportedf("native mission city cannot construct retained source Human state")
		}
	}
	return nativeCityAttachItemsConstruct(objects, unit, member, table, owner, seq, load, ordered, true)
}

func nativeMissionHuman(unit *sav.CityUnitData, member mapload.PartyMember, table *mapload.Table, e sim.Entity) error {
	if e.SourceNow().Class != 0 || member.OriginalHuman != nil || member.Carry == nil || !e.Alive() || e.HP != e.MaxHP || e.Mana != e.MaxMana {
		return originalCityUnsupportedf("native mission city Human lacks its normalized native basis")
	}
	loadout := mapload.PartyLoadout(member, table)
	equipped := mapload.MemberItemEquipment(member, table)
	carried := mapload.MemberCarriedItems(member, table)
	mapload.NormalizeShieldLoadout(&equipped, &carried, table)
	mapload.ApplyItemEffects(&loadout, equipped, member.Profile.Fighter)
	d := member.Hero.RecomputeWithSkillXP(member.Profile, loadout, member.Carry.SkillXP)
	h, err := nativeCityHumanFromDerived(member, table, *unit, d, d.HealthMax, d.ManaMax)
	if err != nil {
		return err
	}
	// The live actor's pools are maintained state. XP awards can leave them
	// different from the next derive result; keep both the current values and
	// the genuine future derive inputs without manufacturing a modifier.
	for _, value := range []int32{e.HP, e.MaxHP, e.Mana, e.MaxMana} {
		if value < 0 || value > 65535 {
			return originalCityUnsupportedf("native mission city pool exceeds uint16")
		}
	}
	h.Health, h.HealthMax, h.Mana, h.ManaMax = uint16(e.HP), uint16(e.MaxHP), uint16(e.Mana), uint16(e.MaxMana)
	h.HealthPeriod, h.ManaPeriod = uint16(e.HealthRegenPeriod), uint16(e.ManaRegenPeriod)
	h.Attack.SecondBase, h.Attack.SecondSpread = e.SecondBase, e.SecondSpread
	if current := member.Carry.LiveLoad; current != nil {
		h.Weight, h.InventoryWeight, h.Load, h.Capacity = uint16(current.Inventory.OwnWeight), current.Inventory.Accumulator, uint16(int16(current.Load)), uint16(int16(current.Capacity))
		if current.Movement.Present {
			h.Speed = uint16(current.Movement.RawSpeed)
		}
	}
	if d.Combat.ToHit != e.ToHit || d.Combat.Defence != e.Defence || d.Combat.Absorption != e.Absorption || d.Skill != e.Skill {
		return originalCityUnsupportedf("native mission city Human combat basis differs from current values")
	}
	if err := nativeCityApplyHumanState(unit, member, h); err != nil {
		return err
	}
	// Reuse the current runtime field mapping used by source Humans.
	unit.Scalar2[28], unit.Scalar2[29] = e.HealthHundredths, e.ManaHundredths
	unit.Scalar2[34], unit.Scalar2[39], unit.Scalar2[40] = e.Reach, uint8(e.AttackCharge), uint8(e.AttackRelax)
	return nil
}
