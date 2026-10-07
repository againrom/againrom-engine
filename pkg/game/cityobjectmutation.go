package game

import (
	"fmt"
	"slices"

	"againrom/pkg/sim"
)

type cityItemLocationKind uint8

const (
	cityItemDetached cityItemLocationKind = iota
	cityItemPack
	cityItemWorn
)

// Pack destinations insert at Index after any source removal. Worn indices
// are zero-based. Detached is a caller-attested table or in-flight occurrence;
// it has no party or index and may share its Item with other live roots.
type cityItemLocation struct {
	PartyID string
	Kind    cityItemLocationKind
	Index   int
}

func cityMutationCandidate(g *cityObjectTopology) (*cityObjectTopology, error) {
	if g == nil {
		return nil, fmt.Errorf("city object topology is absent")
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g.Clone(), nil
}

func cityMutationParty(g *cityObjectTopology, partyID string) (*cityPartyObjectRoots, error) {
	for i := range g.Roots {
		if string(g.Roots[i].PartyID) == partyID {
			return &g.Roots[i], nil
		}
	}
	return nil, fmt.Errorf("city object location has missing party identity")
}

func cityMutationLocation(g *cityObjectTopology, at cityItemLocation) (*cityPartyObjectRoots, error) {
	if at.Kind == cityItemDetached {
		if at.PartyID != "" || at.Index != 0 {
			return nil, fmt.Errorf("detached city object location has party or index")
		}
		return nil, nil
	}
	if at.Kind != cityItemPack && at.Kind != cityItemWorn || at.Index < 0 {
		return nil, fmt.Errorf("invalid city object location")
	}
	return cityMutationParty(g, at.PartyID)
}

func cityMutationSource(g *cityObjectTopology, id sim.SavedObjectID, at cityItemLocation) (int, error) {
	item := -1
	for i := range g.Items {
		if g.Items[i].ID == id {
			item = i
			break
		}
	}
	if id == 0 || item < 0 {
		return -1, fmt.Errorf("city item mutation has missing item identity")
	}
	root, err := cityMutationLocation(g, at)
	if err != nil {
		return -1, err
	}
	switch at.Kind {
	case cityItemPack:
		if at.Index >= len(root.Pack) || root.Pack[at.Index] != id {
			return -1, fmt.Errorf("city item mutation has stale pack source")
		}
	case cityItemWorn:
		if at.Index >= len(root.Worn) || root.Worn[at.Index] != id {
			return -1, fmt.Errorf("city item mutation has stale equipment source")
		}
	}
	return item, nil
}

func cityMutationInsert(g *cityObjectTopology, id sim.SavedObjectID, at cityItemLocation) error {
	root, err := cityMutationLocation(g, at)
	if err != nil {
		return err
	}
	switch at.Kind {
	case cityItemPack:
		if at.Index > len(root.Pack) {
			return fmt.Errorf("city item destination exceeds pack bounds")
		}
		root.Pack = slices.Insert(root.Pack, at.Index, id)
	case cityItemWorn:
		if at.Index >= len(root.Worn) || root.Worn[at.Index] != 0 {
			return fmt.Errorf("city item destination is outside equipment or occupied")
		}
		root.Worn[at.Index] = id
	}
	return nil
}

// Only the selected edge moves. Other aliases, null positions and all child
// identities survive, including nodes temporarily detached onto a shop table.
func moveCityItemRoot(g *cityObjectTopology, id sim.SavedObjectID, from, to cityItemLocation) (*cityObjectTopology, error) {
	n, err := cityMutationCandidate(g)
	if err != nil {
		return nil, err
	}
	if _, err := cityMutationSource(n, id, from); err != nil {
		return nil, err
	}
	root, _ := cityMutationLocation(n, from)
	switch from.Kind {
	case cityItemPack:
		root.Pack = slices.Delete(root.Pack, from.Index, from.Index+1)
	case cityItemWorn:
		root.Worn[from.Index] = 0
	}
	if err := cityMutationInsert(n, id, to); err != nil {
		return nil, err
	}
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return n, nil
}

// The caller has already established a partial quantity move from current
// values. Each non-null Effect occurrence gets a separate child, even when
// the source repeats one Effect ID. No quantity or child value is stored here.
func splitCityItemRoot(g *cityObjectTopology, id sim.SavedObjectID, from, to cityItemLocation) (*cityObjectTopology, sim.SavedObjectID, error) {
	n, err := cityMutationCandidate(g)
	if err != nil {
		return nil, 0, err
	}
	index, err := cityMutationSource(n, id, from)
	if err != nil {
		return nil, 0, err
	}
	source := n.Items[index]
	copy := cityItemTopology{}
	copy.ID, err = mintCityObjectID(&n.NextID)
	if err != nil {
		return nil, 0, err
	}
	if source.Effects != nil {
		copy.Effects = make([]sim.SavedObjectID, len(source.Effects))
	}
	for i, child := range source.Effects {
		if child == 0 {
			continue
		}
		copy.Effects[i], err = mintCityObjectID(&n.NextID)
		if err != nil {
			return nil, 0, err
		}
		n.Effects = append(n.Effects, copy.Effects[i])
	}
	if source.Spell != 0 {
		copy.Spell, err = mintCityObjectID(&n.NextID)
		if err != nil {
			return nil, 0, err
		}
		n.Spells = append(n.Spells, copy.Spell)
	}
	n.Items = append(n.Items, copy)
	if err := cityMutationInsert(n, copy.ID, to); err != nil {
		return nil, 0, err
	}
	if err := n.Validate(); err != nil {
		return nil, 0, err
	}
	return n, copy.ID, nil
}

func cityBookSpellShared(g *cityObjectTopology, id sim.SavedObjectID) bool {
	references := 0
	for _, row := range g.Items {
		if row.Spell == id {
			references++
		}
	}
	for _, row := range g.Books {
		for _, child := range row.Slots {
			if child == id {
				references++
			}
		}
	}
	return references > 1
}

// Prepare one independently edited book slot. Existing unique spells retain
// identity; a null slot or a shared spell receives a fresh node. Scalar changes
// remain the caller's responsibility and must commit with this candidate.
func prepareCityBookMutation(g *cityObjectTopology, partyID string, slot int, expectedID sim.SavedObjectID) (*cityObjectTopology, sim.SavedObjectID, error) {
	n, err := cityMutationCandidate(g)
	if err != nil {
		return nil, 0, err
	}
	if slot < 0 || slot >= len(cityBookTopology{}.Slots) {
		return nil, 0, fmt.Errorf("city book mutation has invalid slot")
	}
	if _, err := cityMutationParty(n, partyID); err != nil {
		return nil, 0, err
	}
	book := -1
	for i := range n.Books {
		if string(n.Books[i].PartyID) == partyID {
			book = i
			break
		}
	}
	if book < 0 {
		n.Books = append(n.Books, cityBookTopology{PartyID: []byte(partyID)})
		book = len(n.Books) - 1
	}
	if n.Books[book].Slots[slot] != expectedID {
		return nil, 0, fmt.Errorf("city book mutation has stale spell source")
	}
	id := expectedID
	if id == 0 || cityBookSpellShared(n, id) {
		id, err = mintCityObjectID(&n.NextID)
		if err != nil {
			return nil, 0, err
		}
		n.Spells = append(n.Spells, id)
	}
	n.Books[book].Slots[slot] = id
	if err := n.Validate(); err != nil {
		return nil, 0, err
	}
	return n, id, nil
}
