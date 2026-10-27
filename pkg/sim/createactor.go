package sim

import "fmt"

// actorSearch finds the cell a created actor stands on. n is the candidate
// World holding the actor at index i, off the map; a search may draw from n's
// stream, and those draws stay drawn when the creation fails.
type actorSearch func(n *World, i int) (x, y int32, ok bool)

// createActor is the one runtime builder of an actor and its group: the cheat
// summon and the second game's Summon spell both reach it. It gives template
// the next entity id, binds a source actor to a generated source key, adds its
// holdings, places it on the cell search finds, and puts it alone in a new
// group: under the Player container of owner's actor standing at the anchor
// in a saved-group World, as a new command group in a native one. Nothing
// changes when it fails, except that keepDraws keeps the draws the candidate
// took: a cast that drew its placement has drawn it.
func (w *World) createActor(template Entity, pack []ItemStack, worn [EquipSlots]ItemInstance, anchorX, anchorY int32, search actorSearch, keepDraws bool) (EntityID, error) {
	id, available := w.NextEntityID()
	if !available || id >= 1<<28 {
		return 0, fmt.Errorf("sim: actor creation identity space exhausted")
	}
	template.ID = id
	template.ActorLoad.Source.HasOwner = template.Owner != 0 && template.ActorLoad.Source.Class != 0
	if template.ActorState == 0 {
		template.ActorState = actorStateGuard
	}
	template.OffMap = true
	template.HasTarget = false
	template.clearAttack()
	template.SourceBinding.ArchiveIndex = 0
	template.SourceBinding.GroupIndex, template.SourceBinding.GroupSelector, template.SourceBinding.GroupOwnerKey = 0, 0, 0
	template.SourceBinding.GroupOwnerSlot, template.SourceBinding.GroupOwnerResolved = 0, false
	if template.ActorLoad.Source.Class != 0 {
		template.SourceBinding.Class = GeneratedUnitBinding
		if template.Humanoid {
			template.SourceBinding.Class = GeneratedHumanBinding
		}
		key := uint32(0x50000000) + uint32(id)*16
		for w.cheatActorKeyUsed(key) {
			key += 16
			if key == 0 {
				return 0, fmt.Errorf("sim: actor creation source key space exhausted")
			}
		}
		template.SourceBinding.Identity, template.SourceBinding.RuntimeID = key, w.allocateRuntimeID()
	} else {
		template.SourceBinding = SourceBinding{}
	}
	var units uint64
	for _, stack := range pack {
		units += uint64(stack.Count)
		if stack.Count > 65535 || units > MaxOriginalHoldingValues {
			return 0, fmt.Errorf("sim: created actor holdings exceed bounded quantity")
		}
	}
	if !w.cheatHoldingCapacity(pack, &worn, false) {
		return 0, fmt.Errorf("sim: created actor holdings exceed bounded item/effect values")
	}
	held, equipped, err := normaliseHoldings([]Entity{template}, []Stock{{ID: id, OrderedStacks: pack, ItemInstances: expandItems(pack), EquippedItems: worn}})
	if err != nil {
		return 0, err
	}
	n := w.sourceMutationCopy(0)
	n.actorTraversal = append([]EntityID(nil), w.actorTraversal...)
	n.casts = append([]scriptCast(nil), w.casts...)
	n.entities = append(n.entities, template)
	n.constructFacing(&n.entities[len(n.entities)-1])
	n.constructMover(n.entities[len(n.entities)-1])
	n.routes = append(n.routes, nil)
	n.carried = append(n.carried, held[0])
	n.equipment = append(n.equipment, equipped[0])
	i := len(n.entities) - 1
	px, py, found := search(&n, i)
	if !found || !n.placeAt(i, px, py) {
		w.keepCandidateDraws(&n, keepDraws)
		return 0, fmt.Errorf("sim: created actor found no free cell")
	}
	n.entities[i].PostX, n.entities[i].PostY = px, py
	if template.ActorLoad.Source.Class != 0 && !n.deriveSource(i) {
		w.keepCandidateDraws(&n, keepDraws)
		return 0, fmt.Errorf("sim: created source actor cannot derive owned state")
	}
	if !n.groupDialectNative() {
		container, owner := w.cheatSummonOwner(template.Owner, anchorX, anchorY)
		if !n.newSavedCommandGroup([]int{i}, orderNone, cell{x: px, y: py}, true, container, false) {
			w.keepCandidateDraws(&n, keepDraws)
			return 0, fmt.Errorf("sim: created actor saved group identity space exhausted")
		}
		if owner.Class == 1 {
			n.savedGroupFor(id).Owner = owner
		}
		n.syncSavedActorCommand(i)
	} else {
		n.groups = append([]groupAI(nil), w.groups...)
		n.commandGroup([]int{i}, orderNone, cell{x: px, y: py})
		if template.Owner != 0 && n.entities[i].CommandGroup == 0 {
			w.keepCandidateDraws(&n, keepDraws)
			return 0, fmt.Errorf("sim: created actor native group identity space exhausted")
		}
		n.entities[i].Group = n.entities[i].CommandGroup
	}
	if _, err := n.MarshalBinary(); err != nil {
		w.keepCandidateDraws(&n, keepDraws)
		return 0, err
	}
	*w = n
	return id, nil
}

func (w *World) keepCandidateDraws(n *World, keep bool) {
	if keep {
		w.rng = n.rng
	}
}

// nearestFreeCell is the cheat summon's search: the free cell nearest the
// anchor by squared distance, the first in row order on a tie.
func nearestFreeCell(x, y int32) actorSearch {
	return func(n *World, i int) (int32, int32, bool) {
		best := int64(1<<63 - 1)
		px, py := int32(0), int32(0)
		for cy := int32(0); cy < n.bounds.Height; cy++ {
			for cx := int32(0); cx < n.bounds.Width; cx++ {
				dx, dy := int64(cx-x), int64(cy-y)
				distance := dx*dx + dy*dy
				if distance < best && n.placeFree(i, cx, cy) {
					best, px, py = distance, cx, cy
				}
			}
		}
		return px, py, best != int64(1<<63-1)
	}
}
