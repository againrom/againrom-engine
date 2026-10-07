package sav

import "fmt"

// CityFromMissionDocument projects explicitly selected current Human objects
// from their own World document into a city. Local object references, Player
// ownership and source records select the graph; item values never identify
// an object. The caller supplies already-normalized current Human/item fields.
// Mission terrain, other actors and their exclusively owned graph are omitted.
func CityFromMissionDocument(data DocumentData, actorIndices []uint16) (*CityProvenance, error) {
	d, err := saveDocumentFromData(data)
	if err != nil {
		return nil, err
	}
	if d.archive.world == nil || len(actorIndices) == 0 {
		return nil, fmt.Errorf("sav: mission city projection needs a World and survivors")
	}
	// Build the same grammar records with an explicit local-index table. This
	// second, bounded graph avoids inferring a source index from an address key
	// or from the archive writer's encounter order.
	objects := make([]*Record, len(data.Objects))
	for i, r := range data.Objects {
		objects[i] = newRecord(r.Class, 0, 0)
	}
	for i, r := range data.Objects {
		if err := documentRecordFromData(r, objects[i], objects, false, 0); err != nil {
			return nil, err
		}
	}
	selected := make(map[*Record]bool)
	for _, index := range actorIndices {
		if index == 0 || int(index) > len(objects) {
			return nil, fmt.Errorf("sav: city survivor outside document")
		}
		r := objects[index-1]
		if r.Class != "Human" || selected[r] || r.Counts["Effects"] != 0 {
			return nil, fmt.Errorf("sav: city survivor is duplicated, non-Human or still affected")
		}
		selected[r] = true
		// The ordinary kept-actor boundary releases this mission reference.
		r.RefSlots["U68"], r.Refs["U68"] = []*Record{nil}, nil
	}
	var player *Record
	for _, index := range data.Players {
		if index == 0 {
			continue
		}
		candidate := objects[index-1]
		if candidate.Value["Participant"] == 0 {
			if player != nil {
				return nil, fmt.Errorf("sav: mission has multiple human participants")
			}
			player = candidate
		}
	}
	if player == nil {
		return nil, fmt.Errorf("sav: mission has no human participant")
	}
	seen := make(map[*Record]bool)
	var groups []*Record
	var heroName string
	for _, group := range player.Groups {
		var actors []*Record
		for _, actor := range group.RefSlots["Actors"] {
			if !selected[actor] {
				continue
			}
			if seen[actor] || actor.Value["Reference"] != player.Value["This"] {
				return nil, fmt.Errorf("sav: city survivor has ambiguous Player ownership")
			}
			seen[actor] = true
			actors = append(actors, actor)
			if actor.Value["Identity"] == player.Value["Hero"] {
				heroName = actor.Text["Name"]
			}
		}
		if len(actors) != 0 {
			group.RefSlots["Actors"], group.Refs["Actors"] = actors, actors
			group.Counts["Actors"] = len(actors)
			groups = append(groups, group)
		}
	}
	if len(seen) != len(selected) || heroName == "" {
		return nil, fmt.Errorf("sav: city survivors lack their exact Player group or starting hero")
	}
	player.Groups, player.Counts["Groups"] = groups, len(groups)
	player.Refs["Actors"] = nil
	player.Counts["Actors"] = 0
	for _, group := range groups {
		player.Refs["Actors"] = append(player.Refs["Actors"], group.RefSlots["Actors"]...)
		player.Counts["Actors"] += group.Counts["Actors"]
	}
	// The completed mission's outcome and placement latch do not enter the town.
	// Campaign records stay unchanged until the caller's town projection.
	player.Value["Outcome"] = 0
	player.Value["F3D"] = 0
	d.archive.players, d.archive.dead, d.archive.world = []*Record{player}, nil, nil
	d.archive.head.Mission, d.archive.head.MapName = 0, ""
	d.archive.marker, d.archive.global = GeneratedCityMarker, 0
	body, err := serializeArchiveDocument(d.archive)
	if err != nil {
		return nil, err
	}
	city, err := parseCityDocument(body)
	if err != nil {
		return nil, err
	}
	state, err := cityStateFromData(NewCityStateData(heroName), 2)
	if err != nil {
		return nil, err
	}
	return newCityProvenance(d.version, city, state, cloneCityCampaign(d.campaign))
}
