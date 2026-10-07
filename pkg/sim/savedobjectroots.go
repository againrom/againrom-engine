package sim

import (
	"fmt"
	"slices"
)

// Locations returns every current occurrence, never a preferred owner.
func (r *SavedObjects) Locations(id SavedObjectID) []SavedItemLocation {
	var out []SavedItemLocation
	if r == nil {
		return out
	}
	for _, c := range r.Containers {
		for i, ref := range c.Items {
			if ref == id && id != 0 {
				out = append(out, SavedItemLocation{c.Owner, uint32(i)})
			}
		}
	}
	for _, root := range r.ItemRoots {
		if root.ID == id {
			out = append(out, SavedItemLocation{Owner: root.Owner})
		}
	}
	return out
}

func (r *SavedObjects) HasLocation(id SavedObjectID, location SavedItemLocation) bool {
	if r == nil || id == 0 {
		return false
	}
	if c := r.container(location.Owner); c != nil {
		return uint64(location.Index) < uint64(len(c.Items)) && c.Items[location.Index] == id
	}
	if location.Index != 0 {
		return false
	}
	return slices.Contains(r.ItemRoots, SavedItemRoot{id, location.Owner})
}

func (r *SavedObjects) HasRoot(id SavedObjectID, owner SavedObjectOwner) bool {
	return r.HasLocation(id, SavedItemLocation{Owner: owner})
}

func savedRootCompare(a, b SavedItemRoot) int {
	for _, pair := range [][2]uint64{{uint64(a.Owner.Kind), uint64(b.Owner.Kind)}, {uint64(a.Owner.Entity), uint64(b.Owner.Entity)}, {uint64(a.Owner.Object), uint64(b.Owner.Object)}, {uint64(a.Owner.Slot), uint64(b.Owner.Slot)}, {a.Owner.SessionHandle, b.Owner.SessionHandle}, {uint64(a.ID), uint64(b.ID)}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	return 0
}

func (r *SavedObjects) addRoot(id SavedObjectID, owner SavedObjectOwner) error {
	if savedOwnerFault(owner) != nil || owner.Kind != SavedOwnerActorWorn && owner.Kind != SavedOwnerSession {
		return fmt.Errorf("sim: invalid item root owner")
	}
	for _, root := range r.ItemRoots {
		if root.Owner == owner {
			return fmt.Errorf("sim: occupied item root")
		}
	}
	r.ItemRoots = append(r.ItemRoots, SavedItemRoot{id, owner})
	slices.SortFunc(r.ItemRoots, savedRootCompare)
	return nil
}

// AddItemRoot constructs an explicit non-container occurrence. The completed
// registry must pass Validate before it is installed or persisted.
func (r *SavedObjects) AddItemRoot(id SavedObjectID, owner SavedObjectOwner) error {
	if r == nil {
		return fmt.Errorf("sim: absent item registry")
	}
	return r.addRoot(id, owner)
}

func (r *SavedObjects) rootCount(id SavedObjectID) uint64 {
	return uint64(len(r.Locations(id))) + uint64(r.item(id).InFlight)
}

func (r *SavedObjects) uniqueLocation(id SavedObjectID, owner SavedObjectOwner) (SavedItemLocation, error) {
	var found SavedItemLocation
	count := 0
	for _, at := range r.Locations(id) {
		if at.Owner == owner {
			found = at
			count++
		}
	}
	if count != 1 {
		return found, fmt.Errorf("sim: item source requires an exact occurrence")
	}
	return found, nil
}

// MigrateLegacyOwners is an input-only, atomic conversion of the old exclusive
// Owner model. Current construction and persistence use explicit roots.
func (r *SavedObjects) MigrateLegacyOwners() error {
	if r == nil || r.Version != 1 {
		return fmt.Errorf("sim: expected legacy item ownership")
	}
	n := r.Clone()
	if len(n.ItemRoots) != 0 {
		return fmt.Errorf("sim: legacy registry has current roots")
	}
	owners := make(map[SavedObjectID]SavedObjectOwner)
	for _, row := range n.Items {
		if row.Owner == (SavedObjectOwner{}) || row.Retired || row.InFlight != 0 || savedOwnerFault(row.Owner) != nil {
			return fmt.Errorf("sim: invalid legacy item owner")
		}
		owners[row.ID] = row.Owner
	}
	seen := make(map[SavedObjectID]bool)
	for _, c := range n.Containers {
		for _, id := range c.Items {
			if id == 0 {
				continue
			}
			if seen[id] || owners[id] != c.Owner {
				return fmt.Errorf("sim: conflicting legacy item ownership")
			}
			seen[id] = true
		}
	}
	for i := range n.Items {
		row := &n.Items[i]
		switch row.Owner.Kind {
		case SavedOwnerActorPack, SavedOwnerSack:
			if !seen[row.ID] {
				return fmt.Errorf("sim: legacy item lacks its container")
			}
		case SavedOwnerActorWorn, SavedOwnerSession:
			if err := n.addRoot(row.ID, row.Owner); err != nil {
				return err
			}
		case SavedOwnerInFlight:
			row.InFlight = 1
		case SavedOwnerRetired:
			row.Retired = true
		}
		row.Owner = SavedObjectOwner{}
	}
	n.Version = SavedObjectsVersion
	if err := n.Validate(); err != nil {
		return err
	}
	*r = *n
	return nil
}

// RefreshChildLiveness counts node edges, including repeated Effect edges.
// Multiple roots of an Item do not duplicate that Item's outgoing edges.
func (r *SavedObjects) RefreshChildLiveness() {
	refs := make(map[SavedObjectID]bool)
	for _, book := range r.BookRoots {
		for _, id := range book.Slots {
			refs[id] = id != 0
		}
	}
	for _, item := range r.Items {
		if item.Retired {
			continue
		}
		for _, id := range item.Effects {
			refs[id] = true
		}
		if item.Spell != 0 {
			refs[item.Spell] = true
		}
	}
	for i := range r.Effects {
		v := &r.Effects[i]
		v.Retired = !refs[v.ID] && v.ExternalReferences == 0
	}
	for i := range r.Spells {
		v := &r.Spells[i]
		v.Retired = !refs[v.ID] && v.ExternalReferences == 0
	}
}

// ReserveExternal assigns a distinct session occurrence from the existing
// monotonic namespace. Allocator gaps are valid; a handle is not an Item ID.
func (r *SavedObjects) ReserveExternal(id SavedObjectID) (uint64, error) {
	var handle uint64
	err := r.mutate(func(n *SavedObjects) error {
		row := n.item(id)
		if row == nil || row.InFlight == 0 {
			return fmt.Errorf("sim: reservation requires an in-flight occurrence")
		}
		for {
			next, err := n.mint()
			if err != nil {
				return err
			}
			handle = uint64(next)
			used := false
			for _, root := range n.ItemRoots {
				used = used || root.Owner.Kind == SavedOwnerSession && root.Owner.SessionHandle == handle
			}
			if !used {
				break
			}
		}
		if err := n.addRoot(id, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: handle}); err != nil {
			return err
		}
		row.InFlight--
		return nil
	})
	return handle, err
}

func (r *SavedObjects) sessionHandle(id SavedObjectID, handles ...uint64) uint64 {
	if len(handles) == 1 {
		if r.HasRoot(id, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: handles[0]}) {
			return handles[0]
		}
		return 0
	}
	if len(handles) != 0 {
		return 0
	}
	var result uint64
	for _, at := range r.Locations(id) {
		if at.Owner.Kind == SavedOwnerSession {
			if result != 0 {
				return 0
			}
			result = at.Owner.SessionHandle
		}
	}
	return result
}
