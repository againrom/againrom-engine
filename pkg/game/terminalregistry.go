package game

import (
	"encoding/binary"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func reconcileSavedTerminalRegistry(ms *Mission) error {
	if ms.savedDocument == nil || ms.savedDocument.Document == nil {
		return nil
	}
	bindings := map[sim.EntityID]uint32{}
	for _, b := range ms.savedDocument.Actors {
		if b.ObjectIndex != 0 && int(b.ObjectIndex) <= len(ms.savedDocument.Document.Objects) {
			bindings[b.EntityID], _ = savedStructureValue(&ms.savedDocument.Document.Objects[b.ObjectIndex-1], "Identity")
		}
	}
	terminal, err := currentTerminalActorBindings(ms.savedDocument, ms.World)
	if err != nil {
		return err
	}
	for id, object := range terminal {
		bindings[id], _ = savedStructureValue(&ms.savedDocument.Document.Objects[object-1], "Identity")
	}
	ms.World.ReconcileTerminalActorRegistry(bindings)
	ms.World.ReconcileNativeActorRegistry(bindings)
	return nil
}

func projectTerminalActorRegistry(doc *sav.DocumentData, removed map[uint32]bool) error {
	keys := map[uint32]bool{}
	for _, index := range doc.DeadActors {
		if index == 0 || int(index) > len(doc.Objects) {
			continue
		}
		r := &doc.Objects[index-1]
		stage, _ := savedStructureValue(r, "Stage")
		health, _ := savedStructureValue(r, "Health")
		key, _ := savedStructureValue(r, "Identity")
		if int16(health) > -10 || stage < 2 && !removed[key] {
			continue
		}
		for _, field := range []string{"U50", "U54"} {
			value, err := savedMotionRaw(r, field, 4)
			if err != nil {
				return err
			}
			binary.LittleEndian.PutUint32(value, 16)
		}
		if key != 0 {
			keys[key] = true
		}
	}
	if doc.World == nil {
		return nil
	}
	touched, last := map[uint16]byte{}, map[uint16]int{}
	for i := range doc.World.Cells {
		c := &doc.World.Cells[i]
		last[c.Cell] = i
		if keys[c.GroundActor] {
			c.GroundActor = 0
			touched[c.Cell] |= 0x40
		}
		if keys[c.AirActor] {
			c.AirActor = 0
			touched[c.Cell] |= 0x80
		}
	}
	for i := range doc.World.Blocks {
		b := &doc.World.Blocks[i]
		mask := touched[b.Cell]
		if mask == 0 {
			continue
		}
		c := doc.World.Cells[last[b.Cell]]
		if c.GroundActor != 0 {
			mask &^= 0x40
		}
		if c.AirActor != 0 {
			mask &^= 0x80
		}
		b.Dyn &^= mask
	}
	return nil
}
