package sim

import "encoding/binary"

func (w *World) ReconcileNativeActorRegistry(bindings map[EntityID]uint32) {
	w.syncNativeActorCells(bindings)
}

func (w *World) syncNativeActorCells(bindings map[EntityID]uint32) {
	if w.savedMotion == nil || w.savedCellPlanes == nil {
		return
	}
	actors := map[EntityID]int{}
	keys := map[EntityID]uint32{}
	unresolved := false
	for i, e := range w.entities {
		m := w.motionFor(e.ID)
		if m == nil || m.Current || !e.Alive() {
			continue
		}
		need := e.OffMap
		if !need {
			for y := int32(0); y < footprintSide(e.TokenSize) && !need; y++ {
				for x := int32(0); x < footprintSide(e.TokenSize); x++ {
					c := w.motionCell(uint16(e.Y+y)<<8 | uint16(e.X+x))
					if c == nil || !motionSlot(c, e.Domain.layer()).Bound || motionSlot(c, e.Domain.layer()).Entity != e.ID {
						need = true
						break
					}
				}
			}
		}
		if !need {
			continue
		}
		actors[e.ID], keys[e.ID] = i, e.SourceBinding.Identity
		if keys[e.ID] == 0 {
			keys[e.ID] = bindings[e.ID]
		}
		unresolved = unresolved || keys[e.ID] == 0
	}
	if len(actors) == 0 {
		return
	}
	if unresolved {
		for _, c := range w.savedMotion.Cells {
			for _, slot := range []SavedActorSlot{c.Ground, c.Air} {
				if _, ok := actors[slot.Entity]; ok && slot.Bound && keys[slot.Entity] == 0 {
					keys[slot.Entity] = slot.Key
				}
			}
		}
	}
	byKey := map[uint32]EntityID{}
	for _, e := range w.entities {
		if _, ok := actors[e.ID]; !ok {
			continue
		}
		if keys[e.ID] == 0 {
			delete(actors, e.ID)
		} else {
			byKey[keys[e.ID]] = e.ID
		}
	}
	if len(actors) == 0 {
		return
	}
	changed := false
	for i := range w.savedMotion.Cells {
		c := &w.savedMotion.Cells[i]
		for layer, slot := range []*SavedActorSlot{&c.Ground, &c.Air} {
			id, ok := byKey[slot.Key]
			if !ok || slot.Bound && slot.Entity != id {
				continue
			}
			e := w.entities[actors[id]]
			if !e.OffMap && layer == e.Domain.layer() && entityCoversCell(&e, int32(c.Cell&255), int32(c.Cell>>8)) {
				continue
			}
			*slot = SavedActorSlot{}
			changed = true
			binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0)
			w.syncCurrentCellActor(c.Cell, layer, *slot)
			reserved := false
			for _, m := range w.savedMotion.Motions {
				if m.Entity == e.ID {
					continue
				}
				other := indexOfEntity(w.entities, m.Entity)
				if other >= 0 && w.entities[other].Domain.layer() == layer && w.motionReserves(&m, int32(c.Cell&255), int32(c.Cell>>8)) {
					reserved = true
					break
				}
			}
			if !reserved {
				w.savedCellPlanes.Dynamic[c.Cell] &^= 0x40 << layer
			}
		}
	}
	for _, e := range w.entities {
		if _, ok := actors[e.ID]; !ok || e.OffMap || keys[e.ID] == 0 {
			continue
		}
		for y := int32(0); y < footprintSide(e.TokenSize); y++ {
			for x := int32(0); x < footprintSide(e.TokenSize); x++ {
				key := uint16(e.Y+y)<<8 | uint16(e.X+x)
				if w.createSavedCell(key) != "" {
					continue
				}
				c := w.motionCell(key)
				slot := motionSlot(c, e.Domain.layer())
				if slot.Key != 0 && slot.Key != keys[e.ID] {
					continue
				}
				if slot.Bound && slot.Entity == e.ID && w.savedCellPlanes.Dynamic[key]&(0x20|0x40<<e.Domain.layer()) == 0x20|0x40<<e.Domain.layer() {
					continue
				}
				changed = true
				*slot = SavedActorSlot{Key: keys[e.ID], Entity: e.ID, Bound: true}
				binary.LittleEndian.PutUint32(c.Payload[4+4*e.Domain.layer():], slot.Key)
				w.syncCurrentCellActor(key, e.Domain.layer(), *slot)
				w.savedCellPlanes.Dynamic[key] |= 0x20 | 0x40<<e.Domain.layer()
			}
		}
	}
	if changed {
		w.refreshSavedPlaneBlocks()
	}
}
