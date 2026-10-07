package game

import (
	"fmt"
	"reflect"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// SnapshotCityReturn is mutable earned state with its own admission rules.
// It never replaces the semantic source or the independently rebuilt baseline.
// Runtime actor IDs and map/action/effect references do not cross this boundary.
type SnapshotCityReturn struct {
	Version  uint32
	Identity uint32
	Mission  int
	Member   mapload.PartyMember
	Holdings *sav.CityItemGraph
	// Worn is only ever present alongside Holdings (Version 3): a member whose
	// pack capture did not succeed falls back to the strict legacy comparison
	// for both container and equipment topology, rather than mixing a loose
	// pack with a strict equipped set or the reverse.
	Worn *sav.CityEquipmentGraph
}

func cloneCityReturn(r *SnapshotCityReturn) *SnapshotCityReturn {
	if r == nil {
		return nil
	}
	n := *r
	n.Member = mapload.CloneParty([]mapload.PartyMember{r.Member})[0]
	n.Holdings = cloneCityItemGraph(r.Holdings)
	n.Worn = cloneCityEquipmentGraph(r.Worn)
	return &n
}

// returnDerived drops the member fields that follow from what he wears, so
// no comparison of two members decides on them. The written return carries
// the current worn set, and every reader derives these fields again from it.
//
// Weapon is slot 1's weapon record, and the starting-weapon latch only
// decides whether an empty slot 1 is drawn holding Weapon; the latch is set
// through its one writer so both sides carry the same value.
//
// Body, BodyDir and Class of a composed member are data.HeroAppearance of
// the figure equipment and Mage, which the mission build, the town faces and
// a SAV restore each recompute. The retained party still holds the values
// computed when the mission opened. A hired member's Class is his row.
func returnDerived(p mapload.PartyMember) mapload.PartyMember {
	p.Weapon = nil
	materializeStartingWeapon(&p)
	if !p.Hired() {
		p.Body, p.BodyDir, p.Class = "", "", 0
	}
	return p
}

func returnItems(items []sim.ItemInstance) []sim.ItemInstance {
	if len(items) == 0 {
		return nil
	}
	out := make([]sim.ItemInstance, len(items))
	for i, item := range items {
		out[i] = item.Clone()
		out[i].ObjectID = 0
	}
	return out
}

// checkReturn admits a captured return as a well-formed record of the
// member's current state: its identity, its graphs against its own member,
// no world handles and a normalized current Human. It does not compare the
// member with the document; the town writer writes whatever changed.
func (b originalCityBinding) checkReturn(r *SnapshotCityReturn) error {
	if r == nil || r.Version < 1 || r.Version > 3 || r.Identity != b.character.Identity || r.Member.ID != b.partyID ||
		(r.Version >= 2) != (r.Holdings != nil) || (r.Version == 3) != (r.Worn != nil) {
		return fmt.Errorf("city return has invalid source identity or version")
	}
	if _, ok := MissionMap(r.Mission); !ok {
		return fmt.Errorf("city return has invalid mission")
	}
	p := r.Member
	if p.Carry == nil || p.Hired() ||
		p.Saved != nil || p.OriginalHuman != nil || p.PotionEffect != nil {
		return fmt.Errorf("city return changed unsupported character structure")
	}
	// A retained original Human has a live class-2 load to validate. A native
	// current member deliberately has no such load: its current Hero, XP and
	// equipment are the producer's authority and the town writer derives the
	// SAV Human from those values. Both shapes are valid return records.
	var source *sim.SourceActor
	if a := p.Carry.LiveLoad; a != nil {
		if err := a.Validate(); err != nil {
			return err
		}
		s := a.Inventory.Source
		if s.Class != 2 || !sim.InPersistBand(int32(s.TypeID)) || !s.EquipmentRuntimePresent || !a.Movement.Present {
			return fmt.Errorf("city return has no source Human load")
		}
		source = &s
	}
	if r.Holdings != nil {
		if err := checkCityItemGraph(r.Holdings, p); err != nil {
			return err
		}
	}
	if r.Worn != nil {
		if err := checkCityEquipmentGraph(r.Worn, p); err != nil {
			return err
		}
	}
	if err := cityWorldHandles(p); err != nil {
		return err
	}
	if source != nil {
		a := p.Carry.LiveLoad
		h := mapload.SourceHumanState(*source, a.Inventory.Accumulator)
		if p.Hero != h.Hero() || source.Stats[8] != source.Stats[9] || source.Stats[11] != source.Stats[12] || int16(source.Stats[8]) <= 0 ||
			source.Stats[4] != uint16(a.Movement.RawSpeed) || source.Stats[5] != uint16(a.Inventory.OwnWeight) ||
			source.Stats[6] != uint16(a.Load) || source.Stats[7] != uint16(a.Capacity) {
			return fmt.Errorf("city return has inconsistent or unnormalized current Human values")
		}
		for i, xp := range p.Carry.SkillXP {
			if uint32(xp) != source.SkillXP[i] {
				return fmt.Errorf("city return XP differs from current Human")
			}
		}
	}
	return p.Book.Validate(p.KnownSpells)
}

// cityWorldHandles refuses a local registry handle inside a town member.
func cityWorldHandles(p mapload.PartyMember) error {
	if p.Carry == nil {
		return nil
	}
	for _, item := range p.Carry.ItemInstances {
		if item.ObjectID != 0 {
			return fmt.Errorf("city return has a world item handle")
		}
	}
	for _, item := range p.Carry.EquippedItems {
		if item.ObjectID != 0 {
			return fmt.Errorf("city return has a world equipment handle")
		}
	}
	for _, item := range p.Carry.OrderedStacks {
		if item.ObjectID != 0 {
			return fmt.Errorf("city return has a world stack handle")
		}
	}
	return nil
}

// captureMissionReturn can mint this representation only at the real frontend
// boundary. The join is character.Identity -> binding.partyID -> mission.party
// -> mission.ids -> live Entity. Neither a name nor a coincident runtime ID is
// an alternative join. A failed admission leaves the ordinary AGS path intact.
func (s *CampaignSession) captureMissionReturn(n int, party []mapload.PartyMember, w *sim.World, ids []sim.EntityID, normalized bool, packs map[string]*sav.CityItemGraph, worn map[string]*sav.CityEquipmentGraph) {
	state := s.originalCity
	if state == nil {
		return
	}
	prior := make(map[string]*SnapshotCityReturn, len(state.bindings))
	for i := range state.bindings {
		b := &state.bindings[i]
		prior[b.partyID] = b.returned
		b.returned = nil
	}
	// A member the mission changed has no Returned reset to write until its
	// return is captured, so the town keeps .ags, also across an AGS LOAD.
	uncaptured := fmt.Errorf("original-compatible town save has a mission return it could not capture")
	if w != nil && state.unavailable == nil {
		state.unavailable = uncaptured
	}
	document, semantic := state.document.(*sav.CityProvenance)
	if !semantic || state.partyImportVersion < 6 {
		return
	}
	if !normalized || w == nil || s.live == nil || s.live.world != w || s.liveMission != n || s.live.mission == nil ||
		!reflect.DeepEqual(party, s.live.mission.party) || !reflect.DeepEqual(ids, s.live.mission.ids) ||
		len(party) != len(ids) || len(party) != len(state.bindings) || len(s.Carried) != len(state.bindings) {
		return
	}
	byID := make(map[string]int, len(party))
	used := make(map[sim.EntityID]bool, len(ids))
	for i, p := range party {
		if p.ID == "" || used[ids[i]] {
			return
		}
		if _, dup := byID[p.ID]; dup {
			return
		}
		byID[p.ID], used[ids[i]] = i, true
	}
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	current := make(map[string]mapload.PartyMember)
	for _, p := range s.Carried {
		if _, dup := current[p.ID]; dup {
			return
		}
		current[p.ID] = p
	}
	var returns []*SnapshotCityReturn
	for _, b := range state.bindings {
		source, err := document.Human(b.character.Identity)
		if err != nil || source.AttachedEffects {
			return
		}
		i, ok := byID[b.partyID]
		if !ok {
			return
		}
		entry, err := b.expectedMember()
		if before := prior[b.partyID]; before != nil {
			if b.checkReturn(before) != nil {
				return
			}
			entry = before.Member
		}
		(&mapWorld{}).resolveWeaponMaterialized(&entry, mapload.EquipmentFromParty(entry))
		start := mapload.CloneParty([]mapload.PartyMember{party[i]})[0]
		(&mapWorld{}).resolveWeaponMaterialized(&start, mapload.EquipmentFromParty(start))
		if err != nil || !reflect.DeepEqual(entry, start) || party[i].StartingHero != b.character.Hero {
			return
		}
		e, ok := entities[ids[i]]
		p, carried := current[b.partyID]
		if !ok || !carried || !e.Alive() || e.Owner != sim.SelfSlot || !sim.InPersistBand(e.TypeID) || !e.Humanoid ||
			p.Carry == nil || !reflect.DeepEqual(p.Carry.LiveLoad, e.CurrentActorLoad()) {
			return
		}
		for _, effect := range w.ActiveEffects() {
			if effect.Target == e.ID {
				return
			}
		}
		r := &SnapshotCityReturn{Version: 1, Identity: b.character.Identity, Mission: n, Member: mapload.CloneParty([]mapload.PartyMember{p})[0]}
		if p.Carry.LiveLoad != nil {
			if graph := packs[b.partyID]; graph != nil {
				r.Version, r.Holdings = 2, graph
				if equipped := worn[b.partyID]; equipped != nil {
					r.Version, r.Worn = 3, equipped
				}
			}
		}
		if b.checkReturn(r) != nil {
			return
		}
		returns = append(returns, r)
	}
	for i := range state.bindings {
		state.bindings[i].returned = returns[i]
	}
	if state.unavailable == uncaptured {
		state.unavailable = nil
	}
}
