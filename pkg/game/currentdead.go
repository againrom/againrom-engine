package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentActorRecordMatches(w *sim.World, e sim.Entity, record sav.DocumentRecordData) bool {
	if currentActorClassMatches(e, record.Class) {
		return true
	}
	if e.SourceBinding.Class != 0 || e.ActorLoad.Source.Class != 0 {
		return false
	}
	key, err := savedStructureValue(&record, "Identity")
	if err != nil || key == 0 {
		return false
	}
	for _, body := range w.OriginalDeadActors() {
		if body.ID == e.ID && body.Source.Identity == key && savedActorClass(body.Source.Class) == record.Class {
			return true
		}
	}
	return false
}

func captureCurrentDead(doc *sav.DocumentData, state *SnapshotSAVDocument, w *sim.World, a *currentActionData) error {
	entities := make(map[sim.EntityID]sim.Entity)
	for _, e := range w.Entities() {
		entities[e.ID] = e
	}
	for _, body := range w.OriginalDeadActors() {
		e, live := entities[body.ID]
		if !live {
			continue
		}
		var record *sav.DocumentRecordData
		for _, b := range state.Actors {
			if b.EntityID == body.ID && !b.Retired && b.ObjectIndex > 0 && int(b.ObjectIndex) <= len(doc.Objects) && slices.Contains(doc.DeadActors, b.ObjectIndex) {
				record = &doc.Objects[b.ObjectIndex-1]
			}
		}
		if record == nil || !currentActorRecordMatches(w, e, *record) {
			return fmt.Errorf("current retained actor lacks an exact ordinary body")
		}
		v := a.Values[body.ID]
		v.RetainedDead = true
		owner, err := savedStructureValue(record, "Reference")
		if err != nil {
			return err
		}
		if body.Source.OwnerKey == 0 && owner != 0 {
			v.RetainedOwnerAbsent = &owner
		}
		credit, err := savedStructureValue(record, "U40")
		if err != nil {
			return err
		}
		if !e.HasKillCredit && credit != 0 {
			v.RetainedCreditAbsent = &credit
		}
		if !currentActorClassMatches(e, record.Class) {
			v.ClassAbsent = &sim.ActorClassAbsence{Wire: body.Source.Class, Humanoid: e.Humanoid}
		}
		if record.Class != "Unit" {
			v.UnitXP = nil
		}
		a.Values[body.ID] = v
	}
	return nil
}

func projectCurrentDeadOwner(doc *sav.DocumentData, record *sav.DocumentRecordData, w *sim.World, e sim.Entity) error {
	if e.SourceBinding.Class != 0 {
		return nil
	}
	for _, body := range w.OriginalDeadActors() {
		if body.ID != e.ID || body.Source.OwnerKey != 0 || e.Owner == 0 {
			continue
		}
		var key uint32
		for _, index := range doc.Players {
			if index == 0 {
				continue
			}
			player := &doc.Objects[index-1]
			slot, _ := savedStructureValue(player, "Slot")
			if uint32(uint16(slot)) != e.Owner {
				continue
			}
			value, _ := savedStructureValue(player, "This")
			if value == 0 || key != 0 && key != value {
				return fmt.Errorf("current retained actor lacks an exact ordinary owner")
			}
			key = value
		}
		if key == 0 {
			return fmt.Errorf("current retained actor lacks its Player")
		}
		return savedActorSetValue(record, "Reference", key)
	}
	return nil
}

func currentRetainedDeadKeys(state *SnapshotSAVDocument) (map[uint32]bool, error) {
	out := map[uint32]bool{}
	if state == nil || state.Document == nil {
		return out, nil
	}
	a, err := readCurrentActions(state.Document)
	if err != nil || a == nil {
		return out, err
	}
	for id, v := range a.Values {
		if !v.RetainedDead {
			continue
		}
		var object uint16
		for _, b := range a.Bindings {
			if b.ID != id || b.Structure {
				continue
			}
			if object != 0 || b.Missing || b.Object == 0 || int(b.Object) > len(state.Document.Objects) {
				return nil, fmt.Errorf("current retained actor has an invalid object binding")
			}
			object = b.Object
		}
		if object == 0 || !slices.Contains(state.Document.DeadActors, object) {
			return nil, fmt.Errorf("current retained actor has no ordinary dead root")
		}
		key, err := savedStructureValue(&state.Document.Objects[object-1], "Identity")
		if err != nil || key == 0 || out[key] {
			return nil, fmt.Errorf("current retained actor has an invalid identity")
		}
		out[key] = true
	}
	return out, nil
}

func currentAbsentDeadClasses(state *SnapshotSAVDocument) (map[uint32]uint8, error) {
	out := map[uint32]uint8{}
	if state == nil || state.Document == nil {
		return out, nil
	}
	a, err := readCurrentActions(state.Document)
	if err != nil || a == nil {
		return out, err
	}
	for _, b := range a.Bindings {
		v := a.Values[b.ID]
		if b.Structure || b.Missing || v.ClassAbsent == nil || !v.RetainedDead || v.SourceBound || v.SourceClass != 0 || v.ClassAbsent.Humanoid || b.Object == 0 || int(b.Object) > len(state.Document.Objects) {
			continue
		}
		r := &state.Document.Objects[b.Object-1]
		if savedActorClass(v.ClassAbsent.Wire) != r.Class {
			continue
		}
		key, err := savedStructureValue(r, "Identity")
		if err != nil {
			return nil, err
		}
		out[key] = v.ClassAbsent.Wire
	}
	return out, nil
}
