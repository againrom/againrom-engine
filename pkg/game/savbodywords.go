package game

import (
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// A body loaded on the dead list and in no Group is never on the tick list
// (SAV-1171), so the original writes its action-run end as LOAD read it. The
// World holds no carrier for a body's run end (DIV-4059).
var loadedBodyWords = []string{"U138"}

// A body that is no entity, a terminal record or a record no map unit binds,
// has no World carrier for its order phase and completion, attack charge,
// health hundredths and credit spell either; they also stay as loaded.
var loadedTerminalWords = []string{"U58", "U136", "U138", "U134", "UA2", "U48"}

// recordBodyWords reads a body's attack charge, health hundredths and credit
// spell from its record. A body joined to an original dead record carries its
// order words there, so the row's References stay empty.
func recordBodyWords(r *sav.DocumentRecordData, id sim.EntityID) sim.BodyReferenceRecord {
	value := func(name string) uint32 { v, _ := savedStructureValue(r, name); return v }
	return sim.BodyReferenceRecord{ID: id, Words: &sim.BodyWords{AttackCharge: uint8(value("U134")), HealthHundredths: uint8(value("UA2")), CreditSpell: int8(value("U48"))}}
}

// restoreBodyWords gives each body the World holds as an entity the words of
// its loaded record. A health remainder above 99 is not a legal World value;
// that body keeps its constructor words (DIV-4059).
func restoreBodyWords(w *sim.World, rows []sim.BodyReferenceRecord) error {
	kept := rows[:0:0]
	for _, row := range rows {
		if e, ok := w.Entity(row.ID); ok && !e.Alive() && !e.Dying() && row.Words.HealthHundredths <= 99 {
			kept = append(kept, row)
		}
	}
	return w.RestoreBodyReferences(kept)
}

// keepLoadedBodyWords writes names into a body's record from the loaded
// document's record of the same Identity and class, when that record is a
// DeadActors root in no Group. Without such a record the words stay as the
// writer built them.
func keepLoadedBodyWords(r *sav.DocumentRecordData, loaded *SnapshotSAVDocument, names []string) error {
	source := loadedDeadRoot(r, loaded)
	if source == nil {
		return nil
	}
	for _, name := range names {
		if name == "U58" {
			raw, err := savedMotionRaw(source, name, 4)
			if err != nil {
				return err
			}
			mustSetRaw(r, name, slices.Clone(raw))
			continue
		}
		value, err := savedStructureValue(source, name)
		if err != nil {
			return err
		}
		savedObjectSetValue(r, name, value)
	}
	return nil
}

// loadedBodyReference is the reference word name of the loaded record with
// r's Identity when that word names a DeadActors root of the current document
// whose body is no World entity: the World holds no reference to such a body,
// and LOAD keeps a reference to any registered object (SAV-908). Otherwise it
// is 0 (DIV-4059).
func loadedBodyReference(state, loaded *SnapshotSAVDocument, r *sav.DocumentRecordData, name string) uint32 {
	if loaded == nil || loaded.Document == nil {
		return 0
	}
	key, err := savedStructureValue(r, "Identity")
	if err != nil || key == 0 {
		return 0
	}
	var source *sav.DocumentRecordData
	for i := range loaded.Document.Objects {
		o := &loaded.Document.Objects[i]
		if id, err := savedStructureValue(o, "Identity"); err != nil || id != key || o.Class != r.Class {
			continue
		}
		if source != nil {
			return 0
		}
		source = o
	}
	if source == nil {
		return 0
	}
	target, err := savedStructureValue(source, name)
	if err != nil || target == 0 {
		return 0
	}
	entity := map[uint16]bool{}
	for _, a := range state.Actors {
		if !a.Retired {
			entity[a.ObjectIndex] = true
		}
	}
	found := false
	for _, root := range state.Document.DeadActors {
		if root == 0 || int(root) > len(state.Document.Objects) {
			continue
		}
		if id, err := savedStructureValue(&state.Document.Objects[root-1], "Identity"); err == nil && id == target {
			if found || entity[root] {
				return 0
			}
			found = true
		}
	}
	if !found {
		return 0
	}
	return target
}

// loadedDeadRoot is the one loaded DeadActors root in no Group with the
// record's Identity and class, or nil.
func loadedDeadRoot(r *sav.DocumentRecordData, loaded *SnapshotSAVDocument) *sav.DocumentRecordData {
	if loaded == nil || loaded.Document == nil {
		return nil
	}
	key, err := savedStructureValue(r, "Identity")
	if err != nil || key == 0 {
		return nil
	}
	doc := loaded.Document
	var found uint16
	for _, root := range doc.DeadActors {
		if root == 0 || int(root) > len(doc.Objects) {
			continue
		}
		source := &doc.Objects[root-1]
		if id, err := savedStructureValue(source, "Identity"); err != nil || id != key || source.Class != r.Class {
			continue
		}
		if found != 0 && found != root {
			return nil
		}
		found = root
	}
	if found == 0 || savedDocumentIncoming(doc)[found] != 1 {
		return nil
	}
	return &doc.Objects[found-1]
}
