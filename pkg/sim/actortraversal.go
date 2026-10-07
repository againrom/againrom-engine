package sim

import (
	"fmt"
	"slices"
)

func (w *World) actorTraversalIDs() []EntityID {
	if w.actorTraversal != nil {
		return w.actorTraversal
	}
	var ids []EntityID
	for _, e := range w.entities {
		if !e.OffMap {
			ids = append(ids, e.ID)
		}
	}
	return ids
}

func (w *World) ActorTraversal() []EntityID { return slices.Clone(w.actorTraversalIDs()) }

func (w *World) RestoreActorTraversal(ids []EntityID) error {
	if len(ids) > len(w.entities) || len(ids) > 65535 {
		return fmt.Errorf("sim: oversized on-map actor traversal")
	}
	seen := make(map[EntityID]bool, len(ids))
	for _, id := range ids {
		i := indexOfEntity(w.entities, id)
		if i < 0 || w.entities[i].OffMap || seen[id] {
			return fmt.Errorf("sim: invalid on-map actor traversal")
		}
		seen[id] = true
	}
	for _, e := range w.entities {
		if !e.OffMap && !seen[e.ID] {
			return fmt.Errorf("sim: incomplete on-map actor traversal")
		}
	}
	w.actorTraversal = append([]EntityID{}, ids...)
	return nil
}

func (w *World) unlinkActorTraversal(id EntityID) {
	w.actorTraversal = slices.DeleteFunc(append([]EntityID{}, w.actorTraversalIDs()...), func(v EntityID) bool { return v == id })
}

func (w *World) restoreActionActorTraversal(ids []EntityID) error {
	if len(ids) > len(w.entities) || len(ids) > 65535 {
		return fmt.Errorf("sim: oversized on-map actor traversal")
	}
	seen := make(map[EntityID]bool, len(ids))
	order := []EntityID{}
	for _, id := range ids {
		i := indexOfEntity(w.entities, id)
		if i < 0 || seen[id] {
			return fmt.Errorf("sim: invalid on-map actor traversal")
		}
		seen[id] = true
		if !w.entities[i].OffMap {
			order = append(order, id)
		}
	}
	for _, e := range w.entities {
		if !e.OffMap && !seen[e.ID] {
			order = append(order, e.ID)
		}
	}
	w.actorTraversal = order
	return nil
}

func (w *World) appendActorTraversal(id EntityID) {
	ids := append([]EntityID{}, w.actorTraversalIDs()...)
	if !slices.Contains(ids, id) {
		ids = append(ids, id)
	}
	w.actorTraversal = ids
}

// AI-363. Actors absent from the retained group graph follow in identity order.
func (w *World) RebuildLoadedActorTraversal() {
	if w.savedGroups == nil {
		w.actorTraversal = nil
		return
	}
	ids := []EntityID{}
	appendGroup := func(g SavedGroup) {
		for _, m := range g.Members {
			i := indexOfEntity(w.entities, m.Entity)
			if m.Bound && i >= 0 && !w.entities[i].OffMap && !slices.Contains(ids, m.Entity) {
				ids = append(ids, m.Entity)
			}
		}
	}
	if w.savedGroups.PlayersPresent {
		for _, p := range w.savedGroups.Players {
			for _, g := range w.savedGroups.Groups {
				if g.ContainerID == p.ID {
					appendGroup(g)
				}
			}
		}
	} else {
		for _, g := range w.savedGroups.Groups {
			appendGroup(g)
		}
	}
	for _, e := range w.entities {
		if !e.OffMap && !slices.Contains(ids, e.ID) {
			ids = append(ids, e.ID)
		}
	}
	w.actorTraversal = ids
}
