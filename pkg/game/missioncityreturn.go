package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Keep the current source graph when an imported World becomes a town. The
// active-mission SAV writer already performs this projection; abandoning it
// here made the later town SAVE incorrectly use the new-campaign constructor.
func (s *CampaignSession) retainMissionCity(in townInstall, mission int, world *sim.World) error {
	source, err := snapshotSavedDocument(s.live.mission.state)
	if err != nil {
		return err
	}
	city := Snapshot{Party: s.Carried, Difficulty: s.Difficulty}
	provenance, err := s.missionCityProvenance(in, city, Snapshot{SavedDocument: source}, world)
	if err != nil {
		return err
	}
	characters, err := provenance.SourceParty()
	if err != nil {
		return err
	}
	// The baseline must be reproducible from the retained document alone, as
	// on a cold original-city LOAD. Live carried values remain separate.
	baseline, report := restorePartyCharacters(append([]sav.Character(nil), characters...), nil, nil, in.bodies, in.table)
	if report.Err != nil || report.Fallback {
		return fmt.Errorf("cannot bind returned source roster: %v", report.Err)
	}
	persistent := characters[:0]
	for _, c := range characters {
		if persistentOriginalCharacter(c, in.table) {
			persistent = append(persistent, c)
		}
	}
	persistent, _, _ = leadFirst(persistent)
	mapload.NameParty(baseline)
	clearRestoredMissionPosition(baseline)
	state := bindOriginalCity(provenance, persistent, baseline, nil, in.table)
	if state.unavailable != nil {
		return state.unavailable
	}
	packs := captureCityItemGraphs(s.live.mission.party, world, s.live.mission.ids)
	worn := captureCityEquipmentGraphs(s.live.mission.party, world, s.live.mission.ids)
	if len(state.bindings) != len(s.Carried) {
		return fmt.Errorf("returned source roster differs from current party")
	}
	for i := range state.bindings {
		binding := &state.bindings[i]
		var member *mapload.PartyMember
		for j := range s.Carried {
			if s.Carried[j].ID == binding.partyID {
				if member != nil {
					return fmt.Errorf("returned party identity repeats")
				}
				member = &s.Carried[j]
			}
		}
		if member == nil {
			return fmt.Errorf("returned source identity %s is missing", binding.partyID)
		}
		if member.Carry == nil {
			// The graph just constructed this town grant. Bind its complete
			// source basis before the next mission's strict entry join.
			if exact, err := binding.replayedMember(*member, in.campaign); err != nil || !exact {
				return fmt.Errorf("new city grant %s differs from its constructor: %v", member.ID, err)
			}
			*member = mapload.CloneParty([]mapload.PartyMember{binding.baseline})[0]
			continue
		}
		returned := &SnapshotCityReturn{Version: 1, Identity: binding.character.Identity,
			Mission: mission, Member: mapload.CloneParty([]mapload.PartyMember{*member})[0]}
		if member.Carry.LiveLoad != nil {
			if graph := packs[member.ID]; graph != nil {
				returned.Version, returned.Holdings = 2, graph
				if equipped := worn[member.ID]; equipped != nil {
					returned.Version, returned.Worn = 3, equipped
				}
			}
		}
		if err := binding.checkReturn(returned); err != nil {
			return fmt.Errorf("%s: %w", member.ID, err)
		}
		binding.returned = returned
	}
	state.captureSession(s)
	// Offered is a transient arrival advisory. A cold city import has none;
	// citySaveAdvisory independently admits the current settled-town value.
	state.baselineOffered = 0
	s.originalCity = state
	return nil
}
