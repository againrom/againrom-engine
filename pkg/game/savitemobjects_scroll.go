package game

import (
	"fmt"

	"againrom/pkg/sim"
)

// A native scroll reservation has an exact owner and value, but no established
// original archive edge. Its registry row remains live; zero ObjectIndex plus
// this marker means detached projection, not object retirement.
const savedActiveScrollUnavailable = "active native scroll has no proven SAV session edge"

const savedItemCountUnavailable = "native Item count exceeds the SAV word"
const savedContainerDetachedUnavailable = "native container includes an Item without a complete SAV projection"

func savedDetachedCoverage(reason string) bool {
	return reason == savedActiveScrollUnavailable || reason == savedItemCountUnavailable
}

func currentItemHasOrdinaryRoot(registry *sim.SavedObjects, id sim.SavedObjectID) bool {
	for _, at := range registry.Locations(id) {
		if at.Owner.Kind != sim.SavedOwnerSession {
			return true
		}
	}
	return false
}

func savedDetachedObjects(world *sim.World, registry *sim.SavedObjects) (map[sim.SavedObjectID]bool, map[sim.SavedObjectID]string, error) {
	detached := make(map[sim.SavedObjectID]bool)
	reasons := make(map[sim.SavedObjectID]string)
	var pairs []sim.SavedExternalItem
	for _, cast := range world.ScrollCasts() {
		id := cast.Item.ObjectID
		if id == 0 {
			continue // An explicitly unbound native item has no archive identity.
		}
		pairs = append(pairs, sim.SavedExternalItem{Handle: cast.Reservation, ID: id, Value: sim.StackItem(cast.Item, 1)})
		if !currentItemHasOrdinaryRoot(registry, id) {
			detached[id] = true
			reasons[id] = savedActiveScrollUnavailable
		}
	}
	// This also refuses any other external session value: a matching numeric
	// handle alone is not a game-owned reservation.
	if err := registry.ValidateExternal(pairs); err != nil {
		return nil, nil, fmt.Errorf("saved SAV scroll reservation differs: %w", err)
	}
	projectedChildren, detachedChildren := make(map[sim.SavedObjectID]bool), make(map[sim.SavedObjectID]bool)
	childReasons := make(map[sim.SavedObjectID]string)
	for _, item := range registry.Items {
		if item.Retired {
			continue
		}
		if item.Value.Count > 65535 {
			detached[item.ID] = true
			reasons[item.ID] = savedItemCountUnavailable
		}
		refs := projectedChildren
		if detached[item.ID] {
			refs = detachedChildren
		}
		for _, id := range item.Effects {
			refs[id] = true
			if detached[item.ID] && (childReasons[id] == "" || reasons[item.ID] < childReasons[id]) {
				childReasons[id] = reasons[item.ID]
			}
		}
		if item.Spell != 0 {
			refs[item.Spell] = true
			if detached[item.ID] && (childReasons[item.Spell] == "" || reasons[item.ID] < childReasons[item.Spell]) {
				childReasons[item.Spell] = reasons[item.ID]
			}
		}
	}
	for _, row := range registry.Effects {
		if !row.Retired && row.ExternalReferences == 0 && detachedChildren[row.ID] && !projectedChildren[row.ID] {
			detached[row.ID] = true
			reasons[row.ID] = childReasons[row.ID]
		}
	}
	for _, row := range registry.Spells {
		if !row.Retired && row.ExternalReferences == 0 && detachedChildren[row.ID] && !projectedChildren[row.ID] {
			detached[row.ID] = true
			reasons[row.ID] = childReasons[row.ID]
		}
	}
	return detached, reasons, nil
}

func savedCurrentContainerCoverage(c sim.SavedObjectContainer, indices map[sim.SavedObjectID]uint16) string {
	for _, id := range c.Items {
		if id != 0 && indices[id] == 0 {
			return savedContainerDetachedUnavailable
		}
	}
	return savedObjectCoverage(c.Coverage)
}
