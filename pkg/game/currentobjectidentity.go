package game

import (
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
)

func validateCurrentObjectIdentityAbsence(row currentOwnedObject, legacy bool) error {
	if row.IdentityPresent != nil && (row.Kind < 1 || row.Kind > 4) {
		return fmt.Errorf("current identity absence has an unsupported object kind")
	}
	absent := row.IdentityPresent != nil && !*row.IdentityPresent
	if row.IdentityAnchor != nil && (!absent || row.Object == 0 || *row.IdentityAnchor == 0) {
		return fmt.Errorf("current identity absence has a conflicting anchor")
	}
	if absent && row.Object != 0 && row.IdentityAnchor == nil && !legacy {
		return fmt.Errorf("current identity absence lacks its ordinary anchor")
	}
	return nil
}

func restoreCurrentObjectIdentity(row currentOwnedObject, identity uint32) uint32 {
	if row.IdentityPresent != nil && !*row.IdentityPresent && (row.IdentityAnchor == nil || *row.IdentityAnchor == identity) {
		return 0
	}
	return identity
}

// An absent native key is anchored after ordinary keys are allocated. A later
// ordinary key edit becomes current state instead of replaying the absence.
func finalizeCurrentObjectIdentityAbsence(doc *sav.DocumentData) error {
	a, err := readCurrentActions(doc)
	if err != nil || a == nil {
		return err
	}
	for i := range a.Ownership {
		row := &a.Ownership[i]
		if err := captureNativeItemRecordAnchor(doc, row); err != nil {
			return err
		}
		row.IdentityAnchor = nil
		if row.IdentityPresent == nil || *row.IdentityPresent || row.Object == 0 {
			continue
		}
		if int(row.Object) > len(doc.Objects) || (row.Kind < 1 || row.Kind > 4) {
			return fmt.Errorf("current identity absence has no ordinary object")
		}
		r := &doc.Objects[row.Object-1]
		if row.Kind == 1 && !savedItemClass(r.Class) || row.Kind == 2 && r.Class != "Effect" || row.Kind == 3 && r.Class != "Spell" || row.Kind == 4 && r.Class != "Sack" {
			return fmt.Errorf("current identity absence has a different ordinary class")
		}
		field := "Identity"
		if row.Kind == 3 {
			field = "This"
		}
		identity, err := savedStructureValue(r, field)
		if err != nil || identity == 0 {
			return fmt.Errorf("current identity absence lacks a completed ordinary key")
		}
		row.IdentityAnchor = &identity
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return sav.SetNativeActions(&doc.State, raw)
}
