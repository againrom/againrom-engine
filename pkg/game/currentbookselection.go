package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
)

func matchCurrentBookSelection(doc *sav.DocumentData, actions *currentActionData) error {
	for i := range actions.Actions.Actors {
		row := &actions.Actions.Actors[i]
		if row.Current == nil || row.Current.AdmittedBookSpell == 0 {
			continue
		}
		var object uint16
		for _, binding := range actions.Bindings {
			if binding.ID == row.Entity && !binding.Structure && !binding.Missing {
				object = binding.Object
				break
			}
		}
		if object == 0 || int(object) > len(doc.Objects) {
			return fmt.Errorf("current book selection has no ordinary actor")
		}
		record := &doc.Objects[object-1]
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
