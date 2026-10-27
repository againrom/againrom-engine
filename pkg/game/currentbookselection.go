package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// currentBookSelectionRecord is the actor record of a row whose actor holds an
// admitted book selection, nil for a row without one.
func currentBookSelectionRecord(doc *sav.DocumentData, actions *currentActionData, row *sim.ActorContinuation) (*sav.DocumentRecordData, error) {
	if row.Current == nil || row.Current.AdmittedBookSpell == 0 {
		return nil, nil
	}
	var object uint16
	for _, binding := range actions.Bindings {
		if binding.ID == row.Entity && !binding.Structure && !binding.Missing {
			object = binding.Object
			break
		}
	}
	if object == 0 || int(object) > len(doc.Objects) {
		return nil, fmt.Errorf("current book selection has no ordinary actor")
	}
	return &doc.Objects[object-1], nil
}

// finalizeCurrentBookSelection runs after the document's keys are complete:
// a selection whose Spell record had no key when the cast wire wrote U44
// takes the completed key, the one LOAD matches against the book slot.
func finalizeCurrentBookSelection(doc *sav.DocumentData) error {
	actions, err := readCurrentActions(doc)
	if err != nil || actions == nil {
		return err
	}
	for i := range actions.Actions.Actors {
		row := &actions.Actions.Actors[i]
		record, err := currentBookSelectionRecord(doc, actions, row)
		if err != nil {
			return err
		}
		if record == nil {
			continue
		}
		if key, err := savedStructureValue(record, "U44"); err != nil {
			return err
		} else if key != 0 {
			continue
		}
		refs, _ := savedObjectRefs(record, "Spells")
		selected := row.Current.AdmittedBookSpell
		if int(selected) > len(refs) || refs[selected-1] == 0 || int(refs[selected-1]) > len(doc.Objects) {
			continue
		}
		key, err := savedStructureValue(&doc.Objects[refs[selected-1]-1], "This")
		if err != nil {
			return err
		}
		savedObjectSetValue(record, "U44", key)
	}
	return nil
}

func matchCurrentBookSelection(doc *sav.DocumentData, actions *currentActionData) error {
	for i := range actions.Actions.Actors {
		row := &actions.Actions.Actors[i]
		record, err := currentBookSelectionRecord(doc, actions, row)
		if err != nil {
			return err
		}
		if record == nil {
			continue
		}
		key, err := savedStructureValue(record, "U44")
		if err != nil {
			return err
		}
		refs, _ := savedObjectRefs(record, "Spells")
		selected := row.Current.AdmittedBookSpell
		if key != 0 && int(selected) <= len(refs) {
			ref := refs[selected-1]
			if ref != 0 && int(ref) <= len(doc.Objects) {
				value, err := savedStructureValue(&doc.Objects[ref-1], "This")
				if err != nil {
					return err
				}
				if value == key {
					continue
				}
			}
		}
		row.Current.AdmittedBookSpell = 0
		for slot, ref := range refs {
			if key == 0 || ref == 0 || int(ref) > len(doc.Objects) {
				continue
			}
			value, err := savedStructureValue(&doc.Objects[ref-1], "This")
			if err != nil {
				return err
			}
			if value == key {
				if row.Current.AdmittedBookSpell != 0 {
					return fmt.Errorf("ordinary book selection has ambiguous sparse slots")
				}
				row.Current.AdmittedBookSpell = uint16(slot + 1)
			}
		}
	}
	return nil
}
