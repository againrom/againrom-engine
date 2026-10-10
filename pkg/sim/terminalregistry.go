package sim

import "encoding/binary"

func terminalRegistryActor(e *Entity) bool {
	return !e.Alive() && e.Decay >= DecayBones
}

func (w *World) detachTerminalRegistry(id EntityID, removed bool) {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].Alive() || w.entities[i].Decay < DecayBones && (!removed || w.entities[i].Decay == DecayNone) {
		return
	}
	w.clearTerminalRegistry(map[EntityID]bool{id: true}, map[uint32]bool{w.entities[i].SourceBinding.Identity: true})
}

// HERO-DEATH-026: terminal bodies release actor slots and stop actor dispatch.
func (w *World) ReconcileTerminalActorRegistry(bindings map[EntityID]uint32) {
	ids, keys := map[EntityID]bool{}, map[uint32]bool{}
	for _, e := range w.entities {
		if terminalRegistryActor(&e) {
			ids[e.ID], keys[e.SourceBinding.Identity] = true, true
		}
	}
	for _, d := range w.OriginalDeadActors() {
		if d.Current.Stage >= uint8(DecayBones) && d.Current.HP <= -10 {
			ids[d.ID], keys[d.Source.Identity] = true, true
		}
	}
	for _, d := range w.currentTerminalActors {
		if d.HP <= -10 {
			ids[d.ID] = true
		}
	}
	for id := range ids {
		keys[bindings[id]] = true
	}
	for _, e := range w.entities {
		if !terminalRegistryActor(&e) {
			delete(ids, e.ID)
			delete(keys, e.SourceBinding.Identity)
			delete(keys, bindings[e.ID])
		}
	}
	w.clearTerminalRegistry(ids, keys)
}

func (w *World) clearTerminalRegistry(ids map[EntityID]bool, keys map[uint32]bool) {
	delete(keys, 0)
	if w.savedMotion != nil {
		for _, c := range w.savedMotion.Cells {
			for _, slot := range []SavedActorSlot{c.Ground, c.Air} {
				if slot.Bound && ids[slot.Entity] && slot.Key != 0 {
					keys[slot.Key] = true
				}
			}
		}
		for i := range w.savedMotion.Motions {
			m := &w.savedMotion.Motions[i]
			if ids[m.Entity] {
				m.ActorAction = 16
			}
		}
	}
	touched := map[uint16]byte{}
	for i := range w.savedCellRecords {
		c := &w.savedCellRecords[i]
		for layer, slot := range []*SavedCellActorSlot{&c.Ground, &c.Air} {
			if keys[slot.Key] {
				*slot = SavedCellActorSlot{}
				touched[c.Cell] |= 0x40 << layer
			}
		}
	}
	if w.savedMotion != nil {
		for i := range w.savedMotion.Cells {
			c := &w.savedMotion.Cells[i]
			for layer, slot := range []*SavedActorSlot{&c.Ground, &c.Air} {
				if keys[slot.Key] {
					*slot = SavedActorSlot{}
					binary.LittleEndian.PutUint32(c.Payload[4+4*layer:], 0)
					touched[c.Cell] |= 0x40 << layer
				}
			}
		}
	}
	for cell, mask := range touched {
		if c := w.motionCell(cell); c != nil {
			if c.Ground.Key != 0 {
				mask &^= 0x40
			}
			if c.Air.Key != 0 {
				mask &^= 0x80
			}
		}
		for _, e := range w.entities {
			if !ids[e.ID] && !e.OffMap && e.OrdinaryTargetable() && entityCoversCell(&e, int32(cell&255), int32(cell>>8)) {
				mask &^= 0x40 << e.Domain.layer()
			}
		}
		if w.savedCellPlanes != nil {
			w.savedCellPlanes.Dynamic[cell] &^= mask
		}
		if w.savedMotion != nil {
			for i := range w.savedMotion.Blocks {
				if w.savedMotion.Blocks[i].Cell == cell {
					w.savedMotion.Blocks[i].Dyn &^= mask
				}
			}
		}
	}
	if len(touched) != 0 {
		w.refreshSavedPlaneBlocks()
	}
}
