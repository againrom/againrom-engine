package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// An absent World setting may still have a current, unambiguous actor owner
// setting. Read that value before using the existing native constructor.
func currentActorOwnerReserve(w *sim.World, slot uint32) uint32 {
	value, found := uint32(95), false
	for _, e := range w.Entities() {
		s := e.ActorLoad.Source
		if e.Owner != slot || s.Class == 0 || !s.HasOwner {
			continue
		}
		if found && value != s.ManaReservePercent {
			return 95
		}
		value, found = s.ManaReservePercent, true
	}
	return value
}

func currentActorOwnerKeys(doc *sav.DocumentData) map[uint16]uint32 {
	keys := map[uint16]uint32{}
	for object, site := range currentActorGroupSites(doc) {
		key, _ := savedStructureValue(&doc.Objects[site.Player-1], "This")
		keys[object] = key
	}
	for i, r := range doc.Objects {
		if _, ok := keys[uint16(i+1)]; !ok {
			if key, err := savedStructureValue(&r, "Reference"); err == nil {
				keys[uint16(i+1)] = key
			}
		}
	}
	return keys
}

func captureCurrentManaReserves(doc *sav.DocumentData, state *SnapshotSAVDocument, w *sim.World, a *currentActionData) {
	owners, percents := currentActorOwnerKeys(doc), map[uint32]uint32{}
	for _, r := range doc.Objects {
		if r.Class == "Player" {
			key, _ := savedStructureValue(&r, "This")
			percent, _ := savedStructureValue(&r, "F58")
			percents[key] = percent
		}
	}
	objects := map[sim.EntityID]uint16{}
	for _, b := range state.Actors {
		if !b.Retired {
			objects[b.EntityID] = b.ObjectIndex
		}
	}
	for _, e := range w.Entities() {
		s := e.ActorLoad.Source
		key := owners[objects[e.ID]]
		if wire, ok := percents[key]; ok && s.Class != 0 && s.HasOwner && wire != s.ManaReservePercent {
			v := a.Values[e.ID]
			v.ManaReserve = &sim.ActorManaReserve{Wire: wire, Value: s.ManaReservePercent, OwnerKey: key}
			a.Values[e.ID] = v
		}
	}
}

func matchCurrentManaReserves(ms *Mission, a *currentActionData) {
	owners := currentActorOwnerKeys(ms.savedDocument.Document)
	for _, b := range ms.savedDocument.Actors {
		v := a.Values[b.EntityID]
		if v.ManaReserve != nil && owners[b.ObjectIndex] != v.ManaReserve.OwnerKey {
			v.ManaReserve = nil
			a.Values[b.EntityID] = v
		}
	}
}
