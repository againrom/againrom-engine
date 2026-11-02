package game

import (
	"bytes"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The order words an actor's action links were saved with. While the written
// words are the ones the loaded record holds, the actor has run no order the
// World writes, and its links stand as LOAD read them.
var (
	loadedOrderRaw    = []string{"U50", "U54", "U58"}
	loadedOrderValues = []string{"U6C", "U136"}
)

// keepLoadedActionLinks writes each action link U5C, U60, U61 and U64 of a
// living actor that the World does not hold as its loaded record holds it.
// LOAD maps U5C and U64 through the archive identity map and keeps U60 and
// U61 raw (SAV-1262). The World holds U5C while it runs a target, all four
// while it runs a pending order, and U60 and U61 in a native basis. Each
// address is renamed to the key its target has in this document: an actor by
// its entity's written key, a dead actor by its record, a Spell of the
// actor's book by the Spell in the same book slot, and 0 for a target the
// document does not hold.
func keepLoadedActionLinks(state, loaded *SnapshotSAVDocument, world *sim.World, entities []sim.Entity, e sim.Entity, r *sav.DocumentRecordData, targetHeld bool) {
	source := loadedActorRecord(loaded, e.ID, r.Class)
	if source == nil || !sameLoadedOrderWords(r, source) {
		return
	}
	value := func(name string) uint32 {
		v, _ := savedStructureValue(source, name)
		return v
	}
	if key, _ := savedStructureValue(r, "U5C"); key == 0 && !targetHeld {
		key = loadedActorLinkKey(state, loaded, world, entities, value("U5C"))
		if key == 0 {
			key = loadedBodyReference(state, loaded, r, "U5C")
		}
		savedObjectSetValue(r, "U5C", key)
	}
	if e.PendingOrder.Kind != sim.PendingNone {
		return
	}
	if !e.NativeBasis.ScalarIsKnown(sim.ScalarU60) {
		savedObjectSetValue(r, "U60", value("U60"))
	}
	if !e.NativeBasis.ScalarIsKnown(sim.ScalarU61) {
		savedObjectSetValue(r, "U61", value("U61"))
	}
	savedObjectSetValue(r, "U64", loadedBookSpellKey(state, loaded, source, r, value("U64")))
}

// loadedActorRecord is the loaded record the entity was restored from.
func loadedActorRecord(loaded *SnapshotSAVDocument, id sim.EntityID, class string) *sav.DocumentRecordData {
	if loaded == nil || loaded.Document == nil {
		return nil
	}
	for _, a := range loaded.Actors {
		if a.EntityID == id && a.ObjectIndex != 0 && int(a.ObjectIndex) <= len(loaded.Document.Objects) {
			if source := &loaded.Document.Objects[a.ObjectIndex-1]; source.Class == class {
				return source
			}
			return nil
		}
	}
	return nil
}

func sameLoadedOrderWords(r, source *sav.DocumentRecordData) bool {
	raw := func(record *sav.DocumentRecordData, name string) []byte {
		for _, field := range record.Raw {
			if field.Name == name {
				return field.Bytes
			}
		}
		return nil
	}
	for _, name := range loadedOrderRaw {
		if a, b := raw(r, name), raw(source, name); a == nil || !bytes.Equal(a, b) {
			return false
		}
	}
	for _, name := range loadedOrderValues {
		a, errA := savedStructureValue(r, name)
		b, errB := savedStructureValue(source, name)
		if errA != nil || errB != nil || a != b {
			return false
		}
	}
	a, b := raw(r, "U158"), raw(source, "U158")
	return len(a) > 9 && len(b) > 9 && a[8] == b[8] && a[9] == b[9]
}

// loadedActorLinkKey is the written key of the actor the loaded Identity key
// names: the written key of the entity restored from that record, or the
// Identity of the dead record the World restored from it; otherwise 0.
func loadedActorLinkKey(state, loaded *SnapshotSAVDocument, world *sim.World, entities []sim.Entity, key uint32) uint32 {
	if key == 0 {
		return 0
	}
	for _, d := range world.OriginalDeadActors() {
		if d.Source.Identity != key {
			continue
		}
		root, err := currentDeadSourceRoot(state, d)
		if err != nil {
			return 0
		}
		written, _ := savedStructureValue(root, "Identity")
		return written
	}
	index := uint16(0)
	for i := range loaded.Document.Objects {
		if id, err := savedStructureValue(&loaded.Document.Objects[i], "Identity"); err == nil && id == key {
			if index != 0 {
				return 0
			}
			index = uint16(i + 1)
		}
	}
	if index == 0 {
		return 0
	}
	target := sim.EntityID(0)
	for _, a := range loaded.Actors {
		if a.ObjectIndex == index {
			target = a.EntityID
		}
	}
	if target == 0 && loaded.GroupBindings != nil {
		for _, m := range loaded.GroupBindings.Members {
			if m.Bound && m.ObjectIndex == index {
				target = m.EntityID
			}
		}
	}
	if target == 0 {
		return 0
	}
	written, err := savedCurrentOrderActorKey(state, entities, target)
	if err != nil {
		return 0
	}
	return written
}

// loadedBookSpellKey is the key of the written Spell in the book slot where
// the loaded actor's book holds the Spell keyed key, or 0. The book slot is
// the Spells position, as the book selection reads it.
func loadedBookSpellKey(state, loaded *SnapshotSAVDocument, source, r *sav.DocumentRecordData, key uint32) uint32 {
	if key == 0 {
		return 0
	}
	old, _ := savedObjectRefs(source, "Spells")
	refs, _ := savedObjectRefs(r, "Spells")
	for slot, ref := range old {
		if ref == 0 || int(ref) > len(loaded.Document.Objects) {
			continue
		}
		if this, err := savedStructureValue(&loaded.Document.Objects[ref-1], "This"); err != nil || this != key {
			continue
		}
		if slot >= len(refs) || refs[slot] == 0 || int(refs[slot]) > len(state.Document.Objects) {
			return 0
		}
		written, _ := savedStructureValue(&state.Document.Objects[refs[slot]-1], "This")
		return written
	}
	return 0
}
