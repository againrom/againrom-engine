package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Values are compared only within the same party slot. They select changed
// edges, never an identity: the topology supplies that slot's exact source ID.
func prepareCityBookChanges(g *cityObjectTopology, before, next mapload.PartyMember, table *mapload.Table) (*cityObjectTopology, error) {
	if g == nil {
		return nil, nil
	}
	n, err := cityMutationCandidate(g)
	if err != nil {
		return nil, err
	}
	if before.ID == "" || next.ID != before.ID {
		return nil, fmt.Errorf("city book mutation changed party identity")
	}
	if err := before.Book.Validate(before.KnownSpells); err != nil {
		return nil, err
	}
	if err := next.Book.Validate(next.KnownSpells); err != nil {
		return nil, err
	}
	if _, err := cityMutationParty(n, before.ID); err != nil {
		return nil, err
	}
	oldValues, newValues := cityMemberBook(before, table), cityMemberBook(next, table)
	for slot, oldValue := range oldValues {
		if oldValue == newValues[slot] {
			continue
		}
		book := slices.IndexFunc(n.Books, func(b cityBookTopology) bool { return string(b.PartyID) == before.ID })
		var expected sim.SavedObjectID
		if book >= 0 {
			expected = n.Books[book].Slots[slot]
		}
		if oldValue.Present != (expected != 0) {
			return nil, fmt.Errorf("city book mutation has missing or stale slot %d", slot)
		}
		if !newValues[slot].Present {
			n.Books[book].Slots[slot] = 0
			continue
		}
		n, _, err = prepareCityBookMutation(n, before.ID, slot, expected)
		if err != nil {
			return nil, err
		}
	}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

func prepareCityPartyBookChanges(g *cityObjectTopology, before, next []mapload.PartyMember, table *mapload.Table) (*cityObjectTopology, error) {
	if len(before) != len(next) {
		return nil, fmt.Errorf("city book action changed party membership")
	}
	n := g
	for i, member := range next {
		if before[i].ID != member.ID {
			return nil, fmt.Errorf("city book action changed party order")
		}
		var err error
		n, err = prepareCityBookChanges(n, before[i], member, table)
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func cloneCityBookHistory(s *originalCitySaveState) *originalCitySaveState {
	if s == nil {
		return nil
	}
	n := *s
	n.bindings = slices.Clone(s.bindings)
	for i := range n.bindings {
		n.bindings[i].training = slices.Clone(s.bindings[i].training)
	}
	return &n
}

func (s *CampaignSession) cityBookCandidate() *CampaignSession {
	n := *s
	n.Carried = mapload.CloneParty(s.Carried)
	town := *s.Town
	town.cityObjects = s.Town.cityObjects.Clone()
	n.Town = &town
	if s.Shop != nil {
		n.Shop = cloneShopMutation(s.Shop)
		if s.Shop.table == nil {
			n.Shop.table = nil
		}
		for i := range s.Shop.shelves {
			if s.Shop.shelves[i] == nil {
				n.Shop.shelves[i] = nil
			}
		}
	}
	n.originalCity = cloneCityBookHistory(s.originalCity)
	return &n
}

func (s *CampaignSession) commitCityBookCandidate(n *CampaignSession) {
	copy(s.Carried, n.Carried)
	s.Town.gold, s.Town.cityObjects = n.Town.gold, n.Town.cityObjects
	if s.Shop != nil && n.Shop != nil {
		*s.Shop = *n.Shop
	}
	if s.originalCity != nil && n.originalCity != nil {
		*s.originalCity = *n.originalCity
	}
}

// Run the existing value operation on detached current state. Graph admission
// precedes publishing the party, purse, table and training history together.
func (t *shopController) withCityBookMutation(action func(*shopController) ui.TownAction) ui.TownAction {
	return cityBookMutation(t, func(c *shopController) **CampaignSession { return &c.sess }, t.in, action)
}

func (t *townScreen) withCityBookMutation(action func(*townScreen) ui.TownAction) ui.TownAction {
	return cityBookMutation(t, func(c *townScreen) **CampaignSession { return &c.sess }, t.in, action)
}

// cityBookMutation runs action on a copy of t whose session is a candidate,
// then publishes the candidate with the graph's book changes.
func cityBookMutation[T any](t *T, sess func(*T) **CampaignSession, in *InstallResources, action func(*T) ui.TownAction) ui.TownAction {
	s := *sess(t)
	if s.Town == nil || s.Town.cityObjects == nil {
		return action(t)
	}
	if err := s.Town.cityObjects.Validate(); err != nil {
		return ui.TownAction{Msg: err.Error()}
	}
	n := *t
	*sess(&n) = s.cityBookCandidate()
	result := action(&n)
	c := *sess(&n)
	graph, err := prepareCityPartyBookChanges(c.Town.cityObjects, s.Carried, c.Carried, in.Table)
	if err != nil {
		return ui.TownAction{Msg: "cannot update city book: " + err.Error()}
	}
	c.Town.cityObjects = graph
	s.commitCityBookCandidate(c)
	*sess(&n) = s
	*t = n
	return result
}
