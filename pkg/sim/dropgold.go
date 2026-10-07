package sim

// KindPlayerDropGold is ROM1 player-command opcode 0x23. Player is the
// addressed roster slot, Group carries the full 32-bit gold amount and X and Y
// carry the requested cell. Like KindPlayerParameter it addresses a roster slot
// rather than an entity and is applied before entity lookup.
const KindPlayerDropGold uint8 = 0x23

// playerFirstActor is the index of the addressed roster slot's first living
// on-map actor, or -1 when the slot has none or its binding is ambiguous.
//
// With a saved Player registry the slot must name exactly one Player
// container; the first living bound member of that Player's Groups, in the
// maintained container order, is the actor. A slot named by two containers is
// ambiguous and has no first actor. Actors the registry does not bind, and
// worlds without a registry, use the maintained actor traversal order.
// Neither the current selection nor the smallest entity ID takes part.
func (w *World) playerFirstActor(slot uint32) int {
	if slot >= relationSlots {
		return -1
	}
	usable := func(i int) bool {
		e := w.entities[i]
		return e.Owner == slot && !e.OffMap && e.Alive()
	}
	if w.savedGroups != nil && w.savedGroups.PlayersPresent {
		var player uint32
		for _, p := range w.savedGroups.Players {
			if p.Slot != slot {
				continue
			}
			if player != 0 {
				return -1
			}
			player = p.ID
		}
		if player != 0 {
			for _, g := range w.savedGroups.Groups {
				if g.ContainerID != player {
					continue
				}
				for _, m := range g.Members {
					if i := indexOfEntity(w.entities, m.Entity); m.Bound && i >= 0 && usable(i) {
						return i
					}
				}
			}
		}
	}
	for _, id := range w.actorTraversalIDs() {
		if i := indexOfEntity(w.entities, id); i >= 0 && usable(i) {
			return i
		}
	}
	return -1
}

// applyPlayerDropGold debits the addressed player's purse and puts the gold on
// the ground. The request is admitted only for a positive amount the purse
// covers and a first actor standing on a cell that can hold a Sack; any other
// request changes nothing, so a refusal never costs gold.
//
// The requested cell is used when it is within two cells of the first actor on
// each axis and can hold a Sack; otherwise the first actor's own cell is used.
// A cell that is merely ground-blocked is kept, and no neighbouring cell is
// searched. The gold merges into a Sack already on the cell.
func (w *World) applyPlayerDropGold(player, amount uint32, at CellPoint) {
	if amount == 0 {
		return
	}
	i := w.playerFirstActor(player)
	if i < 0 || w.purses[player] < amount {
		return
	}
	x, y := w.entities[i].X, w.entities[i].Y
	if sackFault(w.bounds, Sack{X: x, Y: y}) != nil {
		return
	}
	dx, dy := at.X-x, at.Y-y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx <= 2 && dy <= 2 && sackFault(w.bounds, Sack{X: at.X, Y: at.Y}) == nil {
		x, y = at.X, at.Y
	}
	if w.savedObjects != nil {
		n := w.sourceMutationCopy(i)
		n.purses[player] -= amount
		if !n.putGroundObjectAt(i, x, y, amount, ItemStack{}) || !n.savedMutationValid() {
			return
		}
		*w = n
		return
	}
	w.purses[player] -= amount
	w.pourSack(x, y, amount, nil)
}
