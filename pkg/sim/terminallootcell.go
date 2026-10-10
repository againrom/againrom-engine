package sim

// DIV-1441: only an impassable origin relocates. Ordinary ground deaths keep
// their sack merge policy. Rings are ordered by distance, then Y and X.
func (w *World) terminalLootCell(i int) (int32, int32, bool) {
	e := w.entities[i]
	if w.terminalLootGround(e.X, e.Y) {
		return e.X, e.Y, true
	}
	if _, inside := w.cellIndex(e.X, e.Y); !inside {
		return e.X, e.Y, false
	}
	limit := max(e.X, e.Y, w.bounds.Width-1-e.X, w.bounds.Height-1-e.Y)
	for distance := int32(1); distance <= limit; distance++ {
		left, right, top, bottom := e.X-distance, e.X+distance, e.Y-distance, e.Y+distance
		if top >= 0 {
			if x, ok := w.terminalLootRow(i, left, right, top); ok {
				return x, top, true
			}
		}
		if left >= 0 || right < w.bounds.Width {
			for y := max(int32(0), top+1); y <= min(w.bounds.Height-1, bottom-1); y++ {
				if left >= 0 && w.terminalLootFree(i, left, y) {
					return left, y, true
				}
				if right < w.bounds.Width && w.terminalLootFree(i, right, y) {
					return right, y, true
				}
			}
		}
		if bottom < w.bounds.Height {
			if x, ok := w.terminalLootRow(i, left, right, bottom); ok {
				return x, bottom, true
			}
		}
	}
	return e.X, e.Y, false
}

func (w *World) terminalLootRow(self int, left, right, y int32) (int32, bool) {
	for x := max(int32(0), left); x <= min(w.bounds.Width-1, right); x++ {
		if w.terminalLootFree(self, x, y) {
			return x, true
		}
	}
	return 0, false
}

func (w *World) terminalLootGround(x, y int32) bool {
	return w.terrainOpen(DomainGround, x, y) && !w.groundBlockedSackCell(x, y)
}

func (w *World) terminalLootFree(self int, x, y int32) bool {
	if !w.terminalLootGround(x, y) || w.sackAt(x, y) {
		return false
	}
	for j, e := range w.entities {
		if j == self || e.Domain.layer() != DomainGround.layer() {
			continue
		}
		if m := w.motionFor(e.ID); m != nil && m.Current {
			if w.motionOwnsCell(e.ID, DomainGround.layer(), x, y) {
				return false
			}
		} else if counted(&e) && entityCoversCell(&e, x, y) {
			return false
		}
	}
	return true
}

func (w *World) hasTerminalLoot(i int) bool {
	e := w.entities[i]
	if e.TypeID > 0x40 && e.GoldChance > 0 && (e.TreasureMin != 0 || e.TreasureMax != 0) {
		return true
	}
	if e.SuppressCorpseLoot {
		return false
	}
	if len(w.carried[i]) != 0 {
		return true
	}
	for slot, item := range w.equipment[i] {
		if item.Empty() || slot == slotWeapon && item.innateWeapon() || slot >= 2 && e.ActorLoad.Source.Class != 0 && e.ActorLoad.Source.Class != 2 {
			continue
		}
		return true
	}
	return false
}
