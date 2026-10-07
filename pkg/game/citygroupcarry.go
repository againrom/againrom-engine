package game

import (
	"encoding/binary"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// cityHireGroup is the group a tavern hire adds under the Player (SAV-617).
// Every hire group in the lawful town corpus holds 80 AI bytes that are zero
// but for byte 0x45, which is 1, and the pointer at 0x4c, which LOAD replaces
// (SAV-GRPAI-563) and this writes as 0. The selector +1c has no constructor
// value (SAV-GRPALLOC-577); it is written 0, as the constructed group writes it.
func cityHireGroup() sav.CityGroupData {
	raw := make([]byte, 80)
	raw[0x45] = 1
	return sav.CityGroupData{Raw80: raw, F44: nativeCityPlayerIdentity}
}

// cityConstructedGroup is the group a town holds when nothing else places a
// party member: the group of a town with no recorded groups, and the group a
// member joining after the load enters.
func cityConstructedGroup() sav.CityGroupData {
	return sav.CityGroupData{Raw80: make([]byte, 80), F44: nativeCityPlayerIdentity}
}

// cityLiveGroup is one of the Player's groups as current town state: the
// group's own field values and the party members in it, in order. A member is
// named by its party ID, which is stable across maps, saves and loads.
type cityLiveGroup struct {
	Payload sav.CityGroupData
	Members []string
}

// reconcileCityGroups returns the live groups over the current party and the
// party index of each member, group by group. A member who left the party
// leaves its group, and a group left with no member is dropped. A member in no
// group joins one: a hire to the group of its mercenary type, appended in
// party order (SAV-617), any other member to the first group. DIV-1411 names
// the inferences in that placement.
func reconcileCityGroups(groups []cityLiveGroup, party []mapload.PartyMember) ([]cityLiveGroup, [][]int) {
	index := make(map[string]int, len(party))
	for i, p := range party {
		if _, repeated := index[p.ID]; !repeated && p.ID != "" {
			index[p.ID] = i
		}
	}
	placed := make([]bool, len(party))
	var out []cityLiveGroup
	var members [][]int
	for _, g := range groups {
		var held []int
		for _, id := range g.Members {
			if i, ok := index[id]; ok && !placed[i] {
				placed[i] = true
				held = append(held, i)
			}
		}
		if len(held) != 0 {
			out, members = append(out, cityLiveGroup{Payload: g.Payload}), append(members, held)
		}
	}
	if len(out) == 0 {
		out, members = append(out, cityLiveGroup{Payload: cityConstructedGroup()}), append(members, nil)
	}
	hireGroup := map[uint8]int{}
	for i, p := range party {
		if placed[i] {
			continue
		}
		g := 0
		if p.Hired() {
			k, ok := hireGroup[p.MercenaryType]
			if !ok {
				k = len(out)
				hireGroup[p.MercenaryType] = k
				out, members = append(out, cityLiveGroup{Payload: cityHireGroup()}), append(members, nil)
			}
			g = k
		}
		members[g] = append(members[g], i)
	}
	var kept []cityLiveGroup
	var order [][]int
	for g := range out {
		if len(members[g]) == 0 {
			continue
		}
		for _, i := range members[g] {
			out[g].Members = append(out[g].Members, party[i].ID)
		}
		kept, order = append(kept, out[g]), append(order, members[g])
	}
	return kept, order
}

// settleCityGroups records the groups over the current party as the town's
// live membership. It runs wherever the party changes: a hire or return at the
// tavern, a mission's return, a companion's arrival and a load.
func (t *Town) settleCityGroups(party []mapload.PartyMember) {
	if t == nil {
		return
	}
	t.cityGroups, _ = reconcileCityGroups(t.cityGroups, party)
}

// writeCityGroups replaces the constructed single town group with the live
// groups. Each group keeps its own field values and holds its members in live
// order, and a group left with no member is not written. The owner and group
// keys were remapped to the written Player or cleared when the group was
// recorded.
//
// It returns the party index of each written actor in group order, the order
// the document lists them in, or nil when the town has no single Player.
func writeCityGroups(city *sav.CityData, party []mapload.PartyMember, live []cityLiveGroup) []int {
	if city == nil || len(city.Players) != 1 {
		return nil
	}
	written := city.Objects[city.Players[0]-1].Player
	if written == nil || len(written.Groups) != 1 || len(written.Groups[0].Actors) != len(party) {
		return nil
	}
	actors := written.Groups[0].Actors
	groups, order := reconcileCityGroups(live, party)
	written.Groups = make([]sav.CityGroupData, len(groups))
	for g := range groups {
		p := groups[g].Payload
		out := sav.CityGroupData{Words20: slices.Clone(p.Words20), Raw80: slices.Clone(p.Raw80),
			Words3c: slices.Clone(p.Words3c), F1c: p.F1c, F40: p.F40, F44: p.F44}
		for _, i := range order[g] {
			out.Actors = append(out.Actors, actors[i])
		}
		written.Groups[g] = out
	}
	return slices.Concat(order...)
}

// loadedPlayerGroup is one group of a loaded town's Player with its field
// values remapped to the written Player and its actor references unresolved.
type loadedPlayerGroup struct {
	payload sav.CityGroupData
	actors  []uint16
}

// loadedPlayerGroups reads a loaded town's Player groups. The owner and group
// keys are remapped to the written Player or cleared, as LOAD clears a key
// that names nothing. It returns nil when the document has no single Player.
func loadedPlayerGroups(source sav.CityData) []loadedPlayerGroup {
	if len(source.Players) != 1 || source.Players[0] == 0 || int(source.Players[0]) > len(source.Objects) {
		return nil
	}
	player := source.Objects[source.Players[0]-1].Player
	if player == nil || len(player.Fixed) != 51 {
		return nil
	}
	key := binary.LittleEndian.Uint32(player.Fixed[47:51])
	remap := func(v uint32) uint32 {
		if v != 0 && v == key {
			return nativeCityPlayerIdentity
		}
		return 0
	}
	var out []loadedPlayerGroup
	for _, g := range player.Groups {
		out = append(out, loadedPlayerGroup{payload: sav.CityGroupData{Words20: slices.Clone(g.Words20), Raw80: slices.Clone(g.Raw80),
			Words3c: slices.Clone(g.Words3c), F1c: g.F1c, F40: remap(g.F40), F44: remap(g.F44)}, actors: g.Actors})
	}
	return out
}

// cityGroupsFromLoaded is the live membership of a town loaded from an
// original file: each loaded group with the party members it held, in loaded
// order, resolved through the provenance's identity bindings. It is nil when
// the provenance is unavailable.
func cityGroupsFromLoaded(party []mapload.PartyMember, loaded *SnapshotOriginalCity) []cityLiveGroup {
	if loaded == nil || loaded.Unavailable != "" {
		return nil
	}
	source := loaded.Document
	boundTo := map[uint32]string{}
	for _, b := range loaded.Bindings {
		boundTo[b.Identity] = b.PartyID
	}
	member := map[string]bool{}
	for _, p := range party {
		member[p.ID] = true
	}
	return cityGroupsOver(loadedPlayerGroups(source), func(a uint16) (string, bool) {
		if a == 0 || int(a) > len(source.Objects) || source.Objects[a-1].Unit == nil || len(source.Objects[a-1].Unit.Token) != 37 {
			return "", false
		}
		id := boundTo[binary.LittleEndian.Uint32(source.Objects[a-1].Unit.Token[29:33])]
		return id, member[id]
	})
}

// cityGroupsFromCurrent is the live membership of a town loaded from a current
// SAV: each saved group with the party members its actors are bound to. The
// binding is the document's own, so it holds with or without original
// provenance.
func cityGroupsFromCurrent(city sav.CityData, a *currentActionData) []cityLiveGroup {
	if a == nil {
		return nil
	}
	entityID := make(map[sim.EntityID]string, len(a.Party))
	for _, p := range a.Party {
		entityID[p.Entity] = string(p.ID)
	}
	objectID := map[uint16]string{}
	for _, b := range a.Bindings {
		if id, ok := entityID[b.ID]; ok && !b.Structure && !b.Missing {
			objectID[b.Object] = id
		}
	}
	return cityGroupsOver(loadedPlayerGroups(city), func(object uint16) (string, bool) {
		id, ok := objectID[object]
		return id, ok
	})
}

// cityGroupsOver builds live groups from loaded ones, naming each actor's
// member through resolve. A member is placed once, in the first group naming
// it, and a group with no member is dropped.
func cityGroupsOver(loaded []loadedPlayerGroup, resolve func(uint16) (string, bool)) []cityLiveGroup {
	var out []cityLiveGroup
	placed := map[string]bool{}
	for _, g := range loaded {
		live := cityLiveGroup{Payload: g.payload}
		for _, a := range g.actors {
			if id, ok := resolve(a); ok && !placed[id] {
				placed[id] = true
				live.Members = append(live.Members, id)
			}
		}
		if len(live.Members) != 0 {
			out = append(out, live)
		}
	}
	return out
}
